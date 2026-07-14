package analyzer

import (
	"reflect"
	"testing"
)

func TestBuildCodeGraphResolvesDependenciesAndUniqueSymbols(t *testing.T) {
	files := []FileEntry{
		{
			Path:     "src/main.ts",
			Language: "TypeScript",
			Functions: []SymbolRef{
				{Name: "run", Line: 1, Public: true},
				{Name: "run", Line: 8, Public: true},
			},
			Imports: []string{"./util", "lodash/map", "@scope/tools/parser"},
		},
		{Path: "src/util.ts", Language: "TypeScript"},
		{
			Path:     "internal/api/service.go",
			Language: "Go",
			Imports:  []string{"github.com/acme/app/internal/db"},
		},
		{Path: "internal/db/store.go", Language: "Go"},
	}

	graph := BuildCodeGraph(files)
	if graph.Stats.Files != 4 {
		t.Fatalf("files = %d, want 4", graph.Stats.Files)
	}
	if graph.Stats.Functions != 2 {
		t.Fatalf("functions = %d, want 2", graph.Stats.Functions)
	}
	if !hasNode(graph, "fn:src/main.ts#run@1") || !hasNode(graph, "fn:src/main.ts#run@8") {
		t.Fatal("duplicate symbol names should remain distinct by line")
	}
	if !hasEdge(graph, "file:src/main.ts", "file:src/util.ts", "depends_on") {
		t.Fatal("relative import was not resolved to the local file")
	}
	if !hasEdge(graph, "file:internal/api/service.go", "dir:internal/db", "depends_on") {
		t.Fatal("Go module import was not resolved to the internal directory")
	}
	if !hasNode(graph, "pkg:lodash") || !hasNode(graph, "pkg:@scope/tools") {
		t.Fatal("external package imports were not grouped by package root")
	}
	if graph.Stats.InternalDeps != 2 {
		t.Fatalf("internal deps = %d, want 2", graph.Stats.InternalDeps)
	}
	if graph.Stats.ExternalDeps != 2 {
		t.Fatalf("external deps = %d, want 2", graph.Stats.ExternalDeps)
	}
	if graph.Stats.MaxDegree == 0 {
		t.Fatal("max degree should be computed")
	}
}

func TestBuildCodeGraphCountsDependencyCycles(t *testing.T) {
	graph := BuildCodeGraph([]FileEntry{
		{Path: "a.ts", Language: "TypeScript", Imports: []string{"./b"}},
		{Path: "b.ts", Language: "TypeScript", Imports: []string{"./a"}},
	})
	if graph.Stats.Cycles != 1 {
		t.Fatalf("cycles = %d, want 1", graph.Stats.Cycles)
	}
}

func TestBuildCodeGraphIsDeterministic(t *testing.T) {
	files := []FileEntry{
		{Path: "z.ts", Language: "TypeScript", Imports: []string{"./a"}},
		{Path: "a.ts", Language: "TypeScript"},
	}
	first := BuildCodeGraph(files)
	second := BuildCodeGraph(files)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same indexed files should produce byte-stable graph data")
	}
}

func hasNode(graph CodeGraph, id string) bool {
	for _, node := range graph.Nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func hasEdge(graph CodeGraph, source, target, rel string) bool {
	for _, edge := range graph.Edges {
		if edge.Source == source && edge.Target == target && edge.Rel == rel {
			return true
		}
	}
	return false
}
