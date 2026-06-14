package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parseManifests reads recognised dependency files at the repo root and feeds
// their dependencies into the signal collector.
func parseManifests(repoPath string, s *repoSignals) {
	if data, err := os.ReadFile(filepath.Join(repoPath, "package.json")); err == nil {
		s.packageManagers.add("npm")
		parsePackageJSON(data, s)
	}
	if data, err := os.ReadFile(filepath.Join(repoPath, "go.mod")); err == nil {
		s.packageManagers.add("go modules")
		parseGoMod(data, s)
	}
	if data, err := os.ReadFile(filepath.Join(repoPath, "requirements.txt")); err == nil {
		s.packageManagers.add("pip")
		parseRequirements(data, s)
	}
	if data, err := os.ReadFile(filepath.Join(repoPath, "pyproject.toml")); err == nil {
		s.packageManagers.add("pip")
		parseLineDeps(string(data), s)
	}
	if data, err := os.ReadFile(filepath.Join(repoPath, "Cargo.toml")); err == nil {
		s.packageManagers.add("cargo")
		parseCargo(data, s)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "pom.xml")); err == nil {
		s.packageManagers.add("maven")
	}
	if _, err := os.Stat(filepath.Join(repoPath, "Gemfile")); err == nil {
		s.packageManagers.add("bundler")
	}
	if _, err := os.Stat(filepath.Join(repoPath, "composer.json")); err == nil {
		s.packageManagers.add("composer")
	}
	// Infra signals.
	for _, f := range []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml"} {
		if _, err := os.Stat(filepath.Join(repoPath, f)); err == nil {
			s.tools.add("Docker")
		}
	}
	if _, err := os.Stat(filepath.Join(repoPath, ".github", "workflows")); err == nil {
		s.tools.add("GitHub Actions")
	}
}

func parsePackageJSON(data []byte, s *repoSignals) {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}
	for name := range pkg.Dependencies {
		s.recordDep(name)
	}
	for name := range pkg.DevDependencies {
		s.recordDep(name)
	}
}

var reGoRequire = regexp.MustCompile(`^\s*([\w./-]+/[\w./-]+)\s+v`)

func parseGoMod(data []byte, s *repoSignals) {
	for _, line := range strings.Split(string(data), "\n") {
		if m := reGoRequire.FindStringSubmatch(line); m != nil {
			s.recordDep(m[1])
		}
	}
}

var reReq = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)`)

func parseRequirements(data []byte, s *repoSignals) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := reReq.FindStringSubmatch(line); m != nil {
			s.recordDep(m[1])
		}
	}
}

func parseLineDeps(content string, s *repoSignals) {
	for _, line := range strings.Split(content, "\n") {
		if m := reReq.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			s.recordDep(m[1])
		}
	}
}

var reCargoDep = regexp.MustCompile(`^\s*([A-Za-z0-9_-]+)\s*=`)

func parseCargo(data []byte, s *repoSignals) {
	inDeps := false
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			inDeps = strings.Contains(t, "dependencies")
			continue
		}
		if inDeps {
			if m := reCargoDep.FindStringSubmatch(line); m != nil {
				s.recordDep(m[1])
			}
		}
	}
}
