package pull

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// newTestPuller creates a Puller configured for testing with a custom base URL.
func newTestPuller(modelsDir, baseURL string) *Puller {
	p := NewPuller(modelsDir)
	p.baseURL = baseURL
	return p
}

// computeSHA256 returns the hex-encoded SHA256 hash of data.
func computeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// newManifestResponse creates a manifestResponse with the given parameters.
func newManifestResponse(filename string, size int64, sha256Hash string) manifestResponse {
	resp := manifestResponse{
		GGUFFile: &manifestFile{
			Filename: filename,
			Size:     size,
		},
	}
	if sha256Hash != "" {
		resp.GGUFFile.LFS = &manifestLFS{SHA256: sha256Hash}
	}
	return resp
}

// newManifestResponseWithMmproj creates a manifestResponse that includes mmproj file info.
func newManifestResponseWithMmproj(
	filename string, size int64, sha256Hash string,
	mmprojFilename string, mmprojSize int64, mmprojSHA256 string,
) manifestResponse {
	resp := newManifestResponse(filename, size, sha256Hash)
	resp.MmprojFile = &manifestFile{
		Filename: mmprojFilename,
		Size:     mmprojSize,
	}
	if mmprojSHA256 != "" {
		resp.MmprojFile.LFS = &manifestLFS{SHA256: mmprojSHA256}
	}
	return resp
}

func TestPull_Success(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), modelHash)
			json.NewEncoder(w).Encode(resp)

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

	// Act
	result, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}
	if result.Filename != "model-Q4_K_M.gguf" {
		t.Errorf("Filename = %q, want %q", result.Filename, "model-Q4_K_M.gguf")
	}
	if result.Size != int64(len(modelContent)) {
		t.Errorf("Size = %d, want %d", result.Size, len(modelContent))
	}
	if result.MmprojFilename != "" {
		t.Errorf("MmprojFilename = %q, want empty", result.MmprojFilename)
	}
	if result.MmprojFailed {
		t.Error("MmprojFailed = true, want false")
	}

	// Verify file was written
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != string(modelContent) {
		t.Error("downloaded content mismatch")
	}

	// Verify metadata has nil mmproj
	entry := puller.metadata.Find("test/model", "Q4_K_M")
	if entry == nil {
		t.Fatal("metadata entry not found")
	}
	if entry.Mmproj != nil {
		t.Error("metadata Mmproj should be nil for text-only model")
	}
}

func TestPull_RepoNotFound(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.Pull(context.Background(), "nonexistent/repo", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
	if !strings.Contains(err.Error(), "repository not found") {
		t.Errorf("error = %q, want to contain 'repository not found'", err.Error())
	}
}

func TestPull_RepoNotFoundWith404(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.Pull(context.Background(), "nonexistent/repo", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
	if !strings.Contains(err.Error(), "repository not found") {
		t.Errorf("error = %q, want to contain 'repository not found'", err.Error())
	}
}

func TestPull_InvalidQuantization(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.Pull(context.Background(), "test/model", "Q8_0")

	// Assert
	if err == nil {
		t.Fatal("expected error for non-matching quant")
	}
	if !strings.Contains(err.Error(), "invalid quantization") {
		t.Errorf("error = %q, want to contain 'invalid quantization'", err.Error())
	}
}

func TestPull_DownloadError(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("model-Q4_K_M.gguf", 12345, "abc123")
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for download failure")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %q, want to contain 'status 500'", err.Error())
	}
}

func TestGetFileInfo_Success(t *testing.T) {
	// Arrange
	expectedSize := int64(1234567890)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("model-Q4_K_M.gguf", expectedSize, "abc123")
			json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	info, err := puller.GetFileInfo(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}
	if info.Filename != "model-Q4_K_M.gguf" {
		t.Errorf("Filename = %q, want %q", info.Filename, "model-Q4_K_M.gguf")
	}
	if info.Size != expectedSize {
		t.Errorf("Size = %d, want %d", info.Size, expectedSize)
	}
	if info.MmprojFilename != "" {
		t.Errorf("MmprojFilename = %q, want empty", info.MmprojFilename)
	}
	if info.MmprojSize != 0 {
		t.Errorf("MmprojSize = %d, want 0", info.MmprojSize)
	}
}

func TestPull_IntegrityVerificationFailure(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), wrongHash)
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContent)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !strings.Contains(err.Error(), "integrity verification failed") {
		t.Errorf("error = %q, want to contain 'integrity verification failed'", err.Error())
	}

	// Verify file was cleaned up
	modelPath := filepath.Join(tmpDir, "model-Q4_K_M.gguf")
	if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
		t.Error("corrupted file should be removed after hash mismatch")
	}
}

func TestPull_NoHashAvailable(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			// Return manifest WITHOUT lfs field (no hash available)
			resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), "")
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(modelContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(modelContent)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert - should fail because hash is missing (fail-closed)
	if err == nil {
		t.Fatal("Pull() error = nil, want error for missing SHA256 hash")
	}
	if !strings.Contains(err.Error(), "no SHA256 hash available") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "no SHA256 hash available")
	}

	// Verify file was cleaned up
	filePath := filepath.Join(tmpDir, "model-Q4_K_M.gguf")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("downloaded file should have been cleaned up")
	}
}

func TestPull_ManifestMissingGGUFFile(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			fmt.Fprint(w, `{"ggufFile": null}`)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.Pull(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error for manifest with null ggufFile")
	}
	if !strings.Contains(err.Error(), "no GGUF file found") {
		t.Errorf("error = %q, want to contain 'no GGUF file found'", err.Error())
	}
}

func TestGetFileInfo_SendsLlamaCppUserAgent(t *testing.T) {
	// Arrange
	var capturedUserAgent string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		resp := newManifestResponse("model-Q4_K_M.gguf", 1024, "abc123")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	_, err := puller.GetFileInfo(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}
	if capturedUserAgent != "llama-cpp" {
		t.Errorf("User-Agent = %q, want %q", capturedUserAgent, "llama-cpp")
	}
}

// --- Mmproj tests ---

// newMmprojTestServer creates a test server that serves both model and mmproj files.
// mmprojStatus controls the HTTP status for mmproj download requests (0 means 200 OK).
func newMmprojTestServer(t *testing.T, modelContent, mmprojContent []byte, mmprojStatus int) (*httptest.Server, manifestResponse) {
	t.Helper()

	modelHash := computeSHA256(modelContent)
	mmprojHash := computeSHA256(mmprojContent)
	mmprojOriginalFilename := "mmproj-model-f16.gguf"

	manifest := newManifestResponseWithMmproj(
		"model-Q4_K_M.gguf", int64(len(modelContent)), modelHash,
		mmprojOriginalFilename, int64(len(mmprojContent)), mmprojHash,
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			json.NewEncoder(w).Encode(manifest)

		case strings.Contains(r.URL.Path, "/resolve/main/"+mmprojOriginalFilename):
			if mmprojStatus != 0 {
				w.WriteHeader(mmprojStatus)
				return
			}
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

	return srv, manifest
}

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

func TestPull_TextOnlyModel(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")
	modelHash := computeSHA256(modelContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			// No mmprojFile in response
			resp := newManifestResponse("model-Q4_K_M.gguf", int64(len(modelContent)), modelHash)
			json.NewEncoder(w).Encode(resp)

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

	// Act
	result, err := puller.Pull(context.Background(), "test/textonly-model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}
	if result.MmprojFilename != "" {
		t.Errorf("MmprojFilename = %q, want empty", result.MmprojFilename)
	}
	if result.MmprojSize != 0 {
		t.Errorf("MmprojSize = %d, want 0", result.MmprojSize)
	}
	if result.MmprojFailed {
		t.Error("MmprojFailed = true, want false")
	}

	// Verify metadata has nil mmproj
	entry := puller.metadata.Find("test/textonly-model", "Q4_K_M")
	if entry == nil {
		t.Fatal("metadata entry not found")
	}
	if entry.Mmproj != nil {
		t.Error("metadata Mmproj should be nil for text-only model")
	}
}

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

func TestGetFileInfo_WithMmproj(t *testing.T) {
	// Arrange
	expectedModelSize := int64(2489757856)
	expectedMmprojSize := int64(851251104)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponseWithMmproj(
				"gemma-3-4b-it-Q4_K_M.gguf", expectedModelSize, "abc123",
				"mmproj-model-f16.gguf", expectedMmprojSize, "def456",
			)
			json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)
	repo := "ggml-org/gemma-3-4b-it-GGUF"

	// Act
	info, err := puller.GetFileInfo(context.Background(), repo, "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}
	if info.Filename != "gemma-3-4b-it-Q4_K_M.gguf" {
		t.Errorf("Filename = %q, want %q", info.Filename, "gemma-3-4b-it-Q4_K_M.gguf")
	}
	if info.Size != expectedModelSize {
		t.Errorf("Size = %d, want %d", info.Size, expectedModelSize)
	}

	expectedMmprojFilename := "ggml-org_gemma-3-4b-it-GGUF_mmproj-model-f16.gguf"
	if info.MmprojFilename != expectedMmprojFilename {
		t.Errorf("MmprojFilename = %q, want %q", info.MmprojFilename, expectedMmprojFilename)
	}
	if info.MmprojSize != expectedMmprojSize {
		t.Errorf("MmprojSize = %d, want %d", info.MmprojSize, expectedMmprojSize)
	}
}

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

func TestGetFileInfo_WithoutMmproj(t *testing.T) {
	// Arrange
	expectedSize := int64(5030000000)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			resp := newManifestResponse("Qwen3-8B-Q4_K_M.gguf", expectedSize, "abc123")
			json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	puller := newTestPuller(tmpDir, srv.URL)

	// Act
	info, err := puller.GetFileInfo(context.Background(), "unsloth/Qwen3-8B-GGUF", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}
	if info.Filename != "Qwen3-8B-Q4_K_M.gguf" {
		t.Errorf("Filename = %q, want %q", info.Filename, "Qwen3-8B-Q4_K_M.gguf")
	}
	if info.Size != expectedSize {
		t.Errorf("Size = %d, want %d", info.Size, expectedSize)
	}
	if info.MmprojFilename != "" {
		t.Errorf("MmprojFilename = %q, want empty", info.MmprojFilename)
	}
	if info.MmprojSize != 0 {
		t.Errorf("MmprojSize = %d, want 0", info.MmprojSize)
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
