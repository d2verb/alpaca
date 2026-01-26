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
)

const (
	huggingFaceAPIURL = "https://huggingface.co/api/models"
)

// Puller handles model downloads from HuggingFace.
type Puller struct {
	modelsDir string
	client    *http.Client
}

// NewPuller creates a new model puller.
func NewPuller(modelsDir string) *Puller {
	return &Puller{
		modelsDir: modelsDir,
		client:    &http.Client{},
	}
}

// ParseModelSpec parses a model specification (repo:quant).
func ParseModelSpec(spec string) (repo, quant string, err error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid model spec: expected 'repo:quant', got '%s'", spec)
	}
	return parts[0], parts[1], nil
}

// Pull downloads a model from HuggingFace.
func (p *Puller) Pull(ctx context.Context, repo, quant string) (string, error) {
	// Find matching file
	filename, err := p.findMatchingFile(ctx, repo, quant)
	if err != nil {
		return "", err
	}

	// Download file
	destPath := filepath.Join(p.modelsDir, filename)
	if err := p.downloadFile(ctx, repo, filename, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}

func (p *Puller) findMatchingFile(ctx context.Context, repo, quant string) (string, error) {
	url := fmt.Sprintf("%s/%s", huggingFaceAPIURL, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch repo info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("repo not found: %s", repo)
	}

	var repoInfo struct {
		Siblings []struct {
			Filename string `json:"rfilename"`
		} `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", fmt.Errorf("parse repo info: %w", err)
	}

	// Find GGUF file matching quant
	quantUpper := strings.ToUpper(quant)
	for _, sibling := range repoInfo.Siblings {
		if strings.HasSuffix(sibling.Filename, ".gguf") &&
			strings.Contains(strings.ToUpper(sibling.Filename), quantUpper) {
			return sibling.Filename, nil
		}
	}

	return "", fmt.Errorf("no matching file found for quant '%s'", quant)
}

func (p *Puller) downloadFile(ctx context.Context, repo, filename, destPath string) error {
	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repo, filename)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Copy with progress (TODO: add progress reporting)
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(destPath) // Clean up partial download
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
