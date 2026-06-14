package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/NexVed/Cortex/gen/cortex/v1"
)

// Version is the daemon version reported over RPC.
const Version = "0.1.0"

func (h *Handlers) Status(ctx context.Context, req *connect.Request[v1.DaemonStatusRequest]) (*connect.Response[v1.DaemonStatusResponse], error) {
	watcherRunning := false
	if h.WatcherRunning != nil {
		watcherRunning = h.WatcherRunning()
	}
	return connect.NewResponse(&v1.DaemonStatusResponse{
		Ready:           true,
		Version:         Version,
		UptimeSeconds:   int64(time.Since(h.Started).Seconds()),
		WatcherRunning:  watcherRunning,
		SemanticEnabled: h.Embedder != nil && h.Vector != nil,
	}), nil
}
