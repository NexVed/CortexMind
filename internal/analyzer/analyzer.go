// Package analyzer derives a high level understanding of a repository — its
// tech stack, authentication approach, notable features and structure — and
// assembles a knowledge graph from those findings. It is heuristic-first
// (works fully offline) with optional LLM enrichment of the summary.
package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NexVed/Cortex/internal/llm"
)

// TechStack captures the detected technologies grouped by role.
type TechStack struct {
	Languages       []string `json:"languages"`
	Frameworks      []string `json:"frameworks"`
	Databases       []string `json:"databases"`
	Tools           []string `json:"tools"`
	PackageManagers []string `json:"package_managers"`
}

// AuthAnalysis captures detected authentication mechanisms and providers.
type AuthAnalysis struct {
	Detected   bool     `json:"detected"`
	Mechanisms []string `json:"mechanisms"` // jwt, session, oauth2, api-key, basic
	Providers  []string `json:"providers"`  // github, google, auth0, clerk, firebase, supabase ...
	Libraries  []string `json:"libraries"`  // passport, next-auth, golang-jwt ...
}

// Feature is an inferred capability of the project.
type Feature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// StructureNode describes a top-level directory or notable file.
type StructureNode struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // dir | file
	Role string `json:"role"` // source | tests | docs | config | infra | assets
}

// Analysis is the full result persisted to projects.metadata.
type Analysis struct {
	TechStack TechStack       `json:"tech_stack"`
	Auth      AuthAnalysis    `json:"auth"`
	Features  []Feature       `json:"features"`
	Structure []StructureNode `json:"structure"`
	Endpoints []string        `json:"api_endpoints"`
	Summary   string          `json:"summary"`
	Graph     Graph           `json:"graph"`
	Enriched  bool            `json:"enriched"` // true when an LLM refined the summary
}

// Input is the data the analyzer needs about an already-scanned repo.
type Input struct {
	RepoPath    string
	ProjectName string
	Languages   map[string]int // language -> file count (from scanner.Result)
	EntryPoints []string
}

// Analyze performs the full heuristic analysis and, when an LLM client is
// enabled, enriches the summary. The llmClient may be nil.
func Analyze(ctx context.Context, in Input, llmClient llm.Client) (*Analysis, error) {
	a := &Analysis{}

	a.TechStack.Languages = sortedLangs(in.Languages)

	scan := scanRepo(in.RepoPath)

	a.TechStack.Frameworks = scan.frameworks.list()
	a.TechStack.Databases = scan.databases.list()
	a.TechStack.Tools = scan.tools.list()
	a.TechStack.PackageManagers = scan.packageManagers.list()

	a.Auth = scan.auth()
	a.Structure = detectStructure(in.RepoPath)
	a.Endpoints = scan.endpoints.list()
	a.Features = inferFeatures(scan, a)

	a.Summary = buildHeuristicSummary(in, a)

	// Optional LLM enrichment of the summary/features.
	if llmClient != nil && llmClient.Enabled() {
		if enriched, err := enrichSummary(ctx, llmClient, in, a); err == nil && strings.TrimSpace(enriched) != "" {
			a.Summary = enriched
			a.Enriched = true
		}
	}

	a.Graph = buildGraph(in.ProjectName, a)
	return a, nil
}

func sortedLangs(langs map[string]int) []string {
	type kv struct {
		k string
		v int
	}
	var s []kv
	for k, v := range langs {
		s = append(s, kv{k, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	out := make([]string, 0, len(s))
	for _, e := range s {
		out = append(out, e.k)
	}
	return out
}

func detectStructure(repoPath string) []StructureNode {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil
	}
	var nodes []StructureNode
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != ".github" {
			continue
		}
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		nodes = append(nodes, StructureNode{Name: name, Kind: kind, Role: roleOf(name, e.IsDir())})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind == "dir"
		}
		return nodes[i].Name < nodes[j].Name
	})
	return nodes
}

func roleOf(name string, isDir bool) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "test") || lower == "spec" || lower == "__tests__":
		return "tests"
	case lower == "docs" || lower == "doc" || strings.HasSuffix(lower, ".md"):
		return "docs"
	case lower == ".github" || lower == "deploy" || lower == "infra" || lower == "k8s" || lower == "terraform" || lower == "dockerfile" || lower == "docker-compose.yml":
		return "infra"
	case lower == "public" || lower == "assets" || lower == "static" || lower == "images":
		return "assets"
	case isDir && (lower == "src" || lower == "internal" || lower == "lib" || lower == "app" || lower == "pkg" || lower == "cmd" || lower == "components" || lower == "server" || lower == "backend" || lower == "frontend" || lower == "ui"):
		return "source"
	case !isDir && (strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".toml") || strings.Contains(lower, "config") || strings.HasPrefix(lower, ".env")):
		return "config"
	default:
		if isDir {
			return "source"
		}
		return "config"
	}
}

func buildHeuristicSummary(in Input, a *Analysis) string {
	var b strings.Builder
	b.WriteString(in.ProjectName)
	if len(a.TechStack.Languages) > 0 {
		b.WriteString(" is a " + strings.Join(topN(a.TechStack.Languages, 3), "/") + " project")
	} else {
		b.WriteString(" is a repository")
	}
	if len(a.TechStack.Frameworks) > 0 {
		b.WriteString(" built with " + strings.Join(topN(a.TechStack.Frameworks, 4), ", "))
	}
	b.WriteString(".")
	if len(a.TechStack.Databases) > 0 {
		b.WriteString(" Persistence: " + strings.Join(a.TechStack.Databases, ", ") + ".")
	}
	if a.Auth.Detected {
		mech := strings.Join(a.Auth.Mechanisms, ", ")
		if mech == "" {
			mech = "authentication"
		}
		b.WriteString(" Uses " + mech)
		if len(a.Auth.Providers) > 0 {
			b.WriteString(" via " + strings.Join(a.Auth.Providers, ", "))
		}
		b.WriteString(".")
	}
	if len(a.Features) > 0 {
		names := make([]string, 0, len(a.Features))
		for _, f := range a.Features {
			names = append(names, f.Name)
		}
		b.WriteString(" Key features: " + strings.Join(topN(names, 8), ", ") + ".")
	}
	return b.String()
}

func enrichSummary(ctx context.Context, c llm.Client, in Input, a *Analysis) (string, error) {
	system := "You are a senior software architect. Given structured facts about a code repository, write a concise 3-5 sentence technical overview covering its purpose, tech stack, authentication and notable features. Be factual and avoid speculation."
	var u strings.Builder
	u.WriteString("Repository: " + in.ProjectName + "\n")
	u.WriteString("Languages: " + strings.Join(a.TechStack.Languages, ", ") + "\n")
	u.WriteString("Frameworks: " + strings.Join(a.TechStack.Frameworks, ", ") + "\n")
	u.WriteString("Databases: " + strings.Join(a.TechStack.Databases, ", ") + "\n")
	u.WriteString("Tools: " + strings.Join(a.TechStack.Tools, ", ") + "\n")
	if a.Auth.Detected {
		u.WriteString("Auth mechanisms: " + strings.Join(a.Auth.Mechanisms, ", ") + "\n")
		u.WriteString("Auth providers: " + strings.Join(a.Auth.Providers, ", ") + "\n")
	}
	names := make([]string, 0, len(a.Features))
	for _, f := range a.Features {
		names = append(names, f.Name)
	}
	u.WriteString("Detected features: " + strings.Join(names, ", ") + "\n")
	readme := readReadme(in.RepoPath)
	if readme != "" {
		u.WriteString("\nREADME excerpt:\n" + readme + "\n")
	}
	return c.Summarize(ctx, system, u.String())
}

func readReadme(repoPath string) string {
	for _, name := range []string{"README.md", "README", "Readme.md", "readme.md"} {
		data, err := os.ReadFile(filepath.Join(repoPath, name))
		if err == nil {
			if len(data) > 2000 {
				data = data[:2000]
			}
			return string(data)
		}
	}
	return ""
}

func topN(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
