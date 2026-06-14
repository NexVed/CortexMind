package git

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/rs/zerolog/log"
)

// SyncEngine manages the .cortex/ directory inside a project repository.
type SyncEngine struct {
	AuthorName  string
	AuthorEmail string
}

func NewSyncEngine() *SyncEngine {
	return &SyncEngine{AuthorName: "CORTEX", AuthorEmail: "cortex@local"}
}

// CortexDir returns the path to the .cortex directory for a repo.
func CortexDir(repoPath string) string {
	return filepath.Join(repoPath, ".cortex")
}

// authMethod builds a transport auth method from a GitHub token (empty token
// means anonymous, which only works for public repos).
func authMethod(token string) *githttp.BasicAuth {
	if token == "" {
		return nil
	}
	// GitHub accepts the token as the password with any non-empty username.
	return &githttp.BasicAuth{Username: "cortex", Password: token}
}

// Clone performs a shallow clone of cloneURL into repoPath using the optional
// GitHub token for private repositories. It is a no-op (returns nil) when the
// destination already contains a git repository.
func Clone(repoPath, cloneURL, token string) error {
	if _, err := git.PlainOpen(repoPath); err == nil {
		return nil // already cloned
	}
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		return err
	}
	_, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:          cloneURL,
		Auth:         authMethod(token),
		Depth:        1,
		SingleBranch: true,
	})
	if err != nil {
		return fmt.Errorf("clone %s: %w", cloneURL, err)
	}
	log.Info().Str("repo", repoPath).Msg("cloned repository")
	return nil
}

// EnsureRepo clones the repository if it is missing, otherwise fast-forwards it.
// It returns whether a fresh clone was performed.
func EnsureRepo(repoPath, cloneURL, token string) (cloned bool, err error) {
	if _, openErr := git.PlainOpen(repoPath); openErr != nil {
		if err := Clone(repoPath, cloneURL, token); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := pullWithAuth(repoPath, token); err != nil {
		// A failed pull (e.g. detached/shallow) is non-fatal for scanning.
		log.Debug().Err(err).Str("repo", repoPath).Msg("pull failed; scanning existing checkout")
	}
	return false, nil
}

func pullWithAuth(repoPath, token string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = wt.Pull(&git.PullOptions{RemoteName: "origin", Auth: authMethod(token), Depth: 1})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return err
}

// InitCortexDir ensures the .cortex directory and its category subfolders exist.
func (s *SyncEngine) InitCortexDir(repoPath string) error {
	dirs := []string{"architecture", "decisions", "tasks", "handoffs", "roadmaps", "memories"}
	base := CortexDir(repoPath)
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// CommitChanges stages the .cortex directory and commits it. It is a no-op when
// there is nothing to commit or the repo is not a git repository.
func (s *SyncEngine) CommitChanges(repoPath, message string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		log.Debug().Err(err).Str("repo", repoPath).Msg("not a git repo, skipping commit")
		return nil
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if _, err := wt.Add(".cortex"); err != nil {
		return fmt.Errorf("git add .cortex: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		return nil
	}
	_, err = wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  s.AuthorName,
			Email: s.AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	log.Info().Str("repo", repoPath).Str("msg", message).Msg("committed .cortex changes")
	return nil
}

// PullRemote fetches and fast-forwards the current branch.
func (s *SyncEngine) PullRemote(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = wt.Pull(&git.PullOptions{RemoteName: "origin"})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return err
}
