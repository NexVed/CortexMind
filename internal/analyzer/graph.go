package analyzer

import "fmt"

// Node is a single knowledge-graph vertex.
type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"` // project | language | framework | database | tool | auth | provider | feature | module
	Color string `json:"color"`
}

// Edge connects two nodes.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Rel    string `json:"rel"` // uses | has_feature | authenticates_with | contains
}

// Graph is the persisted knowledge graph for a project.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

var typeColors = map[string]string{
	"project":  "#E8326E",
	"language": "#8B5CF6",
	"framework": "#3B82F6",
	"database": "#F59E0B",
	"tool":     "#06B6D4",
	"auth":     "#EF4444",
	"provider": "#EC4899",
	"feature":  "#22C55E",
	"module":   "#64748B",
}

// buildGraph assembles a knowledge graph from the analysis. The project node is
// the hub; every finding becomes a typed node linked back to it.
func buildGraph(projectName string, a *Analysis) Graph {
	g := Graph{}
	idx := map[string]bool{}

	addNode := func(id, label, typ string) string {
		if !idx[id] {
			idx[id] = true
			g.Nodes = append(g.Nodes, Node{ID: id, Label: label, Type: typ, Color: typeColors[typ]})
		}
		return id
	}
	addEdge := func(src, tgt, rel string) {
		g.Edges = append(g.Edges, Edge{Source: src, Target: tgt, Rel: rel})
	}

	root := addNode("project", projectName, "project")

	link := func(prefix, typ, rel string, values []string) {
		for i, v := range values {
			id := fmt.Sprintf("%s_%d", prefix, i)
			addNode(id, v, typ)
			addEdge(root, id, rel)
		}
	}

	link("lang", "language", "uses", a.TechStack.Languages)
	link("fw", "framework", "uses", a.TechStack.Frameworks)
	link("db", "database", "uses", a.TechStack.Databases)
	link("tool", "tool", "uses", a.TechStack.Tools)

	if a.Auth.Detected {
		authID := addNode("auth", "Authentication", "auth")
		addEdge(root, authID, "has_feature")
		for i, m := range a.Auth.Mechanisms {
			id := fmt.Sprintf("authmech_%d", i)
			addNode(id, m, "auth")
			addEdge(authID, id, "uses")
		}
		for i, p := range a.Auth.Providers {
			id := fmt.Sprintf("authprov_%d", i)
			addNode(id, p, "provider")
			addEdge(authID, id, "authenticates_with")
		}
	}

	for i, f := range a.Features {
		id := fmt.Sprintf("feat_%d", i)
		addNode(id, f.Name, "feature")
		addEdge(root, id, "has_feature")
	}

	for _, node := range a.Structure {
		if node.Kind == "dir" && node.Role == "source" {
			id := "mod_" + node.Name
			addNode(id, node.Name, "module")
			addEdge(root, id, "contains")
		}
	}

	return g
}
