package git

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type FileChange struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Index     string `json:"index_status"`
	Worktree  string `json:"worktree_status"`
	Untracked bool   `json:"untracked"`
}

type WorkingTreeChanges struct {
	Path             string       `json:"repo_path"`
	Branch           string       `json:"branch"`
	Head             string       `json:"head"`
	Upstream         string       `json:"upstream"`
	Ahead            int          `json:"ahead"`
	Behind           int          `json:"behind"`
	Files            []FileChange `json:"files"`
	LocalCommits     []string     `json:"local_commits"`
	Diff             string       `json:"diff,omitempty"`
	UpstreamDiff     string       `json:"upstream_diff,omitempty"`
	DiffTruncated    bool         `json:"diff_truncated"`
	FilesTruncated   bool         `json:"files_truncated"`
	CommitsTruncated bool         `json:"commits_truncated"`
	ObservedAt       string       `json:"observed_at"`
	Note             string       `json:"note"`
}

// GitOutput runs only the fixed, read-only commands used by the checkout review.
// Never execute through a shell; bound output and disable external diff/fsmonitor programs.
func GitOutput(ctx context.Context, path string, limit int, args ...string) (string, bool, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-pager", "-c", "core.fsmonitor=false", "-c", "color.ui=false", "-C", path}, args...)...)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(value), "GIT_") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	output, stderr := &boundedOutput{limit: limit}, &boundedOutput{limit: 2048}
	cmd.Stdout, cmd.Stderr = output, stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", false, ctx.Err()
		}
		if _, ok := err.(*exec.Error); ok {
			return "", false, fmt.Errorf("install Git for Windows and restart CortexMind to inspect local checkouts: %w", err)
		}
		return "", false, fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(stderr.String()))
	}
	return output.String(), output.truncated, nil
}

type boundedOutput struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (w *boundedOutput) String() string { return w.buffer.String() }
func (w *boundedOutput) Len() int       { return w.buffer.Len() }

func (w *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := w.limit - w.Len()
	if len(p) > remaining {
		p = p[:remaining]
		w.truncated = true
	}
	_, err := w.buffer.Write(p)
	return n, err
}

func CheckoutRoot(ctx context.Context, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("repo_path must be an absolute path on the machine running CortexMind")
	}
	canonical, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("checkout directory does not exist: %w", err)
	}
	root, _, err := GitOutput(ctx, canonical, 8192, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("select the root of a cloned Git repository: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(root)))
	if err != nil {
		return "", err
	}
	if !SamePath(canonical, resolved) {
		return "", fmt.Errorf("repo_path must be the checkout root: %s", resolved)
	}
	return resolved, nil
}

func SamePath(a, b string) bool {
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	return aErr == nil && bErr == nil && os.SameFile(aInfo, bInfo)
}

// RepositoryIdentity compares HTTPS and SSH clone URLs without retaining credentials.
func RepositoryIdentity(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		if at := strings.Index(raw, "@"); at >= 0 {
			raw = raw[at+1:]
		}
		if colon := strings.Index(raw, ":"); colon >= 0 {
			raw = "ssh://" + raw[:colon] + "/" + raw[colon+1:]
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	if path == "" {
		return ""
	}
	return strings.ToLower(u.Hostname() + "/" + path)
}

func ReviewWorkingTree(ctx context.Context, path string, includeDiff bool) (*WorkingTreeChanges, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	result := &WorkingTreeChanges{Path: path, Files: []FileChange{}, LocalCommits: []string{}, ObservedAt: time.Now().UTC().Format(time.RFC3339), Note: "Local checkout only; upstream is the last fetched ref. Untracked files are listed but their contents are not included in the diff. No fetch, pull, commit, or push is performed."}
	status, truncated, err := GitOutput(ctx, path, 256<<10, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	result.FilesTruncated = truncated
	parts := strings.Split(status, "\x00")
	for i := 0; i < len(parts); i++ {
		row := parts[i]
		if len(row) < 4 {
			continue
		}
		if truncated && i == len(parts)-1 {
			break
		}
		change := FileChange{Path: row[3:], Index: string(row[0]), Worktree: string(row[1]), Untracked: row[:2] == "??"}
		if strings.ContainsAny(row[:2], "RC") && i+1 < len(parts) {
			i++
			change.OldPath = parts[i]
		}
		result.Files = append(result.Files, change)
	}
	if head, _, err := GitOutput(ctx, path, 128, "rev-parse", "--verify", "HEAD"); err == nil {
		result.Head = strings.TrimSpace(head)
	}
	if branch, _, err := GitOutput(ctx, path, 8192, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		result.Branch = strings.TrimSpace(branch)
	} else {
		result.Branch = "(detached HEAD)"
	}
	if upstream, _, err := GitOutput(ctx, path, 8192, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err == nil {
		result.Upstream = strings.TrimSpace(upstream)
		counts, _, err := GitOutput(ctx, path, 128, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
		if err != nil {
			return nil, err
		}
		fields := strings.Fields(counts)
		if len(fields) == 2 {
			result.Ahead, _ = strconv.Atoi(fields[0])
			result.Behind, _ = strconv.Atoi(fields[1])
		}
		commits, truncated, err := GitOutput(ctx, path, 32<<10, "log", "--format=%h %s", "-n", "20", "@{upstream}..HEAD")
		if err != nil {
			return nil, err
		}
		result.CommitsTruncated = truncated || result.Ahead > 20
		if strings.TrimSpace(commits) != "" {
			result.LocalCommits = strings.Split(strings.TrimSpace(commits), "\n")
		}
	}
	if includeDiff {
		args := []string{"diff", "--no-ext-diff", "--no-textconv"}
		if result.Head != "" {
			args = append(args, "HEAD")
		}
		result.Diff, result.DiffTruncated, err = GitOutput(ctx, path, 64<<10, append(args, "--")...)
		if err != nil {
			return nil, err
		}
		if result.Head == "" {
			staged, cut, err := GitOutput(ctx, path, 64<<10, "diff", "--cached", "--no-ext-diff", "--no-textconv", "--")
			if err != nil {
				return nil, err
			}
			combined := result.Diff + staged
			result.DiffTruncated = result.DiffTruncated || cut || len(combined) > 64<<10
			if len(combined) > 64<<10 {
				combined = combined[:64<<10]
			}
			result.Diff = combined
		}
		if result.Upstream != "" {
			var cut bool
			result.UpstreamDiff, cut, err = GitOutput(ctx, path, 64<<10, "diff", "--no-ext-diff", "--no-textconv", "@{upstream}", "HEAD", "--")
			if err != nil {
				return nil, err
			}
			result.DiffTruncated = result.DiffTruncated || cut
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
