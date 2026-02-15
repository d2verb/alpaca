package pull

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
