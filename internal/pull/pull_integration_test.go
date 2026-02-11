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

	// Verify file was written
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != string(modelContent) {
		t.Error("downloaded content mismatch")
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
	filename, size, err := puller.GetFileInfo(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}
	if filename != "model-Q4_K_M.gguf" {
		t.Errorf("filename = %q, want %q", filename, "model-Q4_K_M.gguf")
	}
	if size != expectedSize {
		t.Errorf("size = %d, want %d", size, expectedSize)
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
	_, _, err := puller.GetFileInfo(context.Background(), "test/model", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}
	if capturedUserAgent != "llama-cpp" {
		t.Errorf("User-Agent = %q, want %q", capturedUserAgent, "llama-cpp")
	}
}
