package api

import (
	"fmt"

	"github.com/NexVed/Cortex/internal/auth"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// GitHubRepo is the shape returned to the UI for repo listing/sync. It mirrors
// auth.Repo but adds an "imported" flag so the UI can show which repos already
// exist as CORTEX projects.
type GitHubRepo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	Language    string `json:"language"`
	Imported    bool   `json:"imported"`
}

// SyncResult summarises a GitHub → projects import.
type SyncResult struct {
	Total    int      `json:"total"`
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Names    []string `json:"names"`
}

// ListGitHubRepos returns every repository the user can access, using the
// access token persisted on their record. Fully paginated, so it is not capped
// at a single page the way a naive client-side fetch would be.
func (s *Service) ListGitHubRepos(user *core.Record) ([]GitHubRepo, error) {
	token := user.GetString("github_access_token")
	if token == "" {
		return nil, fmt.Errorf("no GitHub access token on user; sign in with GitHub first")
	}
	repos, err := auth.NewClient(token).ListUserRepos()
	if err != nil {
		return nil, fmt.Errorf("list github repos: %w", err)
	}

	// Mark repos that already exist as projects.
	existing := map[string]bool{}
	if recs, err := s.App.FindRecordsByFilter(db.CollProjects, "github_url != ''", "", 5000, 0, nil); err == nil {
		for _, r := range recs {
			existing[r.GetString("github_url")] = true
		}
	}

	out := make([]GitHubRepo, 0, len(repos))
	for _, r := range repos {
		out = append(out, GitHubRepo{
			ID:          r.ID,
			Name:        r.Name,
			FullName:    r.FullName,
			Description: r.Description,
			Private:     r.Private,
			HTMLURL:     r.HTMLURL,
			CloneURL:    r.CloneURL,
			Language:    r.Language,
			Imported:    existing[r.HTMLURL],
		})
	}
	return out, nil
}

// SyncGitHubRepos imports every accessible GitHub repository as a CORTEX
// project record (without cloning). Existing projects are left untouched, so it
// is safe to call repeatedly. This is the server-side equivalent of the login
// sync, using the stored token and full pagination.
func (s *Service) SyncGitHubRepos(user *core.Record) (*SyncResult, error) {
	repos, err := s.ListGitHubRepos(user)
	if err != nil {
		return nil, err
	}

	coll, err := s.App.FindCollectionByNameOrId(db.CollProjects)
	if err != nil {
		return nil, err
	}

	res := &SyncResult{Total: len(repos)}
	for _, repo := range repos {
		if repo.Imported {
			res.Skipped++
			continue
		}
		// Guard against a race / duplicate github_url.
		if existing, _ := s.App.FindFirstRecordByFilter(db.CollProjects,
			"github_url = {:u} || github_repo_id = {:id}",
			map[string]any{"u": repo.HTMLURL, "id": fmt.Sprintf("%d", repo.ID)}); existing != nil {
			res.Skipped++
			continue
		}

		rec := core.NewRecord(coll)
		rec.Set("name", repo.Name)
		rec.Set("description", firstNonEmpty(repo.Description, "Imported from GitHub"))
		rec.Set("github_url", repo.HTMLURL)
		rec.Set("github_repo_id", fmt.Sprintf("%d", repo.ID))
		rec.Set("owner", user.Id)
		rec.Set("status", "active")
		rec.Set("progress", 0)
		rec.Set("icon_color", colorFromName(repo.Name))
		if repo.Language != "" {
			rec.Set("technologies", []string{repo.Language})
		}
		if err := s.App.Save(rec); err != nil {
			log.Warn().Err(err).Str("repo", repo.FullName).Msg("failed to import repo")
			res.Skipped++
			continue
		}
		res.Imported++
		res.Names = append(res.Names, repo.FullName)
	}

	db.LogActivity(s.App, "", user.Id, "synced_github_repos",
		fmt.Sprintf("Imported %d of %d repositories from GitHub", res.Imported, res.Total), nil)
	return res, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ── HTTP handlers ──────────────────────────────────────

func (s *Service) handleListGitHubRepos(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	repos, err := s.ListGitHubRepos(user)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, repos)
}

func (s *Service) handleSyncGitHubRepos(e *core.RequestEvent) error {
	user, err := s.resolveUser(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	res, err := s.SyncGitHubRepos(user)
	if err != nil {
		return e.InternalServerError(err.Error(), nil)
	}
	return e.JSON(200, res)
}
