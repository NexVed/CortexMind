package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	cgit "github.com/NexVed/Cortex/internal/git"
	"github.com/pocketbase/pocketbase/core"
)

// categoryDir maps a vault category to its .cortex/ subdirectory.
var categoryDir = map[string]string{
	"architecture": "architecture",
	"decision":     "decisions",
	"roadmap":      "roadmaps",
	"task":         "tasks",
	"handoff":      "handoffs",
	"memory":       "memories",
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a title into a filesystem-safe slug.
func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

// ExportEntryToFile writes a shared vault entry to the .cortex/ directory and
// returns the relative file path that was written.
func ExportEntryToFile(entry *core.Record, repoPath string) (string, error) {
	category := entry.GetString("category")
	sub, ok := categoryDir[category]
	if !ok {
		sub = "memories"
	}
	dir := filepath.Join(cgit.CortexDir(repoPath), sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	slug := Slugify(entry.GetString("title"))
	filename := slug + ".md"
	fullPath := filepath.Join(dir, filename)

	content := fmt.Sprintf("# %s\n\n%s\n", entry.GetString("title"), entry.GetString("content"))
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return "", err
	}

	rel := filepath.ToSlash(filepath.Join(".cortex", sub, filename))
	return rel, nil
}
