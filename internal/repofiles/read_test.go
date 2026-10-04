package repofiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileCannotEscapeRepository(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(root, "../secret.txt"); err == nil {
		t.Fatal("repository read escaped its root")
	}
	if _, err := ReadFile(root, filepath.Join(parent, "secret.txt")); err == nil {
		t.Fatal("absolute file path accepted")
	}
}
