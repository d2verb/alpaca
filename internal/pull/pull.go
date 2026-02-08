// Package pull handles downloading models from HuggingFace.
package pull

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/d2verb/alpaca/internal/metadata"
)

const defaultHuggingFaceBaseURL = "https://huggingface.co"

// ProgressFunc is called during download with current and total bytes.
type ProgressFunc func(downloaded, total int64)

// Puller handles model downloads from HuggingFace.
type Puller struct {
	modelsDir  string
	client     *http.Client
	onProgress ProgressFunc
	metadata   *metadata.Manager
	baseURL    string
}

// NewPuller creates a new model puller.
func NewPuller(modelsDir string) *Puller {
	return &Puller{
		modelsDir: modelsDir,
		client:    &http.Client{},
		metadata:  metadata.NewManager(modelsDir),
		baseURL:   defaultHuggingFaceBaseURL,
	}
}

// SetProgressFunc sets the progress callback function.
func (p *Puller) SetProgressFunc(fn ProgressFunc) {
	p.onProgress = fn
}

// PullResult contains information about the downloaded file.
type PullResult struct {
	Path     string
	Filename string
	Size     int64
}

// ggufFileInfo holds a GGUF filename and its optional LFS SHA256 hash.
type ggufFileInfo struct {
	Filename string
	SHA256   string // empty if not available from API
}

// Pull downloads a model from HuggingFace.
func (p *Puller) Pull(ctx context.Context, repo, quant string) (*PullResult, error) {
	// Load existing metadata
	if err := p.metadata.Load(ctx); err != nil {
		return nil, fmt.Errorf("load metadata: %w", err)
	}

	// Find matching file
	fileInfo, err := p.findMatchingFile(ctx, repo, quant)
	if err != nil {
		return nil, err
	}

	// Validate filename (for clear error messages)
	if !filepath.IsLocal(fileInfo.Filename) {
		return nil, fmt.Errorf("invalid filename from API: %s", fileInfo.Filename)
	}

	// Download file with OS-level path confinement
	size, err := p.downloadFile(ctx, repo, fileInfo.Filename)
	if err != nil {
		return nil, err
	}

	// Verify SHA256 integrity if hash is available from API
	if fileInfo.SHA256 != "" {
		if err := p.verifyFileHash(fileInfo.Filename, fileInfo.SHA256); err != nil {
			// Clean up the corrupted file
			p.removeDownloadedFile(fileInfo.Filename)
			return nil, fmt.Errorf("integrity verification failed for %s: %w", fileInfo.Filename, err)
		}
	} else {
		slog.Warn("no SHA256 hash available from API, skipping integrity verification", "filename", fileInfo.Filename)
	}

	destPath := filepath.Join(p.modelsDir, fileInfo.Filename)

	// Save metadata entry
	entry := metadata.ModelEntry{
		Repo:         repo,
		Quant:        quant,
		Filename:     fileInfo.Filename,
		Size:         size,
		DownloadedAt: time.Now().UTC(),
	}
	if err := p.metadata.Add(entry); err != nil {
		return nil, fmt.Errorf("add metadata entry: %w", err)
	}
	if err := p.metadata.Save(ctx); err != nil {
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	return &PullResult{
		Path:     destPath,
		Filename: fileInfo.Filename,
		Size:     size,
	}, nil
}

// GetFileInfo fetches info about the model file without downloading.
func (p *Puller) GetFileInfo(ctx context.Context, repo, quant string) (filename string, size int64, err error) {
	fileInfo, err := p.findMatchingFile(ctx, repo, quant)
	if err != nil {
		return "", 0, err
	}
	filename = fileInfo.Filename

	// Get file size via HEAD request
	url := fmt.Sprintf("%s/%s/resolve/main/%s", p.baseURL, repo, filename)
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("fetch file info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("file not found: status %d", resp.StatusCode)
	}

	return filename, resp.ContentLength, nil
}

func (p *Puller) findMatchingFile(ctx context.Context, repo, quant string) (ggufFileInfo, error) {
	files, err := p.listGGUFFiles(ctx, repo)
	if err != nil {
		return ggufFileInfo{}, err
	}

	// Find GGUF file matching quant
	quantUpper := strings.ToUpper(quant)
	for _, fi := range files {
		if strings.Contains(strings.ToUpper(fi.Filename), quantUpper) {
			return fi, nil
		}
	}

	// Build available quants for error message
	filenames := make([]string, len(files))
	for i, fi := range files {
		filenames[i] = fi.Filename
	}
	available := extractQuants(filenames)
	if len(available) > 0 {
		return ggufFileInfo{}, fmt.Errorf("no matching file found for quant '%s'\nAvailable: %s", quant, strings.Join(available, ", "))
	}
	return ggufFileInfo{}, fmt.Errorf("no GGUF files found in repository '%s'", repo)
}

func (p *Puller) listGGUFFiles(ctx context.Context, repo string) ([]ggufFileInfo, error) {
	url := fmt.Sprintf("%s/api/models/%s", p.baseURL, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch repo info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("repository not found: %s", repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repo info: status %d", resp.StatusCode)
	}

	var repoInfo struct {
		Siblings []struct {
			Filename string `json:"rfilename"`
			LFS      *struct {
				SHA256 string `json:"sha256"`
			} `json:"lfs"`
		} `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return nil, fmt.Errorf("parse repo info: %w", err)
	}

	var files []ggufFileInfo
	for _, sibling := range repoInfo.Siblings {
		if strings.HasSuffix(strings.ToLower(sibling.Filename), ".gguf") {
			fi := ggufFileInfo{Filename: sibling.Filename}
			if sibling.LFS != nil {
				fi.SHA256 = sibling.LFS.SHA256
			}
			files = append(files, fi)
		}
	}
	return files, nil
}

// extractQuants extracts quantization types from GGUF filenames.
func extractQuants(files []string) []string {
	quantPatterns := []string{"Q2_K", "Q3_K_S", "Q3_K_M", "Q3_K_L", "Q4_0", "Q4_K_S", "Q4_K_M", "Q5_0", "Q5_K_S", "Q5_K_M", "Q6_K", "Q8_0"}
	var found []string
	seen := make(map[string]bool)

	for _, file := range files {
		fileUpper := strings.ToUpper(file)
		for _, q := range quantPatterns {
			if strings.Contains(fileUpper, q) && !seen[q] {
				found = append(found, q)
				seen[q] = true
			}
		}
	}
	return found
}

func (p *Puller) downloadFile(ctx context.Context, repo, filename string) (int64, error) {
	partFilename := filename + ".part"
	etagFilename := filename + ".etag"

	// Open models directory with OS-level path confinement.
	// This prevents path traversal attacks even with malicious filenames.
	root, err := os.OpenRoot(p.modelsDir)
	if err != nil {
		return 0, fmt.Errorf("open models dir: %w", err)
	}
	defer root.Close()

	// Retry loop for 416 responses (max 1 retry)
	const maxRetries = 1
	for attempt := 0; attempt <= maxRetries; attempt++ {
		size, retry, err := p.doDownload(ctx, root, repo, filename, partFilename, etagFilename)
		if err != nil {
			return 0, err
		}
		if !retry {
			return size, nil
		}
		// retry == true means we got 416, files are cleaned up, try again
	}

	return 0, fmt.Errorf("download failed: max retries exceeded")
}

// doDownload performs the actual download. Returns (size, retry, error).
// retry=true indicates a 416 response was received and files were cleaned up.
func (p *Puller) doDownload(ctx context.Context, root *os.Root, repo, filename, partFilename, etagFilename string) (int64, bool, error) {
	// Check for existing .part file and .etag
	var existingSize int64
	var existingETag string
	if info, err := root.Stat(partFilename); err == nil {
		existingSize = info.Size()
		existingETag = readETagFile(root, etagFilename)
	}

	// If .part exists but .etag is missing, we cannot safely resume
	// Delete .part and start from beginning
	if existingSize > 0 && existingETag == "" {
		removePartFiles(root, partFilename, etagFilename)
		existingSize = 0
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/%s/resolve/main/%s", p.baseURL, repo, filename)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, false, fmt.Errorf("create request: %w", err)
	}

	// Set Range + If-Range headers for resume
	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
		req.Header.Set("If-Range", existingETag)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	// Handle response codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Server ignored Range, ETag mismatch, or Range not supported
		// Delete .part + .etag and start from beginning
		removePartFiles(root, partFilename, etagFilename)
		existingSize = 0
	case http.StatusPartialContent:
		// Range request successful, validate Content-Range
		rangeStart, err := parseContentRangeStart(resp.Header.Get("Content-Range"))
		if err != nil || rangeStart != existingSize {
			// Content-Range mismatch, restart from beginning
			removePartFiles(root, partFilename, etagFilename)
			// Need to re-request without Range header (defer will close resp.Body)
			return 0, true, nil
		}
	case http.StatusRequestedRangeNotSatisfiable:
		// Range invalid (.part size > server file size)
		// Delete .part + .etag and signal retry
		removePartFiles(root, partFilename, etagFilename)
		return 0, true, nil
	default:
		return 0, false, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Save ETag for new downloads
	if existingSize == 0 {
		if etag := resp.Header.Get("ETag"); etag != "" {
			if f, err := root.Create(etagFilename); err == nil {
				f.Write([]byte(etag))
				f.Close()
			}
		}
	}

	// Open file (append mode for resume, create for new)
	var out *os.File
	if existingSize > 0 {
		out, err = root.OpenFile(partFilename, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		out, err = root.Create(partFilename)
	}
	if err != nil {
		return 0, false, fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Calculate total size for progress reporting
	// ContentLength is -1 when server doesn't provide Content-Length header
	var total int64
	if resp.ContentLength < 0 {
		total = -1 // Unknown size
	} else if resp.StatusCode == http.StatusPartialContent {
		total = existingSize + resp.ContentLength
	} else {
		total = resp.ContentLength
	}

	// Copy with progress reporting
	var written int64
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return 0, false, ctx.Err()
		default:
		}

		nr, readErr := resp.Body.Read(buf)
		if nr > 0 {
			nw, writeErr := out.Write(buf[:nr])
			written += int64(nw)
			if p.onProgress != nil {
				p.onProgress(existingSize+written, total)
			}
			if writeErr != nil {
				return 0, false, fmt.Errorf("write file: %w", writeErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, false, fmt.Errorf("read response: %w", readErr)
		}
	}

	// Sync to ensure data is flushed to disk before rename
	if err := out.Sync(); err != nil {
		return 0, false, fmt.Errorf("sync file: %w", err)
	}

	// Rename .part to final filename and clean up .etag
	if err := root.Rename(partFilename, filename); err != nil {
		return 0, false, fmt.Errorf("rename file: %w", err)
	}
	root.Remove(etagFilename) // Ignore error, file may not exist

	return existingSize + written, false, nil
}

// parseContentRangeStart extracts the start byte from Content-Range header.
// Format: "bytes start-end/total" or "bytes start-end/*"
func parseContentRangeStart(header string) (int64, error) {
	if header == "" {
		return 0, fmt.Errorf("empty Content-Range header")
	}
	var start, end int64
	var total string
	_, err := fmt.Sscanf(header, "bytes %d-%d/%s", &start, &end, &total)
	if err != nil {
		return 0, fmt.Errorf("parse Content-Range: %w", err)
	}
	return start, nil
}

// removePartFiles removes .part and .etag files.
func removePartFiles(root *os.Root, partFilename, etagFilename string) {
	root.Remove(partFilename)
	root.Remove(etagFilename)
}

// readETagFile reads the ETag from file, returning empty string on any error.
func readETagFile(root *os.Root, filename string) string {
	f, err := root.Open(filename)
	if err != nil {
		return ""
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	return string(data)
}

// verifyFileHash computes the SHA256 hash of a downloaded file and compares it
// against the expected hash from the HuggingFace API.
func (p *Puller) verifyFileHash(filename, expectedSHA256 string) error {
	root, err := os.OpenRoot(p.modelsDir)
	if err != nil {
		return fmt.Errorf("open models dir: %w", err)
	}
	defer root.Close()

	f, err := root.Open(filename)
	if err != nil {
		return fmt.Errorf("open file for verification: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("compute hash: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expectedSHA256 {
		return fmt.Errorf("expected SHA256 %s, got %s", expectedSHA256, actual)
	}
	return nil
}

// removeDownloadedFile removes a downloaded file from the models directory.
func (p *Puller) removeDownloadedFile(filename string) {
	root, err := os.OpenRoot(p.modelsDir)
	if err != nil {
		return
	}
	defer root.Close()
	root.Remove(filename)
}
