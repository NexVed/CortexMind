package analyzer

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// ── Code graph types ───────────────────────────────────

// CodeNode is a vertex in the codebase memory graph.
type CodeNode struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"` // dir | file | function | class | package
	Path   string `json:"path,omitempty"`
	Lang   string `json:"lang,omitempty"`
	Line   int    `json:"line,omitempty"`
	Public bool   `json:"public,omitempty"`
	Degree int    `json:"degree"`
}

// CodeEdge is a directed relationship between two code nodes.
type CodeEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Rel    string `json:"rel"` // contains | defines | imports | depends_on
}

// CodeGraphStats summarizes the graph.
type CodeGraphStats struct {
	Dirs         int `json:"dirs"`
	Files        int `json:"files"`
	Functions    int `json:"functions"`
	Classes      int `json:"classes"`
	Packages     int `json:"packages"`
	Edges        int `json:"edges"`
	InternalDeps int `json:"internal_deps"` // file->file / file->dir dependencies
	ExternalDeps int `json:"external_deps"` // file->external package edges
	Orphans      int `json:"orphans"`       // nodes with no relationships
	Cycles       int `json:"cycles"`        // dependency back-edges
	MaxDegree    int `json:"max_degree"`    // highest node connectivity
}

// CodeGraph is the full codebase memory graph for a project.
type CodeGraph struct {
	Nodes []CodeNode     `json:"nodes"`
	Edges []CodeEdge     `json:"edges"`
	Stats CodeGraphStats `json:"stats"`
}

// FileEntry is the per-file input to the code graph builder (sourced from the
// file_index collection).
type FileEntry struct {
	Path      string
	Language  string
	Functions []SymbolRef
	Classes   []SymbolRef
	Imports   []string
}

// SymbolRef is a defined symbol (function/class) within a file.
type SymbolRef struct {
	Name   string
	Line   int
	Public bool
}

// ── Builder ────────────────────────────────────────────

type cgBuilder struct {
	nodes   []CodeNode
	edges   []CodeEdge
	nodeIdx map[string]int // id -> index into nodes
	edgeSet map[string]bool
	stats   CodeGraphStats
}

// BuildCodeGraph assembles a code structure + dependency graph from indexed
// files. It is pure and dependency-free: directories become container nodes,
// files hang off them, symbols hang off files, and imports are resolved to
// local files/dirs (depends_on) or grouped into external package nodes.
func BuildCodeGraph(files []FileEntry) CodeGraph {
	b := &cgBuilder{nodeIdx: map[string]int{}, edgeSet: map[string]bool{}}

	// Known files and directories (for import resolution).
	fileSet := map[string]bool{}
	dirSet := map[string]bool{}
	for _, f := range files {
		p := path.Clean(strings.TrimSpace(f.Path))
		if p == "" || p == "." {
			continue
		}
		fileSet[p] = true
		for d := path.Dir(p); d != "." && d != "/" && d != ""; d = path.Dir(d) {
			dirSet[d] = true
		}
	}

	for _, f := range files {
		fp := path.Clean(strings.TrimSpace(f.Path))
		if fp == "" || fp == "." {
			continue
		}
		fileID := "file:" + fp
		b.addNode(CodeNode{ID: fileID, Label: path.Base(fp), Type: "file", Path: fp, Lang: f.Language})
		b.stats.Files++

		// Directory chain: dir contains subdir ... contains file.
		b.linkDirChain(fp, fileID)

		// Defined symbols.
		for _, fn := range f.Functions {
			id := fmt.Sprintf("fn:%s#%s@%d", fp, fn.Name, fn.Line)
			b.addNode(CodeNode{ID: id, Label: fn.Name, Type: "function", Path: fp, Line: fn.Line, Public: fn.Public})
			b.addEdge(fileID, id, "defines")
			b.stats.Functions++
		}
		for _, cl := range f.Classes {
			id := fmt.Sprintf("cls:%s#%s@%d", fp, cl.Name, cl.Line)
			b.addNode(CodeNode{ID: id, Label: cl.Name, Type: "class", Path: fp, Line: cl.Line, Public: cl.Public})
			b.addEdge(fileID, id, "defines")
			b.stats.Classes++
		}

		// Imports -> depends_on (local) or imports (external package).
		seen := map[string]bool{}
		for _, imp := range f.Imports {
			imp = strings.TrimSpace(imp)
			if imp == "" || seen[imp] || isNoiseImport(imp) {
				continue
			}
			seen[imp] = true

			if strings.HasPrefix(imp, ".") {
				// Relative import — resolve to a local file.
				if target, ok := resolveRelative(fp, imp, fileSet); ok && target != fp {
					if b.addEdge(fileID, "file:"+target, "depends_on") {
						b.stats.InternalDeps++
					}
					continue
				}
			}
			if dir, ok := matchInternalDir(imp, dirSet); ok {
				if b.addEdge(fileID, "dir:"+dir, "depends_on") {
					b.stats.InternalDeps++
				}
				continue
			}
			// External package.
			pkg := externalPackageKey(imp, f.Language)
			pkgID := "pkg:" + pkg
			b.addNode(CodeNode{ID: pkgID, Label: packageLabel(pkg), Type: "package"})
			if b.addEdge(fileID, pkgID, "imports") {
				b.stats.ExternalDeps++
			}
		}
	}

	b.computeDegrees()
	b.stats.Cycles = b.countDependencyCycles()
	b.stats.Edges = len(b.edges)
	for _, n := range b.nodes {
		switch n.Type {
		case "dir":
			b.stats.Dirs++
		case "package":
			b.stats.Packages++
		}
	}

	// Deterministic ordering for stable output.
	sort.SliceStable(b.nodes, func(i, j int) bool { return b.nodes[i].ID < b.nodes[j].ID })
	sort.SliceStable(b.edges, func(i, j int) bool {
		if b.edges[i].Source != b.edges[j].Source {
			return b.edges[i].Source < b.edges[j].Source
		}
		return b.edges[i].Target < b.edges[j].Target
	})

	return CodeGraph{Nodes: b.nodes, Edges: b.edges, Stats: b.stats}
}

// linkDirChain ensures dir nodes exist for the file's path and links
// dir -> subdir -> ... -> file with contains edges.
func (b *cgBuilder) linkDirChain(filePath, fileID string) {
	dir := path.Dir(filePath)
	if dir == "." || dir == "" {
		return
	}
	// Build ordered ancestor list root..leaf.
	var chain []string
	for d := dir; d != "." && d != "/" && d != ""; d = path.Dir(d) {
		chain = append([]string{d}, chain...)
	}
	for i, d := range chain {
		b.addNode(CodeNode{ID: "dir:" + d, Label: path.Base(d), Type: "dir", Path: d})
		if i > 0 {
			b.addEdge("dir:"+chain[i-1], "dir:"+d, "contains")
		}
	}
	b.addEdge("dir:"+chain[len(chain)-1], fileID, "contains")
}

func (b *cgBuilder) addNode(n CodeNode) {
	if _, ok := b.nodeIdx[n.ID]; ok {
		return
	}
	b.nodeIdx[n.ID] = len(b.nodes)
	b.nodes = append(b.nodes, n)
}

// addEdge adds a unique directed edge. Returns true if newly added.
func (b *cgBuilder) addEdge(src, tgt, rel string) bool {
	key := src + "|" + tgt + "|" + rel
	if b.edgeSet[key] {
		return false
	}
	b.edgeSet[key] = true
	b.edges = append(b.edges, CodeEdge{Source: src, Target: tgt, Rel: rel})
	return true
}

func (b *cgBuilder) computeDegrees() {
	deg := map[string]int{}
	for _, e := range b.edges {
		deg[e.Source]++
		deg[e.Target]++
	}
	for i := range b.nodes {
		b.nodes[i].Degree = deg[b.nodes[i].ID]
		if b.nodes[i].Degree == 0 {
			b.stats.Orphans++
		}
		if b.nodes[i].Degree > b.stats.MaxDegree {
			b.stats.MaxDegree = b.nodes[i].Degree
		}
	}
}

func (b *cgBuilder) countDependencyCycles() int {
	adj := map[string][]string{}
	for _, e := range b.edges {
		if e.Rel == "depends_on" {
			adj[e.Source] = append(adj[e.Source], e.Target)
		}
	}
	color := map[string]uint8{}
	cycles := 0
	var visit func(string)
	visit = func(id string) {
		color[id] = 1
		for _, next := range adj[id] {
			switch color[next] {
			case 1:
				cycles++
			case 0:
				visit(next)
			}
		}
		color[id] = 2
	}
	for id := range adj {
		if color[id] == 0 {
			visit(id)
		}
	}
	return cycles
}

// ── import resolution helpers ──────────────────────────

// resolveRelative resolves a relative import against the importing file to a
// concrete local file, trying common extensions and index files.
func resolveRelative(fromPath, imp string, fileSet map[string]bool) (string, bool) {
	base := path.Clean(path.Join(path.Dir(fromPath), imp))
	candidates := []string{
		base,
		base + ".ts", base + ".tsx", base + ".js", base + ".jsx",
		base + ".go", base + ".py", base + ".rs",
		base + "/index.ts", base + "/index.tsx", base + "/index.js",
		base + "/mod.rs", base + "/__init__.py",
	}
	for _, c := range candidates {
		if fileSet[c] {
			return c, true
		}
	}
	return "", false
}

// matchInternalDir checks whether an import path corresponds to a known project
// directory by testing progressively shorter suffixes (longest match wins).
// This maps e.g. a Go module import ".../internal/db" to the "internal/db" dir.
func matchInternalDir(imp string, dirSet map[string]bool) (string, bool) {
	parts := strings.Split(strings.Trim(imp, "/"), "/")
	for i := 0; i < len(parts)-0; i++ {
		cand := strings.Join(parts[i:], "/")
		if cand != "" && dirSet[cand] {
			return cand, true
		}
	}
	return "", false
}

// packageLabel shortens a long import path for display (keeps last 2 segments).
func externalPackageKey(imp, language string) string {
	imp = strings.Trim(imp, "/")
	parts := strings.Split(imp, "/")
	if len(parts) == 0 {
		return imp
	}
	switch language {
	case "TypeScript", "JavaScript":
		if strings.HasPrefix(imp, "@") && len(parts) >= 2 {
			return strings.Join(parts[:2], "/")
		}
		return parts[0]
	case "Go":
		if len(parts) >= 3 && (parts[0] == "github.com" || parts[0] == "gitlab.com") {
			return strings.Join(parts[:3], "/")
		}
	}
	return parts[0]
}

func packageLabel(imp string) string {
	imp = strings.Trim(imp, "/")
	parts := strings.Split(imp, "/")
	if len(parts) <= 2 {
		return imp
	}
	return ".../" + strings.Join(parts[len(parts)-2:], "/")
}

// isNoiseImport filters out tokens the regex symbol extractor captures as
// "imports" but which are actually string literals (e.g. manifest filenames),
// keeping the code graph clean.
func isNoiseImport(imp string) bool {
	switch imp {
	case "go.mod", "go.sum", "package.json", "package-lock.json", "Cargo.toml",
		"Cargo.lock", "requirements.txt", "Gemfile", "pom.xml", "build.gradle",
		"tsconfig.json", ".env", "yarn.lock", "pnpm-lock.yaml":
		return true
	}
	// Bare config-ish filenames with a dot but no path separator and a known
	// data extension are almost never real import targets.
	if !strings.Contains(imp, "/") {
		lower := strings.ToLower(imp)
		for _, ext := range []string{".toml", ".yaml", ".yml", ".lock", ".txt", ".xml", ".cfg", ".ini"} {
			if strings.HasSuffix(lower, ext) {
				return true
			}
		}
	}
	return false
}
