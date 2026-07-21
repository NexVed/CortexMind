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
	_ = ctx
	repo, err := s.DB.Repository(projectID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}
	token := s.GitHubToken
	if token == "" {
		return nil, fmt.Errorf("GitHub is not connected")
	}
	path := filepath.Join(s.Config.DataDirPath(), "repositories", strings.ReplaceAll(repo.FullName, "/", "__"))
	if _, err = cgit.EnsureRepo(path, repo.CloneURL, token); err != nil {
		return nil, err
	}
	count := 0
	langs := map[string]bool{}
	store := database.RecordStore{DB: s.DB}
	err = filepath.Walk(path, func(p string, info os.FileInfo, e error) error {
		if e != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() > int64(s.Config.Scanner.MaxFileSizeKB)*1024 {
			return nil
		}
		if lang := scanner.DetectLanguage(p); lang != "" {
			count++
			langs[lang] = true
			rel, _ := filepath.Rel(path, p)
			_, _ = store.Create("file_index", map[string]any{"project": projectID, "path": filepath.ToSlash(rel), "language": lang, "size_bytes": info.Size(), "last_indexed": time.Now().UTC().Format(time.RFC3339)})
		}
		return nil
	})
	if err != nil {
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
	return &ScanResult{Name: repo.Name, ProjectID: projectID, IndexedFiles: count, Frameworks: frameworks}, nil
}
