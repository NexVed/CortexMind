package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	v1 "github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
)

func (h *Handlers) ListEntries(ctx context.Context, req *connect.Request[v1.ListVaultEntriesRequest]) (*connect.Response[v1.ListVaultEntriesResponse], error) {
	extra := ""
	if req.Msg.ProjectId != "" {
		extra = "project = {:proj}"
	}
	filter, params := scopedFilter(userID(ctx), extra)
	if req.Msg.ProjectId != "" {
		params["proj"] = req.Msg.ProjectId
	}
	if req.Msg.Category != "" {
		filter += " && category = {:cat}"
		params["cat"] = req.Msg.Category
	}
	records, err := h.App.FindRecordsByFilter(db.CollVaultEntries, filter, "-updated", 500, 0, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListVaultEntriesResponse{}
	for _, r := range records {
		out.Entries = append(out.Entries, recordToVaultEntry(r))
	}
	return connect.NewResponse(out), nil
}

func (h *Handlers) CreateEntry(ctx context.Context, req *connect.Request[v1.CreateVaultEntryRequest]) (*connect.Response[v1.CreateVaultEntryResponse], error) {
	e := req.Msg.Entry
	if e == nil || e.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("entry.title is required"))
	}
	coll, err := h.App.FindCollectionByNameOrId(db.CollVaultEntries)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rec := core.NewRecord(coll)
	applyVaultEntry(rec, e)
	rec.Set("version", 1)
	if uid := userID(ctx); uid != "" {
		rec.Set("owner", uid)
	}
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	db.LogActivity(h.App, e.ProjectId, userID(ctx), "created_vault_entry", e.Title, nil)
	return connect.NewResponse(&v1.CreateVaultEntryResponse{Entry: recordToVaultEntry(rec)}), nil
}

func (h *Handlers) UpdateEntry(ctx context.Context, req *connect.Request[v1.UpdateVaultEntryRequest]) (*connect.Response[v1.UpdateVaultEntryResponse], error) {
	e := req.Msg.Entry
	if e == nil || e.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("entry.id is required"))
	}
	rec, err := h.App.FindRecordById(db.CollVaultEntries, e.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	applyVaultEntry(rec, e)
	rec.Set("version", rec.GetInt("version")+1)
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateVaultEntryResponse{Entry: recordToVaultEntry(rec)}), nil
}

func (h *Handlers) DeleteEntry(ctx context.Context, req *connect.Request[v1.DeleteVaultEntryRequest]) (*connect.Response[v1.DeleteVaultEntryResponse], error) {
	rec, err := h.App.FindRecordById(db.CollVaultEntries, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := h.App.Delete(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteVaultEntryResponse{Success: true}), nil
}

func applyVaultEntry(rec *core.Record, e *v1.VaultEntry) {
	if e.ProjectId != "" {
		rec.Set("project", e.ProjectId)
	}
	rec.Set("category", e.Category)
	rec.Set("title", e.Title)
	rec.Set("content", e.Content)
	rec.Set("tags", e.Tags)
	rec.Set("is_shared", e.IsShared)
	rec.Set("source_agent", e.SourceAgent)
}
