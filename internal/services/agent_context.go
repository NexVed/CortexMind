package services

import (
	"fmt"
	"strings"

	"github.com/NexVed/Cortex/internal/database"
)

// AgentContextService owns the project-scoped memory and task operations used
// by MCP tools. Authentication and tool input validation stay at the MCP edge.
type AgentContextService struct {
	DB     *database.DB
	Graphs CodeGraphService
}

func (s AgentContextService) GetContext(projectID string, memoryLimit int) (map[string]any, error) {
	project, err := s.DB.Project(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	if memoryLimit <= 0 || memoryLimit > 100 {
		memoryLimit = 25
	}
	memories, err := (database.RecordStore{DB: s.DB}).List("agent_memories", projectID, memoryLimit)
	if err != nil {
		return nil, err
	}
	response := map[string]any{"project": project, "memories": memories.Items}
	if graph, found, graphErr := s.Graphs.Load(projectID); graphErr == nil && found {
		response["code_graph"] = map[string]any{"built": graph.Built, "generated_at": graph.GeneratedAt, "stats": graph.Stats}
	}
	return response, nil
}

func (s AgentContextService) SaveMemory(projectID string, input map[string]any) (map[string]any, error) {
	if _, err := s.DB.Project(projectID); err != nil {
		return nil, fmt.Errorf("project not found")
	}
	content, _ := input["content"].(string)
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > 20000 {
		return nil, fmt.Errorf("content must be at most 20000 characters")
	}
	category, _ := input["category"].(string)
	if category == "" {
		category = "note"
	}
	if !map[string]bool{"context": true, "progress": true, "decision": true, "note": true, "handoff": true}[category] {
		return nil, fmt.Errorf("unsupported memory category")
	}
	input["project"] = projectID
	input["content"] = content
	input["category"] = category
	store := database.RecordStore{DB: s.DB}
	memory, err := store.Create("agent_memories", input)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(stringFrom(input["title"]))
	if title == "" {
		title = "Agent memory"
	}
	_, _ = store.Create("activity_log", map[string]any{
		"project": projectID, "action": "Saved agent " + category,
		"subject": title, "owner": stringFrom(input["owner"]),
	})
	// A full context hand-off is also durable project knowledge. Mirror it in
	// the Vault so it is visible outside the agent-memory screen, without
	// exposing any token or credential data.
	if category == "context" {
		tags := []string{"session-context"}
		if inputTags, ok := input["tags"].([]string); ok {
			tags = append(tags, inputTags...)
		}
		_, _ = store.Create("vault_entries", map[string]any{
			"project": projectID, "category": "architecture", "title": title,
			"content": content, "tags": tags, "is_shared": false,
			"source_agent": stringFrom(input["owner"]), "source_memory_id": stringFrom(memory["id"]),
		})
		_, _ = store.Create("activity_log", map[string]any{
			"project": projectID, "action": "Added session context to vault",
			"subject": title, "owner": stringFrom(input["owner"]),
		})
	}
	return memory, nil
}

func (s AgentContextService) ListMemories(projectID string, limit int) ([]map[string]any, error) {
	if _, err := s.DB.Project(projectID); err != nil {
		return nil, fmt.Errorf("project not found")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	page, err := (database.RecordStore{DB: s.DB}).List("agent_memories", projectID, limit)
	return page.Items, err
}

func (s AgentContextService) ActiveTasks(projectID string, limit int) ([]map[string]any, error) {
	if _, err := s.DB.Project(projectID); err != nil {
		return nil, fmt.Errorf("project not found")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	page, err := (database.RecordStore{DB: s.DB}).List("tasks", projectID, 100)
	if err != nil {
		return nil, err
	}
	active := make([]map[string]any, 0, len(page.Items))
	for _, task := range page.Items {
		status, _ := task["status"].(string)
		if status == "done" || status == "completed" || status == "cancelled" {
			continue
		}
		active = append(active, task)
		if len(active) == limit {
			break
		}
	}
	return active, nil
}

func (s AgentContextService) SaveSessionDigest(projectID, sessionID, title, summary string) (map[string]any, error) {
	project, err := s.DB.Project(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	if len(summary) > 30000 {
		return nil, fmt.Errorf("summary must be at most 30000 characters")
	}
	if title == "" {
		title = "Session digest" + sessionTitleSuffix(sessionID)
	}
	return (database.RecordStore{DB: s.DB}).Create("session_digests", map[string]any{
		"project": projectID, "project_id": projectID, "project_name": project.Name, "session_id": sessionID, "ide": "mcp",
		"title": title, "summary_md": summary, "digest_json": map[string]any{"session_id": sessionID, "source": "agent"},
		"provider": "agent", "token_count": estimatedTokens(summary), "memory_count": 0,
	})
}
