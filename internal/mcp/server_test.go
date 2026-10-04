package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NexVed/Cortex/internal/database"
	"github.com/NexVed/Cortex/internal/repositories"
	"github.com/NexVed/Cortex/internal/services"
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

func TestAllProjectsConnectionDiscoversFutureProjectsAndKeepsMemorySeparate(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first, _ := db.CreateProject("First", "", "", "")
	second, _ := db.CreateProject("Second", "", "", "")
	service := services.MCPConnectionService{Projects: db, Connections: repositories.MCPConnectionRepository{DB: db}}
	connection, err := service.Create(services.CreateMCPConnectionInput{Scope: "all", IDE: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if !connection.AllProjects() {
		t.Fatal("all-project connection restricted")
	}
	if _, err = service.Create(services.CreateMCPConnectionInput{ProjectID: "*", IDE: "codex"}); err == nil {
		t.Fatal("scope escalation without explicit all scope")
	}
	server := New(db)
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
		r := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+connection.Token)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response["result"].(map[string]any)
	}
	list := call("cortex_list_projects", nil)
	if list["isError"] != false || len(list["structuredContent"].(map[string]any)["projects"].([]any)) != 2 {
		t.Fatalf("project discovery failed: %v", list)
	}
	db.CreateProject("Future", "", "", "")
	list = call("cortex_list_projects", nil)
	if len(list["structuredContent"].(map[string]any)["projects"].([]any)) != 3 {
		t.Fatal("future project inaccessible")
	}
	if call("cortex_get_context", nil)["isError"] != true {
		t.Fatal("missing project ID should not combine memories")
	}
	if call("cortex_get_context", map[string]any{"project_id": "unknown"})["isError"] != true {
		t.Fatal("unknown project allowed")
	}
	for _, project := range []*database.Project{first, second} {
		if call("cortex_save_memory", map[string]any{"project_id": project.ID, "content": project.Name})["isError"] != false {
			t.Fatal("all-project memory write failed")
		}
	}
	for _, project := range []*database.Project{first, second} {
		result := call("cortex_list_memories", map[string]any{"project_id": project.ID})
		if result["isError"] != false {
			t.Fatalf("memory read: %v", result)
		}
		encoded, _ := json.Marshal(result["structuredContent"])
		var items []map[string]any
		json.Unmarshal(encoded, &items)
		if len(items) != 1 || items[0]["content"] != project.Name {
			t.Fatalf("project memory mixed: %s", encoded)
		}
	}
	if err = service.Connections.Delete(connection.ID); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	r.Header.Set("Authorization", "Bearer "+connection.Token)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("revoked all-project token still works")
	}
}
