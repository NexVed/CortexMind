package services

import (
	"fmt"
	"sort"
	"strings"
)

// CodeGraphQuery lets MCP clients request focused context instead of receiving
// an unbounded repository graph on every tool call.
type CodeGraphQuery struct {
	NodeID        string
	Query         string
	NodeTypes     []string
	Relationships []string
	MaxNodes      int
}

// CodeGraphContext is the bounded graph slice returned to coding agents.
type CodeGraphContext struct {
	ProjectID   string      `json:"project_id"`
	ProjectName string      `json:"project_name"`
	GeneratedAt string      `json:"generated_at"`
	Built       bool        `json:"built"`
	Stats       GraphStats  `json:"stats"`
	Nodes       []GraphNode `json:"nodes"`
	Edges       []GraphEdge `json:"edges"`
	Truncated   bool        `json:"truncated"`
}

func (s CodeGraphService) Context(projectID string, query CodeGraphQuery) (*CodeGraphContext, error) {
	graph, found, err := s.Load(projectID)
	if err != nil {
		return nil, err
	}
	if !found || !graph.Built {
		return nil, fmt.Errorf("code graph has not been built for project %q", projectID)
	}
	if query.MaxNodes <= 0 {
		query.MaxNodes = 250
	}
	if query.MaxNodes > 1000 {
		query.MaxNodes = 1000
	}

	validTypes := allowedGraphValues(query.NodeTypes, map[string]bool{"dir": true, "file": true, "function": true, "class": true, "package": true})
	validRelations := allowedGraphValues(query.Relationships, map[string]bool{"contains": true, "defines": true, "imports": true, "depends_on": true})
	nodesByID := make(map[string]GraphNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodesByID[node.ID] = node
	}
	selected := map[string]bool{}
	if query.NodeID != "" {
		if _, exists := nodesByID[query.NodeID]; !exists {
			return nil, fmt.Errorf("code graph node %q was not found", query.NodeID)
		}
		selected[query.NodeID] = true
		for _, edge := range graph.Edges {
			if !matchesRelationship(edge, validRelations) {
				continue
			}
			if edge.Source == query.NodeID {
				selected[edge.Target] = true
			}
			if edge.Target == query.NodeID {
				selected[edge.Source] = true
			}
		}
	} else {
		needle := strings.ToLower(strings.TrimSpace(query.Query))
		candidates := make([]GraphNode, 0, len(graph.Nodes))
		for _, node := range graph.Nodes {
			if len(validTypes) > 0 && !validTypes[node.Type] {
				continue
			}
			if needle != "" && !strings.Contains(strings.ToLower(node.Label), needle) && !strings.Contains(strings.ToLower(node.Path), needle) {
				continue
			}
			candidates = append(candidates, node)
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].Degree != candidates[j].Degree {
				return candidates[i].Degree > candidates[j].Degree
			}
			return candidates[i].ID < candidates[j].ID
		})
		for _, node := range candidates[:minInt(len(candidates), query.MaxNodes)] {
			selected[node.ID] = true
		}
	}

	nodes := make([]GraphNode, 0, len(selected))
	for _, node := range graph.Nodes {
		if selected[node.ID] {
			nodes = append(nodes, node)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	edges := make([]GraphEdge, 0)
	for _, edge := range graph.Edges {
		if selected[edge.Source] && selected[edge.Target] && matchesRelationship(edge, validRelations) {
			edges = append(edges, edge)
		}
	}
	return &CodeGraphContext{
		ProjectID: graph.ProjectID, ProjectName: graph.ProjectName, GeneratedAt: graph.GeneratedAt,
		Built: graph.Built, Stats: graph.Stats, Nodes: nodes, Edges: edges,
		Truncated: query.NodeID == "" && len(selected) < len(graph.Nodes),
	}, nil
}

func allowedGraphValues(values []string, allowed map[string]bool) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		if allowed[value] {
			result[value] = true
		}
	}
	return result
}

func matchesRelationship(edge GraphEdge, requested map[string]bool) bool {
	return len(requested) == 0 || requested[edge.Rel]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
