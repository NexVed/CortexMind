package main

import (
	"net"
	"testing"
	"time"
)

// TestWaitForServer checks the backend-readiness gate that decides whether the
// desktop shell boots its own daemon or reuses a running one. If this logic
// breaks, the window would either load before the server is up or hang forever.
func TestWaitForServer(t *testing.T) {
	// A closed/unused port must be reported dead and time out with an error.
	if serverAlive("127.0.0.1:1") {
		t.Fatal("serverAlive reported a dead port as alive")
	}
	if err := waitForServer("127.0.0.1:1", 300*time.Millisecond, nil); err == nil {
		t.Fatal("waitForServer returned nil for an unreachable address")
	}

	// A real listener must be detected as alive and satisfy the wait.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	if !serverAlive(addr) {
		t.Fatal("serverAlive reported a live listener as dead")
	}
	if err := waitForServer(addr, 2*time.Second, nil); err != nil {
		t.Fatalf("waitForServer failed against a live listener: %v", err)
	}

	// A daemon startup error must abort the wait immediately.
	errCh := make(chan error, 1)
	errCh <- net.ErrClosed
	if err := waitForServer("127.0.0.1:1", 5*time.Second, errCh); err == nil {
		t.Fatal("waitForServer ignored a backend startup error")
	}
}
