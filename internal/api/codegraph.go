package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/analyzer"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// CodeGraphResult is the API/MCP representation of a project's code graph.
type CodeGraphResult struct {
	ProjectID   string                    `json:"project_id"`
	ProjectName string                    `json:"project_name"`
	Nodes       []analyzer.CodeNode       `json:"nodes"`
	Edges       []analyzer.CodeEdge       `json:"edges"`
	Stats       analyzer.CodeGraphStats   `json:"stats"`
	GeneratedAt string                    `json:"generated_at"`
	Built       bool                      `json:"built"`
}

type fnRow struct {
	Name     string `json:"name"`
	Line     int    `json:"line"`
	IsPublic bool   `json:"is_public"`
}

type clsRow struct {
	Name string `json:"name"`
	Line int    `json:"line"`
}

// BuildAndStoreCodeGraph assembles the code graph from the project's indexed
// files and persists it as the project's codebase memory.
func (s *Service) BuildAndStoreCodeGraph(ctx context.Context, user *core.Record, projectID string) (*CodeGraphResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	rows, err := s.App.FindRecordsByFilter(db.CollFileIndex, "project = {:p}", "path", 10000, 0,
		map[string]any{"p": projectID})
	if err != nil {
		return nil, fmt.Errorf("failed to load file index: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("project has no indexed files — run Scan first")
	}

	files := make([]analyzer.FileEntry, 0, len(rows))
	for _, r := range rows {
		var fns []fnRow
		var classes []clsRow
		var imports []string
		_ = r.UnmarshalJSONField("functions", &fns)
		_ = r.UnmarshalJSONField("classes", &classes)
		_ = r.UnmarshalJSONField("imports", &imports)

		fe := analyzer.FileEntry{
			Path:     r.GetString("path"),
			Language: r.GetString("language"),
			Imports:  imports,
		}
		for _, f := range fns {
			fe.Functions = append(fe.Functions, analyzer.SymbolRef{Name: f.Name, Line: f.Line, Public: f.IsPublic})
		}
		for _, c := range classes {
			fe.Classes = append(fe.Classes, analyzer.SymbolRef{Name: c.Name, Line: c.Line, Public: true})
		}
		files = append(files, fe)
	}

	graph := analyzer.BuildCodeGraph(files)
	generatedAt := time.Now().UTC()

	if err := s.persistCodeGraph(project, graph, generatedAt); err != nil {
		return nil, err
	}
	uid := ""
	if user != nil {
		uid = user.Id
	}
	db.LogActivity(s.App, projectID, uid, "code_graph_built",
		fmt.Sprintf("%d files, %d symbols, %d edges", graph.Stats.Files,
			graph.Stats.Functions+graph.Stats.Classes, graph.Stats.Edges), nil)

	return &CodeGraphResult{
		ProjectID:   projectID,
		ProjectName: project.GetString("name"),
		Nodes:       graph.Nodes,
		Edges:       graph.Edges,
		Stats:       graph.Stats,
		GeneratedAt: generatedAt.Format(time.RFC3339),
		Built:       true,
	}, nil
}

// GetCodeGraph returns the stored code graph, or an empty (not-built) result.
func (s *Service) GetCodeGraph(projectID string) (*CodeGraphResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	rec, err := s.App.FindFirstRecordByFilter(db.CollCodeGraphs, "project = {:p}",
		map[string]any{"p": projectID})
	if err != nil || rec == nil {
		return &CodeGraphResult{ProjectID: projectID, ProjectName: project.GetString("name"), Built: false}, nil
	}
	graph := codeGraphFromRecord(rec)
	return &CodeGraphResult{
		ProjectID:   projectID,
		ProjectName: project.GetString("name"),
		Nodes:       graph.Nodes,
		Edges:       graph.Edges,
		Stats:       graph.Stats,
		GeneratedAt: rfc3339(rec, "generated_at"),
		Built:       true,
	}, nil
}

func (s *Service) persistCodeGraph(project *core.Record, graph analyzer.CodeGraph, generatedAt time.Time) error {
	existing, _ := s.App.FindFirstRecordByFilter(db.CollCodeGraphs, "project = {:p}",
		map[string]any{"p": project.Id})
	var rec *core.Record
	if existing != nil {
		rec = existing
	} else {
		coll, err := s.App.FindCollectionByNameOrId(db.CollCodeGraphs)
		if err != nil {
			return err
		}
		rec = core.NewRecord(coll)
		rec.Set("project", project.Id)
	}
	rec.Set("nodes", graph.Nodes)
	rec.Set("edges", graph.Edges)
	rec.Set("stats", graph.Stats)
	rec.Set("node_count", len(graph.Nodes))
	rec.Set("edge_count", len(graph.Edges))
	rec.Set("generated_at", types.NowDateTime())
	return s.App.Save(rec)
}

func codeGraphFromRecord(rec *core.Record) analyzer.CodeGraph {
	var g analyzer.CodeGraph
	_ = rec.UnmarshalJSONField("nodes", &g.Nodes)
	_ = rec.UnmarshalJSONField("edges", &g.Edges)
	_ = rec.UnmarshalJSONField("stats", &g.Stats)
	return g
}

// ── Recall (agent-facing memory queries) ───────────────

// QueryCodeGraph searches the stored graph for a symbol/file and returns a
// compact, human/agent-readable description of it and its neighbours.
func (s *Service) QueryCodeGraph(projectID, query string) (string, error) {
	rec, err := s.App.FindFirstRecordByFilter(db.CollCodeGraphs, "project = {:p}",
		map[string]any{"p": projectID})
	if err != nil || rec == nil {
		return "", fmt.Errorf("no code graph built yet — build it first")
	}
	g := codeGraphFromRecord(rec)
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return "", fmt.Errorf("empty query")
	}

	byID := make(map[string]analyzer.CodeNode, len(g.Nodes))
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	// outgoing/incoming adjacency
	out := map[string][]analyzer.CodeEdge{}
	in := map[string][]analyzer.CodeEdge{}
	for _, e := range g.Edges {
		out[e.Source] = append(out[e.Source], e)
		in[e.Target] = append(in[e.Target], e)
	}

	// Rank matches: exact label first, then substring; prefer symbols/files.
	type match struct {
		n     analyzer.CodeNode
		score int
	}
	var matches []match
	for _, n := range g.Nodes {
		label := strings.ToLower(n.Label)
		score := 0
		if label == q {
			score = 100
		} else if strings.Contains(label, q) {
			score = 50
		} else if strings.Contains(strings.ToLower(n.Path), q) {
			score = 20
		}
		if score == 0 {
			continue
		}
		switch n.Type {
		case "function", "class":
			score += 5
		case "file":
			score += 3
		}
		matches = append(matches, match{n, score})
	}
	if len(matches) == 0 {
		return fmt.Sprintf("No code entity matching %q was found in the graph.", query), nil
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	if len(matches) > 5 {
		matches = matches[:5]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Code graph matches for %q:\n\n", query))
	for _, m := range matches {
		n := m.n
		switch n.Type {
		case "function", "class":
			b.WriteString(fmt.Sprintf("• %s %s", n.Type, n.Label))
			if n.Path != "" {
				b.WriteString(fmt.Sprintf(" — defined in %s:%d", n.Path, n.Line))
			}
			b.WriteString("\n")
			// The file that defines it and that file's dependencies.
			for _, e := range in[n.ID] {
				if e.Rel == "defines" {
					describeFileDeps(&b, byID[e.Source], out, in, byID)
				}
			}
		case "file":
			b.WriteString(fmt.Sprintf("• file %s\n", n.Path))
			describeFileDeps(&b, n, out, in, byID)
		case "dir":
			files := 0
			for _, e := range out[n.ID] {
				if e.Rel == "contains" && strings.HasPrefix(e.Target, "file:") {
					files++
				}
			}
			b.WriteString(fmt.Sprintf("• directory %s — contains %d files\n", n.Path, files))
		case "package":
			importers := 0
			for _, e := range in[n.ID] {
				if e.Rel == "imports" {
					importers++
				}
			}
			b.WriteString(fmt.Sprintf("• package %s — imported by %d files\n", n.Label, importers))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()), nil
}

func describeFileDeps(b *strings.Builder, file analyzer.CodeNode, out, in map[string][]analyzer.CodeEdge, byID map[string]analyzer.CodeNode) {
	if file.ID == "" {
		return
	}
	var defines, deps, importers, dependents []string
	for _, e := range out[file.ID] {
		switch e.Rel {
		case "defines":
			if t, ok := byID[e.Target]; ok {
				defines = append(defines, t.Label)
			}
		case "depends_on":
			if t, ok := byID[e.Target]; ok {
				deps = append(deps, t.Path)
			}
		case "imports":
			if t, ok := byID[e.Target]; ok {
				importers = append(importers, t.Label)
			}
		}
	}
	for _, e := range in[file.ID] {
		if e.Rel == "depends_on" {
			if src, ok := byID[e.Source]; ok {
				dependents = append(dependents, src.Path)
			}
		}
	}
	if len(defines) > 0 {
		b.WriteString("    defines: " + joinCap(defines, 12) + "\n")
	}
	if len(deps) > 0 {
		b.WriteString("    depends on: " + joinCap(deps, 10) + "\n")
	}
	if len(importers) > 0 {
		b.WriteString("    imports pkgs: " + joinCap(importers, 10) + "\n")
	}
	if len(dependents) > 0 {
		b.WriteString("    used by: " + joinCap(dependents, 10) + "\n")
	}
}

// CodeMap returns a compact overview of the project's structure from the graph.
func (s *Service) CodeMap(projectID string) (string, error) {
	rec, err := s.App.FindFirstRecordByFilter(db.CollCodeGraphs, "project = {:p}",
		map[string]any{"p": projectID})
	if err != nil || rec == nil {
		return "", fmt.Errorf("no code graph built yet — build it first")
	}
	g := codeGraphFromRecord(rec)

	byID := make(map[string]analyzer.CodeNode, len(g.Nodes))
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	// Count files per directory.
	fileCountByDir := map[string]int{}
	filesByDir := map[string][]string{}
	for _, e := range g.Edges {
		if e.Rel == "contains" && strings.HasPrefix(e.Target, "file:") {
			d := byID[e.Source]
			fileCountByDir[d.ID]++
			if f, ok := byID[e.Target]; ok && len(filesByDir[d.ID]) < 6 {
				filesByDir[d.ID] = append(filesByDir[d.ID], f.Label)
			}
		}
	}
	type dirCount struct {
		node  analyzer.CodeNode
		count int
	}
	var dirs []dirCount
	for id, c := range fileCountByDir {
		dirs = append(dirs, dirCount{byID[id], c})
	}
	sort.SliceStable(dirs, func(i, j int) bool { return dirs[i].count > dirs[j].count })

	var b strings.Builder
	st := g.Stats
	b.WriteString(fmt.Sprintf("Code map: %d dirs, %d files, %d functions, %d classes, %d external packages.\n",
		st.Dirs, st.Files, st.Functions, st.Classes, st.Packages))
	b.WriteString(fmt.Sprintf("Dependencies: %d internal, %d external.\n\nModules:\n", st.InternalDeps, st.ExternalDeps))
	limit := len(dirs)
	if limit > 15 {
		limit = 15
	}
	for _, d := range dirs[:limit] {
		b.WriteString(fmt.Sprintf("- %s/ (%d files): %s\n", d.node.Path, d.count, joinCap(filesByDir[d.node.ID], 6)))
	}
	return strings.TrimSpace(b.String()), nil
}

func joinCap(items []string, max int) string {
	more := 0
	if len(items) > max {
		more = len(items) - max
		items = items[:max]
	}
	s := strings.Join(items, ", ")
	if more > 0 {
		s += fmt.Sprintf(" (+%d more)", more)
	}
	return s
}
