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

func TestPull_RePull_MmprojFilenameChanged(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)
	mmprojContentA := []byte("mmproj-content-version-A")
	mmprojHashA := computeSHA256(mmprojContentA)
	mmprojContentB := []byte("mmproj-content-version-B")
	mmprojHashB := computeSHA256(mmprojContentB)
	repo := "ggml-org/gemma-3-4b-it-GGUF"

	// First pull: mmproj filename A
	mmprojOriginalA := "mmproj-v1.gguf"
	var pullCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			if pullCount.Load() == 0 {
				resp := newManifestResponseWithMmproj(
					"model-Q4_K_M.gguf", int64(len(modelContent)), modelHash,
					mmprojOriginalA, int64(len(mmprojContentA)), mmprojHashA,
				)
				json.NewEncoder(w).Encode(resp)
			} else {
				resp := newManifestResponseWithMmproj(
					"model-Q4_K_M.gguf", int64(len(modelContent)), modelHash,
					"mmproj-v2.gguf", int64(len(mmprojContentB)), mmprojHashB,
				)
				json.NewEncoder(w).Encode(resp)
			}

		case strings.Contains(r.URL.Path, "/resolve/main/mmproj-v1.gguf"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(mmprojContentA)))
			w.WriteHeader(http.StatusOK)
			w.Write(mmprojContentA)

		case strings.Contains(r.URL.Path, "/resolve/main/mmproj-v2.gguf"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(mmprojContentB)))
			w.WriteHeader(http.StatusOK)
			w.Write(mmprojContentB)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
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
	result1, err := puller.Pull(context.Background(), repo, "Q4_K_M")
	if err != nil {
		t.Fatalf("first Pull() error = %v", err)
	}

	storageFilenameA := result1.MmprojFilename
	if storageFilenameA == "" {
		t.Fatal("first pull should have mmproj filename")
	}

	// Verify file A exists
	if _, err := os.Stat(filepath.Join(tmpDir, storageFilenameA)); os.IsNotExist(err) {
		t.Fatal("mmproj file A should exist after first pull")
	}

	// Act - Second pull with different mmproj filename
	pullCount.Store(1)
	result2, err := puller.Pull(context.Background(), repo, "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("second Pull() error = %v", err)
	}

	expectedStorageB := mmprojStorageFilename(repo, "mmproj-v2.gguf")
	if result2.MmprojFilename != expectedStorageB {
		t.Errorf("MmprojFilename = %q, want %q", result2.MmprojFilename, expectedStorageB)
	}

	// Verify old file A is deleted
	if _, err := os.Stat(filepath.Join(tmpDir, storageFilenameA)); !os.IsNotExist(err) {
		t.Error("old mmproj file A should be deleted after re-pull with different filename")
	}

	// Verify new file B exists
	if _, err := os.Stat(filepath.Join(tmpDir, expectedStorageB)); os.IsNotExist(err) {
		t.Error("new mmproj file B should exist after re-pull")
	}
}

func TestPull_RePull_MmprojRemoved(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)
	mmprojContent := []byte("fake-mmproj-content")
	mmprojHash := computeSHA256(mmprojContent)
	repo := "ggml-org/gemma-3-4b-it-GGUF"

	var pullCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			if pullCount.Load() == 0 {
				// First pull: with mmproj
				resp := newManifestResponseWithMmproj(
					"model-Q4_K_M.gguf", int64(len(modelContent)), modelHash,
					"mmproj-model-f16.gguf", int64(len(mmprojContent)), mmprojHash,
				)
				json.NewEncoder(w).Encode(resp)
			} else {
				// Second pull: text-only (no mmproj)
				resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), modelHash)
				json.NewEncoder(w).Encode(resp)
			}

		case strings.Contains(r.URL.Path, "/resolve/main/mmproj-model-f16.gguf"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(mmprojContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(mmprojContent)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
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

	// First pull (with mmproj)
	result1, err := puller.Pull(context.Background(), repo, "Q4_K_M")
	if err != nil {
		t.Fatalf("first Pull() error = %v", err)
	}

	mmprojFilename := result1.MmprojFilename
	if mmprojFilename == "" {
		t.Fatal("first pull should have mmproj filename")
	}

	// Verify mmproj file exists
	if _, err := os.Stat(filepath.Join(tmpDir, mmprojFilename)); os.IsNotExist(err) {
		t.Fatal("mmproj file should exist after first pull")
	}

	// Act - Second pull (text-only, no mmproj)
	pullCount.Store(1)
	result2, err := puller.Pull(context.Background(), repo, "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("second Pull() error = %v", err)
	}
	if result2.MmprojFilename != "" {
		t.Errorf("MmprojFilename = %q, want empty", result2.MmprojFilename)
	}

	// Verify old mmproj file is deleted
	if _, err := os.Stat(filepath.Join(tmpDir, mmprojFilename)); !os.IsNotExist(err) {
		t.Error("old mmproj file should be deleted when upstream removes mmproj")
	}

	// Verify metadata has nil mmproj
	entry := puller.metadata.Find(repo, "Q4_K_M")
	if entry == nil {
		t.Fatal("metadata entry not found")
	}
	if entry.Mmproj != nil {
		t.Error("metadata Mmproj should be nil after re-pull without mmproj")
	}
}

func TestPull_RePull_SharedMmproj_NotDeleted(t *testing.T) {
	// Arrange: Two entries (different quants) sharing the same mmproj file.
	modelContentQ4 := []byte("fake-model-Q4-content")
	modelHashQ4 := computeSHA256(modelContentQ4)
	modelContentQ8 := []byte("fake-model-Q8-content")
	modelHashQ8 := computeSHA256(modelContentQ8)
	mmprojContent := []byte("shared-mmproj-content")
	mmprojHash := computeSHA256(mmprojContent)
	repo := "ggml-org/gemma-3-4b-it-GGUF"

	var pullCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/Q4_K_M"):
			if pullCount.Load() < 2 {
				resp := newManifestResponseWithMmproj(
					"model-Q4_K_M.gguf", int64(len(modelContentQ4)), modelHashQ4,
					"mmproj-model-f16.gguf", int64(len(mmprojContent)), mmprojHash,
				)
				json.NewEncoder(w).Encode(resp)
			} else {
				// Third pull (re-pull Q4): text-only
				resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContentQ4)), modelHashQ4)
				json.NewEncoder(w).Encode(resp)
			}

		case strings.Contains(r.URL.Path, "/manifests/Q8_0"):
			resp := newManifestResponseWithMmproj(
				"model-Q8_0.gguf", int64(len(modelContentQ8)), modelHashQ8,
				"mmproj-model-f16.gguf", int64(len(mmprojContent)), mmprojHash,
			)
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/mmproj-model-f16.gguf"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(mmprojContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(mmprojContent)

		case strings.Contains(r.URL.Path, "/resolve/main/model-Q4_K_M.gguf"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContentQ4)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContentQ4)

		case strings.Contains(r.URL.Path, "/resolve/main/model-Q8_0.gguf"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContentQ8)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContentQ8)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// First pull: Q4_K_M with mmproj
	pullCount.Store(0)
	result1, err := puller.Pull(context.Background(), repo, "Q4_K_M")
	if err != nil {
		t.Fatalf("first Pull(Q4_K_M) error = %v", err)
	}
	mmprojFilename := result1.MmprojFilename

	// Second pull: Q8_0 with same mmproj
	pullCount.Store(1)
	_, err = puller.Pull(context.Background(), repo, "Q8_0")
	if err != nil {
		t.Fatalf("Pull(Q8_0) error = %v", err)
	}

	// Verify both entries share the same mmproj
	entryQ4 := puller.metadata.Find(repo, "Q4_K_M")
	entryQ8 := puller.metadata.Find(repo, "Q8_0")
	if entryQ4 == nil || entryQ4.Mmproj == nil {
		t.Fatal("Q4_K_M metadata or Mmproj should not be nil after pull with mmproj")
	}
	if entryQ8 == nil || entryQ8.Mmproj == nil {
		t.Fatal("Q8_0 metadata or Mmproj should not be nil after pull with mmproj")
	}
	if entryQ4.Mmproj.Filename != entryQ8.Mmproj.Filename {
		t.Fatalf("entries should share same mmproj filename")
	}

	// Act - Re-pull Q4_K_M as text-only (mmproj removed)
	pullCount.Store(2)
	_, err = puller.Pull(context.Background(), repo, "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("re-pull Q4_K_M error = %v", err)
	}

	// Verify mmproj file is NOT deleted (Q8_0 still references it)
	if _, err := os.Stat(filepath.Join(tmpDir, mmprojFilename)); os.IsNotExist(err) {
		t.Error("shared mmproj file should NOT be deleted when another entry still references it")
	}

	// Verify Q4_K_M metadata no longer has mmproj
	entryQ4 = puller.metadata.Find(repo, "Q4_K_M")
	if entryQ4.Mmproj != nil {
		t.Error("Q4_K_M metadata Mmproj should be nil after re-pull without mmproj")
	}

	// Verify Q8_0 metadata still has mmproj
	entryQ8 = puller.metadata.Find(repo, "Q8_0")
	if entryQ8.Mmproj == nil {
		t.Error("Q8_0 metadata Mmproj should still exist")
	}
}
