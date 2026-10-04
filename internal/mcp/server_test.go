package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NexVed/Cortex/internal/database"
	"github.com/NexVed/Cortex/internal/repositories"
)

func TestBoundProjectTools(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	project, err := db.CreateProject("Windows workspace", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Project(project.ID); err != nil {
		t.Fatalf("local project cannot be loaded: %v", err)
	}
	projects, err := db.ListProjects()
	if err != nil || len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("local project missing from picker: projects=%v, err=%v", projects, err)
	}
	const token = "test-token"
	_, err = (repositories.MCPConnectionRepository{DB: db}).Create(repositories.MCPConnection{
		ID: "test-connection", ProjectID: project.ID, IDE: "codex", Label: "test", Enabled: true,
	}, token)
	if err != nil {
		t.Fatal(err)
	}
	server := New(db)
	call := func(method string, params any) map[string]any {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s returned HTTP %d: %s", method, w.Code, w.Body.String())
		}
		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	listed := call("tools/list", map[string]any{})
	tools := listed["result"].(map[string]any)["tools"].([]any)
	for _, item := range tools {
		definition := item.(map[string]any)
		schema := definition["inputSchema"].(map[string]any)
		required, _ := schema["required"].([]any)
		for _, name := range required {
			if name == "project_id" {
				t.Fatalf("%s still requires project_id", definition["name"])
			}
		}
	}
	result := call("tools/call", map[string]any{"name": "cortex_get_context", "arguments": map[string]any{}})["result"].(map[string]any)
	if result["isError"] != false {
		t.Fatalf("bound project call failed: %v", result)
	}
	context := result["structuredContent"].(map[string]any)
	if context["project"].(map[string]any)["id"] != project.ID {
		t.Fatalf("wrong project returned: %v", context["project"])
	}
	denied := call("tools/call", map[string]any{"name": "cortex_get_context", "arguments": map[string]any{"project_id": "other"}})["result"].(map[string]any)
	if denied["isError"] != true {
		t.Fatalf("cross-project call was allowed: %v", denied)
	}
}
