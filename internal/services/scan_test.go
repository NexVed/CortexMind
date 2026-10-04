package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/database"
)

func TestLocalScanReplacesIndexAndHonorsCancellation(t *testing.T) {
	dir, source := t.TempDir(), t.TempDir()
	db, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	project, err := db.CreateProject("local", source, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(source, "one.go"), []byte("package local\nfunc One() {}"), 0600); err != nil {
		t.Fatal(err)
	}
	s := ScanService{DB: db, Config: &config.Config{Server: config.ServerConfig{DataDir: dir}, Scanner: config.ScannerConfig{MaxFileSizeKB: 500}}}
	for i := 0; i < 2; i++ {
		result, err := s.Scan(context.Background(), project.ID)
		if err != nil || result.IndexedFiles != 1 {
			t.Fatalf("offline scan failed: %v (%v)", result, err)
		}
	}
	store := database.RecordStore{DB: db}
	page, err := store.List("file_index", project.ID, 100)
	if err != nil || len(page.Items) != 1 {
		t.Fatal("rescan accumulated duplicate records")
	}
	if err = os.Remove(filepath.Join(source, "one.go")); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(source, "two.go"), []byte("package local"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Scan(context.Background(), project.ID); err != nil {
		t.Fatal(err)
	}
	page, err = store.List("file_index", project.ID, 100)
	if err != nil || len(page.Items) != 1 || page.Items[0]["path"] != "two.go" {
		t.Fatal("deleted source remained indexed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = s.Scan(ctx, project.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled scan did not stop: %v", err)
	}
}
