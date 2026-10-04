package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NexVed/Cortex/internal/localauth"
)

func TestWaitForServer(t *testing.T) {
	token := "test-local-credential"
	if serverAlive("127.0.0.1:1", token) {
		t.Fatal("unused port reported alive")
	}
	if err := waitForServer("127.0.0.1:1", token, 100*time.Millisecond, nil); err == nil {
		t.Fatal("unreachable backend reported ready")
	}
	unrelated := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("credential sent to an unverified server")
		}
		_, _ = w.Write([]byte(`{"app":"cortexMind","ready":true,"proof":"forged"}`))
	}))
	defer unrelated.Close()
	if serverAlive(strings.TrimPrefix(unrelated.URL, "http://"), token) {
		t.Fatal("unrelated listener accepted as CortexMind")
	}
	trusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"app": "cortexMind", "ready": true, "proof": localauth.Proof(token, r.URL.Query().Get("challenge"))})
	}))
	defer trusted.Close()
	if err := waitForServer(strings.TrimPrefix(trusted.URL, "http://"), token, time.Second, nil); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	errCh <- net.ErrClosed
	if err := waitForServer("127.0.0.1:1", token, time.Second, errCh); err == nil {
		t.Fatal("startup failure ignored")
	}
}
