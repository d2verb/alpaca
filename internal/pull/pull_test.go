package pull

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPuller_SetProgressFunc(t *testing.T) {
	puller := NewPuller("/tmp/models")

	var called bool
	var gotDownloaded, gotTotal int64

	puller.SetProgressFunc(func(downloaded, total int64) {
		called = true
		gotDownloaded = downloaded
		gotTotal = total
	})

	// Simulate progress callback
	if puller.onProgress != nil {
		puller.onProgress(100, 1000)
	}

	if !called {
		t.Error("progress function was not called")
	}
	if gotDownloaded != 100 {
		t.Errorf("downloaded = %d, want 100", gotDownloaded)
	}
	if gotTotal != 1000 {
		t.Errorf("total = %d, want 1000", gotTotal)
	}
}

func TestNewPuller(t *testing.T) {
	modelsDir := "/path/to/models"
	puller := NewPuller(modelsDir)

	if puller.modelsDir != modelsDir {
		t.Errorf("modelsDir = %q, want %q", puller.modelsDir, modelsDir)
	}
	if puller.client == nil {
		t.Error("client should not be nil")
	}
	if puller.onProgress != nil {
		t.Error("onProgress should be nil by default")
	}
	if puller.metadata == nil {
		t.Error("metadata should not be nil")
	}
}

func TestDownloadFile_NewDownload(t *testing.T) {
	const testETag = `"abc123"`
	const fullContent = "0123456789"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", testETag)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	modelsDir := t.TempDir()
	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	size, err := puller.downloadFile(context.Background(), "test/repo", "model.gguf")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if size != int64(len(fullContent)) {
		t.Errorf("size = %d, want %d", size, len(fullContent))
	}

	// Verify final file exists
	content, err := os.ReadFile(filepath.Join(modelsDir, "model.gguf"))
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != fullContent {
		t.Errorf("content = %q, want %q", string(content), fullContent)
	}

	// Verify .part and .etag files are cleaned up
	if _, err := os.Stat(filepath.Join(modelsDir, "model.gguf.part")); !os.IsNotExist(err) {
		t.Error(".part file should not exist after successful download")
	}
	if _, err := os.Stat(filepath.Join(modelsDir, "model.gguf.etag")); !os.IsNotExist(err) {
		t.Error(".etag file should not exist after successful download")
	}
}

func TestDownloadFile_ResumeSuccess(t *testing.T) {
	const testETag = `"abc123"`
	const fullContent = "0123456789"
	const partialContent = "01234"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		ifRange := r.Header.Get("If-Range")

		if rangeHeader != "" && ifRange == testETag {
			var start int
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(fullContent)-1, len(fullContent)))
			w.Header().Set("ETag", testETag)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)-start))
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte(fullContent[start:]))
			return
		}

		w.Header().Set("ETag", testETag)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	modelsDir := t.TempDir()

	// Create existing .part and .etag files
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.part"), []byte(partialContent), 0644); err != nil {
		t.Fatalf("failed to create .part file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.etag"), []byte(testETag), 0644); err != nil {
		t.Fatalf("failed to create .etag file: %v", err)
	}

	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	size, err := puller.downloadFile(context.Background(), "test/repo", "model.gguf")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if size != int64(len(fullContent)) {
		t.Errorf("size = %d, want %d", size, len(fullContent))
	}

	// Verify final file content
	content, err := os.ReadFile(filepath.Join(modelsDir, "model.gguf"))
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != fullContent {
		t.Errorf("content = %q, want %q", string(content), fullContent)
	}
}

func TestDownloadFile_ETagMismatch(t *testing.T) {
	const serverETag = `"newetag"`
	const oldETag = `"oldetag"`
	const fullContent = "0123456789"
	const partialContent = "01234"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ifRange := r.Header.Get("If-Range")

		// ETag mismatch: return full content with 200 OK
		if ifRange != "" && ifRange != serverETag {
			w.Header().Set("ETag", serverETag)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fullContent))
			return
		}

		w.Header().Set("ETag", serverETag)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	modelsDir := t.TempDir()

	// Create existing .part and .etag files with old ETag
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.part"), []byte(partialContent), 0644); err != nil {
		t.Fatalf("failed to create .part file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.etag"), []byte(oldETag), 0644); err != nil {
		t.Fatalf("failed to create .etag file: %v", err)
	}

	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	size, err := puller.downloadFile(context.Background(), "test/repo", "model.gguf")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if size != int64(len(fullContent)) {
		t.Errorf("size = %d, want %d", size, len(fullContent))
	}

	// Verify final file has new content (not partial + remainder)
	content, err := os.ReadFile(filepath.Join(modelsDir, "model.gguf"))
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != fullContent {
		t.Errorf("content = %q, want %q", string(content), fullContent)
	}
}

func TestDownloadFile_RangeNotSatisfiable(t *testing.T) {
	const testETag = `"abc123"`
	const fullContent = "0123456789"
	const oversizedPart = "0123456789EXTRA" // Larger than actual file

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		rangeHeader := r.Header.Get("Range")

		if rangeHeader != "" {
			var start int
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)

			if start >= len(fullContent) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
		}

		w.Header().Set("ETag", testETag)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	modelsDir := t.TempDir()

	// Create oversized .part file
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.part"), []byte(oversizedPart), 0644); err != nil {
		t.Fatalf("failed to create .part file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.etag"), []byte(testETag), 0644); err != nil {
		t.Fatalf("failed to create .etag file: %v", err)
	}

	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	size, err := puller.downloadFile(context.Background(), "test/repo", "model.gguf")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if size != int64(len(fullContent)) {
		t.Errorf("size = %d, want %d", size, len(fullContent))
	}

	// Should have made 2 requests: first got 416, second succeeded
	if requestCount != 2 {
		t.Errorf("requestCount = %d, want 2", requestCount)
	}
}

func TestDownloadFile_InterruptionKeepsPartFiles(t *testing.T) {
	const testETag = `"abc123"`
	const fullContent = "0123456789"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", testETag)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
		w.WriteHeader(http.StatusOK)

		// Send partial content then hang
		w.Write([]byte(fullContent[:5]))
		w.(http.Flusher).Flush()

		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	modelsDir := t.TempDir()
	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := puller.downloadFile(ctx, "test/repo", "model.gguf")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context error, got: %v", err)
	}

	// Verify .part file exists and has partial content
	partContent, err := os.ReadFile(filepath.Join(modelsDir, "model.gguf.part"))
	if err != nil {
		t.Fatalf("failed to read .part file: %v", err)
	}
	if len(partContent) == 0 {
		t.Error(".part file should have partial content")
	}

	// Verify .etag file exists
	if _, err := os.Stat(filepath.Join(modelsDir, "model.gguf.etag")); os.IsNotExist(err) {
		t.Error(".etag file should exist after interruption")
	}

	// Verify final file does not exist
	if _, err := os.Stat(filepath.Join(modelsDir, "model.gguf")); !os.IsNotExist(err) {
		t.Error("final file should not exist after interruption")
	}
}

func TestDownloadFile_ProgressReporting(t *testing.T) {
	const testETag = `"abc123"`
	const fullContent = "0123456789"
	const partialContent = "01234"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		ifRange := r.Header.Get("If-Range")

		if rangeHeader != "" && ifRange == testETag {
			var start int
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(fullContent)-1, len(fullContent)))
			w.Header().Set("ETag", testETag)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)-start))
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte(fullContent[start:]))
			return
		}

		w.Header().Set("ETag", testETag)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	modelsDir := t.TempDir()

	// Create existing .part and .etag files
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.part"), []byte(partialContent), 0644); err != nil {
		t.Fatalf("failed to create .part file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.etag"), []byte(testETag), 0644); err != nil {
		t.Fatalf("failed to create .etag file: %v", err)
	}

	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	var progressCalls []struct{ downloaded, total int64 }
	puller.SetProgressFunc(func(downloaded, total int64) {
		progressCalls = append(progressCalls, struct{ downloaded, total int64 }{downloaded, total})
	})

	_, err := puller.downloadFile(context.Background(), "test/repo", "model.gguf")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if len(progressCalls) == 0 {
		t.Error("progress callback was never called")
	}

	// Verify progress starts from existing size
	lastCall := progressCalls[len(progressCalls)-1]
	if lastCall.downloaded != int64(len(fullContent)) {
		t.Errorf("final downloaded = %d, want %d", lastCall.downloaded, len(fullContent))
	}
	if lastCall.total != int64(len(fullContent)) {
		t.Errorf("final total = %d, want %d", lastCall.total, len(fullContent))
	}
}

func TestDownloadFile_PartWithoutETag(t *testing.T) {
	const testETag = `"abc123"`
	const fullContent = "0123456789"
	const partialContent = "01234"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		rangeHeader := r.Header.Get("Range")

		// Should not receive Range header since .etag is missing
		if rangeHeader != "" {
			t.Errorf("unexpected Range header: %s", rangeHeader)
		}

		w.Header().Set("ETag", testETag)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	modelsDir := t.TempDir()

	// Create .part file WITHOUT .etag file
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.part"), []byte(partialContent), 0644); err != nil {
		t.Fatalf("failed to create .part file: %v", err)
	}
	// Intentionally NOT creating .etag file

	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	size, err := puller.downloadFile(context.Background(), "test/repo", "model.gguf")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if size != int64(len(fullContent)) {
		t.Errorf("size = %d, want %d", size, len(fullContent))
	}

	// Verify final file has full content (downloaded from scratch)
	content, err := os.ReadFile(filepath.Join(modelsDir, "model.gguf"))
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != fullContent {
		t.Errorf("content = %q, want %q", string(content), fullContent)
	}

	// Should have made exactly 1 request (no resume attempt)
	if requestCount != 1 {
		t.Errorf("requestCount = %d, want 1", requestCount)
	}
}

func TestDownloadFile_ContentRangeMismatch(t *testing.T) {
	const testETag = `"abc123"`
	const fullContent = "0123456789"
	const partialContent = "01234"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		rangeHeader := r.Header.Get("Range")

		if requestCount == 1 && rangeHeader != "" {
			// First request: return 206 with WRONG Content-Range start
			// Client requested bytes=5-, but we return bytes=3-
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 3-%d/%d", len(fullContent)-1, len(fullContent)))
			w.Header().Set("ETag", testETag)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)-3))
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte(fullContent[3:]))
			return
		}

		// Second request: full download
		w.Header().Set("ETag", testETag)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	modelsDir := t.TempDir()

	// Create .part and .etag files
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.part"), []byte(partialContent), 0644); err != nil {
		t.Fatalf("failed to create .part file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf.etag"), []byte(testETag), 0644); err != nil {
		t.Fatalf("failed to create .etag file: %v", err)
	}

	puller := NewPuller(modelsDir)
	puller.baseURL = server.URL

	size, err := puller.downloadFile(context.Background(), "test/repo", "model.gguf")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if size != int64(len(fullContent)) {
		t.Errorf("size = %d, want %d", size, len(fullContent))
	}

	// Should have made 2 requests: first got mismatched Content-Range, second succeeded
	if requestCount != 2 {
		t.Errorf("requestCount = %d, want 2", requestCount)
	}

	// Verify final file has correct content
	content, err := os.ReadFile(filepath.Join(modelsDir, "model.gguf"))
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != fullContent {
		t.Errorf("content = %q, want %q", string(content), fullContent)
	}
}

func TestParseContentRangeStart(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    int64
		wantErr bool
	}{
		{
			name:   "valid range with known total",
			header: "bytes 100-199/1000",
			want:   100,
		},
		{
			name:   "valid range with unknown total",
			header: "bytes 0-499/*",
			want:   0,
		},
		{
			name:   "large offset",
			header: "bytes 1048576-2097151/10485760",
			want:   1048576,
		},
		{
			name:    "empty header",
			header:  "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			header:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseContentRangeStart(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseContentRangeStart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseContentRangeStart() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestVerifyFileHash_MatchingHash(t *testing.T) {
	// Arrange
	content := []byte("test file content for hashing")
	h := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(h[:])

	modelsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf"), content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	puller := NewPuller(modelsDir)

	// Act
	err := puller.verifyFileHash("model.gguf", expectedHash)

	// Assert
	if err != nil {
		t.Errorf("verifyFileHash() error = %v, want nil", err)
	}
}

func TestVerifyFileHash_MismatchedHash(t *testing.T) {
	// Arrange
	content := []byte("test file content for hashing")
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	modelsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modelsDir, "model.gguf"), content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	puller := NewPuller(modelsDir)

	// Act
	err := puller.verifyFileHash("model.gguf", wrongHash)

	// Assert
	if err == nil {
		t.Fatal("verifyFileHash() error = nil, want error for mismatched hash")
	}
	if !strings.Contains(err.Error(), "expected SHA256") {
		t.Errorf("error = %q, want to contain 'expected SHA256'", err.Error())
	}
}

func TestVerifyFileHash_FileNotFound(t *testing.T) {
	// Arrange
	modelsDir := t.TempDir()
	puller := NewPuller(modelsDir)

	// Act
	err := puller.verifyFileHash("nonexistent.gguf", "abc123")

	// Assert
	if err == nil {
		t.Fatal("verifyFileHash() error = nil, want error for missing file")
	}
	if !strings.Contains(err.Error(), "open file for verification") {
		t.Errorf("error = %q, want to contain 'open file for verification'", err.Error())
	}
}

// TestFilepathIsLocalBehavior documents the expected behavior of filepath.IsLocal
// which is used to validate filenames before passing to os.Root.Create.
func TestFilepathIsLocalBehavior(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantOK   bool
	}{
		// Valid filenames
		{name: "simple filename", filename: "model.gguf", wantOK: true},
		{name: "filename with dashes and dots", filename: "codellama-7b.Q4_K_M.gguf", wantOK: true},
		{name: "hidden file", filename: ".hidden.gguf", wantOK: true},
		{name: "double dots in name", filename: "model..v2.gguf", wantOK: true},
		{name: "filename starting with double dots", filename: "..model.gguf", wantOK: true},

		// Invalid filenames (path traversal attempts)
		{name: "path traversal", filename: "../../../.bashrc", wantOK: false},
		{name: "path traversal mid-path", filename: "foo/../../../.bashrc", wantOK: false},
		{name: "absolute path", filename: "/etc/passwd", wantOK: false},
		{name: "double dot only", filename: "..", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filepath.IsLocal(tt.filename)
			if got != tt.wantOK {
				t.Errorf("filepath.IsLocal(%q) = %v, want %v", tt.filename, got, tt.wantOK)
			}
		})
	}
}
