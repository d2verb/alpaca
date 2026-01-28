// Package pull handles downloading models from HuggingFace.
package pull

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/d2verb/alpaca/internal/metadata"
)

const (
	huggingFaceAPIURL = "https://huggingface.co/api/models"
)

// ProgressFunc is called during download with current and total bytes.
type ProgressFunc func(downloaded, total int64)

// Puller handles model downloads from HuggingFace.
type Puller struct {
	modelsDir  string
	client     *http.Client
	onProgress ProgressFunc
	metadata   *metadata.Manager
}

// NewPuller creates a new model puller.
func NewPuller(modelsDir string) *Puller {
	return &Puller{
		modelsDir: modelsDir,
		client:    &http.Client{},
		metadata:  metadata.NewManager(modelsDir),
	}
}

// SetProgressFunc sets the progress callback function.
func (p *Puller) SetProgressFunc(fn ProgressFunc) {
	p.onProgress = fn
}

// ParseModelSpec parses a model specification (repo:quant).
func ParseModelSpec(spec string) (repo, quant string, err error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid model spec: expected 'repo:quant', got '%s'", spec)
	}
	return parts[0], parts[1], nil
}

// PullResult contains information about the downloaded file.
type PullResult struct {
	Path     string
	Filename string
	Size     int64
}

// Pull downloads a model from HuggingFace.
func (p *Puller) Pull(ctx context.Context, repo, quant string) (*PullResult, error) {
	// Load existing metadata
	if err := p.metadata.Load(ctx); err != nil {
		return nil, fmt.Errorf("load metadata: %w", err)
	}

	// Find matching file
	filename, err := p.findMatchingFile(ctx, repo, quant)
	if err != nil {
		return nil, err
	}

	// Download file
	destPath := filepath.Join(p.modelsDir, filename)
	size, err := p.downloadFile(ctx, repo, filename, destPath)
	if err != nil {
		return nil, err
	}

	// Save metadata entry
	entry := metadata.ModelEntry{
		Repo:         repo,
		Quant:        quant,
		Filename:     filename,
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
		Filename: filename,
		Size:     size,
	}, nil
}

// GetFileInfo fetches info about the model file without downloading.
func (p *Puller) GetFileInfo(ctx context.Context, repo, quant string) (filename string, size int64, err error) {
	filename, err = p.findMatchingFile(ctx, repo, quant)
	if err != nil {
		return "", 0, err
	}

	// Get file size via HEAD request
	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repo, filename)
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

func (p *Puller) findMatchingFile(ctx context.Context, repo, quant string) (string, error) {
	files, err := p.listGGUFFiles(ctx, repo)
	if err != nil {
		return "", err
	}

	// Find GGUF file matching quant
	quantUpper := strings.ToUpper(quant)
	for _, filename := range files {
		if strings.Contains(strings.ToUpper(filename), quantUpper) {
			return filename, nil
		}
	}

	// Build available quants for error message
	available := extractQuants(files)
	if len(available) > 0 {
		return "", fmt.Errorf("no matching file found for quant '%s'\nAvailable: %s", quant, strings.Join(available, ", "))
	}
	return "", fmt.Errorf("no GGUF files found in repository '%s'", repo)
}

func (p *Puller) listGGUFFiles(ctx context.Context, repo string) ([]string, error) {
	url := fmt.Sprintf("%s/%s", huggingFaceAPIURL, repo)
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
		} `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return nil, fmt.Errorf("parse repo info: %w", err)
	}

	var files []string
	for _, sibling := range repoInfo.Siblings {
		if strings.HasSuffix(strings.ToLower(sibling.Filename), ".gguf") {
			files = append(files, sibling.Filename)
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

func (p *Puller) downloadFile(ctx context.Context, repo, filename, destPath string) (int64, error) {
	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repo, filename)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Copy with progress reporting
	var written int64
	total := resp.ContentLength
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			os.Remove(destPath)
			return 0, ctx.Err()
		default:
		}

		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := out.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
				if p.onProgress != nil {
					p.onProgress(written, total)
				}
			}
			if ew != nil {
				os.Remove(destPath)
				return 0, fmt.Errorf("write file: %w", ew)
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			os.Remove(destPath)
			return 0, fmt.Errorf("read response: %w", er)
		}
	}

	return written, nil
}
