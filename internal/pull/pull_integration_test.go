package pull

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// newTestPuller creates a Puller configured for testing with a custom base URL.
func newTestPuller(modelsDir, baseURL string) *Puller {
	p := NewPuller(modelsDir)
	p.baseURL = baseURL
	return p
}

func TestPull_Success(t *testing.T) {
	// Arrange
	modelContent := []byte("fake-model-binary-content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/models/"):
			// Return repo info with GGUF files
			resp := struct {
				Siblings []struct {
					Filename string `json:"rfilename"`
				} `json:"siblings"`
			}{
				Siblings: []struct {
					Filename string `json:"rfilename"`
				}{
					{Filename: "model-Q4_K_M.gguf"},
					{Filename: "model-Q8_0.gguf"},
					{Filename: "README.md"},
				},
			}
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			// Return model file
			w.Header().Set("Content-Length", "25")
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

func TestPull_NoMatchingQuant(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Siblings []struct {
				Filename string `json:"rfilename"`
			} `json:"siblings"`
		}{
			Siblings: []struct {
				Filename string `json:"rfilename"`
			}{
				{Filename: "model-Q4_K_M.gguf"},
			},
		}
		json.NewEncoder(w).Encode(resp)
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
	if !strings.Contains(err.Error(), "no matching file found") {
		t.Errorf("error = %q, want to contain 'no matching file found'", err.Error())
	}
}

func TestPull_DownloadError(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/models/"):
			resp := struct {
				Siblings []struct {
					Filename string `json:"rfilename"`
				} `json:"siblings"`
			}{
				Siblings: []struct {
					Filename string `json:"rfilename"`
				}{
					{Filename: "model-Q4_K_M.gguf"},
				},
			}
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/models/"):
			resp := struct {
				Siblings []struct {
					Filename string `json:"rfilename"`
				} `json:"siblings"`
			}{
				Siblings: []struct {
					Filename string `json:"rfilename"`
				}{
					{Filename: "model-Q4_K_M.gguf"},
				},
			}
			json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/resolve/main/"):
			if r.Method == "HEAD" {
				w.Header().Set("Content-Length", "1234567890")
				w.WriteHeader(http.StatusOK)
			}
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
	if size != 1234567890 {
		t.Errorf("size = %d, want 1234567890", size)
	}
}
