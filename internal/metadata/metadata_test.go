package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	// Arrange
	modelsDir := "/tmp/models"

	// Act
	mgr := NewManager(modelsDir)

	// Assert
	expected := filepath.Join(modelsDir, ".metadata.json")
	if mgr.filePath != expected {
		t.Errorf("expected filePath %s, got %s", expected, mgr.filePath)
	}
	if len(mgr.data.Models) != 0 {
		t.Errorf("expected empty models, got %d", len(mgr.data.Models))
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Act
	err := mgr.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got %v", err)
	}
	if len(mgr.data.Models) != 0 {
		t.Errorf("expected empty models, got %d", len(mgr.data.Models))
	}
}

func TestLoadEmptyFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	metaPath := filepath.Join(tmpDir, ".metadata.json")
	if err := os.WriteFile(metaPath, []byte(""), 0644); err != nil {
		t.Fatalf("create empty file: %v", err)
	}

	// Act
	err := mgr.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error for empty file, got %v", err)
	}
	if len(mgr.data.Models) != 0 {
		t.Errorf("expected empty models, got %d", len(mgr.data.Models))
	}
}

func TestLoadCorruptFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	metaPath := filepath.Join(tmpDir, ".metadata.json")
	if err := os.WriteFile(metaPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("create corrupt file: %v", err)
	}

	// Act
	err := mgr.Load()

	// Assert
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	entry := ModelEntry{
		Repo:         "TheBloke/CodeLlama-7B-GGUF",
		Quant:        "Q4_K_M",
		Filename:     "codellama-7b.Q4_K_M.gguf",
		Size:         4100000000,
		DownloadedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := mgr.Add(entry); err != nil {
		t.Fatalf("add entry: %v", err)
	}

	// Act - Save
	if err := mgr.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Act - Load in new manager
	mgr2 := NewManager(tmpDir)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	// Assert
	if len(mgr2.data.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(mgr2.data.Models))
	}
	loaded := mgr2.data.Models[0]
	if loaded.Repo != entry.Repo {
		t.Errorf("expected repo %s, got %s", entry.Repo, loaded.Repo)
	}
	if loaded.Quant != entry.Quant {
		t.Errorf("expected quant %s, got %s", entry.Quant, loaded.Quant)
	}
	if loaded.Filename != entry.Filename {
		t.Errorf("expected filename %s, got %s", entry.Filename, loaded.Filename)
	}
	if loaded.Size != entry.Size {
		t.Errorf("expected size %d, got %d", entry.Size, loaded.Size)
	}
	if !loaded.DownloadedAt.Equal(entry.DownloadedAt) {
		t.Errorf("expected time %v, got %v", entry.DownloadedAt, loaded.DownloadedAt)
	}
}

func TestAddNewEntry(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())
	entry := ModelEntry{
		Repo:  "repo1",
		Quant: "Q4_K_M",
	}

	// Act
	if err := mgr.Add(entry); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Assert
	if len(mgr.data.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(mgr.data.Models))
	}
	if mgr.data.Models[0].Repo != "repo1" {
		t.Errorf("expected repo1, got %s", mgr.data.Models[0].Repo)
	}
}

func TestAddReplacesExisting(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())
	entry1 := ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "old.gguf",
		Size:     1000,
	}
	entry2 := ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "new.gguf",
		Size:     2000,
	}
	if err := mgr.Add(entry1); err != nil {
		t.Fatalf("add entry1: %v", err)
	}

	// Act
	if err := mgr.Add(entry2); err != nil {
		t.Fatalf("add entry2: %v", err)
	}

	// Assert
	if len(mgr.data.Models) != 1 {
		t.Fatalf("expected 1 model after replacement, got %d", len(mgr.data.Models))
	}
	if mgr.data.Models[0].Filename != "new.gguf" {
		t.Errorf("expected new.gguf, got %s", mgr.data.Models[0].Filename)
	}
	if mgr.data.Models[0].Size != 2000 {
		t.Errorf("expected size 2000, got %d", mgr.data.Models[0].Size)
	}
}

func TestRemoveExisting(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())
	entry := ModelEntry{Repo: "repo1", Quant: "Q4_K_M"}
	if err := mgr.Add(entry); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Act
	if err := mgr.Remove("repo1", "Q4_K_M"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// Assert
	if len(mgr.data.Models) != 0 {
		t.Errorf("expected 0 models after remove, got %d", len(mgr.data.Models))
	}
}

func TestRemoveNonExistent(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())

	// Act
	err := mgr.Remove("nonexistent", "Q4_K_M")

	// Assert
	if err != nil {
		t.Errorf("expected no error for non-existent remove, got %v", err)
	}
}

func TestFindExisting(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())
	entry := ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "test.gguf",
	}
	if err := mgr.Add(entry); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Act
	found := mgr.Find("repo1", "Q4_K_M")

	// Assert
	if found == nil {
		t.Fatal("expected to find entry")
	}
	if found.Filename != "test.gguf" {
		t.Errorf("expected test.gguf, got %s", found.Filename)
	}
}

func TestFindNonExistent(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())

	// Act
	found := mgr.Find("nonexistent", "Q4_K_M")

	// Assert
	if found != nil {
		t.Error("expected nil for non-existent entry")
	}
}

func TestListEmpty(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())

	// Act
	entries := mgr.List()

	// Assert
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListMultiple(t *testing.T) {
	// Arrange
	mgr := NewManager(t.TempDir())
	entry1 := ModelEntry{Repo: "repo1", Quant: "Q4_K_M"}
	entry2 := ModelEntry{Repo: "repo2", Quant: "Q8_0"}
	if err := mgr.Add(entry1); err != nil {
		t.Fatalf("add entry1: %v", err)
	}
	if err := mgr.Add(entry2); err != nil {
		t.Fatalf("add entry2: %v", err)
	}

	// Act
	entries := mgr.List()

	// Assert
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify copy (mutation shouldn't affect internal data)
	entries[0].Repo = "modified"
	if mgr.data.Models[0].Repo == "modified" {
		t.Error("List() should return a copy, not internal slice")
	}
}

func TestGetFilePathFound(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	modelFile := filepath.Join(tmpDir, "model.gguf")
	if err := os.WriteFile(modelFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("create model file: %v", err)
	}
	entry := ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "model.gguf",
	}
	if err := mgr.Add(entry); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Act
	path, err := mgr.GetFilePath(tmpDir, "repo1", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != modelFile {
		t.Errorf("expected path %s, got %s", modelFile, path)
	}
}

func TestGetFilePathNotInMetadata(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Act
	_, err := mgr.GetFilePath(tmpDir, "nonexistent", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for model not in metadata")
	}
}

func TestGetFilePathFileNotFound(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	entry := ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "nonexistent.gguf",
	}
	if err := mgr.Add(entry); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Act
	_, err := mgr.GetFilePath(tmpDir, "repo1", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
