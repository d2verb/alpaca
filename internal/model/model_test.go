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

func TestRemoveModelWithoutMmproj(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	modelFile := filepath.Join(tmpDir, "model.gguf")
	if err := os.WriteFile(modelFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("create model file: %v", err)
	}

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
	if _, err := os.Stat(modelFile); !os.IsNotExist(err) {
		t.Error("model file should be deleted")
	}
}

func TestRemoveModelWithMmprojNoOtherReferences(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	modelFile := filepath.Join(tmpDir, "model.gguf")
	if err := os.WriteFile(modelFile, []byte("dummy model"), 0644); err != nil {
		t.Fatalf("create model file: %v", err)
	}
	mmprojFile := filepath.Join(tmpDir, "repo1_mmproj-f16.gguf")
	if err := os.WriteFile(mmprojFile, []byte("dummy mmproj"), 0644); err != nil {
		t.Fatalf("create mmproj file: %v", err)
	}

	metaMgr := metadata.NewManager(tmpDir)
	entry := metadata.ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "model.gguf",
		Mmproj: &metadata.MmprojEntry{
			Filename: "repo1_mmproj-f16.gguf",
			Size:     1000,
		},
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
	if _, err := os.Stat(modelFile); !os.IsNotExist(err) {
		t.Error("model file should be deleted")
	}
	if _, err := os.Stat(mmprojFile); !os.IsNotExist(err) {
		t.Error("mmproj file should be deleted when no other references exist")
	}
}

func TestRemoveModelWithMmprojSharedByOtherQuant(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	modelFile := filepath.Join(tmpDir, "model-q4.gguf")
	if err := os.WriteFile(modelFile, []byte("dummy model q4"), 0644); err != nil {
		t.Fatalf("create model file: %v", err)
	}
	otherModelFile := filepath.Join(tmpDir, "model-q8.gguf")
	if err := os.WriteFile(otherModelFile, []byte("dummy model q8"), 0644); err != nil {
		t.Fatalf("create other model file: %v", err)
	}
	mmprojFile := filepath.Join(tmpDir, "repo1_mmproj-f16.gguf")
	if err := os.WriteFile(mmprojFile, []byte("dummy mmproj"), 0644); err != nil {
		t.Fatalf("create mmproj file: %v", err)
	}

	metaMgr := metadata.NewManager(tmpDir)
	entry1 := metadata.ModelEntry{
		Repo:     "repo1",
		Quant:    "Q4_K_M",
		Filename: "model-q4.gguf",
		Mmproj: &metadata.MmprojEntry{
			Filename: "repo1_mmproj-f16.gguf",
			Size:     1000,
		},
	}
	entry2 := metadata.ModelEntry{
		Repo:     "repo1",
		Quant:    "Q8_0",
		Filename: "model-q8.gguf",
		Mmproj: &metadata.MmprojEntry{
			Filename: "repo1_mmproj-f16.gguf",
			Size:     1000,
		},
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

	// Act - remove Q4_K_M which shares mmproj with Q8_0
	err := mgr.Remove(ctx, "repo1", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(modelFile); !os.IsNotExist(err) {
		t.Error("model file should be deleted")
	}
	if _, err := os.Stat(mmprojFile); err != nil {
		t.Error("mmproj file should be kept when other entries still reference it")
	}
	if _, err := os.Stat(otherModelFile); err != nil {
		t.Error("other model file should not be affected")
	}
}

func TestRemoveNonExistentModel(t *testing.T) {
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

func TestGetDetailsWithMmproj(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	ctx := context.Background()

	metaMgr := metadata.NewManager(tmpDir)
	entry := metadata.ModelEntry{
		Repo:     "ggml-org/gemma-3-4b-it-GGUF",
		Quant:    "Q4_K_M",
		Filename: "gemma-3-4b-it-Q4_K_M.gguf",
		Size:     2489757856,
		Mmproj: &metadata.MmprojEntry{
			Filename: "ggml-org_gemma-3-4b-it-GGUF_mmproj-model-f16.gguf",
			Size:     851251104,
		},
		DownloadedAt: time.Now().UTC(),
	}
	if err := metaMgr.Add(entry); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := metaMgr.Save(ctx); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	// Act
	details, err := mgr.GetDetails(ctx, "ggml-org/gemma-3-4b-it-GGUF", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetDetails() error = %v", err)
	}
	if details.Mmproj == nil {
		t.Fatal("expected mmproj to be non-nil")
	}
	if details.Mmproj.Filename != "ggml-org_gemma-3-4b-it-GGUF_mmproj-model-f16.gguf" {
		t.Errorf("mmproj filename = %s, want ggml-org_gemma-3-4b-it-GGUF_mmproj-model-f16.gguf", details.Mmproj.Filename)
	}
	if details.Mmproj.Size != 851251104 {
		t.Errorf("mmproj size = %d, want 851251104", details.Mmproj.Size)
	}
}
