package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportBundleREADMEIsMetadataOnly(t *testing.T) {
	repo := t.TempDir()
	memoryText := strings.Repeat("durable memory content ", 20)
	_, err := ExportBundle(Bundle{
		Project: ProjectMeta{Name: "Example"},
		VaultEntries: []BundleEntry{{
			Category: "decision",
			Title:    "Use compact exports",
			Content:  memoryText,
		}},
		SessionDigests: []SessionDigest{{
			Title:       "Session one",
			SummaryMD:   memoryText,
			MemoryCount: 1,
			TokenCount:  12,
		}},
	}, repo)
	if err != nil {
		t.Fatal(err)
	}

	jsonData, err := os.ReadFile(filepath.Join(repo, ".cortex", "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	readmeData, err := os.ReadFile(filepath.Join(repo, ".cortex", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonData), memoryText) {
		t.Fatal("memory.json should contain the canonical content")
	}
	if strings.Contains(string(readmeData), memoryText) {
		t.Fatal("README.md should not duplicate memory content")
	}
	if !strings.Contains(string(readmeData), "Use compact exports") {
		t.Fatal("README.md should retain the memory index")
	}
}

func TestExportBundleFullModeIndexesRawMemories(t *testing.T) {
	repo := t.TempDir()
	memoryText := "full raw session memory"
	_, err := ExportBundle(Bundle{
		ExportMode: "full",
		Project:    ProjectMeta{Name: "Example"},
		AgentMemories: []AgentMemory{{
			Category: "handoff",
			Title:    "Raw handoff",
			Content:  memoryText,
		}},
	}, repo)
	if err != nil {
		t.Fatal(err)
	}

	jsonData, err := os.ReadFile(filepath.Join(repo, ".cortex", "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	readmeData, err := os.ReadFile(filepath.Join(repo, ".cortex", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonData), memoryText) {
		t.Fatal("full memory.json should contain raw memory content")
	}
	if strings.Contains(string(readmeData), memoryText) {
		t.Fatal("README.md should remain an index in full mode")
	}
}
