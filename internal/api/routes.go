package api

import (
	"fmt"

	"github.com/NexVed/Cortex/internal/db"
	"github.com/NexVed/Cortex/internal/llm"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes mounts the CORTEX JSON endpoints on the PocketBase router.
func (s *Service) RegisterRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/cortex/scan/{project}", s.handleScanProject)
	se.Router.POST("/api/cortex/system-prompt/{project}", s.handleSystemPrompt)
	se.Router.GET("/api/cortex/providers", s.handleGetProviders)
	se.Router.POST("/api/cortex/providers", s.handleSetProviders)
	se.Router.GET("/api/cortex/knowledge-graph/{project}", s.handleKnowledgeGraph)

	// GitHub repository listing/import (uses the stored access token, fully
	// paginated so all repos are visible — not just the first page).
	se.Router.GET("/api/cortex/github/repos", s.handleListGitHubRepos)
	se.Router.POST("/api/cortex/github/sync", s.handleSyncGitHubRepos)

	// Portable memory bundle export (.cortex/memory.json + README.md). Pass
	// ?push=true to commit AND push it to GitHub for teammates.
	se.Router.POST("/api/cortex/export/{project}", s.handleExportMemory)

	// Session digests (compressed agent-session summaries).
	se.Router.POST("/api/cortex/session-digest/{project}", s.handleGenerateDigest)
	se.Router.GET("/api/cortex/session-digests/{project}", s.handleListDigests)

	// Code graph (codebase memory: structure + dependency graph).
	se.Router.POST("/api/cortex/code-graph/{project}", s.handleBuildCodeGraph)
	se.Router.GET("/api/cortex/code-graph/{project}", s.handleGetCodeGraph)

	// MCP connection management (per-IDE authorization tokens).
	se.Router.GET("/api/cortex/mcp/connections", s.handleListConnections)
	se.Router.POST("/api/cortex/mcp/connections", s.handleCreateConnection)
	se.Router.DELETE("/api/cortex/mcp/connections/{id}", s.handleDeleteConnection)
	se.Router.GET("/api/cortex/mcp/connections/{id}/status", s.handleConnectionStatus)
}

// resolveUser returns the acting user: the authenticated record when present,
// otherwise (local single-user mode) the first user that has a GitHub token.
func (s *Service) resolveUser(e *core.RequestEvent) (*core.Record, error) {
	if e.Auth != nil && e.Auth.Collection().Name == db.CollUsers {
		return e.Auth, nil
	}
	recs, err := s.App.FindRecordsByFilter(db.CollUsers, "github_access_token != ''", "-updated", 1, 0, nil)
	if err == nil && len(recs) > 0 {
		return recs[0], nil
	}
	// Fall back to any user so provider config still works without GitHub.
	recs, err = s.App.FindRecordsByFilter(db.CollUsers, "id != ''", "-updated", 1, 0, nil)
	if err != nil || len(recs) == 0 {
		return nil, fmt.Errorf("no user found; sign in first")
	}
	return recs[0], nil
}

func (s *Service) handleScanProject(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	projectID := e.Request.PathValue("project")
	res, err := s.ScanProjectDeep(e.Request.Context(), user, projectID)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, res)
}

func (s *Service) handleSystemPrompt(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	projectID := e.Request.PathValue("project")
	var opts PromptOptions
	if err := e.BindBody(&opts); err != nil {
		// Default to including everything when no body is provided.
		opts = PromptOptions{IncludeTasks: true, IncludeVault: true}
	}
	res, err := s.GenerateSystemPrompt(e.Request.Context(), user, projectID, opts)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, res)
}

func (s *Service) handleGetProviders(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.JSON(200, llm.ProviderConfig{}.Redacted())
	}
	cfg := LoadProviderConfig(user)
	return e.JSON(200, cfg.Redacted())
}

func (s *Service) handleSetProviders(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	var incoming llm.ProviderConfig
	if err := e.BindBody(&incoming); err != nil {
		return e.BadRequestError("invalid provider config", err)
	}
	if err := SaveProviderConfig(s.App, user, incoming); err != nil {
		return e.InternalServerError("failed to save provider config", err)
	}
	return e.JSON(200, LoadProviderConfig(user).Redacted())
}

func (s *Service) handleKnowledgeGraph(e *core.RequestEvent) error {
	projectID := e.Request.PathValue("project")
	rec, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	var meta map[string]any
	if err := rec.UnmarshalJSONField("metadata", &meta); err != nil || meta == nil {
		return e.JSON(200, map[string]any{"nodes": []any{}, "edges": []any{}})
	}
	graph, ok := meta["graph"]
	if !ok {
		return e.JSON(200, map[string]any{"nodes": []any{}, "edges": []any{}})
	}
	return e.JSON(200, graph)
}

// ── Session digests ────────────────────────────────────

func (s *Service) handleGenerateDigest(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	projectID := e.Request.PathValue("project")
	var body struct {
		SessionID string `json:"session_id"`
	}
	_ = e.BindBody(&body)
	res, err := s.GenerateSessionDigest(e.Request.Context(), user, projectID, body.SessionID)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, res)
}

func (s *Service) handleListDigests(e *core.RequestEvent) error {
	projectID := e.Request.PathValue("project")
	res, err := s.ListSessionDigests(projectID, 50)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, res)
}

// ── Code graph ─────────────────────────────────────────

func (s *Service) handleBuildCodeGraph(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	projectID := e.Request.PathValue("project")
	res, err := s.BuildAndStoreCodeGraph(e.Request.Context(), user, projectID)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, res)
}

func (s *Service) handleGetCodeGraph(e *core.RequestEvent) error {
	projectID := e.Request.PathValue("project")
	res, err := s.GetCodeGraph(projectID)
	if err != nil {
		return e.NotFoundError(err.Error(), nil)
	}
	return e.JSON(200, res)
}

// ── MCP connection management ──────────────────────────

func (s *Service) handleListConnections(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.JSON(200, []any{})
	}
	recs, err := s.App.FindRecordsByFilter(db.CollMCPTokens, "owner = {:o}", "-created", 100, 0,
		map[string]any{"o": user.Id})
	if err != nil {
		return e.InternalServerError("failed to list connections", err)
	}
	out := make([]*MCPConnection, 0, len(recs))
	for _, r := range recs {
		out = append(out, s.connectionFromRecord(r))
	}
	return e.JSON(200, out)
}

func (s *Service) handleCreateConnection(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	var body struct {
		ProjectID string `json:"project_id"`
		IDE       string `json:"ide"`
		Label     string `json:"label"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("invalid body", err)
	}
	if body.IDE == "" {
		body.IDE = "generic"
	}
	conn, err := s.CreateMCPConnection(user, body.ProjectID, body.IDE, body.Label)
	if err != nil {
		return e.InternalServerError("failed to create connection", err)
	}
	return e.JSON(200, conn)
}

func (s *Service) handleDeleteConnection(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	id := e.Request.PathValue("id")
	rec, err := s.App.FindRecordById(db.CollMCPTokens, id)
	if err != nil {
		return e.NotFoundError("connection not found", err)
	}
	if rec.GetString("owner") != user.Id {
		return e.ForbiddenError("not your connection", nil)
	}
	if err := s.App.Delete(rec); err != nil {
		return e.InternalServerError("failed to delete", err)
	}
	return e.JSON(200, map[string]any{"success": true})
}

func (s *Service) handleConnectionStatus(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	rec, err := s.App.FindRecordById(db.CollMCPTokens, id)
	if err != nil {
		return e.NotFoundError("connection not found", err)
	}
	return e.JSON(200, s.connectionFromRecord(rec))
}
