package pull

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestPull_WithMmproj_Success(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	mmprojContent := []byte("fake-mmproj-binary-content")

	srv, _ := newMmprojTestServer(t, modelContent, mmprojContent, 0)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)
	repo := "ggml-org/gemma-3-4b-it-GGUF"

	// Act
	result, err := puller.Pull(context.Background(), repo, "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}
	if result.Filename != "model-Q4_K_M.gguf" {
		t.Errorf("Filename = %q, want %q", result.Filename, "model-Q4_K_M.gguf")
	}

	expectedMmprojFilename := "ggml-org_gemma-3-4b-it-GGUF_mmproj-model-f16.gguf"
	if result.MmprojFilename != expectedMmprojFilename {
		t.Errorf("MmprojFilename = %q, want %q", result.MmprojFilename, expectedMmprojFilename)
	}
	if result.MmprojSize != int64(len(mmprojContent)) {
		t.Errorf("MmprojSize = %d, want %d", result.MmprojSize, len(mmprojContent))
	}
	if result.MmprojFailed {
		t.Error("MmprojFailed = true, want false")
	}

	// Verify model file exists
	modelPath := filepath.Join(tmpDir, "model-Q4_K_M.gguf")
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("failed to read model file: %v", err)
	}
	if string(content) != string(modelContent) {
		t.Error("model content mismatch")
	}

	// Verify mmproj file exists with prefixed name
	mmprojPath := filepath.Join(tmpDir, expectedMmprojFilename)
	content, err = os.ReadFile(mmprojPath)
	if err != nil {
		t.Fatalf("failed to read mmproj file: %v", err)
	}
	if string(content) != string(mmprojContent) {
		t.Error("mmproj content mismatch")
	}

	// Verify metadata has mmproj
	entry := puller.metadata.Find(repo, "Q4_K_M")
	if entry == nil {
		t.Fatal("metadata entry not found")
	}
	if entry.Mmproj == nil {
		t.Fatal("metadata Mmproj should not be nil")
	}
	if entry.Mmproj.Filename != expectedMmprojFilename {
		t.Errorf("metadata Mmproj.Filename = %q, want %q", entry.Mmproj.Filename, expectedMmprojFilename)
	}
	if entry.Mmproj.Size != int64(len(mmprojContent)) {
		t.Errorf("metadata Mmproj.Size = %d, want %d", entry.Mmproj.Size, len(mmprojContent))
	}
}

func TestPull_WithMmproj_DownloadFailure(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	mmprojContent := []byte("fake-mmproj-binary-content")

	srv, _ := newMmprojTestServer(t, modelContent, mmprojContent, http.StatusInternalServerError)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)
	repo := "ggml-org/gemma-3-4b-it-GGUF"

	// Act
	result, err := puller.Pull(context.Background(), repo, "Q4_K_M")

	// Assert - should succeed (model downloaded) but with MmprojFailed=true
	if err != nil {
		t.Fatalf("Pull() error = %v, want nil (partial success)", err)
	}
	if result.Filename != "model-Q4_K_M.gguf" {
		t.Errorf("Filename = %q, want %q", result.Filename, "model-Q4_K_M.gguf")
	}
	if !result.MmprojFailed {
		t.Error("MmprojFailed = false, want true")
	}
	if result.MmprojFilename != "" {
		t.Errorf("MmprojFilename = %q, want empty (download failed)", result.MmprojFilename)
	}

	// Verify model file exists
	modelPath := filepath.Join(tmpDir, "model-Q4_K_M.gguf")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("model file should exist after partial success")
	}

	// Verify metadata has NO mmproj
	entry := puller.metadata.Find(repo, "Q4_K_M")
	if entry == nil {
		t.Fatal("metadata entry not found")
	}
	if entry.Mmproj != nil {
		t.Error("metadata Mmproj should be nil when mmproj download failed")
	}
}

func TestMmprojStorageFilename(t *testing.T) {
	tests := []struct {
		name             string
		repo             string
		originalFilename string
		want             string
	}{
		{
			name:             "standard repo with slash",
			repo:             "ggml-org/gemma-3-4b-it-GGUF",
			originalFilename: "mmproj-model-f16.gguf",
			want:             "ggml-org_gemma-3-4b-it-GGUF_mmproj-model-f16.gguf",
		},
		{
			name:             "simple repo",
			repo:             "user/model",
			originalFilename: "projector.gguf",
			want:             "user_model_projector.gguf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mmprojStorageFilename(tt.repo, tt.originalFilename)
			if got != tt.want {
				t.Errorf("mmprojStorageFilename(%q, %q) = %q, want %q", tt.repo, tt.originalFilename, got, tt.want)
			}
		})
	}
}
