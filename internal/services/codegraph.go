package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/database"
	"github.com/NexVed/Cortex/internal/scanner"
)

// CodeGraph is the stable API contract consumed by the SolidJS code-graph view.
type CodeGraph struct {
	ProjectID   string      `json:"project_id"`
	ProjectName string      `json:"project_name"`
	Nodes       []GraphNode `json:"nodes"`
	Edges       []GraphEdge `json:"edges"`
	Stats       GraphStats  `json:"stats"`
	GeneratedAt string      `json:"generated_at"`
	Built       bool        `json:"built"`
}

type GraphNode struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	Path   string `json:"path,omitempty"`
	Lang   string `json:"lang,omitempty"`
	Line   int    `json:"line,omitempty"`
	Public bool   `json:"public,omitempty"`
	Degree int    `json:"degree"`
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Rel    string `json:"rel"`
}

type GraphStats struct {
	Dirs         int `json:"dirs"`
	Files        int `json:"files"`
	Functions    int `json:"functions"`
	Classes      int `json:"classes"`
	Packages     int `json:"packages"`
	Edges        int `json:"edges"`
	InternalDeps int `json:"internal_deps"`
	ExternalDeps int `json:"external_deps"`
	Orphans      int `json:"orphans"`
	Cycles       int `json:"cycles"`
	MaxDegree    int `json:"max_degree"`
}

type sourceFile struct {
	Path string
	Lang string
}

// CodeGraphService owns source analysis and its persisted SQLite snapshot. It
// deliberately depends on the scanner's parser but not on the HTTP layer.
type CodeGraphService struct{ DB *database.DB }

func (s CodeGraphService) Load(projectID string) (*CodeGraph, bool, error) {
	raw, ok, err := s.DB.LoadCodeGraph(projectID)
	if err != nil || !ok {
		return nil, ok, err
	}
	var graph CodeGraph
	if err := json.Unmarshal(raw, &graph); err != nil {
		return nil, false, fmt.Errorf("decode stored code graph: %w", err)
	}
	return &graph, true, nil
}

// Build parses the scanned working tree into files, symbols, package manifests,
// and resolved local/external dependencies, then saves the resulting snapshot.
func (s CodeGraphService) Build(projectID string) (*CodeGraph, error) {
	repoPath, err := s.DB.RepositoryPath(projectID)
	if err != nil {
		return nil, fmt.Errorf("project must be scanned before building its graph: %w", err)
	}
	projectName := projectID
	if repo, repoErr := s.DB.Repository(projectID); repoErr == nil && repo.Name != "" {
		projectName = repo.Name
	}
	page, err := (database.RecordStore{DB: s.DB}).List("file_index", projectID, 10000)
	if err != nil {
		return nil, err
	}

	files := make(map[string]sourceFile, len(page.Items))
	for _, item := range page.Items {
		filePath, _ := item["path"].(string)
		language, _ := item["language"].(string)
		filePath = cleanRepoPath(filePath)
		if filePath != "" && language != "" {
			files[filePath] = sourceFile{Path: filePath, Lang: language}
		}
	}
	graph := newCodeGraph(projectID, projectName)
	graph.Built = len(files) > 0
	if len(files) == 0 {
		graph.finalize()
		return s.save(&graph.CodeGraph)
	}

	filePaths := make([]string, 0, len(files))
	for filePath := range files {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)
	goModule := readGoModule(repoPath)
	manifestPackages := readManifestPackages(repoPath)
	for _, packageName := range manifestPackages {
		graph.addPackage(packageName)
	}

	for _, filePath := range filePaths {
		file := files[filePath]
		fileID := "file:" + filePath
		graph.addFileTree(file)
		content, readErr := os.ReadFile(path.Join(repoPath, filepathFromSlash(filePath)))
		if readErr != nil {
			continue
		}
		symbols := scanner.ExtractSymbols(string(content), file.Lang)
		for _, fn := range symbols.Functions {
			id := fmt.Sprintf("function:%s:%s:%d", filePath, fn.Name, fn.Line)
			graph.addNode(GraphNode{ID: id, Label: fn.Name, Type: "function", Path: filePath, Lang: file.Lang, Line: fn.Line, Public: fn.IsPublic})
			graph.link(fileID, id, "defines")
		}
		for _, class := range symbols.Classes {
			id := fmt.Sprintf("class:%s:%s:%d", filePath, class.Name, class.Line)
			graph.addNode(GraphNode{ID: id, Label: class.Name, Type: "class", Path: filePath, Lang: file.Lang, Line: class.Line})
			graph.link(fileID, id, "defines")
		}
		for _, imported := range uniqueStrings(symbols.Imports) {
			if target := resolveLocalImport(filePath, imported, file.Lang, goModule, files); target != "" {
				graph.link(fileID, "file:"+target, "imports")
				graph.Stats.InternalDeps++
				continue
			}
			name := externalPackageName(imported, file.Lang)
			if name == "" || isStandardLibrary(imported, file.Lang) {
				continue
			}
			packageID := graph.addPackage(name)
			if graph.link(fileID, packageID, "depends_on") {
				graph.Stats.ExternalDeps++
			}
		}
	}
	graph.finalize()
	return s.save(&graph.CodeGraph)
}

func (s CodeGraphService) save(graph *CodeGraph) (*CodeGraph, error) {
	graph.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.Marshal(graph)
	if err != nil {
		return nil, err
	}
	if err := s.DB.SaveCodeGraph(graph.ProjectID, raw); err != nil {
		return nil, err
	}
	return graph, nil
}

type graphBuilder struct {
	CodeGraph
	nodes map[string]*GraphNode
	seen  map[string]bool
}

func newCodeGraph(projectID, projectName string) *graphBuilder {
	root := &GraphNode{ID: "dir:.", Label: projectName, Type: "dir", Path: "."}
	return &graphBuilder{CodeGraph: CodeGraph{ProjectID: projectID, ProjectName: projectName, Nodes: []GraphNode{}, Edges: []GraphEdge{}, Stats: GraphStats{}}, nodes: map[string]*GraphNode{root.ID: root}, seen: map[string]bool{}}
}

func (g *graphBuilder) addNode(node GraphNode) string {
	if _, exists := g.nodes[node.ID]; !exists {
		g.nodes[node.ID] = &node
	}
	return node.ID
}

func (g *graphBuilder) addFileTree(file sourceFile) {
	parent := "dir:."
	segments := strings.Split(file.Path, "/")
	current := ""
	for _, segment := range segments[:len(segments)-1] {
		if current == "" {
			current = segment
		} else {
			current += "/" + segment
		}
		id := "dir:" + current
		g.addNode(GraphNode{ID: id, Label: segment, Type: "dir", Path: current})
		g.link(parent, id, "contains")
		parent = id
	}
	fileID := "file:" + file.Path
	g.addNode(GraphNode{ID: fileID, Label: path.Base(file.Path), Type: "file", Path: file.Path, Lang: file.Lang})
	g.link(parent, fileID, "contains")
}

func (g *graphBuilder) addPackage(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return g.addNode(GraphNode{ID: "package:" + name, Label: name, Type: "package", Path: name})
}

func (g *graphBuilder) link(source, target, rel string) bool {
	if source == "" || target == "" || g.nodes[source] == nil || g.nodes[target] == nil {
		return false
	}
	key := source + "\x00" + target + "\x00" + rel
	if g.seen[key] {
		return false
	}
	g.seen[key] = true
	g.Edges = append(g.Edges, GraphEdge{Source: source, Target: target, Rel: rel})
	g.nodes[source].Degree++
	g.nodes[target].Degree++
	return true
}

func (g *graphBuilder) finalize() {
	ids := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	g.Nodes = make([]GraphNode, 0, len(ids))
	for _, id := range ids {
		node := *g.nodes[id]
		g.Nodes = append(g.Nodes, node)
		switch node.Type {
		case "dir":
			g.Stats.Dirs++
		case "file":
			g.Stats.Files++
		case "function":
			g.Stats.Functions++
		case "class":
			g.Stats.Classes++
		case "package":
			g.Stats.Packages++
		}
		if node.Degree == 0 {
			g.Stats.Orphans++
		}
		if node.Degree > g.Stats.MaxDegree {
			g.Stats.MaxDegree = node.Degree
		}
	}
	g.Stats.Edges = len(g.Edges)
	g.Stats.Cycles = countImportCycles(g.Edges)
}

func cleanRepoPath(value string) string {
	value = strings.TrimPrefix(strings.ReplaceAll(value, "\\", "/"), "/")
	if value == "" || value == "." || strings.HasPrefix(value, "../") {
		return ""
	}
	return path.Clean(value)
}

func filepathFromSlash(value string) string {
	return strings.ReplaceAll(value, "/", string(os.PathSeparator))
}

func resolveLocalImport(sourcePath, imported, language, goModule string, files map[string]sourceFile) string {
	imported = strings.TrimSpace(imported)
	if imported == "" {
		return ""
	}
	if language == "TypeScript" || language == "JavaScript" {
		if !strings.HasPrefix(imported, ".") {
			return ""
		}
		base := path.Clean(path.Join(path.Dir(sourcePath), imported))
		candidates := []string{base, base + ".ts", base + ".tsx", base + ".js", base + ".jsx", path.Join(base, "index.ts"), path.Join(base, "index.tsx"), path.Join(base, "index.js"), path.Join(base, "index.jsx")}
		for _, candidate := range candidates {
			if _, exists := files[candidate]; exists {
				return candidate
			}
		}
	}
	if language == "Go" && goModule != "" && (imported == goModule || strings.HasPrefix(imported, goModule+"/")) {
		dir := strings.TrimPrefix(strings.TrimPrefix(imported, goModule), "/")
		for candidate, file := range files {
			if file.Lang == "Go" && path.Dir(candidate) == dir {
				return candidate
			}
		}
	}
	if language == "Python" && strings.HasPrefix(imported, ".") {
		base := strings.TrimPrefix(imported, ".")
		candidate := path.Join(path.Dir(sourcePath), strings.ReplaceAll(base, ".", "/")) + ".py"
		if _, exists := files[candidate]; exists {
			return candidate
		}
	}
	return ""
}

func readGoModule(repoPath string) string {
	content, err := os.ReadFile(path.Join(repoPath, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

func readManifestPackages(repoPath string) []string {
	packages := map[string]bool{}
	add := func(name string) {
		if name != "" {
			packages[name] = true
		}
	}
	if raw, err := os.ReadFile(path.Join(repoPath, "package.json")); err == nil {
		var manifest struct {
			Dependencies         map[string]any `json:"dependencies"`
			DevDependencies      map[string]any `json:"devDependencies"`
			PeerDependencies     map[string]any `json:"peerDependencies"`
			OptionalDependencies map[string]any `json:"optionalDependencies"`
		}
		if json.Unmarshal(raw, &manifest) == nil {
			for _, group := range []map[string]any{manifest.Dependencies, manifest.DevDependencies, manifest.PeerDependencies, manifest.OptionalDependencies} {
				for name := range group {
					add(name)
				}
			}
		}
	}
	if raw, err := os.ReadFile(path.Join(repoPath, "go.mod")); err == nil {
		insideRequire := false
		for _, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "require (") {
				insideRequire = true
				continue
			}
			if insideRequire && trimmed == ")" {
				insideRequire = false
				continue
			}
			if strings.HasPrefix(trimmed, "require ") {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "require "))
			}
			if insideRequire || strings.Contains(trimmed, " v") {
				fields := strings.Fields(trimmed)
				if len(fields) >= 2 && strings.HasPrefix(fields[1], "v") {
					add(fields[0])
				}
			}
		}
	}
	if raw, err := os.ReadFile(path.Join(repoPath, "requirements.txt")); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			name := strings.TrimSpace(strings.Split(line, "#")[0])
			name = strings.SplitN(name, "=", 2)[0]
			name = strings.SplitN(name, ">", 2)[0]
			name = strings.SplitN(name, "<", 2)[0]
			add(strings.TrimSpace(name))
		}
	}
	out := make([]string, 0, len(packages))
	for name := range packages {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func externalPackageName(imported, language string) string {
	imported = strings.Trim(strings.TrimSpace(imported), "\"'")
	if imported == "" || strings.HasPrefix(imported, ".") {
		return ""
	}
	if language == "Go" {
		return imported
	}
	if strings.HasPrefix(imported, "@") {
		parts := strings.Split(imported, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return strings.Split(imported, "/")[0]
}

func isStandardLibrary(imported, language string) bool {
	if language == "Go" {
		return !strings.Contains(imported, ".")
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func countImportCycles(edges []GraphEdge) int {
	adj := map[string][]string{}
	for _, edge := range edges {
		if edge.Rel == "imports" {
			adj[edge.Source] = append(adj[edge.Source], edge.Target)
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	cycles := 0
	var visit func(string)
	visit = func(node string) {
		if visiting[node] {
			cycles++
			return
		}
		if visited[node] {
			return
		}
		visiting[node] = true
		for _, next := range adj[node] {
			visit(next)
		}
		visiting[node] = false
		visited[node] = true
	}
	for node := range adj {
		visit(node)
	}
	return cycles
}

var _ = regexp.MustCompile
