// Package mcp implements a Model Context Protocol (MCP) server over Streamable
// HTTP (JSON-RPC 2.0). IDEs connect with a per-connection Bearer token that
// binds the session to a CORTEX user and project, so the server can serve that
// project's characterization (generated system prompt) and persist/recall the
// AI agent's working memory with awareness of which IDE is connected.
package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/rs/zerolog/log"
)

const (
	protocolVersion = "2025-06-18"
	serverName      = "cortex"
	serverVersion   = "0.1.0"
)

// Server is the MCP endpoint handler.
type Server struct {
	App core.App

	mu       sync.Mutex
	sessions map[string]*session // keyed by mcp_tokens record id
}

// session tracks a live IDE connection in memory for the daemon's lifetime.
type session struct {
	ID         string
	IDE        string
	ClientName string
	Started    time.Time
}

func New(app core.App) *Server {
	return &Server{App: app, sessions: map[string]*session{}}
}

// ── JSON-RPC envelope ──────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// authContext is resolved from the connection token.
type authContext struct {
	token   *core.Record
	ownerID string
	project *core.Record // may be nil if the token is not project-bound
}

// Handler returns the http.Handler for the MCP endpoint.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		// SSE stream open is not required for request/response usage.
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	auth, err := s.authenticate(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32001, Message: "unauthorized: " + err.Error()},
		})
		return
	}

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = json.NewEncoder(w).Encode(rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32700, Message: "parse error"},
		})
		return
	}

	// Notifications (no id) get acknowledged with 202 and no body.
	isNotification := len(req.ID) == 0 || string(req.ID) == "null"

	result, rerr := s.dispatch(r, auth, &req)

	if isNotification {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	if rerr != nil {
		resp.Error = rerr
	} else {
		resp.Result = result
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// authenticate resolves the Bearer token to an MCP token record + owner/project.
func (s *Server) authenticate(r *http.Request) (*authContext, error) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		// Some clients send the key via a custom header.
		token = strings.TrimSpace(r.Header.Get("X-Cortex-Token"))
	}
	if token == "" {
		return nil, errMissingToken
	}
	rec, err := s.App.FindFirstRecordByFilter(db.CollMCPTokens, "token = {:t}", map[string]any{"t": token})
	if err != nil || rec == nil {
		return nil, errInvalidToken
	}
	if !rec.GetBool("enabled") {
		return nil, errDisabledToken
	}

	ctx := &authContext{token: rec, ownerID: rec.GetString("owner")}
	if pid := rec.GetString("project"); pid != "" {
		if proj, err := s.App.FindRecordById(db.CollProjects, pid); err == nil {
			ctx.project = proj
		}
	}
	return ctx, nil
}

// dispatch routes a JSON-RPC method to its handler.
func (s *Server) dispatch(r *http.Request, auth *authContext, req *rpcRequest) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(auth, req)
	case "notifications/initialized", "notifications/cancelled":
		return map[string]any{}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return s.toolsList(), nil
	case "tools/call":
		return s.toolsCall(auth, req)
	case "prompts/list":
		return s.promptsList(), nil
	case "prompts/get":
		return s.promptsGet(auth, req)
	case "resources/list":
		return s.resourcesList(auth), nil
	case "resources/read":
		return s.resourcesRead(auth, req)
	default:
		return nil, &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
}

func (s *Server) handleInitialize(auth *authContext, req *rpcRequest) (any, *rpcError) {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}
	_ = json.Unmarshal(req.Params, &params)

	clientName := params.ClientInfo.Name
	if clientName == "" {
		clientName = "unknown"
	}

	// Start (or refresh) the in-memory session and persist connection metadata.
	sess := &session{ID: newID(), IDE: auth.token.GetString("ide"), ClientName: clientName, Started: time.Now()}
	s.mu.Lock()
	s.sessions[auth.token.Id] = sess
	s.mu.Unlock()

	auth.token.Set("client_name", clientName)
	auth.token.Set("last_used", types.NowDateTime())
	if err := s.App.Save(auth.token); err != nil {
		log.Warn().Err(err).Msg("mcp: failed to update token on initialize")
	}

	pv := params.ProtocolVersion
	if pv == "" {
		pv = protocolVersion
	}

	projectName := ""
	if auth.project != nil {
		projectName = auth.project.GetString("name")
	}
	log.Info().Str("ide", sess.IDE).Str("client", clientName).Str("project", projectName).Msg("mcp client connected")

	return map[string]any{
		"protocolVersion": pv,
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"prompts":   map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    serverName,
			"version": serverVersion,
		},
		"instructions": "CortexMind shared brain. Call the cortex_get_context tool first to load this project's characterization and the memory of previous AI sessions. Use cortex_save_memory to record progress, decisions and context as you work.",
	}, nil
}

// session returns the live session for a token, creating a fallback if the
// daemon restarted mid-connection.
func (s *Server) sessionFor(auth *authContext) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[auth.token.Id]; ok {
		return sess
	}
	sess := &session{ID: newID(), IDE: auth.token.GetString("ide"), ClientName: auth.token.GetString("client_name"), Started: time.Now()}
	s.sessions[auth.token.Id] = sess
	return sess
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// errors
var (
	errMissingToken = mcpErr("missing bearer token")
	errInvalidToken = mcpErr("invalid token")
	errDisabledToken = mcpErr("token disabled")
)

type mcpErr string

func (e mcpErr) Error() string { return string(e) }
