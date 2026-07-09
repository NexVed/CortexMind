package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/NexVed/Cortex/internal/api"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
)

// ── Tool definitions ───────────────────────────────────

func (s *Server) toolsList() map[string]any {
	return map[string]any{
		"tools": []map[string]any{
			{
				"name":        "cortex_get_context",
				"description": "Load this project's characterization (its generated system prompt, tech stack and architecture) plus the memory of previous AI sessions. Call this FIRST when starting work on the project.",
				"inputSchema": map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
			{
				"name":        "cortex_save_memory",
				"description": "Persist a memory for this project (progress, a decision, a note or context) so the next AI session — in any IDE — can recall it. Recorded with the current IDE and session.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":    map[string]any{"type": "string", "description": "Short title for the memory"},
						"content":  map[string]any{"type": "string", "description": "The memory content to remember"},
						"category": map[string]any{"type": "string", "enum": []string{"context", "progress", "decision", "note", "handoff"}, "description": "Type of memory"},
						"tags":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required": []string{"content"},
				},
			},
			{
				"name":        "cortex_list_memories",
				"description": "List the stored memories for this project from all previous AI sessions and IDEs.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"limit":    map[string]any{"type": "number", "description": "Max memories to return (default 30)"},
						"category": map[string]any{"type": "string", "description": "Filter by category"},
					},
				},
			},
			{
				"name":        "cortex_get_tasks",
				"description": "List the active (not done) tasks for this project.",
				"inputSchema": map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
			{
				"name":        "cortex_summarize_session",
				"description": "Compress everything YOU did in this session (all memories you saved) into a stored session digest: a readable markdown note plus a compact, token-efficient JSON. Call this at the END of your session so the next agent can catch up fast. Returns the compact JSON.",
				"inputSchema": map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
			{
				"name":        "cortex_get_session_digests",
				"description": "Get compact, token-efficient JSON digests of previous work sessions for this project. Use this to catch up quickly on what other agents did without reading full memory. Much cheaper than cortex_get_context.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"limit": map[string]any{"type": "number", "description": "Max digests to return (default 5)"},
					},
				},
			},
			{
				"name":        "cortex_recall_code",
				"description": "Recall where a symbol, file or module lives in this project's codebase graph, and its relationships: what defines it, what it depends on, and what depends on it. Use this instead of grepping to understand code structure fast.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string", "description": "A function, class, file or directory name to look up"},
					},
					"required": []string{"query"},
				},
			},
			{
				"name":        "cortex_code_map",
				"description": "Get a compact high-level map of the project's code structure from the codebase graph: module directories, file counts, symbol totals and dependency counts. Call this to orient before diving in.",
				"inputSchema": map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
	}
}

func (s *Server) toolsCall(auth *authContext, req *rpcRequest) (any, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params"}
	}

	switch p.Name {
	case "cortex_get_context":
		return s.textResult(s.buildContext(auth)), nil
	case "cortex_save_memory":
		return s.saveMemory(auth, p.Arguments)
	case "cortex_list_memories":
		return s.listMemories(auth, p.Arguments)
	case "cortex_get_tasks":
		return s.getTasks(auth)
	case "cortex_summarize_session":
		return s.summarizeSession(auth)
	case "cortex_get_session_digests":
		return s.getSessionDigests(auth, p.Arguments)
	case "cortex_recall_code":
		return s.recallCode(auth, p.Arguments)
	case "cortex_code_map":
		return s.codeMap(auth)
	default:
		return nil, &rpcError{Code: -32602, Message: "unknown tool: " + p.Name}
	}
}

// textResult wraps text as an MCP tool result.
func (s *Server) textResult(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}
}

func (s *Server) errResult(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": true,
	}
}

// ── Tool implementations ───────────────────────────────

func (s *Server) saveMemory(auth *authContext, raw json.RawMessage) (any, *rpcError) {
	if auth.project == nil {
		return s.errResult("This MCP connection is not bound to a project. Re-create the connection in CORTEX with a project selected."), nil
	}
	var args struct {
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	_ = json.Unmarshal(raw, &args)
	if strings.TrimSpace(args.Content) == "" {
		return s.errResult("content is required"), nil
	}
	if args.Category == "" {
		args.Category = "note"
	}
	if args.Title == "" {
		args.Title = firstLine(args.Content, 60)
	}

	sess := s.sessionFor(auth)
	coll, err := s.App.FindCollectionByNameOrId(db.CollAgentMemories)
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	rec := core.NewRecord(coll)
	rec.Set("project", auth.project.Id)
	rec.Set("owner", auth.ownerID)
	rec.Set("ide", sess.IDE)
	rec.Set("client_name", sess.ClientName)
	rec.Set("session_id", sess.ID)
	rec.Set("category", args.Category)
	rec.Set("title", args.Title)
	rec.Set("content", args.Content)
	rec.Set("tags", args.Tags)
	if err := s.App.Save(rec); err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	db.LogActivity(s.App, auth.project.Id, auth.ownerID, "agent_memory_saved",
		fmt.Sprintf("%s via %s: %s", args.Category, displayIDE(sess), args.Title), nil)

	return s.textResult(fmt.Sprintf("Saved memory %q (%s) for project %s.", args.Title, args.Category, auth.project.GetString("name"))), nil
}

func (s *Server) listMemories(auth *authContext, raw json.RawMessage) (any, *rpcError) {
	if auth.project == nil {
		return s.errResult("connection not bound to a project"), nil
	}
	var args struct {
		Limit    int    `json:"limit"`
		Category string `json:"category"`
	}
	_ = json.Unmarshal(raw, &args)
	if args.Limit <= 0 || args.Limit > 200 {
		args.Limit = 30
	}

	filter := "project = {:p}"
	params := map[string]any{"p": auth.project.Id}
	if args.Category != "" {
		filter += " && category = {:c}"
		params["c"] = args.Category
	}
	recs, err := s.App.FindRecordsByFilter(db.CollAgentMemories, filter, "-created", args.Limit, 0, params)
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	if len(recs) == 0 {
		return s.textResult("No memories stored for this project yet."), nil
	}
	var b strings.Builder
	for _, r := range recs {
		b.WriteString(fmt.Sprintf("• [%s] %s (via %s, %s)\n  %s\n",
			r.GetString("category"), r.GetString("title"),
			r.GetString("client_name"), r.GetString("ide"),
			truncate(r.GetString("content"), 500)))
	}
	return s.textResult(b.String()), nil
}

func (s *Server) getTasks(auth *authContext) (any, *rpcError) {
	if auth.project == nil {
		return s.errResult("connection not bound to a project"), nil
	}
	recs, err := s.App.FindRecordsByFilter(db.CollTasks, "project = {:p} && status != 'done'", "-updated", 50, 0,
		map[string]any{"p": auth.project.Id})
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	if len(recs) == 0 {
		return s.textResult("No active tasks."), nil
	}
	var b strings.Builder
	for _, r := range recs {
		b.WriteString(fmt.Sprintf("- [%s] %s\n", r.GetString("status"), r.GetString("title")))
	}
	return s.textResult(b.String()), nil
}

// ── Session digests ────────────────────────────────────

func (s *Server) summarizeSession(auth *authContext) (any, *rpcError) {
	if auth.project == nil {
		return s.errResult("connection not bound to a project"), nil
	}
	sess := s.sessionFor(auth)
	user, _ := s.App.FindRecordById(db.CollUsers, auth.ownerID)

	svc := &api.Service{App: s.App}
	res, err := svc.GenerateSessionDigest(context.Background(), user, auth.project.Id, sess.ID)
	if err != nil {
		return s.errResult("could not summarize session: " + err.Error()), nil
	}

	compact, _ := json.Marshal(res.DigestJSON)
	msg := fmt.Sprintf("Stored session digest %q (%d memories, ~%d tokens, provider=%s).\nCompact JSON for the next agent:\n%s",
		res.Title, res.MemoryCount, res.TokenCount, res.Provider, string(compact))
	return s.textResult(msg), nil
}

func (s *Server) getSessionDigests(auth *authContext, raw json.RawMessage) (any, *rpcError) {
	if auth.project == nil {
		return s.errResult("connection not bound to a project"), nil
	}
	var args struct {
		Limit int `json:"limit"`
	}
	_ = json.Unmarshal(raw, &args)
	if args.Limit <= 0 || args.Limit > 50 {
		args.Limit = 5
	}

	svc := &api.Service{App: s.App}
	digests, err := svc.ListSessionDigests(auth.project.Id, args.Limit)
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: err.Error()}
	}
	if len(digests) == 0 {
		return s.textResult("No session digests stored for this project yet. Ask an agent to call cortex_summarize_session at the end of its session."), nil
	}
	compactList := make([]map[string]any, 0, len(digests))
	for _, d := range digests {
		compactList = append(compactList, d.DigestJSON)
	}
	out, _ := json.Marshal(compactList)
	return s.textResult(string(out)), nil
}

// ── Code graph recall ──────────────────────────────────

func (s *Server) recallCode(auth *authContext, raw json.RawMessage) (any, *rpcError) {
	if auth.project == nil {
		return s.errResult("connection not bound to a project"), nil
	}
	var args struct {
		Query string `json:"query"`
	}
	_ = json.Unmarshal(raw, &args)
	if strings.TrimSpace(args.Query) == "" {
		return s.errResult("query is required"), nil
	}
	svc := &api.Service{App: s.App}
	text, err := svc.QueryCodeGraph(auth.project.Id, args.Query)
	if err != nil {
		return s.errResult(err.Error()), nil
	}
	return s.textResult(text), nil
}

func (s *Server) codeMap(auth *authContext) (any, *rpcError) {
	if auth.project == nil {
		return s.errResult("connection not bound to a project"), nil
	}
	svc := &api.Service{App: s.App}
	text, err := svc.CodeMap(auth.project.Id)
	if err != nil {
		return s.errResult(err.Error()), nil
	}
	return s.textResult(text), nil
}

// ── Context assembly ───────────────────────────────────

func (s *Server) buildContext(auth *authContext) string {
	if auth.project == nil {
		return "This MCP connection is not bound to a project. Open CORTEX and create a project-scoped connection."
	}
	var b strings.Builder
	name := auth.project.GetString("name")
	b.WriteString("# Project Characterization: " + name + "\n\n")

	prompt := s.systemPrompt(auth.project)
	if prompt != "" {
		b.WriteString(prompt + "\n\n")
	} else {
		b.WriteString("(No system prompt generated yet — generate one in CORTEX → AI Agents.)\n\n")
	}

	// Compressed digests of previous work sessions (token-efficient recap).
	digests, _ := s.App.FindRecordsByFilter(db.CollSessionDigests, "project = {:p}", "-created", 3, 0,
		map[string]any{"p": auth.project.Id})
	if len(digests) > 0 {
		b.WriteString("## Recent session digests\n")
		for _, d := range digests {
			b.WriteString(fmt.Sprintf("- %s (via %s): %s\n",
				d.GetString("title"), d.GetString("ide"),
				truncate(digestSummary(d), 400)))
		}
		b.WriteString("\n")
	}

	// Memory of previous AI sessions.
	recs, _ := s.App.FindRecordsByFilter(db.CollAgentMemories, "project = {:p}", "-created", 30, 0,
		map[string]any{"p": auth.project.Id})
	if len(recs) > 0 {
		b.WriteString("## Memory from previous sessions\n")
		for _, r := range recs {
			b.WriteString(fmt.Sprintf("- [%s] %s (via %s): %s\n",
				r.GetString("category"), r.GetString("title"),
				r.GetString("client_name"), truncate(r.GetString("content"), 300)))
		}
		b.WriteString("\n")
	}

	// Architectural decisions / notes from the vault.
	vault, _ := s.App.FindRecordsByFilter(db.CollVaultEntries,
		"project = {:p} && (category = 'architecture' || category = 'decision')", "-updated", 10, 0,
		map[string]any{"p": auth.project.Id})
	if len(vault) > 0 {
		b.WriteString("## Architectural notes\n")
		for _, r := range vault {
			b.WriteString("- " + r.GetString("title") + ": " + truncate(r.GetString("content"), 300) + "\n")
		}
	}
	return b.String()
}

// systemPrompt returns the generated characterization for a project.
func (s *Server) systemPrompt(project *core.Record) string {
	var meta map[string]any
	if err := project.UnmarshalJSONField("metadata", &meta); err == nil && meta != nil {
		if sp, ok := meta["system_prompt"].(string); ok && strings.TrimSpace(sp) != "" {
			return sp
		}
	}
	// Fall back to the architecture overview vault entry.
	rec, err := s.App.FindFirstRecordByFilter(db.CollVaultEntries,
		"project = {:p} && category = 'architecture'", map[string]any{"p": project.Id})
	if err == nil && rec != nil {
		return rec.GetString("content")
	}
	return ""
}

// ── Prompts ────────────────────────────────────────────

func (s *Server) promptsList() map[string]any {
	return map[string]any{
		"prompts": []map[string]any{
			{
				"name":        "project_characterization",
				"description": "The CORTEX-generated system prompt characterizing the connected project, including prior AI memory.",
			},
		},
	}
}

func (s *Server) promptsGet(auth *authContext, req *rpcRequest) (any, *rpcError) {
	var p struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(req.Params, &p)
	if p.Name != "project_characterization" {
		return nil, &rpcError{Code: -32602, Message: "unknown prompt: " + p.Name}
	}
	text := s.buildContext(auth)
	return map[string]any{
		"description": "Project characterization and memory",
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": map[string]any{"type": "text", "text": text},
			},
		},
	}, nil
}

// ── Resources ──────────────────────────────────────────

func (s *Server) resourcesList(auth *authContext) map[string]any {
	return map[string]any{
		"resources": []map[string]any{
			{"uri": "cortex://project/characterization", "name": "Project Characterization", "mimeType": "text/markdown"},
			{"uri": "cortex://project/memory", "name": "Project Memory", "mimeType": "text/markdown"},
		},
	}
}

func (s *Server) resourcesRead(auth *authContext, req *rpcRequest) (any, *rpcError) {
	var p struct {
		URI string `json:"uri"`
	}
	_ = json.Unmarshal(req.Params, &p)

	var text string
	switch p.URI {
	case "cortex://project/characterization":
		text = s.buildContext(auth)
	case "cortex://project/memory":
		text = s.memoryDump(auth)
	default:
		return nil, &rpcError{Code: -32602, Message: "unknown resource: " + p.URI}
	}
	return map[string]any{
		"contents": []map[string]any{
			{"uri": p.URI, "mimeType": "text/markdown", "text": text},
		},
	}, nil
}

func (s *Server) memoryDump(auth *authContext) string {
	if auth.project == nil {
		return "no project bound"
	}
	recs, _ := s.App.FindRecordsByFilter(db.CollAgentMemories, "project = {:p}", "-created", 100, 0,
		map[string]any{"p": auth.project.Id})
	if len(recs) == 0 {
		return "No memories stored yet."
	}
	var b strings.Builder
	for _, r := range recs {
		b.WriteString(fmt.Sprintf("## %s [%s]\nvia %s (%s)\n\n%s\n\n",
			r.GetString("title"), r.GetString("category"),
			r.GetString("client_name"), r.GetString("ide"), r.GetString("content")))
	}
	return b.String()
}

// ── helpers ────────────────────────────────────────────

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func firstLine(s string, n int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return truncate(s, n)
}

// digestSummary extracts the human-readable summary ("sum") from a stored
// session digest's compact JSON, falling back to the title.
func digestSummary(rec *core.Record) string {
	var dj map[string]any
	if err := rec.UnmarshalJSONField("digest_json", &dj); err == nil {
		if sum, ok := dj["sum"].(string); ok && strings.TrimSpace(sum) != "" {
			return sum
		}
	}
	return rec.GetString("title")
}

func displayIDE(sess *session) string {
	if sess.ClientName != "" {
		return sess.ClientName
	}
	if sess.IDE != "" {
		return sess.IDE
	}
	return "unknown IDE"
}
