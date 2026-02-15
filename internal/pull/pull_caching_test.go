package pull

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPull_SkipsDownloadWhenAlreadyUpToDate(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)

	var downloadCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), modelHash)
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			downloadCount.Add(1)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContent)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// First pull: should download
	result1, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")
	if err != nil {
		t.Fatalf("first Pull() error = %v", err)
	}
	if result1.AlreadyUpToDate {
		t.Error("first pull should not be AlreadyUpToDate")
	}
	if downloadCount.Load() != 1 {
		t.Errorf("download count after first pull = %d, want 1", downloadCount.Load())
	}

	// Act - Second pull: should skip download
	result2, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("second Pull() error = %v", err)
	}
	if !result2.AlreadyUpToDate {
		t.Error("second pull should be AlreadyUpToDate")
	}
	if downloadCount.Load() != 1 {
		t.Errorf("download count after second pull = %d, want 1 (should not re-download)", downloadCount.Load())
	}
	if result2.Filename != "model-Q4_K_M.gguf" {
		t.Errorf("Filename = %q, want %q", result2.Filename, "model-Q4_K_M.gguf")
	}
	if result2.Size != int64(len(modelContent)) {
		t.Errorf("Size = %d, want %d", result2.Size, len(modelContent))
	}
}

func TestPull_SkipsDownloadWithMmprojWhenAlreadyUpToDate(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)
	mmprojContent := []byte("fake-mmproj-binary-content")
	mmprojHash := computeSHA256(mmprojContent)
	mmprojOriginalFilename := "mmproj-model-f16.gguf"
	repo := "ggml-org/gemma-3-4b-it-GGUF"

	var downloadCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponseWithMmproj(
				"model-Q4_K_M.gguf", int64(len(modelContent)), modelHash,
				mmprojOriginalFilename, int64(len(mmprojContent)), mmprojHash,
			)
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"+mmprojOriginalFilename):
			downloadCount.Add(1)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(mmprojContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(mmprojContent)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			downloadCount.Add(1)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContent)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// First pull: should download both files
	result1, err := puller.Pull(context.Background(), repo, "Q4_K_M")
	if err != nil {
		t.Fatalf("first Pull() error = %v", err)
	}
	if result1.AlreadyUpToDate {
		t.Error("first pull should not be AlreadyUpToDate")
	}
	if downloadCount.Load() != 2 {
		t.Errorf("download count after first pull = %d, want 2", downloadCount.Load())
	}

	// Act - Second pull: should skip both downloads
	result2, err := puller.Pull(context.Background(), repo, "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("second Pull() error = %v", err)
	}
	if !result2.AlreadyUpToDate {
		t.Error("second pull should be AlreadyUpToDate")
	}
	if downloadCount.Load() != 2 {
		t.Errorf("download count after second pull = %d, want 2 (should not re-download)", downloadCount.Load())
	}
	expectedMmproj := mmprojStorageFilename(repo, mmprojOriginalFilename)
	if result2.MmprojFilename != expectedMmproj {
		t.Errorf("MmprojFilename = %q, want %q", result2.MmprojFilename, expectedMmproj)
	}
}

func TestPull_RedownloadsWhenFileCorrupted(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)

	var downloadCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), modelHash)
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			downloadCount.Add(1)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContent)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// First pull
	_, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")
	if err != nil {
		t.Fatalf("first Pull() error = %v", err)
	}

	// Corrupt the file
	if err := os.WriteFile(filepath.Join(tmpDir, "model-Q4_K_M.gguf"), []byte("corrupted"), 0644); err != nil {
		t.Fatalf("failed to corrupt file: %v", err)
	}

	// Act - Second pull: should re-download because hash doesn't match
	result, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("second Pull() error = %v", err)
	}
	if result.AlreadyUpToDate {
		t.Error("should not be AlreadyUpToDate when file is corrupted")
	}
	if downloadCount.Load() != 2 {
		t.Errorf("download count = %d, want 2 (should re-download corrupted file)", downloadCount.Load())
	}
}

func TestPull_RegistersMetadataWhenFileExistsButMetadataMissing(t *testing.T) {
	// Arrange: model file exists on disk with correct hash, but metadata is empty.
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)

	var downloadCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), modelHash)
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			downloadCount.Add(1)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContent)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()

	// Pre-place the model file (simulating deleted metadata)
	if err := os.WriteFile(filepath.Join(tmpDir, "model-Q4_K_M.gguf"), modelContent, 0644); err != nil {
		t.Fatalf("failed to write model file: %v", err)
	}

	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	result, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert - should NOT return AlreadyUpToDate because metadata is missing
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}
	if result.AlreadyUpToDate {
		t.Error("should not be AlreadyUpToDate when metadata is missing")
	}

	// Verify download actually happened
	if downloadCount.Load() != 1 {
		t.Errorf("download count = %d, want 1", downloadCount.Load())
	}

	// Verify metadata was saved
	entry := puller.metadata.Find("test/model", "Q4_K_M")
	if entry == nil {
		t.Fatal("metadata entry should exist after pull")
	}
	if entry.Filename != "model-Q4_K_M.gguf" {
		t.Errorf("metadata Filename = %q, want %q", entry.Filename, "model-Q4_K_M.gguf")
	}
}
