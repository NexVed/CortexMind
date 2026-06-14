package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	v1 "github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
)

func (h *Handlers) ListHandoffs(ctx context.Context, req *connect.Request[v1.ListHandoffsRequest]) (*connect.Response[v1.ListHandoffsResponse], error) {
	extra := ""
	if req.Msg.ProjectId != "" {
		extra = "project = {:proj}"
	}
	filter, params := scopedFilter(userID(ctx), extra)
	if req.Msg.ProjectId != "" {
		params["proj"] = req.Msg.ProjectId
	}
	if req.Msg.ToAgent != "" {
		filter += " && to_agent = {:ta}"
		params["ta"] = req.Msg.ToAgent
	}
	records, err := h.App.FindRecordsByFilter(db.CollHandoffs, filter, "-updated", 200, 0, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListHandoffsResponse{}
	for _, r := range records {
		out.Handoffs = append(out.Handoffs, recordToHandoff(r))
	}
	return connect.NewResponse(out), nil
}

func (h *Handlers) CreateHandoff(ctx context.Context, req *connect.Request[v1.CreateHandoffRequest]) (*connect.Response[v1.CreateHandoffResponse], error) {
	ho := req.Msg.Handoff
	if ho == nil || ho.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("handoff.title is required"))
	}
	coll, err := h.App.FindCollectionByNameOrId(db.CollHandoffs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rec := core.NewRecord(coll)
	if ho.ProjectId != "" {
		rec.Set("project", ho.ProjectId)
	}
	rec.Set("from_agent", ho.FromAgent)
	rec.Set("to_agent", ho.ToAgent)
	rec.Set("title", ho.Title)
	rec.Set("context", ho.Context)
	rec.Set("status", "active")
	rec.Set("included_files", ho.IncludedFiles)
	rec.Set("prompt_preview", buildPromptPreview(ho))
	rec.Set("token_count", estimateTokens(ho.Context))
	if uid := userID(ctx); uid != "" {
		rec.Set("owner", uid)
	}
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	db.LogActivity(h.App, ho.ProjectId, userID(ctx), "created_handoff", ho.Title,
		map[string]any{"to_agent": ho.ToAgent})
	return connect.NewResponse(&v1.CreateHandoffResponse{Handoff: recordToHandoff(rec)}), nil
}

func (h *Handlers) ConsumeHandoff(ctx context.Context, req *connect.Request[v1.ConsumeHandoffRequest]) (*connect.Response[v1.ConsumeHandoffResponse], error) {
	rec, err := h.App.FindRecordById(db.CollHandoffs, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	rec.Set("status", "consumed")
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ConsumeHandoffResponse{Handoff: recordToHandoff(rec)}), nil
}

func buildPromptPreview(ho *v1.Handoff) string {
	return "# Handoff: " + ho.Title + "\n\nFrom: " + ho.FromAgent + " → " + ho.ToAgent + "\n\n" + ho.Context
}

// estimateTokens is a rough heuristic (~4 chars per token).
func estimateTokens(text string) int {
	return len(text) / 4
}
