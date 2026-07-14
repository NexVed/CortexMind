package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/NexVed/Cortex/internal/analyzer"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// PromptOptions toggles which knowledge sources feed the system prompt.
type PromptOptions struct {
	IncludeTasks    bool `json:"include_tasks"`
	IncludeVault    bool `json:"include_vault"`
	IncludeActivity bool `json:"include_activity"`
	Preview         bool `json:"preview"`
}

// PromptResult is returned to the UI after generation.
type PromptResult struct {
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	Provider      string `json:"provider"` // always "cortex" — built locally, no LLM
	Prompt        string `json:"prompt"`
	TokenEstimate int    `json:"token_estimate"`
}

// GetSystemPrompt and SaveSystemPrompt expose the stored project prompt.
// from the stored analysis (tech stack, auth, features, structure) plus the
// selected knowledge sources (vault decisions, active tasks, recent activity).
//
// It deliberately does NOT call any LLM/API: CORTEX's job is to STORE the
// codebase and agent memory, and the prompt it emits is a clean, ready-to-paste
// system prompt for ChatGPT, Claude or any coding agent. The result is
// persisted as a memory vault entry and on the project record.
func (s *Service) GetSystemPrompt(projectID string) (*PromptResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	prompt := storedSystemPrompt(project)
	result := &PromptResult{
		ProjectID:     project.Id,
		ProjectName:   project.GetString("name"),
		Provider:      "custom",
		Prompt:        prompt,
		TokenEstimate: len(prompt) / 4,
	}
	return result, nil
}

func (s *Service) SaveSystemPrompt(user *core.Record, projectID, prompt string) (*PromptResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	if owner := project.GetString("owner"); owner != "" && owner != user.Id {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	prompt = strings.TrimSpace(prompt)
	if err := s.persistSystemPromptAs(user, project, prompt, "saved_system_prompt", "user"); err != nil {
		return nil, err
	}
	return &PromptResult{
		ProjectID:     project.Id,
		ProjectName:   project.GetString("name"),
		Provider:      "custom",
		Prompt:        prompt,
		TokenEstimate: len(prompt) / 4,
	}, nil
}

// GenerateSystemPrompt builds a project-specific agent system prompt entirely
// from the stored analysis and selected knowledge sources.
func (s *Service) GenerateSystemPrompt(ctx context.Context, user *core.Record, projectID string, opts PromptOptions) (*PromptResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	var analysis analyzer.Analysis
	if err := project.UnmarshalJSONField("metadata", &analysis); err != nil || len(analysis.TechStack.Languages) == 0 && analysis.Summary == "" {
		return nil, fmt.Errorf("project has not been scanned yet — run Scan first")
	}

	result := &PromptResult{
		ProjectID:   project.Id,
		ProjectName: project.GetString("name"),
		Provider:    "cortex",
	}
	result.Prompt = s.buildSystemPrompt(project, &analysis, opts)
	result.TokenEstimate = len(result.Prompt) / 4

	if !opts.Preview {
		if err := s.persistSystemPrompt(user, project, result.Prompt); err != nil {
			log.Warn().Err(err).Str("project", project.Id).Msg("failed to persist system prompt")
		}
	}
	return result, nil
}

// buildSystemPrompt assembles a clean, structured, copy-paste-ready system
// prompt for an AI coding agent purely from stored data — no LLM involved.
func (s *Service) buildSystemPrompt(project *core.Record, a *analyzer.Analysis, opts PromptOptions) string {
	name := project.GetString("name")
	var b strings.Builder

	// ── Role ──
	b.WriteString(fmt.Sprintf("You are an expert software engineer and pair-programmer for the %q project. ", name))
	b.WriteString("Everything below is authoritative context about this codebase, compiled by CORTEX from a scan of the repository. Use it as your ground truth. When you write or change code, stay consistent with the stack, structure and conventions described here.\n")

	// ── Overview ──
	b.WriteString("\n## Project overview\n")
	if d := project.GetString("description"); d != "" {
		b.WriteString(d + "\n")
	}
	if a.Summary != "" {
		b.WriteString(a.Summary + "\n")
	}
	if project.GetString("description") == "" && a.Summary == "" {
		b.WriteString(name + " — see the technology stack and structure below.\n")
	}

	// ── Tech stack ──
	stackLines := []string{}
	stackLines = appendStackLine(stackLines, "Languages", a.TechStack.Languages)
	stackLines = appendStackLine(stackLines, "Frameworks & libraries", a.TechStack.Frameworks)
	stackLines = appendStackLine(stackLines, "Databases & storage", a.TechStack.Databases)
	stackLines = appendStackLine(stackLines, "Tooling", a.TechStack.Tools)
	stackLines = appendStackLine(stackLines, "Package managers", a.TechStack.PackageManagers)
	if len(stackLines) > 0 {
		b.WriteString("\n## Technology stack\n")
		b.WriteString("Work within this stack. Do not introduce alternative languages or frameworks without explicit instruction.\n")
		for _, l := range stackLines {
			b.WriteString(l)
		}
	}

	// ── Architecture / structure ──
	if len(a.Structure) > 0 {
		b.WriteString("\n## Project structure\n")
		b.WriteString("Key directories and where code lives:\n")
		for _, n := range a.Structure {
			suffix := "/"
			if n.Kind == "file" {
				suffix = ""
			}
			role := n.Role
			if role == "" {
				role = n.Kind
			}
			b.WriteString(fmt.Sprintf("- `%s%s` — %s\n", n.Name, suffix, role))
		}
	}

	// ── Authentication ──
	if a.Auth.Detected {
		b.WriteString("\n## Authentication\n")
		if len(a.Auth.Mechanisms) > 0 {
			b.WriteString("Mechanisms: " + strings.Join(a.Auth.Mechanisms, ", ") + ".\n")
		}
		if len(a.Auth.Providers) > 0 {
			b.WriteString("Providers: " + strings.Join(a.Auth.Providers, ", ") + ".\n")
		}
		if len(a.Auth.Libraries) > 0 {
			b.WriteString("Libraries: " + strings.Join(a.Auth.Libraries, ", ") + ".\n")
		}
		b.WriteString("Respect the existing auth flow; never weaken or bypass it.\n")
	}

	// ── Features ──
	if len(a.Features) > 0 {
		b.WriteString("\n## Key features\n")
		for _, f := range a.Features {
			if f.Description != "" {
				b.WriteString(fmt.Sprintf("- **%s** — %s\n", f.Name, f.Description))
			} else {
				b.WriteString("- **" + f.Name + "**\n")
			}
		}
	}

	// ── API surface ──
	if len(a.Endpoints) > 0 {
		b.WriteString("\n## API surface (sample)\n")
		for i, e := range a.Endpoints {
			if i >= 25 {
				b.WriteString(fmt.Sprintf("- …and %d more endpoints\n", len(a.Endpoints)-25))
				break
			}
			b.WriteString("- `" + e + "`\n")
		}
	}

	// ── Knowledge sources (opt-in) ──
	if opts.IncludeVault {
		s.appendVault(&b, project.Id)
	}
	if opts.IncludeTasks {
		s.appendTasks(&b, project.Id)
	}
	if opts.IncludeActivity {
		s.appendActivity(&b, project.Id)
	}

	// ── Working agreement ──
	b.WriteString("\n## How to work on this project\n")
	b.WriteString("- Match the existing code style, naming and file layout.\n")
	b.WriteString("- Make minimal, focused changes; prefer editing existing files over adding new abstractions.\n")
	b.WriteString("- Read the relevant code before changing it, and keep changes consistent with the stack above.\n")
	b.WriteString("- Explain non-obvious decisions, and ask before destructive or wide-reaching changes.\n")
	b.WriteString("- When you finish a unit of work, summarize what changed and why.\n")

	return b.String()
}

func (s *Service) appendVault(b *strings.Builder, projectID string) {
	recs, err := s.App.FindRecordsByFilter(db.CollVaultEntries,
		"project = {:p} && (category = 'architecture' || category = 'decision')", "-updated", 20, 0,
		map[string]any{"p": projectID})
	if err != nil || len(recs) == 0 {
		return
	}
	b.WriteString("\n## Architectural decisions & notes\n")
	b.WriteString("Honor these decisions unless explicitly told otherwise:\n")
	for _, r := range recs {
		content := r.GetString("content")
		if len(content) > 400 {
			content = content[:400] + "…"
		}
		b.WriteString("- **" + r.GetString("title") + "**: " + content + "\n")
	}
}

func (s *Service) appendTasks(b *strings.Builder, projectID string) {
	recs, err := s.App.FindRecordsByFilter(db.CollTasks,
		"project = {:p} && status != 'done'", "-updated", 20, 0,
		map[string]any{"p": projectID})
	if err != nil || len(recs) == 0 {
		return
	}
	b.WriteString("\n## Active tasks\n")
	b.WriteString("Current work in progress you should be aware of:\n")
	for _, r := range recs {
		b.WriteString(fmt.Sprintf("- [%s] %s\n", r.GetString("status"), r.GetString("title")))
	}
}

func (s *Service) appendActivity(b *strings.Builder, projectID string) {
	recs, err := s.App.FindRecordsByFilter(db.CollActivityLog,
		"project = {:p}", "-created", 10, 0, map[string]any{"p": projectID})
	if err != nil || len(recs) == 0 {
		return
	}
	b.WriteString("\n## Recent activity\n")
	for _, r := range recs {
		b.WriteString("- " + r.GetString("action") + ": " + r.GetString("subject") + "\n")
	}
}

// appendStackLine adds a "- Label: a, b, c" line when items are present.
func appendStackLine(lines []string, label string, items []string) []string {
	if len(items) == 0 {
		return lines
	}
	return append(lines, "- "+label+": "+strings.Join(items, ", ")+"\n")
}

func storedSystemPrompt(project *core.Record) string {
	var meta map[string]any
	if err := project.UnmarshalJSONField("metadata", &meta); err != nil || meta == nil {
		return ""
	}
	prompt, _ := meta["system_prompt"].(string)
	return prompt
}

func (s *Service) persistSystemPrompt(user *core.Record, project *core.Record, prompt string) error {
	return s.persistSystemPromptAs(user, project, prompt, "generated_system_prompt", "cortex-prompt-generator")
}

func (s *Service) persistSystemPromptAs(user *core.Record, project *core.Record, prompt, activity, sourceAgent string) error {
	var meta map[string]any
	if err := project.UnmarshalJSONField("metadata", &meta); err != nil || meta == nil {
		meta = map[string]any{}
	}
	if prompt == "" {
		delete(meta, "system_prompt")
	} else {
		meta["system_prompt"] = prompt
	}
	project.Set("metadata", meta)
	if err := s.App.Save(project); err != nil {
		return err
	}

	title := project.GetString("name") + " — Agent System Prompt"
	existing, _ := s.App.FindFirstRecordByFilter(db.CollVaultEntries,
		"project = {:p} && category = 'memory' && title = {:t}",
		map[string]any{"p": project.Id, "t": title})

	if prompt == "" {
		if existing != nil {
			return s.App.Delete(existing)
		}
		return nil
	}

	var rec *core.Record
	if existing != nil {
		rec = existing
		rec.Set("version", existing.GetInt("version")+1)
	} else {
		coll, err := s.App.FindCollectionByNameOrId(db.CollVaultEntries)
		if err != nil {
			return err
		}
		rec = core.NewRecord(coll)
		rec.Set("project", project.Id)
		rec.Set("owner", user.Id)
		rec.Set("category", "memory")
		rec.Set("title", title)
		rec.Set("version", 1)
	}
	rec.Set("content", prompt)
	rec.Set("is_shared", true)
	rec.Set("source_agent", sourceAgent)
	if err := s.App.Save(rec); err != nil {
		return err
	}
	db.LogActivity(s.App, project.Id, user.Id, activity, title, nil)
	return nil
}
