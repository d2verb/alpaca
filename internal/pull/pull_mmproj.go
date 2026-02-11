package pull

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/d2verb/alpaca/internal/metadata"
)

// cleanupOldMmproj removes an outdated mmproj file when re-pulling a model.
// It handles two cases:
//   - The new manifest has a different mmproj filename than the existing entry.
//   - The new manifest has no mmproj but the existing entry does.
//
// The old file is only deleted if no other metadata entries reference it.
func (p *Puller) cleanupOldMmproj(repo, quant, newMmprojFilename string) {
	existingEntry := p.metadata.Find(repo, quant)
	if existingEntry == nil || existingEntry.Mmproj == nil {
		return
	}

	oldFilename := existingEntry.Mmproj.Filename

	// No cleanup needed if the filename hasn't changed
	if oldFilename == newMmprojFilename {
		return
	}

	// Check reference count before deleting.
	// If count == 1, only this entry references the file, so it's safe to delete.
	if p.metadata.MmprojReferenceCount(oldFilename) > 1 {
		return
	}

	slog.Info("removing outdated mmproj", "filename", oldFilename)
	p.removeDownloadedFile(oldFilename)
}

// downloadMmproj downloads and verifies an mmproj file.
// On failure, it cleans up partial files and returns an error.
func (p *Puller) downloadMmproj(ctx context.Context, repo string, fileInfo ggufFileInfo) (*metadata.MmprojEntry, error) {
	// Validate both filenames against path traversal
	if !filepath.IsLocal(fileInfo.MmprojOriginalFilename) {
		return nil, fmt.Errorf("invalid mmproj filename from API: %s", fileInfo.MmprojOriginalFilename)
	}
	if !filepath.IsLocal(fileInfo.MmprojFilename) {
		return nil, fmt.Errorf("invalid mmproj storage filename: %s", fileInfo.MmprojFilename)
	}

	// Download mmproj file using the original filename for the URL path
	size, err := p.downloadFile(ctx, repo, fileInfo.MmprojOriginalFilename)
	if err != nil {
		return nil, fmt.Errorf("download mmproj: %w", err)
	}

	// Rename from original filename to storage filename (with repo prefix)
	if fileInfo.MmprojOriginalFilename != fileInfo.MmprojFilename {
		root, err := os.OpenRoot(p.modelsDir)
		if err != nil {
			p.removeDownloadedFile(fileInfo.MmprojOriginalFilename)
			return nil, fmt.Errorf("open models dir for mmproj rename: %w", err)
		}
		defer root.Close()

		if err := root.Rename(fileInfo.MmprojOriginalFilename, fileInfo.MmprojFilename); err != nil {
			p.removeDownloadedFile(fileInfo.MmprojOriginalFilename)
			return nil, fmt.Errorf("rename mmproj file: %w", err)
		}
	}

	// Verify SHA256 integrity (fail-closed)
	if fileInfo.MmprojSHA256 == "" {
		p.removeDownloadedFile(fileInfo.MmprojFilename)
		return nil, fmt.Errorf("integrity verification failed for %s: no SHA256 hash available from API", fileInfo.MmprojFilename)
	}
	if err := p.verifyFileHash(fileInfo.MmprojFilename, fileInfo.MmprojSHA256); err != nil {
		p.removeDownloadedFile(fileInfo.MmprojFilename)
		return nil, fmt.Errorf("integrity verification failed for %s: %w", fileInfo.MmprojFilename, err)
	}

	return &metadata.MmprojEntry{
		Filename: fileInfo.MmprojFilename,
		Size:     size,
	}, nil
}

// mmprojStorageFilename generates a prefixed storage filename for mmproj files
// to avoid collisions between different repositories that use the same mmproj filename.
// Example: repo="ggml-org/gemma-3-4b-it-GGUF", filename="mmproj-model-f16.gguf"
// returns "ggml-org_gemma-3-4b-it-GGUF_mmproj-model-f16.gguf"
func mmprojStorageFilename(repo, originalFilename string) string {
	prefix := strings.ReplaceAll(repo, "/", "_")
	return prefix + "_" + originalFilename
}
