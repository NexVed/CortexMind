package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/rs/zerolog/log"
)

// Scanner analyzes a local repository and populates the file_index and
// scan_results collections.
type Scanner struct {
	App    core.App
	Config *config.ScannerConfig
}

func New(app core.App, cfg *config.ScannerConfig) *Scanner {
	return &Scanner{App: app, Config: cfg}
}

// Job describes a single scan request.
type Job struct {
	ProjectID string
	RepoPath  string
	OwnerID   string
	FullScan  bool
}

// Result is the summary returned by Run.
type Result struct {
	TotalFiles   int            `json:"total_files"`
	IndexedFiles int            `json:"indexed_files"`
	Languages    map[string]int `json:"languages"`
	Dependencies map[string]any `json:"dependencies"`
	EntryPoints  []string       `json:"entry_points"`
	Summary      string         `json:"summary"`
}

// Run executes a scan job: walks the repo, indexes files, aggregates stats,
// and writes scan_results.
func (s *Scanner) Run(job Job) (*Result, error) {
	info, err := os.Stat(job.RepoPath)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("invalid repo path %q: %w", job.RepoPath, err)
	}

	res := &Result{Languages: map[string]int{}, Dependencies: map[string]any{}, EntryPoints: []string{}}
	maxBytes := int64(s.Config.MaxFileSizeKB) * 1024

	walkErr := filepath.Walk(job.RepoPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if fi.IsDir() {
			if s.isIgnoredDir(fi.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		res.TotalFiles++

		if s.isIgnoredExt(path) || (maxBytes > 0 && fi.Size() > maxBytes) {
			return nil
		}
		lang := DetectLanguage(path)
		if lang == "" {
			return nil
		}

		rel, _ := filepath.Rel(job.RepoPath, path)
		rel = filepath.ToSlash(rel)

		if err := s.indexFile(job, path, rel, lang, fi.Size()); err != nil {
			log.Warn().Err(err).Str("file", rel).Msg("failed to index file")
			return nil
		}
		res.IndexedFiles++
		res.Languages[lang]++
		if isEntryPoint(rel) {
			res.EntryPoints = append(res.EntryPoints, rel)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	s.detectDependencies(job.RepoPath, res)
	res.Summary = buildSummary(res)

	if err := s.writeScanResult(job, res); err != nil {
		return nil, err
	}
	s.updateProject(job, res)

	db.LogActivity(s.App, job.ProjectID, job.OwnerID, "indexed_files",
		fmt.Sprintf("Indexed %d files", res.IndexedFiles),
		map[string]any{"languages": res.Languages})

	return res, nil
}

func (s *Scanner) indexFile(job Job, absPath, rel, lang string, size int64) error {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	coll, err := s.App.FindCollectionByNameOrId(db.CollFileIndex)
	if err != nil {
		return err
	}

	// Upsert: reuse the existing record (skip if unchanged) or create new.
	existing, _ := s.App.FindFirstRecordByFilter(db.CollFileIndex,
		"project = {:p} && path = {:path}",
		map[string]any{"p": job.ProjectID, "path": rel})

	var rec *core.Record
	if existing != nil {
		if !job.FullScan && existing.GetString("checksum") == checksum {
			return nil // unchanged
		}
		rec = existing
	} else {
		rec = core.NewRecord(coll)
		rec.Set("project", job.ProjectID)
		rec.Set("path", rel)
	}

	syms := ExtractSymbols(string(data), lang)
	rec.Set("language", lang)
	rec.Set("size_bytes", size)
	rec.Set("functions", syms.Functions)
	rec.Set("classes", syms.Classes)
	rec.Set("imports", syms.Imports)
	rec.Set("checksum", checksum)
	rec.Set("last_indexed", types.NowDateTime())
	return s.App.Save(rec)
}

func (s *Scanner) writeScanResult(job Job, res *Result) error {
	coll, err := s.App.FindCollectionByNameOrId(db.CollScanResults)
	if err != nil {
		return err
	}
	rec := core.NewRecord(coll)
	rec.Set("project", job.ProjectID)
	rec.Set("scanned_at", types.NowDateTime())
	rec.Set("total_files", res.TotalFiles)
	rec.Set("indexed_files", res.IndexedFiles)
	rec.Set("languages", res.Languages)
	rec.Set("dependencies", res.Dependencies)
	rec.Set("entry_points", res.EntryPoints)
	rec.Set("summary", res.Summary)
	return s.App.Save(rec)
}

func (s *Scanner) updateProject(job Job, res *Result) {
	proj, err := s.App.FindRecordById(db.CollProjects, job.ProjectID)
	if err != nil {
		return
	}
	techs := make([]string, 0, len(res.Languages))
	for lang := range res.Languages {
		techs = append(techs, lang)
	}
	sort.Strings(techs)
	proj.Set("technologies", techs)
	proj.Set("last_scanned", types.NowDateTime())
	proj.Set("last_activity", types.NowDateTime())
	if proj.GetString("status") == "scanning" {
		proj.Set("status", "active")
	}
	if err := s.App.Save(proj); err != nil {
		log.Warn().Err(err).Msg("failed to update project after scan")
	}
}

func (s *Scanner) detectDependencies(repoPath string, res *Result) {
	files := map[string]string{
		"go.mod":           "go",
		"package.json":     "npm",
		"Cargo.toml":       "cargo",
		"requirements.txt": "pip",
	}
	for name, eco := range files {
		p := filepath.Join(repoPath, name)
		if data, err := os.ReadFile(p); err == nil {
			res.Dependencies[eco] = strings.Count(string(data), "\n")
		}
	}
}

func (s *Scanner) isIgnoredDir(name string) bool {
	for _, d := range s.Config.IgnoredDirs {
		if name == d {
			return true
		}
	}
	return strings.HasPrefix(name, ".") && name != ".cortex"
}

func (s *Scanner) isIgnoredExt(path string) bool {
	lower := strings.ToLower(path)
	for _, e := range s.Config.IgnoredExtensions {
		if strings.HasSuffix(lower, e) {
			return true
		}
	}
	return false
}

func isEntryPoint(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	switch base {
	case "main.go", "index.ts", "index.js", "main.rs", "main.py", "__main__.py", "app.tsx", "index.tsx":
		return true
	}
	return false
}

func buildSummary(res *Result) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Indexed %d of %d files at %s.\n\n", res.IndexedFiles, res.TotalFiles, time.Now().Format("2006-01-02 15:04")))
	if len(res.Languages) > 0 {
		b.WriteString("Languages:\n")
		type kv struct {
			k string
			v int
		}
		var langs []kv
		for k, v := range res.Languages {
			langs = append(langs, kv{k, v})
		}
		sort.Slice(langs, func(i, j int) bool { return langs[i].v > langs[j].v })
		for _, l := range langs {
			b.WriteString(fmt.Sprintf("  - %s: %d files\n", l.k, l.v))
		}
	}
	if len(res.EntryPoints) > 0 {
		b.WriteString("\nEntry points:\n")
		for _, e := range res.EntryPoints {
			b.WriteString("  - " + e + "\n")
		}
	}
	return b.String()
}
