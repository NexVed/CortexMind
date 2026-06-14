package rpc

import (
	v1 "github.com/NexVed/Cortex/gen/cortex/v1"
	"github.com/pocketbase/pocketbase/core"
)

func rfc3339(rec *core.Record, field string) string {
	dt := rec.GetDateTime(field)
	if dt.IsZero() {
		return ""
	}
	return dt.Time().Format("2006-01-02T15:04:05Z07:00")
}

func recordToProject(r *core.Record) *v1.Project {
	return &v1.Project{
		Id:           r.Id,
		Name:         r.GetString("name"),
		Path:         r.GetString("path"),
		Description:  r.GetString("description"),
		GithubUrl:    r.GetString("github_url"),
		Status:       r.GetString("status"),
		Progress:     int32(r.GetInt("progress")),
		Technologies: r.GetStringSlice("technologies"),
		LastScanned:  rfc3339(r, "last_scanned"),
		LastActivity: rfc3339(r, "last_activity"),
		IconColor:    r.GetString("icon_color"),
	}
}

func recordToVaultEntry(r *core.Record) *v1.VaultEntry {
	return &v1.VaultEntry{
		Id:          r.Id,
		ProjectId:   r.GetString("project"),
		Category:    r.GetString("category"),
		Title:       r.GetString("title"),
		Content:     r.GetString("content"),
		Tags:        r.GetStringSlice("tags"),
		IsShared:    r.GetBool("is_shared"),
		SourceAgent: r.GetString("source_agent"),
		FilePath:    r.GetString("file_path"),
		Version:     int32(r.GetInt("version")),
	}
}

func recordToTask(r *core.Record) *v1.Task {
	return &v1.Task{
		Id:          r.Id,
		ProjectId:   r.GetString("project"),
		Title:       r.GetString("title"),
		Description: r.GetString("description"),
		Status:      r.GetString("status"),
		Priority:    r.GetString("priority"),
		AssignedTo:  r.GetString("assigned_to"),
		DueDate:     rfc3339(r, "due_date"),
		LinkedFiles: r.GetStringSlice("linked_files"),
		Tags:        r.GetStringSlice("tags"),
	}
}

func recordToHandoff(r *core.Record) *v1.Handoff {
	return &v1.Handoff{
		Id:            r.Id,
		ProjectId:     r.GetString("project"),
		FromAgent:     r.GetString("from_agent"),
		ToAgent:       r.GetString("to_agent"),
		Title:         r.GetString("title"),
		Context:       r.GetString("context"),
		Status:        r.GetString("status"),
		IncludedFiles: r.GetStringSlice("included_files"),
		PromptPreview: r.GetString("prompt_preview"),
		TokenCount:    int32(r.GetInt("token_count")),
	}
}

func recordToActivity(r *core.Record) *v1.ActivityLogEntry {
	// metadata is stored as JSON, but the proto expects a JSON string.
	// Since PocketBase returns it as a string internally or we can just fetch it as raw string:
	metadata := r.GetString("metadata")
	if metadata == "" {
		metadata = "{}"
	}
	return &v1.ActivityLogEntry{
		Id:           r.Id,
		ProjectId:    r.GetString("project"),
		OwnerId:      r.GetString("owner"),
		Action:       r.GetString("action"),
		Subject:      r.GetString("subject"),
		MetadataJson: metadata,
		CreatedAt:    rfc3339(r, "created"),
	}
}
