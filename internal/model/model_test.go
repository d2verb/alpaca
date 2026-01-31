package model

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/d2verb/alpaca/internal/metadata"
)

func TestNewManager(t *testing.T) {
	// Arrange
	modelsDir := "/tmp/models"

	// Act
	mgr := NewManager(modelsDir)

	// Assert
	if mgr.modelsDir != modelsDir {
		t.Errorf("modelsDir = %s, want %s", mgr.modelsDir, modelsDir)
	}
	if mgr.metadata == nil {
		t.Error("metadata should not be nil")
	}
}

func TestListEmpty(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Act
	entries, err := mgr.List(ctx)

	// Assert
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListMultiple(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Add entries to metadata
	metaMgr := metadata.NewManager(tmpDir)
	entry1 := metadata.ModelEntry{
		Repo:         "repo1",
		Quant:        "Q4_K_M",
		Filename:     "model1.gguf",
		Size:         1000,
		DownloadedAt: time.Now().UTC(),
	}
	entry2 := metadata.ModelEntry{
		Repo:         "repo2",
		Quant:        "Q8_0",
		Filename:     "model2.gguf",
		Size:         2000,
		DownloadedAt: time.Now().UTC(),
	}
	if err := metaMgr.Add(entry1); err != nil {
		t.Fatalf("add entry1: %v", err)
	}
	if err := metaMgr.Add(entry2); err != nil {
		t.Fatalf("add entry2: %v", err)
	}
	if err := metaMgr.Save(ctx); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	// Act
	entries, err := mgr.List(ctx)

	// Assert
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestExistsTrue(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Add entry to metadata
	metaMgr := metadata.NewManager(tmpDir)
	entry := metadata.ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "model.gguf",
	}
	if err := metaMgr.Add(entry); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := metaMgr.Save(ctx); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	// Act
	exists, err := mgr.Exists(ctx, "repo1", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("expected model to exist")
	}
}

func TestExistsFalse(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Act
	exists, err := mgr.Exists(ctx, "nonexistent", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("expected model to not exist")
	}
}

func TestRemoveSuccess(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Create model file
	modelFile := filepath.Join(tmpDir, "model.gguf")
	if err := os.WriteFile(modelFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("create model file: %v", err)
	}

	// Add entry to metadata
	metaMgr := metadata.NewManager(tmpDir)
	entry := metadata.ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "model.gguf",
	}
	if err := metaMgr.Add(entry); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := metaMgr.Save(ctx); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	// Act
	err := mgr.Remove(ctx, "repo1", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify file deleted
	if _, err := os.Stat(modelFile); !os.IsNotExist(err) {
		t.Error("model file should be deleted")
	}

	// Verify metadata removed
	entries, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after remove, got %d", len(entries))
	}
}

func TestRemoveNonExistent(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Act
	err := mgr.Remove(ctx, "nonexistent", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent model")
	}
}

func TestRemoveFileAlreadyDeleted(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Add entry to metadata (but no actual file)
	metaMgr := metadata.NewManager(tmpDir)
	entry := metadata.ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "nonexistent.gguf",
	}
	if err := metaMgr.Add(entry); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := metaMgr.Save(ctx); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	// Act - should fail because metadata.GetFilePath checks file existence
	err := mgr.Remove(ctx, "repo1", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error when file doesn't exist")
	}
}

func TestGetFilePathSuccess(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Create model file
	modelFile := filepath.Join(tmpDir, "model.gguf")
	if err := os.WriteFile(modelFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("create model file: %v", err)
	}

	// Add entry to metadata
	metaMgr := metadata.NewManager(tmpDir)
	entry := metadata.ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "model.gguf",
	}
	if err := metaMgr.Add(entry); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := metaMgr.Save(ctx); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	// Act
	path, err := mgr.GetFilePath(ctx, "repo1", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetFilePath() error = %v", err)
	}
	if path != modelFile {
		t.Errorf("path = %s, want %s", path, modelFile)
	}
}

func TestGetFilePathNotFound(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Act
	_, err := mgr.GetFilePath(ctx, "nonexistent", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent model")
	}
}

func TestGetDetailsSuccess(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	metaMgr := metadata.NewManager(tmpDir)
	entry := metadata.ModelEntry{
		Repo:         "TheBloke/CodeLlama-7B-GGUF",
		Quant:        "Q4_K_M",
		Filename:     "codellama-7b.Q4_K_M.gguf",
		Size:         4000000000,
		DownloadedAt: time.Now().UTC(),
	}
	if err := metaMgr.Add(entry); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := metaMgr.Save(ctx); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	// Act
	result, err := mgr.GetDetails(ctx, "TheBloke/CodeLlama-7B-GGUF", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetDetails() error = %v", err)
	}
	if result.Repo != entry.Repo {
		t.Errorf("Repo = %s, want %s", result.Repo, entry.Repo)
	}
	if result.Quant != entry.Quant {
		t.Errorf("Quant = %s, want %s", result.Quant, entry.Quant)
	}
	if result.Filename != entry.Filename {
		t.Errorf("Filename = %s, want %s", result.Filename, entry.Filename)
	}
	if result.Size != entry.Size {
		t.Errorf("Size = %d, want %d", result.Size, entry.Size)
	}
}

func TestGetDetailsNotFound(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	// Act
	_, err := mgr.GetDetails(ctx, "nonexistent/repo", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent model")
	}
}
