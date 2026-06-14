package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/NexVed/Cortex/internal/analyzer"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/NexVed/Cortex/internal/llm"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// PromptOptions toggles which knowledge sources feed the system prompt.
type PromptOptions struct {
	IncludeTasks    bool `json:"include_tasks"`
	IncludeVault    bool `json:"include_vault"`
	IncludeActivity bool `json:"include_activity"`
}

// PromptResult is returned to the UI after generation.
type PromptResult struct {
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	Provider      string `json:"provider"` // mistral | ollama | heuristic
	Prompt        string `json:"prompt"`
	TokenEstimate int    `json:"token_estimate"`
}

const promptSystemInstruction = `You are generating a SYSTEM PROMPT that will be given to an AI coding agent which will work on the project described below.
Output ONLY the system prompt text — no preamble, no explanation, no markdown code fences.
The system prompt must:
- Assign the agent a clear role as an expert engineer on this specific project.
- State the project's purpose and the technology stack it must use.
- Summarize the architecture, authentication approach and key features.
- Describe the directory structure so the agent knows where code lives.
- List conventions the agent should follow and current tasks/decisions it must respect.
Be specific, factual and actionable. Write in second person ("You are...").`

// GenerateSystemPrompt builds a project-specific agent system prompt using the
// user's configured LLM provider (Mistral/Ollama), falling back to a heuristic
// prompt when no provider is available. The result is persisted as a memory
// vault entry and on the project record.
func (s *Service) GenerateSystemPrompt(ctx context.Context, user *core.Record, projectID string, opts PromptOptions) (*PromptResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	var analysis analyzer.Analysis
	if err := project.UnmarshalJSONField("metadata", &analysis); err != nil || len(analysis.TechStack.Languages) == 0 && analysis.Summary == "" {
		return nil, fmt.Errorf("project has not been scanned yet — run Scan first")
	}

	facts := s.gatherFacts(project, &analysis, opts)

	provCfg := LoadProviderConfig(user)
	llmClient := llm.New(provCfg)

	result := &PromptResult{ProjectID: project.Id, ProjectName: project.GetString("name")}

	if llmClient.Enabled() {
		generated, gerr := llmClient.Summarize(ctx, promptSystemInstruction, facts)
		if gerr == nil && strings.TrimSpace(generated) != "" {
			result.Prompt = strings.TrimSpace(generated)
			result.Provider = provCfg.LLMProvider
		} else if gerr != nil {
			log.Warn().Err(gerr).Str("project", project.Id).Msg("llm prompt generation failed; using heuristic")
		}
	}
	if result.Prompt == "" {
		result.Prompt = heuristicSystemPrompt(project.GetString("name"), &analysis, facts)
		result.Provider = "heuristic"
	}
	result.TokenEstimate = len(result.Prompt) / 4

	if err := s.persistSystemPrompt(user, project, result.Prompt); err != nil {
		log.Warn().Err(err).Str("project", project.Id).Msg("failed to persist system prompt")
	}
	return result, nil
}

// gatherFacts assembles the structured context the LLM uses to write the prompt.
func (s *Service) gatherFacts(project *core.Record, a *analyzer.Analysis, opts PromptOptions) string {
	var b strings.Builder
	b.WriteString("PROJECT: " + project.GetString("name") + "\n")
	if d := project.GetString("description"); d != "" {
		b.WriteString("DESCRIPTION: " + d + "\n")
	}
	if a.Summary != "" {
		b.WriteString("OVERVIEW: " + a.Summary + "\n")
	}

	b.WriteString("\nTECH STACK:\n")
	writeFact(&b, "Languages", a.TechStack.Languages)
	writeFact(&b, "Frameworks", a.TechStack.Frameworks)
	writeFact(&b, "Databases", a.TechStack.Databases)
	writeFact(&b, "Tools", a.TechStack.Tools)
	writeFact(&b, "Package managers", a.TechStack.PackageManagers)

	if a.Auth.Detected {
		b.WriteString("\nAUTHENTICATION:\n")
		writeFact(&b, "Mechanisms", a.Auth.Mechanisms)
		writeFact(&b, "Providers", a.Auth.Providers)
		writeFact(&b, "Libraries", a.Auth.Libraries)
	}

	if len(a.Features) > 0 {
		b.WriteString("\nFEATURES:\n")
		for _, f := range a.Features {
			b.WriteString(fmt.Sprintf("- %s: %s\n", f.Name, f.Description))
		}
	}

	if len(a.Structure) > 0 {
		b.WriteString("\nSTRUCTURE (top level):\n")
		for _, n := range a.Structure {
			b.WriteString(fmt.Sprintf("- %s (%s, %s)\n", n.Name, n.Kind, n.Role))
		}
	}

	if len(a.Endpoints) > 0 {
		b.WriteString("\nAPI ENDPOINTS (sample):\n")
		for i, e := range a.Endpoints {
			if i >= 25 {
				break
			}
			b.WriteString("- " + e + "\n")
		}
	}

	if opts.IncludeVault {
		s.appendVault(&b, project.Id)
	}
	if opts.IncludeTasks {
		s.appendTasks(&b, project.Id)
	}
	if opts.IncludeActivity {
		s.appendActivity(&b, project.Id)
	}
	return b.String()
}

func (s *Service) appendVault(b *strings.Builder, projectID string) {
	recs, err := s.App.FindRecordsByFilter(db.CollVaultEntries,
		"project = {:p} && (category = 'architecture' || category = 'decision')", "-updated", 20, 0,
		map[string]any{"p": projectID})
	if err != nil || len(recs) == 0 {
		return
	}
	b.WriteString("\nARCHITECTURAL DECISIONS & NOTES:\n")
	for _, r := range recs {
		content := r.GetString("content")
		if len(content) > 400 {
			content = content[:400] + "…"
		}
		b.WriteString("- " + r.GetString("title") + ": " + content + "\n")
	}
}

func (s *Service) appendTasks(b *strings.Builder, projectID string) {
	recs, err := s.App.FindRecordsByFilter(db.CollTasks,
		"project = {:p} && status != 'done'", "-updated", 20, 0,
		map[string]any{"p": projectID})
	if err != nil || len(recs) == 0 {
		return
	}
	b.WriteString("\nACTIVE TASKS:\n")
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
	b.WriteString("\nRECENT ACTIVITY:\n")
	for _, r := range recs {
		b.WriteString("- " + r.GetString("action") + ": " + r.GetString("subject") + "\n")
	}
}

func (s *Service) persistSystemPrompt(user *core.Record, project *core.Record, prompt string) error {
	// Store on the project metadata for quick retrieval.
	var meta map[string]any
	if err := project.UnmarshalJSONField("metadata", &meta); err != nil || meta == nil {
		meta = map[string]any{}
	}
	meta["system_prompt"] = prompt
	project.Set("metadata", meta)
	if err := s.App.Save(project); err != nil {
		return err
	}

	// Upsert a memory vault entry so the prompt becomes shareable IDE memory.
	title := project.GetString("name") + " — Agent System Prompt"
	existing, _ := s.App.FindFirstRecordByFilter(db.CollVaultEntries,
		"project = {:p} && category = 'memory' && title = {:t}",
		map[string]any{"p": project.Id, "t": title})

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
	rec.Set("source_agent", "cortex-prompt-generator")
	if err := s.App.Save(rec); err != nil {
		return err
	}
	db.LogActivity(s.App, project.Id, user.Id, "generated_system_prompt", title, nil)
	return nil
}

// heuristicSystemPrompt produces a usable system prompt without an LLM.
func heuristicSystemPrompt(name string, a *analyzer.Analysis, facts string) string {
	var b strings.Builder
	stack := strings.Join(append(append([]string{}, a.TechStack.Languages...), a.TechStack.Frameworks...), ", ")
	b.WriteString(fmt.Sprintf("You are an expert software engineer working on the %q project.\n\n", name))
	if a.Summary != "" {
		b.WriteString(a.Summary + "\n\n")
	}
	if stack != "" {
		b.WriteString("You must work within this stack: " + stack + ". Do not introduce alternative frameworks without explicit instruction.\n\n")
	}
	if len(a.TechStack.Databases) > 0 {
		b.WriteString("Data is persisted using: " + strings.Join(a.TechStack.Databases, ", ") + ".\n\n")
	}
	if a.Auth.Detected {
		b.WriteString("Authentication uses " + strings.Join(a.Auth.Mechanisms, ", "))
		if len(a.Auth.Providers) > 0 {
			b.WriteString(" via " + strings.Join(a.Auth.Providers, ", "))
		}
		b.WriteString(". Respect the existing auth flow.\n\n")
	}
	if len(a.Structure) > 0 {
		b.WriteString("Key directories:\n")
		for _, n := range a.Structure {
			if n.Kind == "dir" && n.Role == "source" {
				b.WriteString("- " + n.Name + "/\n")
			}
		}
		b.WriteString("\n")
	}
	if len(a.Features) > 0 {
		b.WriteString("The project provides: ")
		names := make([]string, 0, len(a.Features))
		for _, f := range a.Features {
			names = append(names, f.Name)
		}
		b.WriteString(strings.Join(names, ", ") + ".\n\n")
	}
	b.WriteString("Follow the existing code style and conventions. Prefer minimal, focused changes. Explain non-obvious decisions. Ask before making destructive or wide-reaching changes.\n\n")
	b.WriteString("--- PROJECT FACTS ---\n")
	b.WriteString(facts)
	return b.String()
}

func writeFact(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("- " + label + ": " + strings.Join(items, ", ") + "\n")
}
