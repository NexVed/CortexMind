package rpc

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/NexVed/Cortex/internal/scanner"
	"github.com/pocketbase/pocketbase/core"
)

// scopedFilter returns a filter restricting results to the current user, or a
// tautology when running in local (unauthenticated) mode.
func scopedFilter(uid, extra string) (string, map[string]any) {
	params := map[string]any{}
	base := "id != ''"
	if uid != "" {
		base = "owner = {:uid}"
		params["uid"] = uid
	}
	if extra != "" {
		base = "(" + base + ") && (" + extra + ")"
	}
	return base, params
}

func (h *Handlers) ListProjects(ctx context.Context, req *connect.Request[v1.ListProjectsRequest]) (*connect.Response[v1.ListProjectsResponse], error) {
	filter, params := scopedFilter(userID(ctx), "")
	records, err := h.App.FindRecordsByFilter(db.CollProjects, filter, "-last_activity", 200, 0, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListProjectsResponse{}
	for _, r := range records {
		out.Projects = append(out.Projects, recordToProject(r))
	}
	return connect.NewResponse(out), nil
}

func (h *Handlers) GetProject(ctx context.Context, req *connect.Request[v1.GetProjectRequest]) (*connect.Response[v1.GetProjectResponse], error) {
	rec, err := h.App.FindRecordById(db.CollProjects, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", req.Msg.Id))
	}
	return connect.NewResponse(&v1.GetProjectResponse{Project: recordToProject(rec)}), nil
}

func (h *Handlers) CreateProject(ctx context.Context, req *connect.Request[v1.CreateProjectRequest]) (*connect.Response[v1.CreateProjectResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	coll, err := h.App.FindCollectionByNameOrId(db.CollProjects)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rec := core.NewRecord(coll)
	rec.Set("name", req.Msg.Name)
	rec.Set("path", req.Msg.Path)
	rec.Set("description", req.Msg.Description)
	rec.Set("github_url", req.Msg.GithubUrl)
	rec.Set("status", "active")
	rec.Set("progress", 0)
	rec.Set("icon_color", colorFromName(req.Msg.Name))
	if uid := userID(ctx); uid != "" {
		rec.Set("owner", uid)
	}
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	db.LogActivity(h.App, rec.Id, userID(ctx), "created_project", req.Msg.Name, nil)
	return connect.NewResponse(&v1.CreateProjectResponse{Project: recordToProject(rec)}), nil
}

func (h *Handlers) UpdateProject(ctx context.Context, req *connect.Request[v1.UpdateProjectRequest]) (*connect.Response[v1.UpdateProjectResponse], error) {
	p := req.Msg.Project
	if p == nil || p.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project.id is required"))
	}
	rec, err := h.App.FindRecordById(db.CollProjects, p.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	rec.Set("name", p.Name)
	rec.Set("path", p.Path)
	rec.Set("description", p.Description)
	rec.Set("github_url", p.GithubUrl)
	if p.Status != "" {
		rec.Set("status", p.Status)
	}
	rec.Set("progress", p.Progress)
	if err := h.App.Save(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateProjectResponse{Project: recordToProject(rec)}), nil
}

func (h *Handlers) DeleteProject(ctx context.Context, req *connect.Request[v1.DeleteProjectRequest]) (*connect.Response[v1.DeleteProjectResponse], error) {
	rec, err := h.App.FindRecordById(db.CollProjects, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := h.App.Delete(rec); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteProjectResponse{Success: true}), nil
}

func (h *Handlers) ScanProject(ctx context.Context, req *connect.Request[v1.ScanProjectRequest]) (*connect.Response[v1.ScanProjectResponse], error) {
	rec, err := h.App.FindRecordById(db.CollProjects, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	path := rec.GetString("path")
	if path == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("project has no local path"))
	}
	res, err := h.Scanner.Run(scanner.Job{
		ProjectID: rec.Id,
		RepoPath:  path,
		OwnerID:   userID(ctx),
		FullScan:  req.Msg.FullScan,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	langs := map[string]int32{}
	for k, v := range res.Languages {
		langs[k] = int32(v)
	}
	return connect.NewResponse(&v1.ScanProjectResponse{
		TotalFiles:   int32(res.TotalFiles),
		IndexedFiles: int32(res.IndexedFiles),
		Languages:    langs,
		Summary:      res.Summary,
	}), nil
}

func (h *Handlers) GetProjectStats(ctx context.Context, req *connect.Request[v1.ProjectStatsRequest]) (*connect.Response[v1.ProjectStatsResponse], error) {
	recs, err := h.App.FindRecordsByFilter(db.CollScanResults, "project = {:p}", "-scanned_at", 1, 0, map[string]any{"p": req.Msg.Id})
	if err != nil || len(recs) == 0 {
		return connect.NewResponse(&v1.ProjectStatsResponse{Languages: map[string]int32{}}), nil
	}
	r := recs[0]
	langs := map[string]int32{}
	raw := map[string]int{}
	if err := r.UnmarshalJSONField("languages", &raw); err == nil {
		for k, v := range raw {
			langs[k] = int32(v)
		}
	}
	return connect.NewResponse(&v1.ProjectStatsResponse{
		TotalFiles:   int32(r.GetInt("total_files")),
		IndexedFiles: int32(r.GetInt("indexed_files")),
		Languages:    langs,
	}), nil
}

func colorFromName(name string) string {
	sum := sha1.Sum([]byte(name))
	return "#" + hex.EncodeToString(sum[:])[:6]
}
