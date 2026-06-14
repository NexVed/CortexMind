package db

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/rs/zerolog/log"
)

// Collection name constants used across the codebase.
const (
	CollUsers         = "users"
	CollProjects      = "projects"
	CollVaultEntries  = "vault_entries"
	CollTasks         = "tasks"
	CollHandoffs      = "handoffs"
	CollScanResults   = "scan_results"
	CollFileIndex     = "file_index"
	CollSearchHistory = "search_history"
	CollActivityLog   = "activity_log"
	CollMCPTokens     = "mcp_tokens"
	CollAgentMemories = "agent_memories"
)

// EnsureCollections creates every CORTEX collection if it does not already
// exist. It is safe to call on every boot — existing collections are skipped.
func EnsureCollections(app core.App) error {
	if err := ensureUserFields(app); err != nil {
		return err
	}

	usersColl, err := app.FindCollectionByNameOrId(CollUsers)
	if err != nil {
		return err
	}
	usersID := usersColl.Id

	// projects must be created before collections that relate to it.
	if err := ensureProjects(app, usersID); err != nil {
		return err
	}

	projectsColl, err := app.FindCollectionByNameOrId(CollProjects)
	if err != nil {
		return err
	}
	projectsID := projectsColl.Id

	builders := []func(core.App, string, string) error{
		ensureVaultEntries,
		ensureTasks,
		ensureHandoffs,
		ensureScanResults,
		ensureFileIndex,
		ensureActivityLog,
		ensureMCPTokens,
		ensureAgentMemories,
	}
	for _, b := range builders {
		if err := b(app, usersID, projectsID); err != nil {
			return err
		}
	}

	if err := ensureSearchHistory(app, usersID); err != nil {
		return err
	}

	return nil
}

// ensureUserfields adds the GitHub-related fields to the built-in users
// auth collection.
func ensureUserFields(app core.App) error {
	coll, err := app.FindCollectionByNameOrId(CollUsers)
	if err != nil {
		log.Warn().Msg("users collection not found; skipping field extension")
		return nil
	}

	changed := false
	add := func(f core.Field) {
		if coll.Fields.GetByName(f.GetName()) == nil {
			coll.Fields.Add(f)
			changed = true
		}
	}

	add(&core.TextField{Name: "github_access_token", Hidden: true})
	add(&core.TextField{Name: "github_username"})
	add(&core.URLField{Name: "github_avatar_url"})
	add(&core.TextField{Name: "display_name"})
	add(&core.JSONField{Name: "preferences", MaxSize: 2000000})

	if !changed {
		return nil
	}
	return app.Save(coll)
}

func ownerRelation(collectionID string) *core.RelationField {
	return &core.RelationField{
		Name:          "owner",
		CollectionId:  collectionID,
		CascadeDelete: false,
		MaxSelect:     1,
	}
}

func projectRelation(collectionID string) *core.RelationField {
	return &core.RelationField{
		Name:          "project",
		CollectionId:  collectionID,
		CascadeDelete: true,
		MaxSelect:     1,
	}
}

func timestamps(c *core.Collection) {
	c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
}

// newBase returns a base collection with owner-scoped CRUD rules already set,
// or nil if the collection already exists.
func newBase(app core.App, name string) *core.Collection {
	if _, err := app.FindCollectionByNameOrId(name); err == nil {
		return nil // already exists
	}
	c := core.NewBaseCollection(name)
	ownerRule := "@request.auth.id != '' && @request.auth.id = owner"
	c.ListRule = types.Pointer(ownerRule)
	c.ViewRule = types.Pointer(ownerRule)
	c.CreateRule = types.Pointer("@request.auth.id != ''")
	c.UpdateRule = types.Pointer(ownerRule)
	c.DeleteRule = types.Pointer(ownerRule)
	return c
}

// newProjectScoped returns a base collection scoped through its project's owner
// (for collections that have no direct owner field, e.g. scanner outputs).
// Writes are reserved for the daemon (superusers) since these are populated
// programmatically via app.Save which bypasses API rules.
func newProjectScoped(app core.App, name string) *core.Collection {
	if _, err := app.FindCollectionByNameOrId(name); err == nil {
		return nil // already exists
	}
	c := core.NewBaseCollection(name)
	readRule := "@request.auth.id != '' && project.owner = @request.auth.id"
	c.ListRule = types.Pointer(readRule)
	c.ViewRule = types.Pointer(readRule)
	// CreateRule/UpdateRule/DeleteRule left nil => superuser-only.
	return c
}

func ensureProjects(app core.App, usersID string) error {
	c := newBase(app, CollProjects)
	if c == nil {
		return nil
	}
	c.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 200})
	c.Fields.Add(&core.TextField{Name: "path", Max: 1000})
	c.Fields.Add(&core.TextField{Name: "description", Max: 5000})
	c.Fields.Add(&core.URLField{Name: "github_url"})
	c.Fields.Add(&core.TextField{Name: "github_repo_id"})
	c.Fields.Add(ownerRelation(usersID))
	c.Fields.Add(&core.SelectField{Name: "status", Values: []string{"active", "archived", "scanning"}, MaxSelect: 1})
	c.Fields.Add(&core.NumberField{Name: "progress"})
	c.Fields.Add(&core.JSONField{Name: "technologies", MaxSize: 200000})
	c.Fields.Add(&core.DateField{Name: "last_scanned"})
	c.Fields.Add(&core.DateField{Name: "last_activity"})
	c.Fields.Add(&core.TextField{Name: "icon_color"})
	c.Fields.Add(&core.JSONField{Name: "metadata", MaxSize: 2000000})
	timestamps(c)
	c.AddIndex("idx_projects_owner", false, "owner", "")
	return app.Save(c)
}

func ensureVaultEntries(app core.App, usersID, projectsID string) error {
	c := newBase(app, CollVaultEntries)
	if c == nil {
		return nil
	}
	c.Fields.Add(projectRelation(projectsID))
	c.Fields.Add(ownerRelation(usersID))
	c.Fields.Add(&core.SelectField{Name: "category", Values: []string{"architecture", "decision", "roadmap", "task", "handoff", "memory"}, MaxSelect: 1})
	c.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 300})
	c.Fields.Add(&core.EditorField{Name: "content"})
	c.Fields.Add(&core.JSONField{Name: "tags", MaxSize: 100000})
	c.Fields.Add(&core.BoolField{Name: "is_shared"})
	c.Fields.Add(&core.TextField{Name: "source_agent"})
	c.Fields.Add(&core.TextField{Name: "file_path", Max: 1000})
	c.Fields.Add(&core.NumberField{Name: "version"})
	timestamps(c)
	c.AddIndex("idx_vault_project", false, "project", "")
	c.AddIndex("idx_vault_category", false, "category", "")
	return app.Save(c)
}

func ensureTasks(app core.App, usersID, projectsID string) error {
	c := newBase(app, CollTasks)
	if c == nil {
		return nil
	}
	c.Fields.Add(projectRelation(projectsID))
	c.Fields.Add(ownerRelation(usersID))
	c.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 300})
	c.Fields.Add(&core.EditorField{Name: "description"})
	c.Fields.Add(&core.SelectField{Name: "status", Values: []string{"todo", "in_progress", "done"}, MaxSelect: 1})
	c.Fields.Add(&core.SelectField{Name: "priority", Values: []string{"low", "medium", "high"}, MaxSelect: 1})
	c.Fields.Add(&core.TextField{Name: "assigned_to"})
	c.Fields.Add(&core.DateField{Name: "due_date"})
	c.Fields.Add(&core.JSONField{Name: "linked_files", MaxSize: 200000})
	c.Fields.Add(&core.EditorField{Name: "ai_notes"})
	c.Fields.Add(&core.JSONField{Name: "tags", MaxSize: 100000})
	timestamps(c)
	c.AddIndex("idx_tasks_project", false, "project", "")
	c.AddIndex("idx_tasks_status", false, "status", "")
	return app.Save(c)
}

func ensureHandoffs(app core.App, usersID, projectsID string) error {
	c := newBase(app, CollHandoffs)
	if c == nil {
		return nil
	}
	c.Fields.Add(projectRelation(projectsID))
	c.Fields.Add(ownerRelation(usersID))
	c.Fields.Add(&core.TextField{Name: "from_agent"})
	c.Fields.Add(&core.TextField{Name: "to_agent"})
	c.Fields.Add(&core.TextField{Name: "title", Max: 300})
	c.Fields.Add(&core.EditorField{Name: "context"})
	c.Fields.Add(&core.SelectField{Name: "status", Values: []string{"draft", "active", "consumed"}, MaxSelect: 1})
	c.Fields.Add(&core.JSONField{Name: "included_files", MaxSize: 200000})
	c.Fields.Add(&core.JSONField{Name: "decision_refs", MaxSize: 200000})
	c.Fields.Add(&core.EditorField{Name: "prompt_preview"})
	c.Fields.Add(&core.NumberField{Name: "token_count"})
	timestamps(c)
	c.AddIndex("idx_handoffs_project", false, "project", "")
	c.AddIndex("idx_handoffs_to_agent", false, "to_agent", "")
	return app.Save(c)
}

func ensureScanResults(app core.App, usersID, projectsID string) error {
	c := newProjectScoped(app, CollScanResults)
	if c == nil {
		return nil
	}
	c.Fields.Add(projectRelation(projectsID))
	c.Fields.Add(&core.DateField{Name: "scanned_at"})
	c.Fields.Add(&core.NumberField{Name: "total_files"})
	c.Fields.Add(&core.NumberField{Name: "indexed_files"})
	c.Fields.Add(&core.JSONField{Name: "languages", MaxSize: 500000})
	c.Fields.Add(&core.JSONField{Name: "modules", MaxSize: 1000000})
	c.Fields.Add(&core.JSONField{Name: "dependencies", MaxSize: 1000000})
	c.Fields.Add(&core.JSONField{Name: "entry_points", MaxSize: 200000})
	c.Fields.Add(&core.JSONField{Name: "api_endpoints", MaxSize: 500000})
	c.Fields.Add(&core.EditorField{Name: "summary"})
	timestamps(c)
	c.AddIndex("idx_scan_project", false, "project", "")
	return app.Save(c)
}

func ensureFileIndex(app core.App, usersID, projectsID string) error {
	c := newProjectScoped(app, CollFileIndex)
	if c == nil {
		return nil
	}
	c.Fields.Add(projectRelation(projectsID))
	c.Fields.Add(&core.TextField{Name: "path", Required: true, Max: 1000})
	c.Fields.Add(&core.TextField{Name: "language"})
	c.Fields.Add(&core.NumberField{Name: "size_bytes"})
	c.Fields.Add(&core.JSONField{Name: "functions", MaxSize: 2000000})
	c.Fields.Add(&core.JSONField{Name: "classes", MaxSize: 1000000})
	c.Fields.Add(&core.JSONField{Name: "imports", MaxSize: 500000})
	c.Fields.Add(&core.TextField{Name: "summary", Max: 2000})
	c.Fields.Add(&core.TextField{Name: "embedding_id"})
	c.Fields.Add(&core.DateField{Name: "last_indexed"})
	c.Fields.Add(&core.TextField{Name: "checksum"})
	timestamps(c)
	c.AddIndex("idx_file_project_path", true, "project,path", "")
	c.AddIndex("idx_file_language", false, "language", "")
	return app.Save(c)
}

func ensureSearchHistory(app core.App, usersID string) error {
	c := newBase(app, CollSearchHistory)
	if c == nil {
		return nil
	}
	c.Fields.Add(ownerRelation(usersID))
	c.Fields.Add(&core.TextField{Name: "query", Max: 1000})
	c.Fields.Add(&core.JSONField{Name: "scope", MaxSize: 100000})
	c.Fields.Add(&core.NumberField{Name: "results"})
	timestamps(c)
	return app.Save(c)
}

func ensureActivityLog(app core.App, usersID, projectsID string) error {
	c := newBase(app, CollActivityLog)
	if c == nil {
		return nil
	}
	c.Fields.Add(projectRelation(projectsID))
	c.Fields.Add(ownerRelation(usersID))
	c.Fields.Add(&core.TextField{Name: "action"})
	c.Fields.Add(&core.TextField{Name: "subject", Max: 1000})
	c.Fields.Add(&core.JSONField{Name: "metadata", MaxSize: 500000})
	timestamps(c)
	c.AddIndex("idx_activity_project", false, "project", "")
	return app.Save(c)
}

// ensureMCPTokens stores per-IDE MCP connection credentials. Each token binds
// an IDE connection to a user and (optionally) a specific project, so the MCP
// server can identify who and what is connecting.
func ensureMCPTokens(app core.App, usersID, projectsID string) error {
	if _, err := app.FindCollectionByNameOrId(CollMCPTokens); err == nil {
		return nil
	}
	c := core.NewBaseCollection(CollMCPTokens)
	ownerRule := "@request.auth.id != '' && @request.auth.id = owner"
	c.ListRule = types.Pointer(ownerRule)
	c.ViewRule = types.Pointer(ownerRule)
	c.CreateRule = types.Pointer("@request.auth.id != ''")
	c.UpdateRule = types.Pointer(ownerRule)
	c.DeleteRule = types.Pointer(ownerRule)

	c.Fields.Add(ownerRelation(usersID))
	projRel := projectRelation(projectsID)
	projRel.CascadeDelete = true
	c.Fields.Add(projRel)
	c.Fields.Add(&core.TextField{Name: "ide"})         // e.g. cursor, claude, windsurf, vscode
	c.Fields.Add(&core.TextField{Name: "label", Max: 200})
	c.Fields.Add(&core.TextField{Name: "token", Hidden: true, Required: true})
	c.Fields.Add(&core.TextField{Name: "client_name"}) // reported by the IDE on initialize
	c.Fields.Add(&core.BoolField{Name: "enabled"})
	c.Fields.Add(&core.DateField{Name: "last_used"})
	timestamps(c)
	c.AddIndex("idx_mcp_token", true, "token", "")
	c.AddIndex("idx_mcp_owner", false, "owner", "")
	return app.Save(c)
}

// ensureAgentMemories captures what an AI agent learned/did while working on a
// project through an IDE, so the next session can be primed with that memory.
func ensureAgentMemories(app core.App, usersID, projectsID string) error {
	c := newBase(app, CollAgentMemories)
	if c == nil {
		return nil
	}
	c.Fields.Add(projectRelation(projectsID))
	c.Fields.Add(ownerRelation(usersID))
	c.Fields.Add(&core.TextField{Name: "ide"})
	c.Fields.Add(&core.TextField{Name: "client_name"})
	c.Fields.Add(&core.TextField{Name: "session_id"})
	c.Fields.Add(&core.SelectField{Name: "category", Values: []string{"context", "progress", "decision", "note", "handoff"}, MaxSelect: 1})
	c.Fields.Add(&core.TextField{Name: "title", Max: 300})
	c.Fields.Add(&core.EditorField{Name: "content"})
	c.Fields.Add(&core.JSONField{Name: "tags", MaxSize: 100000})
	timestamps(c)
	c.AddIndex("idx_agentmem_project", false, "project", "")
	c.AddIndex("idx_agentmem_session", false, "session_id", "")
	return app.Save(c)
}
