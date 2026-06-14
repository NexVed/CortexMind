package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	v1 "github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
)

func (h *Handlers) ListTasks(ctx context.Context, req *connect.Request[v1.ListTasksRequest]) (*connect.Response[v1.ListTasksResponse], error) {
	extra := ""
	if req.Msg.ProjectId != "" {
		extra = "project = {:proj}"
	}
	filter, params := scopedFilter(userID(ctx), extra)
	if req.Msg.ProjectId != "" {
		params["proj"] = req.Msg.ProjectId
	}
	if req.Msg.Status != "" {
		filter += " && status = {:st}"
		params["st"] = req.Msg.Status
	}
	records, err := h.App.FindRecordsByFilter(db.CollTasks, filter, "-updated", 500, 0, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListTasksResponse{}
	for _, r := range records {
		out.Tasks = append(out.Tasks, recordToTask(r))
	}
	return connect.NewResponse(out), nil
}

func (h *Handlers) CreateTask(ctx context.Context, req *connect.Request[v1.CreateTaskRequest]) (*connect.Response[v1.CreateTaskResponse], error) {
	t := req.Msg.Task
	if t == nil || t.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("task.title is required"))
	}
	coll, err := h.App.FindCollectionByNameOrId(db.CollTasks)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rec := core.NewRecord(coll)
	applyTask(rec, t)
	if t.Status == "" {
		rec.Set("status", "todo")
	}
	if uid := userID(ctx); uid != "" {
		rec.Set("owner", uid)
	}
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateTaskResponse{Task: recordToTask(rec)}), nil
}

func (h *Handlers) UpdateTask(ctx context.Context, req *connect.Request[v1.UpdateTaskRequest]) (*connect.Response[v1.UpdateTaskResponse], error) {
	t := req.Msg.Task
	if t == nil || t.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("task.id is required"))
	}
	rec, err := h.App.FindRecordById(db.CollTasks, t.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	applyTask(rec, t)
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateTaskResponse{Task: recordToTask(rec)}), nil
}

func (h *Handlers) DeleteTask(ctx context.Context, req *connect.Request[v1.DeleteTaskRequest]) (*connect.Response[v1.DeleteTaskResponse], error) {
	rec, err := h.App.FindRecordById(db.CollTasks, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := h.App.Delete(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteTaskResponse{Success: true}), nil
}

func applyTask(rec *core.Record, t *v1.Task) {
	if t.ProjectId != "" {
		rec.Set("project", t.ProjectId)
	}
	rec.Set("title", t.Title)
	rec.Set("description", t.Description)
	if t.Status != "" {
		rec.Set("status", t.Status)
	}
	if t.Priority != "" {
		rec.Set("priority", t.Priority)
	}
	rec.Set("assigned_to", t.AssignedTo)
	rec.Set("linked_files", t.LinkedFiles)
	rec.Set("tags", t.Tags)
}
