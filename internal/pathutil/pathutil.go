// Package pathutil provides path manipulation utilities.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// expandTilde expands ~ to home directory.
// Returns the path unchanged if it doesn't start with ~/.
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand home dir: %w", err)
	}

	return filepath.Join(home, path[2:]), nil
}

// ResolvePath resolves a path with tilde expansion and relative path resolution.
// - ~/... paths are expanded to home directory
// - Absolute paths are returned as-is
// - Relative paths are resolved from baseDir
// - Empty paths are not allowed and return an error
func ResolvePath(path, baseDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Handle tilde expansion first
	if strings.HasPrefix(path, "~/") {
		return expandTilde(path)
	}

	// Absolute paths are returned as-is
	if filepath.IsAbs(path) {
		return path, nil
	}

	// All relative paths are resolved from baseDir
	return filepath.Join(baseDir, path), nil
}
