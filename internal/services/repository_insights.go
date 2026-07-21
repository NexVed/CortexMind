package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NexVed/Cortex/internal/database"
	"github.com/NexVed/Cortex/internal/scanner"
	gogit "github.com/go-git/go-git/v5"
)

type LanguageStat struct {
	Name       string  `json:"name"`
	Bytes      int64   `json:"bytes"`
	Percentage float64 `json:"percentage"`
}
type RepositoryInsights struct {
	ProjectID   string         `json:"project_id"`
	Language    string         `json:"language"`
	SizeBytes   int64          `json:"size_bytes"`
	Files       int            `json:"files"`
	LinesOfCode int            `json:"lines_of_code"`
	LastCommit  string         `json:"last_commit"`
	License     string         `json:"license"`
	Available   bool           `json:"available"`
	Languages   []LanguageStat `json:"languages"`
}
type RepositoryInsightsService struct{ DB *database.DB }

func (s RepositoryInsightsService) Get(projectID string) (RepositoryInsights, error) {
	if _, err := s.DB.Project(projectID); err != nil {
		return RepositoryInsights{}, fmt.Errorf("project not found")
	}
	path, err := s.DB.RepositoryPath(projectID)
	if err != nil {
		return RepositoryInsights{ProjectID: projectID, Language: "Not scanned", License: "Not detected"}, nil
	}
	result := RepositoryInsights{ProjectID: projectID, Language: "Unknown", License: "Not detected", Available: true}
	languageCount := map[string]int{}
	languageBytes := map[string]int64{}
	var sourceBytes int64
	_ = filepath.Walk(path, func(file string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		result.Files++
		result.SizeBytes += info.Size()
		if language := scanner.DetectLanguage(file); language != "" {
			languageCount[language]++
			languageBytes[language] += info.Size()
			sourceBytes += info.Size()
		}
		name := strings.ToLower(info.Name())
		if result.License == "Not detected" && (name == "license" || strings.HasPrefix(name, "license.") || name == "copying") {
			result.License = info.Name()
		}
		if info.Size() > 2*1024*1024 {
			return nil
		}
		handle, openErr := os.Open(file)
		if openErr != nil {
			return nil
		}
		defer handle.Close()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := handle.Read(buf)
			result.LinesOfCode += strings.Count(string(buf[:n]), "\n")
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				break
			}
		}
		return nil
	})
	type pair struct {
		name  string
		count int
	}
	ranked := make([]pair, 0, len(languageCount))
	for name, count := range languageCount {
		ranked = append(ranked, pair{name, count})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].count > ranked[j].count })
	if len(ranked) > 0 {
		result.Language = ranked[0].name
	}
	for name, bytes := range languageBytes {
		percent := 0.0
		if sourceBytes > 0 {
			percent = float64(bytes) * 100 / float64(sourceBytes)
		}
		result.Languages = append(result.Languages, LanguageStat{Name: name, Bytes: bytes, Percentage: percent})
	}
	sort.Slice(result.Languages, func(i, j int) bool { return result.Languages[i].Bytes > result.Languages[j].Bytes })
	if repo, openErr := gogit.PlainOpen(path); openErr == nil {
		if head, headErr := repo.Head(); headErr == nil {
			if commit, commitErr := repo.CommitObject(head.Hash()); commitErr == nil {
				result.LastCommit = commit.Committer.When.UTC().Format("2006-01-02 15:04 UTC")
			}
		}
	}
	return result, nil
}
