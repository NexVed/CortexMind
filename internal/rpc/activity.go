package rpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/NexVed/Cortex/gen/cortex/v1/cortexv1connect"
	"github.com/NexVed/Cortex/internal/db"
)

// ActivityServiceHandler implements the ConnectRPC ActivityService.
type ActivityServiceHandler struct {
	*Handlers
}

var _ cortexv1connect.ActivityServiceHandler = (*ActivityServiceHandler)(nil)

func (h *ActivityServiceHandler) ListActivity(
	ctx context.Context,
	req *connect.Request[cortexv1.ListActivityRequest],
) (*connect.Response[cortexv1.ListActivityResponse], error) {
	uid := userID(ctx)
	filter := ""
	if req.Msg.ProjectId != "" {
		filter = "project = '" + req.Msg.ProjectId + "'"
	}

	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	// db.CollActivityLog already has ListRules that restrict to the user's projects
	// but we enforce an owner check anyway for safety.
	if uid != "" && filter == "" {
		filter = "owner = '" + uid + "'"
	} else if uid != "" {
		filter += " && owner = '" + uid + "'"
	}

	records, err := h.App.FindRecordsByFilter(
		db.CollActivityLog,
		filter,
		"-created",
		limit,
		0,
		nil,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var entries []*cortexv1.ActivityLogEntry
	for _, rec := range records {
		entries = append(entries, recordToActivity(rec))
	}

	return connect.NewResponse(&cortexv1.ListActivityResponse{
		Entries: entries,
	}), nil
}
