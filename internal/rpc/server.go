package rpc

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/NexVed/Cortex/gen/cortex/v1/cortexv1connect"
	"github.com/NexVed/Cortex/internal/scanner"
	"github.com/NexVed/Cortex/internal/search"
	"github.com/NexVed/Cortex/internal/vector"
	"github.com/pocketbase/pocketbase/core"
)

// Handlers implements every CORTEX ConnectRPC service. A single type backs all
// services so they can share the PocketBase app and engine dependencies.
type Handlers struct {
	App          core.App
	Scanner      *scanner.Scanner
	SearchEngine *search.Engine
	Embedder     vector.Embedder // nil when semantic search is disabled
	Vector       vector.Store    // nil when no vector sidecar is configured

	Started        time.Time
	WatcherRunning func() bool
}

// Mount registers all ConnectRPC service handlers onto the provided router
// registrar (PocketBase's router). The auth interceptor is applied to every
// service.
func (h *Handlers) Mount(register func(pattern string, handler http.Handler)) {
	opts := connect.WithInterceptors(AuthInterceptor(h.App))

	mounts := []struct {
		path    string
		handler http.Handler
	}{}

	p, hd := cortexv1connect.NewProjectServiceHandler(h, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	p, hd = cortexv1connect.NewVaultServiceHandler(h, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	p, hd = cortexv1connect.NewTaskServiceHandler(h, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	p, hd = cortexv1connect.NewHandoffServiceHandler(h, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	p, hd = cortexv1connect.NewSearchServiceHandler(h, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	p, hd = cortexv1connect.NewDaemonServiceHandler(h, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	p, hd = cortexv1connect.NewActivityServiceHandler(&ActivityServiceHandler{h}, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	p, hd = cortexv1connect.NewAuthServiceHandler(&AuthServiceHandler{h}, opts)
	mounts = append(mounts, struct {
		path    string
		handler http.Handler
	}{p, hd})

	for _, m := range mounts {
		// Connect handler paths look like "/cortex.v1.ProjectService/".
		// Mount with a trailing wildcard so the sub-path (method name) is
		// passed through to the Connect handler.
		register(m.path, m.handler)
	}
}
