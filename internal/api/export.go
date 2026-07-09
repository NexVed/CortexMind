package api

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	cgit "github.com/NexVed/Cortex/internal/git"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/NexVed/Cortex/internal/memory"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// ExportResult summarises a memory-export run.
type ExportResult struct {
	ProjectID      string   `json:"project_id"`
	Project        string   `json:"project"`
	RepoPath       string   `json:"repo_path"`
	Files          []string `json:"files"`
	VaultEntries   int      `json:"vault_entries"`
	AgentMemories  int      `json:"agent_memories"`
	SessionDigests int      `json:"session_digests"`
	Committed      bool     `json:"committed"`
	Pushed         bool     `json:"pushed"`
	PushError      string   `json:"push_error,omitempty"`
}

// ExportProjectMemory collects all of a project's agentic memory (vault
// entries, per-session agent memories and session digests) plus its metadata,
// writes a portable bundle into the repository's .cortex/ directory as
// memory.json + README.md, commits it, and optionally pushes it to GitHub so
// teammates receive the shared context.
func (s *Service) ExportProjectMemory(ctx context.Context, user *core.Record, projectID string, push bool) (*ExportResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	bundle := memory.Bundle{
		Project: memory.ProjectMeta{
			Name:         project.GetString("name"),
			Description:  project.GetString("description"),
			GitHubURL:    project.GetString("github_url"),
			Technologies: project.GetStringSlice("technologies"),
		},
	}

	// 1. Vault entries (architecture, decisions, roadmaps, …).
	if recs, err := s.App.FindRecordsByFilter(db.CollVaultEntries, "project = {:p}", "-updated", 2000, 0,
		map[string]any{"p": project.Id}); err == nil {
		for _, r := range recs {
			bundle.VaultEntries = append(bundle.VaultEntries, memory.BundleEntry{
				Category:    r.GetString("category"),
				Title:       r.GetString("title"),
				Content:     r.GetString("content"),
				Tags:        r.GetStringSlice("tags"),
				SourceAgent: r.GetString("source_agent"),
				Version:     r.GetInt("version"),
				UpdatedAt:   isoTime(r, "updated"),
			})
		}
	}

	// 2. Per-session agent memories.
	if recs, err := s.App.FindRecordsByFilter(db.CollAgentMemories, "project = {:p}", "-updated", 5000, 0,
		map[string]any{"p": project.Id}); err == nil {
		for _, r := range recs {
			bundle.AgentMemories = append(bundle.AgentMemories, memory.AgentMemory{
				IDE:       r.GetString("ide"),
				Client:    r.GetString("client_name"),
				SessionID: r.GetString("session_id"),
				Category:  r.GetString("category"),
				Title:     r.GetString("title"),
				Content:   r.GetString("content"),
				Tags:      r.GetStringSlice("tags"),
				UpdatedAt: isoTime(r, "updated"),
			})
		}
	}

	// 3. Session digests.
	if recs, err := s.App.FindRecordsByFilter(db.CollSessionDigests, "project = {:p}", "-updated", 2000, 0,
		map[string]any{"p": project.Id}); err == nil {
		for _, r := range recs {
			bundle.SessionDigests = append(bundle.SessionDigests, memory.SessionDigest{
				SessionID:   r.GetString("session_id"),
				IDE:         r.GetString("ide"),
				Title:       r.GetString("title"),
				SummaryMD:   r.GetString("summary_md"),
				TokenCount:  r.GetInt("token_count"),
				MemoryCount: r.GetInt("memory_count"),
				UpdatedAt:   isoTime(r, "updated"),
			})
		}
	}

	// 4. Resolve a local checkout to write .cortex/ into (clone if necessary).
	repoPath, err := s.resolveRepoPath(project, user)
	if err != nil {
		return nil, err
	}

	files, err := memory.ExportBundle(bundle, repoPath)
	if err != nil {
		return nil, fmt.Errorf("write bundle: %w", err)
	}

	res := &ExportResult{
		ProjectID:      project.Id,
		Project:        project.GetString("name"),
		RepoPath:       repoPath,
		Files:          files,
		VaultEntries:   len(bundle.VaultEntries),
		AgentMemories:  len(bundle.AgentMemories),
		SessionDigests: len(bundle.SessionDigests),
	}

	// 5. Commit the .cortex directory so it travels with the repo.
	if err := s.Git.CommitChanges(repoPath, "chore(cortex): export agent memory bundle"); err != nil {
		log.Warn().Err(err).Str("project", project.Id).Msg("failed to commit memory bundle")
	} else {
		res.Committed = true
	}

	// 6. Optionally push to origin so teammates receive it.
	if push {
		token := user.GetString("github_access_token")
		if token == "" {
			res.PushError = "no GitHub access token on user; sign in with GitHub first"
			log.Warn().Str("project", project.Id).Msg(res.PushError)
		} else if err := s.Git.PushRemote(repoPath, token); err != nil {
			res.PushError = err.Error()
			log.Warn().Err(err).Str("project", project.Id).Msg("failed to push memory bundle")
		} else {
			res.Pushed = true
		}
	}

	db.LogActivity(s.App, project.Id, user.Id, "exported_memory",
		fmt.Sprintf("Exported memory bundle (%d entries, %d agent memories)", res.VaultEntries, res.AgentMemories), nil)
	return res, nil
}

// resolveRepoPath returns a local directory containing the project's checkout,
// cloning from GitHub when no usable local path is recorded.
func (s *Service) resolveRepoPath(project *core.Record, user *core.Record) (string, error) {
	localPath := project.GetString("path")
	if localPath != "" && isDir(localPath) {
		return localPath, nil
	}
	ghURL := project.GetString("github_url")
	if ghURL == "" {
		return "", fmt.Errorf("project has no local path or GitHub URL to export into")
	}
	reposDir := filepath.Join(s.Config.DataDirPath(), "repos")
	localPath = filepath.Join(reposDir, nameSanitizer.ReplaceAllString(repoSlug(ghURL), "__"))
	token := user.GetString("github_access_token")
	if _, err := cgit.EnsureRepo(localPath, cloneURLFromHTML(ghURL), token); err != nil {
		return "", fmt.Errorf("clone repo: %w", err)
	}
	project.Set("path", localPath)
	_ = s.App.Save(project)
	return localPath, nil
}

func isoTime(r *core.Record, field string) string {
	dt := r.GetDateTime(field)
	if dt.IsZero() {
		return ""
	}
	return dt.Time().UTC().Format(time.RFC3339)
}

// ── HTTP handler ───────────────────────────────────────

func (s *Service) handleExportMemory(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	projectID := e.Request.PathValue("project")
	push := e.Request.URL.Query().Get("push") == "true"
	res, err := s.ExportProjectMemory(e.Request.Context(), user, projectID, push)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, res)
}
