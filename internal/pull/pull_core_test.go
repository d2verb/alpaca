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
	"testing"
)

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
