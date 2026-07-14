package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/NexVed/Cortex/internal/analyzer"
	"github.com/NexVed/Cortex/internal/auth"
	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/db"
	cgit "github.com/NexVed/Cortex/internal/git"
	"github.com/NexVed/Cortex/internal/llm"
	"github.com/NexVed/Cortex/internal/memory"
	"github.com/NexVed/Cortex/internal/scanner"
	"github.com/NexVed/Cortex/internal/vector"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/rs/zerolog/log"
)

// Service carries the dependencies shared by the CORTEX HTTP routes.
type Service struct {
	App     core.App
	Scanner *scanner.Scanner
	Config  *config.Config
	Git     *cgit.SyncEngine
	Vector  vector.Store // optional; nil disables vector upserts
}

// RepoResult is the per-repository outcome of a full scan.
type RepoResult struct {
	Name         string   `json:"name"`
	ProjectID    string   `json:"project_id"`
	IndexedFiles int      `json:"indexed_files"`
	Frameworks   []string `json:"frameworks"`
	AuthDetected bool     `json:"auth_detected"`
	Features     int      `json:"features"`
	Error        string   `json:"error,omitempty"`
}

// ScanReport summarises a ScanAll run.
type ScanReport struct {
	TotalRepos int          `json:"total_repos"`
	Scanned    int          `json:"scanned"`
	Failed     int          `json:"failed"`
	Enriched   bool         `json:"enriched"`
	Results    []RepoResult `json:"results"`
}

var nameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// ScanAll lists every GitHub repository the user can access, clones each one,
// scans and analyzes it, and persists the resulting knowledge into PocketBase.
func (s *Service) ScanAll(ctx context.Context, user *core.Record) (*ScanReport, error) {
	token := user.GetString("github_access_token")
	if token == "" {
		return nil, fmt.Errorf("no GitHub access token on user; sign in with GitHub first")
	}

	repos, err := auth.NewClient(token).ListUserRepos()
	if err != nil {
		return nil, fmt.Errorf("list github repos: %w", err)
	}

	provCfg := LoadProviderConfig(user)
	llmClient := llm.New(provCfg)
	embedder := llm.NewEmbedder(provCfg)

	reposDir := filepath.Join(s.Config.DataDirPath(), "repos")
	report := &ScanReport{TotalRepos: len(repos), Enriched: llmClient.Enabled()}

	for _, repo := range repos {
		res := RepoResult{Name: repo.FullName}
		if err := s.scanOne(ctx, user, repo, reposDir, token, llmClient, embedder, &res); err != nil {
			res.Error = err.Error()
			report.Failed++
			log.Warn().Err(err).Str("repo", repo.FullName).Msg("scan failed")
		} else {
			report.Scanned++
		}
		report.Results = append(report.Results, res)
	}

	db.LogActivity(s.App, "", user.Id, "scanned_all_repos",
		fmt.Sprintf("Scanned %d of %d repositories", report.Scanned, report.TotalRepos), nil)
	return report, nil
}

func (s *Service) scanOne(ctx context.Context, user *core.Record, repo auth.Repo, reposDir, token string,
	llmClient llm.Client, embedder vector.Embedder, res *RepoResult) error {

	dest := filepath.Join(reposDir, nameSanitizer.ReplaceAllString(repo.FullName, "__"))
	if _, err := cgit.EnsureRepo(dest, repo.CloneURL, token); err != nil {
		return err
	}

	project, err := s.findOrCreateProject(user, repo, dest)
	if err != nil {
		return err
	}
	res.ProjectID = project.Id
	return s.runFullScan(ctx, user, project, dest, llmClient, embedder, res)
}

// ScanProjectDeep clones (if needed), scans and analyzes a single existing
// project, persisting its knowledge graph and memory. This backs the per-repo
// "Scan" button in the UI.
func (s *Service) ScanProjectDeep(ctx context.Context, user *core.Record, projectID string) (*RepoResult, error) {
	project, err := s.App.FindRecordById(db.CollProjects, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	res := &RepoResult{Name: project.GetString("name"), ProjectID: project.Id}
	token := user.GetString("github_access_token")

	provCfg := LoadProviderConfig(user)
	llmClient := llm.New(provCfg)
	embedder := llm.NewEmbedder(provCfg)

	localPath := project.GetString("path")
	if localPath == "" || !isDir(localPath) {
		ghURL := project.GetString("github_url")
		if ghURL == "" {
			return nil, fmt.Errorf("project has no local path or GitHub URL to scan")
		}
		reposDir := filepath.Join(s.Config.DataDirPath(), "repos")
		localPath = filepath.Join(reposDir, nameSanitizer.ReplaceAllString(repoSlug(ghURL), "__"))
		if _, err := cgit.EnsureRepo(localPath, cloneURLFromHTML(ghURL), token); err != nil {
			return nil, fmt.Errorf("clone repo: %w", err)
		}
		project.Set("path", localPath)
		_ = s.App.Save(project)
	}

	if err := s.runFullScan(ctx, user, project, localPath, llmClient, embedder, res); err != nil {
		return nil, err
	}
	return res, nil
}

// runFullScan executes the scanner, the deep analyzer and persistence for an
// already-resolved project whose code is available at repoPath.
func (s *Service) runFullScan(ctx context.Context, user *core.Record, project *core.Record, repoPath string,
	llmClient llm.Client, embedder vector.Embedder, res *RepoResult) error {

	res.Name = project.GetString("name")
	res.ProjectID = project.Id

	scanRes, err := s.Scanner.Run(scanner.Job{
		ProjectID: project.Id,
		RepoPath:  repoPath,
		OwnerID:   user.Id,
		FullScan:  true,
	})
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	res.IndexedFiles = scanRes.IndexedFiles

	analysis, err := analyzer.Analyze(ctx, analyzer.Input{
		RepoPath:    repoPath,
		ProjectName: project.GetString("name"),
		Languages:   scanRes.Languages,
		EntryPoints: scanRes.EntryPoints,
	}, llmClient)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	res.Frameworks = analysis.TechStack.Frameworks
	res.AuthDetected = analysis.Auth.Detected
	res.Features = len(analysis.Features)

	if err := s.persistAnalysis(ctx, user, project, repoPath, analysis, embedder); err != nil {
		log.Warn().Err(err).Str("project", project.Id).Msg("failed to persist analysis")
	}
	// Keep the code graph in sync with every successful scan so graph consumers
	// never depend on a separate manual rebuild step.
	if _, err := s.BuildAndStoreCodeGraph(ctx, user, project.Id); err != nil {
		log.Warn().Err(err).Str("project", project.Id).Msg("failed to rebuild code graph")
	}
	return nil
}

func (s *Service) findOrCreateProject(user *core.Record, repo auth.Repo, localPath string) (*core.Record, error) {
	existing, _ := s.App.FindFirstRecordByFilter(db.CollProjects,
		"github_url = {:u} || github_repo_id = {:id}",
		map[string]any{"u": repo.HTMLURL, "id": fmt.Sprintf("%d", repo.ID)})
	if existing != nil {
		existing.Set("path", localPath)
		existing.Set("github_repo_id", fmt.Sprintf("%d", repo.ID))
		if existing.GetString("github_url") == "" {
			existing.Set("github_url", repo.HTMLURL)
		}
		return existing, s.App.Save(existing)
	}

	coll, err := s.App.FindCollectionByNameOrId(db.CollProjects)
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(coll)
	rec.Set("name", repo.Name)
	rec.Set("path", localPath)
	rec.Set("description", "Imported from GitHub")
	rec.Set("github_url", repo.HTMLURL)
	rec.Set("github_repo_id", fmt.Sprintf("%d", repo.ID))
	rec.Set("owner", user.Id)
	rec.Set("status", "active")
	rec.Set("progress", 0)
	rec.Set("icon_color", colorFromName(repo.Name))
	if err := s.App.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// persistAnalysis stores the analysis on the project record, enriches the
// latest scan_results, writes a shareable architecture memory entry, exports it
// to the repo's .cortex directory and (best-effort) builds an embedding.
func (s *Service) persistAnalysis(ctx context.Context, user *core.Record, project *core.Record,
	repoPath string, analysis *analyzer.Analysis, embedder vector.Embedder) error {

	// 1. Project metadata + technologies.
	project.Set("metadata", analysis)
	techs := append([]string{}, analysis.TechStack.Frameworks...)
	techs = append(techs, analysis.TechStack.Languages...)
	if len(techs) > 0 {
		project.Set("technologies", dedupe(techs))
	}
	project.Set("last_activity", types.NowDateTime())
	if err := s.App.Save(project); err != nil {
		return err
	}

	// 2. Enrich the latest scan_results record.
	if recs, err := s.App.FindRecordsByFilter(db.CollScanResults, "project = {:p}", "-scanned_at", 1, 0,
		map[string]any{"p": project.Id}); err == nil && len(recs) > 0 {
		sr := recs[0]
		sr.Set("modules", map[string]any{
			"features":  analysis.Features,
			"structure": analysis.Structure,
			"auth":      analysis.Auth,
		})
		sr.Set("api_endpoints", analysis.Endpoints)
		if analysis.Summary != "" {
			sr.Set("summary", analysis.Summary)
		}
		_ = s.App.Save(sr)
	}

	// 3. Architecture memory entry (the IDE-facing "brain" record).
	entry, err := s.upsertArchitectureEntry(user, project, analysis)
	if err != nil {
		return err
	}

	// 4. Export to .cortex and commit so it travels with the repo.
	if entry != nil {
		if rel, err := memory.ExportEntryToFile(entry, repoPath); err == nil {
			entry.Set("file_path", rel)
			_ = s.App.Save(entry)
			_ = s.Git.CommitChanges(repoPath, "chore(cortex): update architecture memory")
		}
	}

	// 5. Optional embedding for semantic memory.
	if embedder != nil && s.Vector != nil {
		if vec, err := embedder.Embed(ctx, analysis.Summary); err == nil {
			_ = s.Vector.Upsert(ctx, vector.Record{
				ID:         "project_" + project.Id,
				ProjectID:  project.Id,
				Collection: "project",
				Text:       analysis.Summary,
				Vector:     vec,
				Metadata:   map[string]string{"name": project.GetString("name")},
			})
		}
	}
	return nil
}

func (s *Service) upsertArchitectureEntry(user *core.Record, project *core.Record, analysis *analyzer.Analysis) (*core.Record, error) {
	title := project.GetString("name") + " — Architecture Overview"
	content := renderArchitectureMarkdown(analysis)

	existing, _ := s.App.FindFirstRecordByFilter(db.CollVaultEntries,
		"project = {:p} && category = 'architecture' && title = {:t}",
		map[string]any{"p": project.Id, "t": title})

	var rec *core.Record
	if existing != nil {
		rec = existing
		rec.Set("version", existing.GetInt("version")+1)
	} else {
		coll, err := s.App.FindCollectionByNameOrId(db.CollVaultEntries)
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(coll)
		rec.Set("project", project.Id)
		rec.Set("owner", user.Id)
		rec.Set("category", "architecture")
		rec.Set("title", title)
		rec.Set("version", 1)
	}
	rec.Set("content", content)
	rec.Set("is_shared", true)
	rec.Set("source_agent", "cortex-scanner")
	rec.Set("tags", analysis.TechStack.Frameworks)
	if err := s.App.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func renderArchitectureMarkdown(a *analyzer.Analysis) string {
	var b strings.Builder
	b.WriteString(a.Summary + "\n\n")
	b.WriteString("## Tech Stack\n")
	writeList(&b, "Languages", a.TechStack.Languages)
	writeList(&b, "Frameworks", a.TechStack.Frameworks)
	writeList(&b, "Databases", a.TechStack.Databases)
	writeList(&b, "Tools", a.TechStack.Tools)
	writeList(&b, "Package Managers", a.TechStack.PackageManagers)

	b.WriteString("\n## Authentication\n")
	if a.Auth.Detected {
		writeList(&b, "Mechanisms", a.Auth.Mechanisms)
		writeList(&b, "Providers", a.Auth.Providers)
		writeList(&b, "Libraries", a.Auth.Libraries)
	} else {
		b.WriteString("- No authentication detected\n")
	}

	b.WriteString("\n## Features\n")
	for _, f := range a.Features {
		b.WriteString(fmt.Sprintf("- **%s** — %s\n", f.Name, f.Description))
	}

	if len(a.Endpoints) > 0 {
		b.WriteString("\n## API Endpoints\n")
		for i, e := range a.Endpoints {
			if i >= 40 {
				b.WriteString(fmt.Sprintf("- …and %d more\n", len(a.Endpoints)-40))
				break
			}
			b.WriteString("- `" + e + "`\n")
		}
	}
	return b.String()
}

func writeList(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("- **" + label + "**: " + strings.Join(items, ", ") + "\n")
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func colorFromName(name string) string {
	var hash int32
	for _, c := range name {
		hash = (hash << 5) - hash + c
	}
	if hash < 0 {
		hash = -hash
	}
	h := hash % 360
	return fmt.Sprintf("hsl(%d, 65%%, 55%%)", h)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// repoSlug extracts "owner/name" from a GitHub HTML URL.
func repoSlug(htmlURL string) string {
	s := strings.TrimSuffix(htmlURL, "/")
	s = strings.TrimPrefix(s, "https://github.com/")
	s = strings.TrimPrefix(s, "http://github.com/")
	s = strings.TrimSuffix(s, ".git")
	if s == "" {
		return "repo"
	}
	return s
}

// cloneURLFromHTML turns a GitHub HTML URL into a clonable .git URL.
func cloneURLFromHTML(htmlURL string) string {
	s := strings.TrimSuffix(htmlURL, "/")
	if strings.HasSuffix(s, ".git") {
		return s
	}
	return s + ".git"
}
