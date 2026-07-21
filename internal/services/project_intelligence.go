package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/database"
)

// ProjectIntelligenceService provides the read models consumed by the Digests,
// Agents, and Agent Memory screens. All data stays in local SQLite records.
type ProjectIntelligenceService struct{ DB *database.DB }

func (s ProjectIntelligenceService) ListAgentMemories(projectID string, limit int) ([]map[string]any, error) {
	if _, err := s.DB.Project(projectID); err != nil {
		return nil, fmt.Errorf("project not found")
	}
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	page, err := (database.RecordStore{DB: s.DB}).List("agent_memories", projectID, limit)
	return page.Items, err
}

func (s ProjectIntelligenceService) GenerateDigest(projectID, sessionID string) (map[string]any, error) {
	project, err := s.DB.Project(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	memories, err := s.ListAgentMemories(projectID, 500)
	if err != nil {
		return nil, err
	}
	selected := make([]map[string]any, 0, len(memories))
	for _, memory := range memories {
		memorySession, _ := memory["session_id"].(string)
		if sessionID == "" || memorySession == sessionID {
			selected = append(selected, memory)
		}
	}
	if sessionID == "" && len(selected) > 0 {
		sessionID, _ = selected[0]["session_id"].(string)
	}
	ide := "local"
	if len(selected) > 0 {
		if value, ok := selected[0]["ide"].(string); ok && value != "" {
			ide = value
		}
	}
	lines := []string{"# Session digest", "", "## Summary"}
	compact := make([]map[string]any, 0, len(selected))
	for _, memory := range selected {
		category, _ := memory["category"].(string)
		title, _ := memory["title"].(string)
		content, _ := memory["content"].(string)
		if title == "" {
			title = truncateText(content, 120)
		}
		lines = append(lines, fmt.Sprintf("- **%s** — %s", category, title))
		compact = append(compact, map[string]any{"category": category, "title": title, "content": truncateText(content, 300), "agent": agentFromMemory(memory)})
	}
	if len(selected) == 0 {
		lines = append(lines, "- No agent memories have been recorded for this session yet.")
	}
	summary := strings.Join(lines, "\n")
	digest := map[string]any{
		"project": projectID, "project_id": projectID, "project_name": project.Name, "session_id": sessionID, "ide": ide,
		"title": "Session digest" + sessionTitleSuffix(sessionID), "summary_md": summary,
		"digest_json": map[string]any{"session_id": sessionID, "generated_from": len(selected), "memories": compact},
		"provider":    "heuristic", "token_count": estimatedTokens(summary), "memory_count": len(selected),
	}
	return (database.RecordStore{DB: s.DB}).Create("session_digests", digest)
}

func (s ProjectIntelligenceService) ListDigests(projectID string) ([]map[string]any, error) {
	project, err := s.DB.Project(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	page, err := (database.RecordStore{DB: s.DB}).List("session_digests", projectID, 200)
	if err != nil {
		return nil, err
	}
	for _, digest := range page.Items {
		digest["project_id"] = projectID
		digest["project_name"] = project.Name
		if _, ok := digest["summary_md"].(string); !ok {
			digest["summary_md"], _ = digest["content"].(string)
		}
		if _, ok := digest["digest_json"]; !ok {
			digest["digest_json"] = map[string]any{}
		}
		if _, ok := digest["provider"].(string); !ok {
			digest["provider"] = "heuristic"
		}
		if _, ok := digest["memory_count"]; !ok {
			digest["memory_count"] = 0
		}
		if _, ok := digest["token_count"]; !ok {
			digest["token_count"] = estimatedTokens(stringFrom(digest["summary_md"]))
		}
	}
	return page.Items, nil
}

func (s ProjectIntelligenceService) ActiveAgents(projectID string) ([]map[string]any, error) {
	memories, err := s.ListAgentMemories(projectID, 500)
	if err != nil {
		return nil, err
	}
	type aggregate struct {
		Name, LastSeen string
		Memories       int
		Sessions       map[string]bool
	}
	byName := map[string]*aggregate{}
	for _, memory := range memories {
		name := agentFromMemory(memory)
		if name == "" {
			continue
		}
		item := byName[name]
		if item == nil {
			item = &aggregate{Name: name, Sessions: map[string]bool{}}
			byName[name] = item
		}
		item.Memories++
		when := stringFrom(memory["updated"])
		if when == "" {
			when = stringFrom(memory["created"])
		}
		if when > item.LastSeen {
			item.LastSeen = when
		}
		if session := stringFrom(memory["session_id"]); session != "" {
			item.Sessions[session] = true
		}
	}
	result := make([]map[string]any, 0, len(byName))
	for _, item := range byName {
		result = append(result, map[string]any{"name": item.Name, "status": agentStatus(item.LastSeen), "last_seen": item.LastSeen, "memory_count": item.Memories, "session_count": len(item.Sessions)})
	}
	sort.Slice(result, func(i, j int) bool { return stringFrom(result[i]["last_seen"]) > stringFrom(result[j]["last_seen"]) })
	return result, nil
}

func agentFromMemory(memory map[string]any) string {
	for _, field := range []string{"owner", "agent", "client_name", "ide"} {
		if value := strings.TrimSpace(stringFrom(memory[field])); value != "" {
			return value
		}
	}
	return ""
}
func agentStatus(lastSeen string) string {
	when, err := time.Parse(time.RFC3339, lastSeen)
	if err != nil {
		return "offline"
	}
	age := time.Since(when)
	if age < time.Hour {
		return "active"
	}
	if age < 24*time.Hour {
		return "connected"
	}
	return "idle"
}
func truncateText(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max-1] + "…"
}
func estimatedTokens(value string) int {
	if value == "" {
		return 0
	}
	return (len(value) + 3) / 4
}
func stringFrom(value any) string { text, _ := value.(string); return text }
func sessionTitleSuffix(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	return " · " + sessionID
}

// SystemPrompt returns the project's persisted instructions. It intentionally
// returns an empty prompt for first use so the Context page is usable before a
// user has authored or generated any instructions.
func (s ProjectIntelligenceService) SystemPrompt(projectID string) (map[string]any, error) {
	project, err := s.DB.Project(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	page, err := (database.RecordStore{DB: s.DB}).List("system_prompts", projectID, 1)
	if err != nil {
		return nil, err
	}
	result := map[string]any{"project_id": projectID, "project_name": project.Name, "provider": "custom", "prompt": "", "token_estimate": 0}
	if len(page.Items) == 0 {
		return result, nil
	}
	item := page.Items[0]
	result["prompt"] = stringFrom(item["prompt"])
	if provider := stringFrom(item["provider"]); provider != "" {
		result["provider"] = provider
	}
	result["token_estimate"] = estimatedTokens(stringFrom(result["prompt"]))
	return result, nil
}

func (s ProjectIntelligenceService) SaveSystemPrompt(projectID, prompt string) (map[string]any, error) {
	if _, err := s.DB.Project(projectID); err != nil {
		return nil, fmt.Errorf("project not found")
	}
	prompt = strings.TrimSpace(prompt)
	store := database.RecordStore{DB: s.DB}
	page, err := store.List("system_prompts", projectID, 1)
	if err != nil {
		return nil, err
	}
	if len(page.Items) > 0 {
		if _, err = store.Update("system_prompts", stringFrom(page.Items[0]["id"]), map[string]any{"prompt": prompt, "provider": "custom"}); err != nil {
			return nil, err
		}
	} else if _, err = store.Create("system_prompts", map[string]any{"project": projectID, "prompt": prompt, "provider": "custom"}); err != nil {
		return nil, err
	}
	return s.SystemPrompt(projectID)
}

// GenerateSystemPrompt builds a local baseline. It includes the most recent
// session-context memory so agents get the full hand-off without credentials
// or remote services.
func (s ProjectIntelligenceService) GenerateSystemPrompt(projectID string, includeTasks, includeVault, includeActivity bool) (map[string]any, error) {
	project, err := s.DB.Project(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	lines := []string{"# " + project.Name + " — AI coding-agent context", "", "You are working locally in this project. Preserve existing behavior, make focused changes, and verify your work."}
	memories, err := s.ListAgentMemories(projectID, 500)
	if err != nil {
		return nil, err
	}
	for _, memory := range memories {
		if stringFrom(memory["category"]) == "context" {
			lines = append(lines, "", "## Latest session context", stringFrom(memory["content"]))
			break
		}
	}
	if includeTasks {
		page, listErr := (database.RecordStore{DB: s.DB}).List("tasks", projectID, 25)
		if listErr == nil && len(page.Items) > 0 {
			lines = append(lines, "", "## Active tasks")
			for _, task := range page.Items {
				if stringFrom(task["status"]) != "done" {
					lines = append(lines, "- "+stringFrom(task["title"]))
				}
			}
		}
	}
	if includeVault {
		page, listErr := (database.RecordStore{DB: s.DB}).List("vault_entries", projectID, 20)
		if listErr == nil && len(page.Items) > 0 {
			lines = append(lines, "", "## Saved project knowledge")
			for _, item := range page.Items {
				lines = append(lines, "- "+stringFrom(item["title"]))
			}
		}
	}
	if includeActivity {
		lines = append(lines, "", "## Recent activity", "Review current files and git status before relying on previous activity.")
	}
	prompt := strings.Join(lines, "\n")
	return map[string]any{"project_id": projectID, "project_name": project.Name, "provider": "cortex", "prompt": prompt, "token_estimate": estimatedTokens(prompt)}, nil
}
