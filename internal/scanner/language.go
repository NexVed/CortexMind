package scanner

import (
	"path/filepath"
	"strings"
)

// extLang maps a file extension to a human-readable language name.
var extLang = map[string]string{
	".go":    "Go",
	".ts":    "TypeScript",
	".tsx":   "TypeScript",
	".js":    "JavaScript",
	".jsx":   "JavaScript",
	".mjs":   "JavaScript",
	".cjs":   "JavaScript",
	".rs":    "Rust",
	".py":    "Python",
	".c":     "C",
	".h":     "C",
	".cpp":   "C++",
	".cc":    "C++",
	".cxx":   "C++",
	".hpp":   "C++",
	".java":  "Java",
	".kt":    "Kotlin",
	".rb":    "Ruby",
	".php":   "PHP",
	".cs":    "C#",
	".swift": "Swift",
	".scala": "Scala",
	".sh":    "Shell",
	".sql":   "SQL",
	".html":  "HTML",
	".css":   "CSS",
	".scss":  "SCSS",
	".json":  "JSON",
	".yaml":  "YAML",
	".yml":   "YAML",
	".toml":  "TOML",
	".md":    "Markdown",
}

// DetectLanguage returns the language name for the given path, or "" if unknown.
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := extLang[ext]; ok {
		return lang
	}
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "dockerfile":
		return "Dockerfile"
	case "makefile":
		return "Makefile"
	}
	return ""
}
