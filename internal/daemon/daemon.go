package daemon

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/auth"
	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/database"
	gh "github.com/NexVed/Cortex/internal/github"
	"github.com/NexVed/Cortex/internal/keychain"
	"github.com/NexVed/Cortex/internal/mcp"
	"github.com/NexVed/Cortex/internal/repositories"
	"github.com/NexVed/Cortex/internal/services"
	"github.com/NexVed/Cortex/internal/web"
)

type Daemon struct {
	Config *config.Config
	DB     *database.DB
	Auth   *auth.Service
	server *http.Server
}

func New(cfg *config.Config) *Daemon {
	db, err := database.Open(cfg.DataDirPath())
	if err != nil {
		panic(fmt.Errorf("open local SQLite: %w", err))
	}
	onboarding := services.Onboarding{Users: repositories.UserRepository{DB: db}, GitHub: gh.Client{ClientID: cfg.GitHub.ClientID, ClientSecret: cfg.GitHub.ClientSecret}, DB: db}
	return &Daemon{Config: cfg, DB: db, Auth: &auth.Service{ClientID: cfg.GitHub.ClientID, GitHub: onboarding, Tokens: keychain.Store{}}}
}
func (d *Daemon) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.New(d.DB))
	mux.HandleFunc("/api/cortex/mcp/connections", d.mcpConnections)
	mux.HandleFunc("/api/cortex/mcp/connections/{id}", d.mcpConnection)
	mux.HandleFunc("/api/cortex/mcp/connections/{id}/status", d.mcpConnectionStatus)
	mux.HandleFunc("/api/auth/session", d.session)
	mux.HandleFunc("/api/auth/github/start", d.startGitHub)
	mux.HandleFunc("/api/auth/offline", d.offline)
	mux.HandleFunc("/api/auth/logout", d.logout)
	mux.HandleFunc("/api/projects", d.projects)
	mux.HandleFunc("/api/projects/{id}", d.project)
	mux.HandleFunc("/api/github/repositories", d.githubRepositories)
	mux.HandleFunc("/api/github/sync", d.syncGitHub)
	mux.HandleFunc("/api/cortex/scan/{id}", d.scanProject)
	mux.HandleFunc("/api/cortex/code-graph/{id}", d.codeGraph)
	mux.HandleFunc("/api/cortex/repository-insights/{id}", d.repositoryInsights)
	mux.HandleFunc("/api/cortex/session-digest/{id}", d.generateSessionDigest)
	mux.HandleFunc("/api/cortex/session-digests/{id}", d.sessionDigests)
	mux.HandleFunc("/api/cortex/agent-memories/{id}", d.agentMemories)
	mux.HandleFunc("/api/cortex/agents/{id}", d.activeAgents)
	mux.HandleFunc("/api/cortex/system-prompt/{id}", d.systemPrompt)
	mux.HandleFunc("/api/collections/{collection}/records", d.collectionRecords)
	mux.HandleFunc("/api/collections/{collection}/records/{id}", d.collectionRecord)
	if web.Available() {
		mux.Handle("/", spaFileServer(web.FS()))
	}
	d.server = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", d.Config.Server.Port), Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return d.server.ListenAndServe()
}
func spaFileServer(ui fs.FS) http.Handler {
	files := http.FileServer(http.FS(ui))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			files.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(ui, path); err == nil {
			files.ServeHTTP(w, r)
			return
		}
		r2 := new(http.Request)
		*r2 = *r
		r2.URL.Path = "/"
		files.ServeHTTP(w, r2)
	})
}
func (d *Daemon) session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	u, err := d.DB.CurrentUser()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}
func (d *Daemon) startGitHub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	v, err := d.Auth.StartGitHub()
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (d *Daemon) offline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, fmt.Errorf("invalid offline profile"), http.StatusBadRequest)
		return
	}
	u, err := d.Auth.GitHub.ContinueOffline(strings.TrimSpace(input.DisplayName))
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}
func (d *Daemon) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := d.Auth.Logout(); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (d *Daemon) projects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects, err := d.DB.ListProjects()
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, projects)
	case http.MethodPost:
		var input struct {
			Name        string `json:"name"`
			Path        string `json:"path"`
			Description string `json:"description"`
			GitHubURL   string `json:"github_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, fmt.Errorf("invalid project payload"), http.StatusBadRequest)
			return
		}
		project, err := d.DB.CreateProject(input.Name, input.Path, input.Description, input.GitHubURL)
		if err != nil {
			writeError(w, err, http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, http.StatusCreated, project)
	default:
		methodNotAllowed(w)
	}
}
func (d *Daemon) project(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	project, err := d.DB.Project(r.PathValue("id"))
	if err != nil {
		writeError(w, fmt.Errorf("project not found"), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, project)
}
func (d *Daemon) githubRepositories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	projects, err := d.DB.ListProjects()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}
func (d *Daemon) syncGitHub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	token, err := d.Auth.CurrentGitHubToken()
	if err != nil {
		writeError(w, fmt.Errorf("GitHub is not connected"), http.StatusUnauthorized)
		return
	}
	if _, err = d.Auth.GitHub.CompleteGitHub(r.Context(), token); err != nil {
		writeError(w, err, http.StatusBadGateway)
		return
	}
	projects, err := d.DB.ListProjects()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(projects), "imported": len(projects), "updated": 0, "skipped": 0, "names": names})
}

func (d *Daemon) scanProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	token, err := d.Auth.CurrentGitHubToken()
	if err != nil {
		writeError(w, err, http.StatusUnauthorized)
		return
	}
	result, err := (services.ScanService{DB: d.DB, Config: d.Config, GitHubToken: token}).Scan(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *Daemon) collectionRecords(w http.ResponseWriter, r *http.Request) {
	collection := r.PathValue("collection")
	if !allowedCollection(collection) {
		writeError(w, fmt.Errorf("unknown local collection"), http.StatusNotFound)
		return
	}
	store := database.RecordStore{DB: d.DB}
	switch r.Method {
	case http.MethodGet:
		projectID := projectFilter(r.URL.Query().Get("filter"))
		page, err := store.List(collection, projectID, 500)
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, page)
	case http.MethodPost:
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, fmt.Errorf("invalid record payload"), http.StatusBadRequest)
			return
		}
		record, err := store.Create(collection, input)
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, record)
	default:
		methodNotAllowed(w)
	}
}
func (d *Daemon) collectionRecord(w http.ResponseWriter, r *http.Request) {
	collection, id := r.PathValue("collection"), r.PathValue("id")
	if !allowedCollection(collection) {
		writeError(w, fmt.Errorf("unknown local collection"), http.StatusNotFound)
		return
	}
	store := database.RecordStore{DB: d.DB}
	switch r.Method {
	case http.MethodGet:
		record, err := store.Get(collection, id)
		if err != nil {
			writeError(w, fmt.Errorf("record not found"), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, record)
	case http.MethodPatch:
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, fmt.Errorf("invalid record payload"), http.StatusBadRequest)
			return
		}
		record, err := store.Update(collection, id, patch)
		if err != nil {
			writeError(w, fmt.Errorf("record not found"), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, record)
	case http.MethodDelete:
		if err := store.Delete(collection, id); err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	default:
		methodNotAllowed(w)
	}
}
func allowedCollection(name string) bool {
	switch name {
	case "tasks", "handoffs", "vault_entries", "activity_log", "file_index", "agent_memories", "session_digests", "search_history", "mcp_tokens", "notifications":
		return true
	}
	return false
}
func projectFilter(filter string) string {
	marker := `project="`
	start := strings.Index(filter, marker)
	if start < 0 {
		return ""
	}
	rest := filter[start+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error, status int) {
	writeJSON(w, status, map[string]string{"error": err.Error(), "message": err.Error()})
}
func methodNotAllowed(w http.ResponseWriter) { w.WriteHeader(http.StatusMethodNotAllowed) }
