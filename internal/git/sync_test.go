package git

import (
	"context"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

func TestFailedPullDoesNotReportSuccessfulScanPreparation(t *testing.T) {
	dir := t.TempDir()
	if _, err := gogit.PlainInit(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureRepo(context.Background(), dir, "https://github.com/unused/repository.git", ""); err == nil {
		t.Fatal("failed update accepted as current repository")
	}
}
