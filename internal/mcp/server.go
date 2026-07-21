// Package mcp exposes cortexMind's local project memory to MCP-compatible
// coding agents over a loopback HTTP JSON-RPC endpoint.
package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/NexVed/Cortex/internal/database"
	"github.com/NexVed/Cortex/internal/repositories"
	"github.com/NexVed/Cortex/internal/services"
)

const protocolVersion = "2025-03-26"

type Server struct {
	graphs      services.CodeGraphService
	context     services.AgentContextService
	connections repositories.MCPConnectionRepository
}

func New(db *database.DB) *Server {
	graphs := services.CodeGraphService{DB: db}
	return &Server{graphs: graphs, context: services.AgentContextService{DB: db, Graphs: graphs}, connections: repositories.MCPConnectionRepository{DB: db}}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type toolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}
type graphArguments struct {
	ProjectID     string   `json:"project_id"`
	NodeID        string   `json:"node_id"`
	Query         string   `json:"query"`
	NodeTypes     []string `json:"node_types"`
	Relationships []string `json:"relationships"`
	MaxNodes      int      `json:"max_nodes"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	connection, err := s.authenticate(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="cortexMind"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, nil, -32700, "invalid JSON-RPC request")
		return
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		s.writeError(w, req.ID, -32600, "invalid JSON-RPC request")
		return
	}
	if strings.HasPrefix(req.Method, "notifications/") {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	switch req.Method {
	case "initialize":
		s.writeResult(w, req.ID, map[string]any{"protocolVersion": protocolVersion, "capabilities": map[string]any{"tools": map[string]bool{"listChanged": false}}, "serverInfo": map[string]string{"name": "cortexMind", "version": "0.1.0"}, "instructions": "Use cortex_get_context first, then call cortex_get_code_graph for code structure. Persist useful work with cortex_save_memory and cortex_summarize_session."})
	case "ping":
		s.writeResult(w, req.ID, map[string]any{})
	case "tools/list":
		s.writeResult(w, req.ID, map[string]any{"tools": toolDefinitions()})
	case "tools/call":
		s.callTool(w, req, connection)
	default:
		s.writeError(w, req.ID, -32601, "method not found")
	}
}

func (s *Server) callTool(w http.ResponseWriter, req request, connection *repositories.MCPConnection) {
	var call toolCall
	if err := json.Unmarshal(req.Params, &call); err != nil || call.Name == "" {
		s.writeError(w, req.ID, -32602, "tools/call requires a tool name")
		return
	}
	args := map[string]any{}
	if len(call.Arguments) > 0 && string(call.Arguments) != "null" {
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			s.writeError(w, req.ID, -32602, "invalid tool arguments")
			return
		}
	}
	projectID, _ := args["project_id"].(string)
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		s.writeError(w, req.ID, -32602, "project_id is required")
		return
	}
	if projectID != connection.ProjectID {
		s.writeToolError(w, req.ID, "connection is not authorized for this project")
		return
	}
	if call.Name == "cortex_save_memory" {
		if stringValue(args["ide"]) == "" {
			args["ide"] = connection.IDE
		}
		if stringValue(args["agent"]) == "" {
			args["agent"] = connection.IDE
		}
	}
	result, err := s.executeTool(call.Name, projectID, args)
	if err != nil {
		s.writeToolError(w, req.ID, err.Error())
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		s.writeError(w, req.ID, -32603, "failed to encode tool result")
		return
	}
	_ = s.connections.Touch(connection.ID)
	s.writeResult(w, req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": string(raw)}}, "structuredContent": result, "isError": false})
}

func (s *Server) executeTool(name, projectID string, args map[string]any) (any, error) {
	switch name {
	case "cortex_get_code_graph":
		raw, _ := json.Marshal(args)
		var input graphArguments
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, fmt.Errorf("invalid graph arguments")
		}
		if input.MaxNodes < 0 || input.MaxNodes > 1000 {
			return nil, fmt.Errorf("max_nodes must be between 1 and 1000")
		}
		if !validValues(input.NodeTypes, map[string]bool{"dir": true, "file": true, "function": true, "class": true, "package": true}) {
			return nil, fmt.Errorf("node_types contains an unsupported value")
		}
		if !validValues(input.Relationships, map[string]bool{"contains": true, "defines": true, "imports": true, "depends_on": true}) {
			return nil, fmt.Errorf("relationships contains an unsupported value")
		}
		return s.graphs.Context(projectID, services.CodeGraphQuery{NodeID: input.NodeID, Query: input.Query, NodeTypes: input.NodeTypes, Relationships: input.Relationships, MaxNodes: input.MaxNodes})
	case "cortex_get_context":
		return s.context.GetContext(projectID, integer(args["memory_limit"]))
	case "cortex_save_memory":
		memory := map[string]any{"title": stringValue(args["title"]), "content": stringValue(args["content"]), "category": stringValue(args["category"]), "session_id": stringValue(args["session_id"]), "agent": stringValue(args["agent"]), "owner": stringValue(args["agent"]), "ide": stringValue(args["ide"]), "client_name": stringValue(args["ide"]), "tags": args["tags"]}
		return s.context.SaveMemory(projectID, memory)
	case "cortex_list_memories":
		return s.context.ListMemories(projectID, integer(args["limit"]))
	case "cortex_get_tasks":
		return s.context.ActiveTasks(projectID, integer(args["limit"]))
	case "cortex_summarize_session":
		return s.context.SaveSessionDigest(projectID, stringValue(args["session_id"]), stringValue(args["title"]), stringValue(args["summary"]))
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) authenticate(r *http.Request) (*repositories.MCPConnection, error) {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(value, "Bearer ") {
		return nil, fmt.Errorf("missing bearer token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
	if token == "" {
		return nil, fmt.Errorf("missing bearer token")
	}
	return s.connections.Authenticate(token)
}
func (s *Server) writeToolError(w http.ResponseWriter, id json.RawMessage, message string) {
	s.writeResult(w, id, map[string]any{"content": []map[string]string{{"type": "text", "text": message}}, "isError": true})
}
func (s *Server) writeResult(w http.ResponseWriter, id json.RawMessage, result any) {
	s.write(w, response{JSONRPC: "2.0", ID: id, Result: result})
}
func (s *Server) writeError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	s.write(w, response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}
func (s *Server) write(w http.ResponseWriter, value response) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
func validValues(values []string, allowed map[string]bool) bool {
	for _, value := range values {
		if !allowed[value] {
			return false
		}
	}
	return true
}
func stringValue(value any) string { text, _ := value.(string); return strings.TrimSpace(text) }
func integer(value any) int        { number, _ := value.(float64); return int(number) }

func toolDefinitions() []any {
	return []any{
		tool("cortex_get_context", "Load the bound project's profile, code-graph statistics, and recent memories. Call this first.", map[string]any{"memory_limit": integerSchema("Maximum recent memories, default 25.")}, nil),
		tool("cortex_get_code_graph", "Query the persisted code graph for files, functions, classes, packages, and dependencies.", map[string]any{"node_id": stringSchema("Optional node ID; returns direct relationships."), "query": stringSchema("Case-insensitive label/path search."), "node_types": enumArray("dir", "file", "function", "class", "package"), "relationships": enumArray("contains", "defines", "imports", "depends_on"), "max_nodes": integerSchema("Maximum nodes, default 250.")}, nil),
		tool("cortex_save_memory", "Persist a project memory so later coding sessions can recall progress, decisions, notes, context, or handoffs.", map[string]any{"title": stringSchema("Short memory title."), "content": stringSchema("Memory content."), "category": enumSchema("context", "progress", "decision", "note", "handoff"), "session_id": stringSchema("Optional client session ID."), "agent": stringSchema("Optional agent name."), "ide": stringSchema("Optional IDE/client name."), "tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}, []string{"content"}),
		tool("cortex_list_memories", "List recent stored project memories from prior agents and IDE sessions.", map[string]any{"limit": integerSchema("Maximum memories, default 50.")}, nil),
		tool("cortex_get_tasks", "List active project tasks; completed and cancelled tasks are excluded.", map[string]any{"limit": integerSchema("Maximum tasks, default 50.")}, nil),
		tool("cortex_summarize_session", "Save an agent-authored, token-efficient session digest for later sessions. The agent supplies the summary.", map[string]any{"session_id": stringSchema("Optional client session ID."), "title": stringSchema("Optional digest title."), "summary": stringSchema("Concise session summary to persist.")}, []string{"summary"}),
	}
}
func tool(name, description string, properties map[string]any, required []string) map[string]any {
	properties["project_id"] = stringSchema("Project ID bound to this connection.")
	required = append(required, "project_id")
	return map[string]any{"name": name, "description": description, "inputSchema": map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required}}
}
func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}
func integerSchema(description string) map[string]any {
	return map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "description": description}
}
func enumSchema(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}
func enumArray(values ...string) map[string]any {
	return map[string]any{"type": "array", "items": enumSchema(values...)}
}
