package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/database"
)

func TestLocalCloneScanSeesUnpushedWorkWithoutChangingCheckout(t *testing.T) {
	root, data := t.TempDir(), t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + filepath.Join(root, "empty-hooks")}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-b", "main")
	git("remote", "add", "origin", "git@github.com:fixture/repo.git")
	write("main.go", "package fixture\nfunc Before() {}\n")
	write(".gitignore", "ignored.go\n")
	git("add", "main.go", ".gitignore")
	git("commit", "-m", "Baseline")
	git("update-ref", "refs/remotes/origin/main", "HEAD")
	git("config", "branch.main.remote", "origin")
	git("config", "branch.main.merge", "refs/heads/main")
	write("committed.go", "package fixture\nfunc LocalCommit() {}\n")
	git("add", "committed.go")
	git("commit", "-m", "Not pushed yet")
	write("main.go", "package fixture\nfunc Staged() {}\n")
	git("add", "main.go")
	write("main.go", "package fixture\nfunc Unstaged() {}\n")
	write("new file.go", "package fixture\nfunc NewFile() {}\n")
	write("ignored.go", "package fixture\nfunc IgnoredSecret() {}\n")
	statusBefore, indexBefore, headBefore := git("status", "--porcelain=v1"), git("ls-files", "--stage"), git("rev-parse", "HEAD")
	db, err := database.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.ReplaceGitHubData(nil, []database.Repository{{GitHubID: 42, Name: "repo", FullName: "fixture/repo", Private: true, HTMLURL: "https://github.com/fixture/repo", CloneURL: "https://github.com/fixture/repo.git"}}); err != nil {
		t.Fatal(err)
	}
	s := ScanService{DB: db, Config: &config.Config{Server: config.ServerConfig{DataDir: data}, Scanner: config.ScannerConfig{MaxFileSizeKB: 512}}}
	if id, err := s.ResolveWorkingTree(context.Background(), root); err != nil || id != "42" {
		t.Fatalf("automatic origin match: %q %v", id, err)
	}
	result, err := s.ScanWorkingTree(context.Background(), "42", root, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scan.IndexedFiles != 3 {
		t.Fatalf("ignored or new file indexing incorrect: %+v", result.Scan)
	}
	changes := result.Changes
	if changes.Ahead != 1 || changes.Behind != 0 || changes.Upstream != "origin/main" || len(changes.LocalCommits) != 1 {
		t.Fatalf("unpushed commits missing: %+v", changes)
	}
	if !strings.Contains(changes.Diff, "Unstaged") || !strings.Contains(changes.UpstreamDiff, "LocalCommit") {
		t.Fatalf("checkout/commit patches missing: %+v", changes)
	}
	if len(changes.Files) != 2 {
		t.Fatalf("expected modified and untracked files: %+v", changes.Files)
	}
	var modified, untracked bool
	for _, file := range changes.Files {
		if file.Path == "main.go" && file.Index == "M" && file.Worktree == "M" {
			modified = true
		}
		if file.Path == "new file.go" && file.Untracked {
			untracked = true
		}
	}
	if !modified || !untracked {
		t.Fatalf("staged/unstaged/untracked status incorrect: %+v", changes.Files)
	}
	graph, _, err := (CodeGraphService{DB: db}).Load("42")
	if err != nil {
		t.Fatal(err)
	}
	var current, newFile bool
	for _, node := range graph.Nodes {
		current = current || node.Label == "Unstaged"
		newFile = newFile || node.Label == "NewFile"
		if node.Label == "Before" || node.Label == "IgnoredSecret" {
			t.Fatal("graph contains stale or ignored source")
		}
	}
	if !current || !newFile {
		t.Fatal("graph did not use the live working tree")
	}
	project, err := db.Project("42")
	if err != nil || project.Path == "" {
		t.Fatalf("checkout binding missing: %v %v", project, err)
	}
	write("main.go", "package fixture\nfunc LaterEdit() {}\n")
	if _, err = s.Scan(context.Background(), "42"); err != nil {
		t.Fatalf("registered private clone needed a token or network: %v", err)
	}
	write("main.go", "package fixture\nfunc Unstaged() {}\n")
	write("main.go", "package fixture\n"+strings.Repeat("// Large pending change for bounded diff verification\n", 10000))
	bounded, err := s.WorkingTreeChanges(context.Background(), "42", "", true)
	if err != nil || !bounded.DiffTruncated || len(bounded.Diff) > 64<<10 {
		t.Fatalf("diff output was not bounded: %v", err)
	}
	write("main.go", "package fixture\nfunc Unstaged() {}\n")
	if statusBefore != git("status", "--porcelain=v1") || indexBefore != git("ls-files", "--stage") || headBefore != git("rev-parse", "HEAD") {
		t.Fatal("scan mutated the worktree, index, or HEAD")
	}
	if err = db.ReplaceGitHubData(nil, []database.Repository{{GitHubID: 42, Name: "repo", HTMLURL: "https://github.com/fixture/other"}}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ScanWorkingTree(context.Background(), "42", root, true); err == nil {
		t.Fatal("mismatched origin was accepted")
	}
	if _, err = s.Scan(context.Background(), "42"); err == nil {
		t.Fatal("saved checkout origin was not revalidated")
	}
}

func TestWorkingTreeWithoutCommitsAndLocalPathRestrictions(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("init: %v %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, "first.go"), []byte("package first\nfunc First() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "add", "first.go").CombinedOutput(); err != nil {
		t.Fatalf("add: %v %s", err, out)
	}
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	project, err := db.CreateProject("first", root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s := ScanService{DB: db, Config: &config.Config{Scanner: config.ScannerConfig{MaxFileSizeKB: 512}}}
	result, err := s.ScanWorkingTree(context.Background(), project.ID, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changes.Head != "" || result.Changes.Upstream != "" || !strings.Contains(result.Changes.Diff, "First") {
		t.Fatalf("unborn branch unsupported: %+v", result.Changes)
	}
	other := t.TempDir()
	exec.Command("git", "-C", other, "init", "-b", "main").Run()
	if _, err = s.ValidateWorkingTree(context.Background(), project.ID, other); err == nil {
		t.Fatal("local project accepted another folder")
	}
	if _, err = s.ValidateWorkingTree(context.Background(), project.ID, "."); err == nil {
		t.Fatal("relative folder accepted")
	}
}
