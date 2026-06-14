package analyzer

import "strings"

// directoryFeatures maps a common directory name to an inferred feature.
var directoryFeatures = map[string]Feature{
	"api":           {"REST/HTTP API", "Exposes an HTTP API surface"},
	"graphql":       {"GraphQL API", "GraphQL schema and resolvers"},
	"auth":          {"Authentication", "Dedicated authentication module"},
	"migrations":    {"Database Migrations", "Versioned database schema migrations"},
	"models":        {"Data Models", "Domain/data model definitions"},
	"components":    {"UI Components", "Reusable frontend component library"},
	"pages":         {"Page Routing", "Page-based UI routing"},
	"hooks":         {"State/Logic Hooks", "Reusable logic hooks"},
	"services":      {"Service Layer", "Business-logic service layer"},
	"workers":       {"Background Workers", "Asynchronous/background processing"},
	"jobs":          {"Background Jobs", "Scheduled or queued jobs"},
	"cmd":           {"CLI / Entrypoints", "Command-line entrypoints"},
	"mcp":           {"MCP Integration", "Model Context Protocol server/tools"},
	"vector":        {"Vector Search", "Embedding-based semantic search"},
	"scanner":       {"Code Scanning", "Repository scanning/indexing"},
	"watcher":       {"File Watching", "Live filesystem change watching"},
	"webhooks":      {"Webhooks", "Inbound webhook handling"},
	"notifications": {"Notifications", "User notifications"},
	"payments":      {"Payments", "Payment processing integration"},
	"billing":       {"Billing", "Subscription/billing logic"},
	"search":        {"Search", "Search functionality"},
	"admin":         {"Admin Panel", "Administrative interface"},
	"dashboard":     {"Dashboard", "Analytics/overview dashboard"},
}

// inferFeatures combines directory-name signals, content signals and stack
// findings into a deduplicated feature list.
func inferFeatures(s *repoSignals, a *Analysis) []Feature {
	seen := map[string]bool{}
	var out []Feature
	add := func(f Feature) {
		key := strings.ToLower(f.Name)
		if !seen[key] {
			seen[key] = true
			out = append(out, f)
		}
	}

	for _, node := range a.Structure {
		if node.Kind != "dir" {
			continue
		}
		if f, ok := directoryFeatures[strings.ToLower(node.Name)]; ok {
			add(f)
		}
	}

	for _, sig := range s.featureSignals.list() {
		add(Feature{Name: sig, Description: "Detected from source code"})
	}

	if a.Auth.Detected {
		add(Feature{Name: "Authentication", Description: "User authentication and authorization"})
	}
	if len(s.endpoints.list()) > 0 {
		add(Feature{Name: "HTTP API", Description: "Defines HTTP/REST endpoints"})
	}
	if len(a.TechStack.Databases) > 0 {
		add(Feature{Name: "Data Persistence", Description: "Stores data in " + strings.Join(a.TechStack.Databases, ", ")})
	}
	for _, t := range a.TechStack.Tools {
		if t == "GraphQL" {
			add(Feature{Name: "GraphQL API", Description: "GraphQL API layer"})
		}
		if t == "Docker" {
			add(Feature{Name: "Containerized", Description: "Ships with Docker container configuration"})
		}
	}
	return out
}
