package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	gh "github.com/NexVed/Cortex/internal/github"
)

var githubRepoName = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func (d *Daemon) githubContent(r *http.Request) (gh.Client, string, string, error) {
	project, err := d.DB.Project(r.PathValue("id"))
	if err != nil {
		return gh.Client{}, "", "", fmt.Errorf("project not found")
	}
	u, err := url.Parse(project.GitHubURL)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" {
		return gh.Client{}, "", "", fmt.Errorf("project has no valid GitHub repository")
	}
	name := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	if !githubRepoName.MatchString(name) || strings.Contains(name, "..") {
		return gh.Client{}, "", "", fmt.Errorf("invalid GitHub repository name")
	}
	token, _ := d.Auth.CurrentGitHubToken()
	return d.Auth.GitHub.GitHub, token, name, nil
}

func githubMissing(err error) bool {
	var apiErr *gh.APIError
	return errors.As(err, &apiErr) && (apiErr.Status == 404 || apiErr.Status == 409)
}

func (d *Daemon) repositoryView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	client, token, name, err := d.githubContent(r)
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	resource := func(suffix, accept string) ([]byte, error) {
		return client.RepositoryResource(ctx, token, name, suffix, accept)
	}
	metadata, err := resource("", "application/vnd.github+json")
	if err != nil {
		writeError(w, fmt.Errorf("load repository: %w; reconnect GitHub if this repository is private", err), http.StatusBadGateway)
		return
	}
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err = json.Unmarshal(metadata, &repo); err != nil {
		writeError(w, err, http.StatusBadGateway)
		return
	}
	ref := "?ref=" + url.QueryEscape(repo.DefaultBranch)
	files := []map[string]any{}
	raw, err := resource("/contents"+ref, "application/vnd.github+json")
	if err != nil && !githubMissing(err) {
		writeError(w, fmt.Errorf("load repository files: %w", err), http.StatusBadGateway)
		return
	}
	if err == nil {
		if err = json.Unmarshal(raw, &files); err != nil {
			writeError(w, err, http.StatusBadGateway)
			return
		}
	}
	var commit json.RawMessage = []byte("null")
	raw, err = resource("/commits?per_page=1", "application/vnd.github+json")
	if err != nil && !githubMissing(err) {
		writeError(w, fmt.Errorf("load repository commit: %w", err), http.StatusBadGateway)
		return
	}
	if err == nil {
		var commits []json.RawMessage
		if err = json.Unmarshal(raw, &commits); err != nil {
			writeError(w, err, http.StatusBadGateway)
			return
		}
		if len(commits) > 0 {
			commit = commits[0]
		}
	}
	readmePath, html := "", ""
	raw, err = resource("/readme"+ref, "application/vnd.github+json")
	if err != nil && !githubMissing(err) {
		writeError(w, fmt.Errorf("load README: %w", err), http.StatusBadGateway)
		return
	}
	if err == nil {
		var readme struct {
			Path string `json:"path"`
		}
		if err = json.Unmarshal(raw, &readme); err != nil {
			writeError(w, err, http.StatusBadGateway)
			return
		}
		readmePath = readme.Path
		raw, err = resource("/contents/"+gh.EscapePath(readmePath)+ref, "application/vnd.github.html+json")
		if err != nil {
			writeError(w, fmt.Errorf("render README: %w", err), http.StatusBadGateway)
			return
		}
		html = string(raw)
	}
	writeJSON(w, http.StatusOK, map[string]any{"repo": json.RawMessage(metadata), "commit": commit, "files": files, "readme_html": html, "readme_path": readmePath})
}

func (d *Daemon) repositoryImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	client, token, name, err := d.githubContent(r)
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	file := r.URL.Query().Get("path")
	ref := r.URL.Query().Get("ref")
	if file == "" || path.IsAbs(file) || path.Clean(file) != file || file == ".." || strings.HasPrefix(file, "../") || strings.Contains(file, "\\") || len(file) > 1024 || len(ref) > 200 {
		writeError(w, fmt.Errorf("invalid repository image path"), http.StatusBadRequest)
		return
	}
	suffix := "/contents/" + gh.EscapePath(file)
	if ref != "" {
		suffix += "?ref=" + url.QueryEscape(ref)
	}
	body, err := client.RepositoryResource(r.Context(), token, name, suffix, "application/vnd.github.raw+json")
	if err != nil {
		writeError(w, err, http.StatusBadGateway)
		return
	}
	typeName := http.DetectContentType(body)
	if strings.EqualFold(path.Ext(file), ".svg") && strings.Contains(string(body), "<svg") {
		typeName = "image/svg+xml"
	}
	if !strings.HasPrefix(typeName, "image/") {
		writeError(w, fmt.Errorf("requested repository file is not an image"), http.StatusUnsupportedMediaType)
		return
	}
	w.Header().Set("Content-Type", typeName)
	w.Header().Set("Content-Disposition", "inline")
	_, _ = w.Write(body)
}
