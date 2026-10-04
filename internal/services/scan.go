package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/database"
	cgit "github.com/NexVed/Cortex/internal/git"
	"github.com/NexVed/Cortex/internal/scanner"
)

type ScanService struct {
	DB          *database.DB
	Config      *config.Config
	GitHubToken string
}
type ScanResult struct {
	Name         string   `json:"name"`
	ProjectID    string   `json:"project_id"`
	IndexedFiles int      `json:"indexed_files"`
	Frameworks   []string `json:"frameworks"`
	AuthDetected bool     `json:"auth_detected"`
	Features     int      `json:"features"`
}

func (s ScanService) Scan(ctx context.Context, projectID string) (*ScanResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	project, err := s.DB.Project(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}
	path := project.Path
	if repo, repoErr := s.DB.Repository(projectID); repoErr == nil {
		if s.GitHubToken == "" && repo.Private {
			return nil, fmt.Errorf("connect GitHub before scanning this private repository")
		}
		path = filepath.Join(s.Config.DataDirPath(), "repositories", strings.ReplaceAll(repo.FullName, "/", "__"))
		if _, err = cgit.EnsureRepo(ctx, path, repo.CloneURL, s.GitHubToken); err != nil {
			return nil, err
		}
	}
	if path == "" {
		return nil, fmt.Errorf("select a local project directory before scanning")
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("project directory does not exist")
	}
	count := 0
	langs := map[string]bool{}
	files := []map[string]any{}
	store := database.RecordStore{DB: s.DB}
	err = filepath.Walk(path, func(p string, info os.FileInfo, e error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e != nil {
			return e
		}
		if info.IsDir() {
			if p != path {
				for _, ignored := range append([]string{".git", "node_modules", "vendor"}, s.Config.Scanner.IgnoredDirs...) {
					if info.Name() == ignored {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}
		if !info.Mode().IsRegular() || info.Size() > int64(s.Config.Scanner.MaxFileSizeKB)*1024 {
			return nil
		}
		if lang := scanner.DetectLanguage(p); lang != "" {
			count++
			langs[lang] = true
			rel, _ := filepath.Rel(path, p)
			files = append(files, map[string]any{"project": projectID, "path": filepath.ToSlash(rel), "language": lang, "size_bytes": info.Size(), "last_indexed": time.Now().UTC().Format(time.RFC3339)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err = store.ReplaceProjectFiles(ctx, projectID, files); err != nil {
		return nil, err
	}
	if err := s.DB.SaveRepositoryScan(projectID, path, count); err != nil {
		return nil, err
	}
	if _, err := (CodeGraphService{DB: s.DB}).Build(projectID); err != nil {
		return nil, fmt.Errorf("build code graph: %w", err)
	}
	_, _ = store.Create("activity_log", map[string]any{
		"project": projectID, "action": "Scanned repository",
		"subject": fmt.Sprintf("%d files indexed and code graph updated", count),
	})
	frameworks := make([]string, 0, len(langs))
	for lang := range langs {
		frameworks = append(frameworks, lang)
	}
	return &ScanResult{Name: project.Name, ProjectID: projectID, IndexedFiles: count, Frameworks: frameworks}, nil
}
