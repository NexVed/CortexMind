package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/db"
	"github.com/NexVed/Cortex/internal/llm"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// SessionDigestResult is returned to callers (UI + MCP) after a digest is built.
type SessionDigestResult struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	ProjectName string         `json:"project_name"`
	SessionID   string         `json:"session_id"`
	IDE         string         `json:"ide"`
	Title       string         `json:"title"`
	SummaryMD   string         `json:"summary_md"`   // Obsidian/Notion-style readable note
	DigestJSON  map[string]any `json:"digest_json"`  // compact, token-efficient agent-to-agent form
	Provider    string         `json:"provider"`     // mistral | ollama | heuristic
	TokenCount  int            `json:"token_count"`  // estimate for the compact JSON
	MemoryCount int            `json:"memory_count"` // number of memories compressed
	Created     string         `json:"created"`
}

const digestSystemInstruction = `You are compressing an AI coding agent's work session into a SHORT summary.
Write ONE dense paragraph (max 90 words) describing what the agent actually did, decided and changed this session.
Be factual and specific. No preamble, no markdown, no bullet points — just the paragraph.`

// GenerateSessionDigest compresses the agent memories of a single session into a
// readable markdown note plus a compact JSON form, and persists it. When
// sessionID is empty, the most recent memories for the project are used.
func (s *Service) GenerateSessionDigest(ctx context.Context, user *core.Record, projectID, sessionID string) (*SessionDigestResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	filter := "project = {:p}"
	params := map[string]any{"p": projectID}
	if sessionID != "" {
		filter += " && session_id = {:s}"
		params["s"] = sessionID
	}
	memories, err := s.App.FindRecordsByFilter(db.CollAgentMemories, filter, "created", 200, 0, params)
	if err != nil {
		return nil, fmt.Errorf("failed to load memories: %w", err)
	}
	if len(memories) == 0 {
		return nil, fmt.Errorf("no agent memories found to summarize for this session")
	}

	// Group memory content by category.
	var progress, decisions, notes, next []string
	ide, clientName := "", ""
	for _, m := range memories {
		if ide == "" {
			ide = m.GetString("ide")
		}
		if clientName == "" {
			clientName = m.GetString("client_name")
		}
		line := compactMemory(m)
		switch m.GetString("category") {
		case "progress":
			progress = append(progress, line)
		case "decision":
			decisions = append(decisions, line)
		case "handoff":
			next = append(next, line)
		default: // note, context, anything else
			notes = append(notes, line)
		}
	}

	// Build a factual summary paragraph (LLM if available, heuristic otherwise).
	provCfg := LoadProviderConfig(user)
	llmClient := llm.New(provCfg)
	provider := "heuristic"
	summary := ""
	if llmClient.Enabled() {
		facts := digestFacts(project.GetString("name"), progress, decisions, notes, next)
		if gen, gerr := llmClient.Summarize(ctx, digestSystemInstruction, facts); gerr == nil && strings.TrimSpace(gen) != "" {
			summary = strings.TrimSpace(gen)
			provider = provCfg.LLMProvider
		} else if gerr != nil {
			log.Warn().Err(gerr).Str("project", projectID).Msg("digest llm summary failed; using heuristic")
		}
	}
	if summary == "" {
		summary = heuristicDigestSummary(progress, decisions, notes, next)
	}

	name := project.GetString("name")
	created := time.Now().UTC()

	// Compact, token-efficient JSON (short keys) for agent-to-agent transfer.
	digestJSON := buildCompactDigest(name, sessionID, ide, created, summary, progress, decisions, notes, next)
	tokenCount := estimateDigestTokens(digestJSON)
	digestJSON["tok"] = tokenCount

	// Readable Obsidian/Notion-style markdown note.
	title := fmt.Sprintf("%s - session digest (%s)", name, created.Format("2006-01-02 15:04"))
	summaryMD := buildDigestMarkdown(name, ide, created, len(memories), tokenCount, summary, progress, decisions, notes, next)

	rec, err := s.persistSessionDigest(user, project, sessionID, ide, clientName, title, summaryMD, digestJSON, provider, tokenCount, len(memories))
	if err != nil {
		return nil, err
	}

	return &SessionDigestResult{
		ID:          rec.Id,
		ProjectID:   projectID,
		ProjectName: name,
		SessionID:   sessionID,
		IDE:         ide,
		Title:       title,
		SummaryMD:   summaryMD,
		DigestJSON:  digestJSON,
		Provider:    provider,
		TokenCount:  tokenCount,
		MemoryCount: len(memories),
		Created:     created.Format(time.RFC3339),
	}, nil
}

// ListSessionDigests returns stored digests for a project, newest first.
func (s *Service) ListSessionDigests(projectID string, limit int) ([]*SessionDigestResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	recs, err := s.App.FindRecordsByFilter(db.CollSessionDigests, "project = {:p}", "-created", limit, 0,
		map[string]any{"p": projectID})
	if err != nil {
		return nil, err
	}
	out := make([]*SessionDigestResult, 0, len(recs))
	for _, r := range recs {
		var dj map[string]any
		_ = r.UnmarshalJSONField("digest_json", &dj)
		out = append(out, &SessionDigestResult{
			ID:          r.Id,
			ProjectID:   projectID,
			SessionID:   r.GetString("session_id"),
			IDE:         r.GetString("ide"),
			Title:       r.GetString("title"),
			SummaryMD:   r.GetString("summary_md"),
			DigestJSON:  dj,
			Provider:    r.GetString("provider"),
			TokenCount:  r.GetInt("token_count"),
			MemoryCount: r.GetInt("memory_count"),
			Created:     rfc3339(r, "created"),
		})
	}
	return out, nil
}

func (s *Service) persistSessionDigest(user, project *core.Record, sessionID, ide, clientName, title, summaryMD string, digestJSON map[string]any, provider string, tokenCount, memoryCount int) (*core.Record, error) {
	// Upsert by session so re-summarizing the same session refreshes its digest.
	var existing *core.Record
	if sessionID != "" {
		existing, _ = s.App.FindFirstRecordByFilter(db.CollSessionDigests,
			"project = {:p} && session_id = {:s}", map[string]any{"p": project.Id, "s": sessionID})
	}
	var rec *core.Record
	if existing != nil {
		rec = existing
	} else {
		coll, err := s.App.FindCollectionByNameOrId(db.CollSessionDigests)
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(coll)
		rec.Set("project", project.Id)
		if user != nil {
			rec.Set("owner", user.Id)
		}
		rec.Set("session_id", sessionID)
	}
	rec.Set("ide", ide)
	rec.Set("client_name", clientName)
	rec.Set("title", title)
	rec.Set("summary_md", summaryMD)
	rec.Set("digest_json", digestJSON)
	rec.Set("provider", provider)
	rec.Set("token_count", tokenCount)
	rec.Set("memory_count", memoryCount)
	if err := s.App.Save(rec); err != nil {
		return nil, err
	}
	uid := ""
	if user != nil {
		uid = user.Id
	}
	db.LogActivity(s.App, project.Id, uid, "session_digest_created", title, map[string]any{"ide": ide, "tokens": tokenCount})
	return rec, nil
}

// ── builders ───────────────────────────────────────────

func buildCompactDigest(project, sessionID, ide string, ts time.Time, summary string, progress, decisions, notes, next []string) map[string]any {
	d := map[string]any{
		"v":    1,
		"proj": project,
		"ide":  ide,
		"ts":   ts.Format(time.RFC3339),
		"sum":  summary,
	}
	if sessionID != "" {
		d["sess"] = sessionID
	}
	if len(progress) > 0 {
		d["did"] = progress
	}
	if len(decisions) > 0 {
		d["dec"] = decisions
	}
	if len(notes) > 0 {
		d["note"] = notes
	}
	if len(next) > 0 {
		d["next"] = next
	}
	return d
}

func buildDigestMarkdown(project, ide string, ts time.Time, memCount, tokenCount int, summary string, progress, decisions, notes, next []string) string {
	var b strings.Builder
	b.WriteString("# Session Digest - " + project + "\n\n")
	ideLabel := ide
	if ideLabel == "" {
		ideLabel = "unknown"
	}
	b.WriteString(fmt.Sprintf("> %s · via %s · %d memories · ~%d tokens\n\n",
		ts.Format("2006-01-02 15:04 MST"), ideLabel, memCount, tokenCount))

	b.WriteString("## Summary\n" + summary + "\n\n")
	writeSection(&b, "What was done", progress)
	writeSection(&b, "Decisions", decisions)
	writeSection(&b, "Notes & context", notes)
	writeSection(&b, "Next steps", next)

	tag := strings.ToLower(strings.ReplaceAll(ideLabel, " ", "-"))
	b.WriteString(fmt.Sprintf("\n#session #%s", tag))
	return b.String()
}

func writeSection(b *strings.Builder, heading string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("## " + heading + "\n")
	for _, it := range items {
		b.WriteString("- " + it + "\n")
	}
	b.WriteString("\n")
}

func digestFacts(project string, progress, decisions, notes, next []string) string {
	var b strings.Builder
	b.WriteString("PROJECT: " + project + "\n")
	appendFactList(&b, "PROGRESS", progress)
	appendFactList(&b, "DECISIONS", decisions)
	appendFactList(&b, "NOTES", notes)
	appendFactList(&b, "NEXT", next)
	return b.String()
}

func appendFactList(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("\n" + label + ":\n")
	for _, it := range items {
		b.WriteString("- " + it + "\n")
	}
}

func heuristicDigestSummary(progress, decisions, notes, next []string) string {
	parts := []string{}
	if len(progress) > 0 {
		parts = append(parts, fmt.Sprintf("Made progress on %d item(s): %s", len(progress), joinTrim(progress, 2)))
	}
	if len(decisions) > 0 {
		parts = append(parts, fmt.Sprintf("Recorded %d decision(s): %s", len(decisions), joinTrim(decisions, 2)))
	}
	if len(notes) > 0 {
		parts = append(parts, fmt.Sprintf("Captured %d note(s): %s", len(notes), joinTrim(notes, 2)))
	}
	if len(next) > 0 {
		parts = append(parts, "Next: "+joinTrim(next, 2))
	}
	if len(parts) == 0 {
		return "No significant activity was recorded this session."
	}
	return strings.Join(parts, ". ") + "."
}

// compactMemory reduces a memory record to a short, information-dense line.
func compactMemory(m *core.Record) string {
	title := strings.TrimSpace(m.GetString("title"))
	content := strings.TrimSpace(m.GetString("content"))
	if title != "" && !strings.EqualFold(title, content) {
		// Include a little content when it adds information beyond the title.
		short := squash(content, 140)
		if short != "" && !strings.EqualFold(short, title) {
			return title + ": " + short
		}
		return title
	}
	return squash(content, 180)
}

// squash collapses whitespace/newlines and truncates to n characters.
func squash(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func joinTrim(items []string, max int) string {
	if len(items) > max {
		items = items[:max]
	}
	return strings.Join(items, "; ")
}

// estimateDigestTokens roughly estimates tokens for the compact JSON (~4 chars/token).
func estimateDigestTokens(d map[string]any) int {
	total := 0
	for k, v := range d {
		total += len(k) + 4
		switch val := v.(type) {
		case string:
			total += len(val)
		case []string:
			for _, s := range val {
				total += len(s) + 4
			}
		case int:
			total += 6
		}
	}
	return total / 4
}
