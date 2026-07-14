package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	cgit "github.com/NexVed/Cortex/internal/git"
	"github.com/pocketbase/pocketbase/core"
)

// categoryDir maps a vault category to its .cortex/ subdirectory.
var categoryDir = map[string]string{
	"architecture": "architecture",
	"decision":     "decisions",
	"roadmap":      "roadmaps",
	"task":         "tasks",
	"handoff":      "handoffs",
	"memory":       "memories",
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a title into a filesystem-safe slug.
func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

// ExportEntryToFile writes a shared vault entry to the .cortex/ directory and
// returns the relative file path that was written.
func ExportEntryToFile(entry *core.Record, repoPath string) (string, error) {
	category := entry.GetString("category")
	sub, ok := categoryDir[category]
	if !ok {
		sub = "memories"
	}
	dir := filepath.Join(cgit.CortexDir(repoPath), sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	slug := Slugify(entry.GetString("title"))
	filename := slug + ".md"
	fullPath := filepath.Join(dir, filename)

	content := fmt.Sprintf("# %s\n\n%s\n", entry.GetString("title"), entry.GetString("content"))
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return "", err
	}

	rel := filepath.ToSlash(filepath.Join(".cortex", sub, filename))
	return rel, nil
}

// ── Portable memory bundle (.cortex/memory.json + README.md) ───────────────
//
// A Bundle is the complete, shareable snapshot of a project's agentic memory.
// It is written to the repo's .cortex/ directory as both a machine-readable
// JSON file (memory.json) and a human-readable index (README.md) so it can be
// committed and pushed for teammates to consume.

const BundleSchemaVersion = "1.1"

// ProjectMeta is the lightweight project description carried in a bundle.
type ProjectMeta struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	GitHubURL    string   `json:"github_url,omitempty"`
	Technologies []string `json:"technologies,omitempty"`
}

// BundleEntry is a single vault entry (architecture, decision, roadmap, …).
type BundleEntry struct {
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
	SourceAgent string   `json:"source_agent,omitempty"`
	Version     int      `json:"version,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

// AgentMemory is what an AI agent learned/did in an IDE session.
type AgentMemory struct {
	IDE       string   `json:"ide,omitempty"`
	Client    string   `json:"client_name,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	Category  string   `json:"category"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// SessionDigest is a compressed summary of a single agent session.
type SessionDigest struct {
	SessionID   string `json:"session_id,omitempty"`
	IDE         string `json:"ide,omitempty"`
	Title       string `json:"title"`
	SummaryMD   string `json:"summary_md,omitempty"`
	TokenCount  int    `json:"token_count,omitempty"`
	MemoryCount int    `json:"memory_count,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// Bundle is the full portable memory snapshot for a project.
type Bundle struct {
	SchemaVersion  string          `json:"schema_version"`
	GeneratedAt    string          `json:"generated_at"`
	Generator      string          `json:"generator"`
	ExportMode     string          `json:"export_mode,omitempty"`
	Project        ProjectMeta     `json:"project"`
	VaultEntries   []BundleEntry   `json:"vault_entries,omitempty"`
	AgentMemories  []AgentMemory   `json:"agent_memories,omitempty"`
	SessionDigests []SessionDigest `json:"session_digests,omitempty"`
}

// ExportBundle writes a Bundle to the repo's .cortex/ directory as memory.json
// and a metadata-only README.md. Keeping one canonical content file avoids
// doubling repository size and reduces noisy Git diffs.
func ExportBundle(bundle Bundle, repoPath string) ([]string, error) {
	if bundle.SchemaVersion == "" {
		bundle.SchemaVersion = BundleSchemaVersion
	}
	if bundle.GeneratedAt == "" {
		bundle.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if bundle.Generator == "" {
		bundle.Generator = "cortex"
	}
	if bundle.ExportMode == "" {
		bundle.ExportMode = "compact"
	}

	base := cgit.CortexDir(repoPath)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}

	written := []string{}

	// 1. Machine-readable JSON.
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal bundle: %w", err)
	}
	jsonPath := filepath.Join(base, "memory.json")
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return nil, err
	}
	written = append(written, filepath.ToSlash(filepath.Join(".cortex", "memory.json")))

	// 2. Human-readable README.
	readmePath := filepath.Join(base, "README.md")
	if err := os.WriteFile(readmePath, []byte(renderBundleReadme(bundle)), 0o644); err != nil {
		return nil, err
	}
	written = append(written, filepath.ToSlash(filepath.Join(".cortex", "README.md")))

	return written, nil
}

func renderBundleReadme(b Bundle) string {
	var w strings.Builder
	fmt.Fprintf(&w, "# %s — CORTEX Memory\n\n", b.Project.Name)
	w.WriteString("> Portable agentic memory exported by CORTEX. This directory travels with the\n")
	w.WriteString("> repository so teammates and their AI agents share the same context.\n\n")

	if b.Project.Description != "" {
		w.WriteString(b.Project.Description + "\n\n")
	}
	w.WriteString("| Field | Value |\n|---|---|\n")
	fmt.Fprintf(&w, "| Generated | %s |\n", b.GeneratedAt)
	fmt.Fprintf(&w, "| Schema | v%s |\n", b.SchemaVersion)
	fmt.Fprintf(&w, "| Export mode | %s |\n", b.ExportMode)
	if b.Project.GitHubURL != "" {
		fmt.Fprintf(&w, "| Repository | %s |\n", b.Project.GitHubURL)
	}
	if len(b.Project.Technologies) > 0 {
		fmt.Fprintf(&w, "| Tech | %s |\n", strings.Join(b.Project.Technologies, ", "))
	}
	fmt.Fprintf(&w, "| Vault entries | %d |\n", len(b.VaultEntries))
	fmt.Fprintf(&w, "| Agent memories | %d |\n", len(b.AgentMemories))
	fmt.Fprintf(&w, "| Session digests | %d |\n\n", len(b.SessionDigests))

	w.WriteString("Machine-readable form: [`memory.json`](./memory.json)\n\n")
	w.WriteString("The JSON file is canonical. This README contains metadata only so memory content is not duplicated.\n\n")
	if b.ExportMode != "full" {
		w.WriteString("Raw agent memories are kept in local storage; use the full export option only when an archive is required.\n\n")
	}

	if len(b.VaultEntries) > 0 {
		w.WriteString("## Knowledge Vault\n\n")
		for _, e := range b.VaultEntries {
			meta := []string{e.Category}
			if e.Version > 0 {
				meta = append(meta, fmt.Sprintf("v%d", e.Version))
			}
			if e.UpdatedAt != "" {
				meta = append(meta, e.UpdatedAt)
			}
			fmt.Fprintf(&w, "- [%s] %s\n", strings.Join(meta, " · "), e.Title)
		}
		w.WriteString("\n")
	}

	if len(b.AgentMemories) > 0 {
		w.WriteString("## Raw Agent Memories\n\n")
		w.WriteString("This was a full export. Complete content is stored only in `memory.json`.\n\n")
		for _, m := range b.AgentMemories {
			title := m.Title
			if title == "" {
				title = "(untitled)"
			}
			meta := []string{}
			if m.IDE != "" {
				meta = append(meta, m.IDE)
			}
			if m.SessionID != "" {
				meta = append(meta, "session "+m.SessionID)
			}
			if len(meta) > 0 {
				fmt.Fprintf(&w, "- [%s] %s (%s)\n", m.Category, title, strings.Join(meta, " · "))
			} else {
				fmt.Fprintf(&w, "- [%s] %s\n", m.Category, title)
			}
		}
		w.WriteString("\n")
	}

	if len(b.SessionDigests) > 0 {
		w.WriteString("## Session Digests\n\n")
		for _, d := range b.SessionDigests {
			title := d.Title
			if title == "" {
				title = "(untitled session)"
			}
			fmt.Fprintf(&w, "- %s (%d memories, ~%d tokens)\n", title, d.MemoryCount, d.TokenCount)
		}
		w.WriteString("\n")
	}

	return w.String()
}
