package daemon

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NexVed/Cortex/internal/auth"
	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/database"
	gh "github.com/NexVed/Cortex/internal/github"
	"github.com/NexVed/Cortex/internal/keychain"
	"github.com/NexVed/Cortex/internal/repositories"
	"github.com/NexVed/Cortex/internal/services"
)

type memoryTokens map[string]string

func (s memoryTokens) Get(key string) (string, error) {
	if v, ok := s[key]; ok {
		return v, nil
	}
	return "", keychain.ErrNotFound
}
func (s memoryTokens) Set(key, value string) error { s[key] = value; return nil }
func (s memoryTokens) Delete(key string) error     { delete(s, key); return nil }

func testDaemon(t *testing.T) *Daemon {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Daemon{Config: &config.Config{Server: config.ServerConfig{Port: 8090, DataDir: dir}}, DB: db, APIToken: "test-api-token", started: time.Now(), Auth: &auth.Service{GitHub: services.Onboarding{DB: db, Users: repositories.UserRepository{DB: db}}, Tokens: memoryTokens{}}}
}

func call(t *testing.T, d *Daemon, method, path, body, token, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://127.0.0.1:8090"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	res := httptest.NewRecorder()
	d.Handler().ServeHTTP(res, req)
	return res
}

func TestLocalAPISecurity(t *testing.T) {
	d := testDaemon(t)
	for _, path := range []string{"/api/projects", "/api/auth/session", "/api/collections/vault_entries/records", "/api/cortex/mcp/connections"} {
		if res := call(t, d, "GET", path, "", "", ""); res.Code != http.StatusUnauthorized {
			t.Fatalf("%s without credential: %d", path, res.Code)
		}
	}
	if res := call(t, d, "POST", "/api/cortex/mcp/connections", `{}`, "", ""); res.Code != http.StatusUnauthorized {
		t.Fatal("unauthenticated token creation accepted")
	}
	if res := call(t, d, "GET", "/api/projects", "", "wrong", ""); res.Code != http.StatusUnauthorized {
		t.Fatal("wrong token accepted")
	}
	if res := call(t, d, "POST", "/api/auth/offline", `{"display_name":"attacker"}`, d.APIToken, "https://attacker.example"); res.Code != http.StatusForbidden {
		t.Fatal("cross-origin write accepted")
	}
	if res := call(t, d, "POST", "/mcp", `{}`, "mcp-token", "https://attacker.example"); res.Code != http.StatusForbidden {
		t.Fatal("cross-origin MCP request accepted")
	}
	request := httptest.NewRequest("GET", "http://attacker.example:8090/api/health", nil)
	res := httptest.NewRecorder()
	d.Handler().ServeHTTP(res, request)
	if res.Code != http.StatusForbidden {
		t.Fatal("DNS rebinding host accepted")
	}
	request = httptest.NewRequest("POST", "http://127.0.0.1:8090/api/auth/offline", strings.NewReader(`{"display_name":"x"}`))
	request.Header.Set("Content-Type", "text/plain")
	request.Header.Set("Authorization", "Bearer "+d.APIToken)
	res = httptest.NewRecorder()
	d.Handler().ServeHTTP(res, request)
	if res.Code != http.StatusUnsupportedMediaType {
		t.Fatal("simple cross-site request content type accepted")
	}
	if res := call(t, d, "GET", "/api/cortex/status", "", d.APIToken, "http://127.0.0.1:8090"); res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"ready":true`) {
		t.Fatal("authenticated status unavailable")
	}
	if res := call(t, d, "GET", "/api/unknown", "", d.APIToken, ""); res.Code != http.StatusNotFound || !strings.Contains(res.Header().Get("Content-Type"), "json") {
		t.Fatal("unknown API returned the SPA")
	}
	oversized := `{"payload":"` + strings.Repeat("x", maxRequestBytes) + `"}`
	if res := call(t, d, "POST", "/api/collections/vault_entries/records", oversized, d.APIToken, ""); res.Code < 400 {
		t.Fatal("unbounded API payload accepted")
	}
}

func TestResetRevokesCredentialsAndClearsAllData(t *testing.T) {
	d := testDaemon(t)
	u, err := d.Auth.GitHub.ContinueOffline("tester")
	if err != nil {
		t.Fatal(err)
	}
	project, err := d.DB.CreateProject("sample", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	store := database.RecordStore{DB: d.DB}
	for _, collection := range []string{"tasks", "file_index", "system_prompts", "notifications", "agent_memories", "session_digests", "vault_entries"} {
		if _, err = store.Create(collection, map[string]any{"project": project.ID}); err != nil {
			t.Fatal(err)
		}
	}
	connection, err := d.mcpConnectionService().Create(services.CreateMCPConnectionInput{ProjectID: project.ID, IDE: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.DB.SaveRepositoryScan(project.ID, "unused", 1); err != nil {
		t.Fatal(err)
	}
	if err = d.DB.SaveSetting("providers:"+u.ID, map[string]string{"llm_provider": "mistral"}); err != nil {
		t.Fatal(err)
	}
	if _, err = d.DB.Exec(`INSERT INTO code_graphs(project_id,payload) VALUES(?, '{}')`, project.ID); err != nil {
		t.Fatal(err)
	}
	if err = d.Auth.Tokens.Set("mistral:"+u.ID, "secret"); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(d.Config.Server.DataDir, "repositories")
	if err = os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(cache, "cached.txt"), []byte("cache"), 0600); err != nil {
		t.Fatal(err)
	}
	userDirectory := t.TempDir()
	if err = d.DB.BindWorkingTree(project.ID, userDirectory); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(userDirectory, "source.go"), []byte("package sample"), 0600); err != nil {
		t.Fatal(err)
	}
	res := call(t, d, "POST", "/api/cortex/reset", "{}", d.APIToken, "")
	if res.Code != http.StatusOK {
		t.Fatal(res.Body.String())
	}
	for _, table := range []string{"users", "active_session", "projects", "local_records", "mcp_connections", "repository_scans", "code_graphs", "project_worktrees", "app_settings"} {
		var count int
		if err = d.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("reset left %s: %d (%v)", table, count, err)
		}
	}
	if _, err = d.Auth.Tokens.Get("mistral:" + u.ID); err == nil {
		t.Fatal("provider secret survived reset")
	}
	if _, err = d.mcpConnectionService().Connections.Authenticate(connection.Token); err == nil {
		t.Fatal("MCP token survived reset")
	}
	if _, err = os.Stat(cache); !os.IsNotExist(err) {
		t.Fatal("clone cache survived reset")
	}
	if _, err = os.Stat(filepath.Join(userDirectory, "source.go")); err != nil {
		t.Fatal("user project was touched by reset")
	}
}

func TestFailedResetRollsBackAndReturnsError(t *testing.T) {
	d := testDaemon(t)
	project, err := d.DB.CreateProject("preserved", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = (database.RecordStore{DB: d.DB}).Create("tasks", map[string]any{"project": project.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err = d.DB.Exec(`CREATE TRIGGER block_reset BEFORE DELETE ON projects BEGIN SELECT RAISE(FAIL, 'blocked'); END;`); err != nil {
		t.Fatal(err)
	}
	res := call(t, d, "POST", "/api/cortex/reset", "{}", d.APIToken, "")
	if res.Code != http.StatusInternalServerError {
		t.Fatal("failed reset reported success")
	}
	page, err := (database.RecordStore{DB: d.DB}).List("tasks", project.ID, 100)
	if err != nil || len(page.Items) != 1 {
		t.Fatal("failed reset did not roll back earlier deletes")
	}
}

func TestProviderConfigurationStoresOnlyRedactedSecretsInResponses(t *testing.T) {
	d := testDaemon(t)
	u, err := d.Auth.GitHub.ContinueOffline("provider-test")
	if err != nil {
		t.Fatal(err)
	}
	res := call(t, d, "POST", "/api/cortex/providers", `{"llm_provider":"mistral","mistral_key":"private-mistral-key"}`, d.APIToken, "")
	if res.Code != 200 || strings.Contains(res.Body.String(), "private-mistral-key") {
		t.Fatal("provider save failed or disclosed its secret")
	}
	var raw string
	if err = d.DB.QueryRow(`SELECT payload FROM app_settings WHERE id=?`, "providers:"+u.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "private-mistral-key") {
		t.Fatal("provider key was stored in SQLite")
	}
	if key, err := d.Auth.Tokens.Get("mistral:" + u.ID); err != nil || key != "private-mistral-key" {
		t.Fatal("provider key was not stored in the credential store")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPrivateRepositoryContentUsesBackendCredential(t *testing.T) {
	d := testDaemon(t)
	u := database.User{ID: "github:tester", Provider: "github", Username: "tester"}
	if err := d.DB.SaveUser(u); err != nil {
		t.Fatal(err)
	}
	if err := d.DB.SetActiveUser(u.ID); err != nil {
		t.Fatal(err)
	}
	if err := d.Auth.Tokens.Set("github:"+u.ID, "github-secret"); err != nil {
		t.Fatal(err)
	}
	project, err := d.DB.CreateProject("private", "", "", "https://github.com/tester/private")
	if err != nil {
		t.Fatal(err)
	}
	d.Auth.GitHub.GitHub = gh.Client{HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "Bearer github-secret" {
			t.Error("repository request lost or misdirected its credential")
		}
		body := ""
		switch r.URL.Path {
		case "/repos/tester/private":
			body = `{"full_name":"tester/private","private":true,"default_branch":"main"}`
		case "/repos/tester/private/contents":
			body = `[{"name":"README.md","type":"file"}]`
		case "/repos/tester/private/commits":
			body = `[{"sha":"1234"}]`
		case "/repos/tester/private/readme":
			body = `{"path":"README.md"}`
		case "/repos/tester/private/contents/README.md":
			if r.Header.Get("Accept") != "application/vnd.github.html+json" {
				t.Error("README requested with the wrong media type")
			}
			body = "<h1>Private readme</h1>"
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	res := call(t, d, "GET", "/api/github/repositories/"+project.ID+"/view", "", d.APIToken, "")
	if res.Code != 200 {
		t.Fatal(res.Body.String())
	}
	var view map[string]any
	if err = json.Unmarshal(res.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view["readme_html"] != "<h1>Private readme</h1>" {
		t.Fatal("private readme missing")
	}
	if strings.Contains(res.Body.String(), "github-secret") {
		t.Fatal("GitHub credential exposed to UI")
	}
}
