package receipt

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectInstallSource determines the install source by examining the binary path.
// It resolves symlinks before checking to handle cases like /usr/local/bin/alpaca
// pointing to /opt/homebrew/Cellar/...
func DetectInstallSource(binaryPath string) InstallSource {
	// Resolve symlinks to get the real path
	resolved, err := filepath.EvalSymlinks(binaryPath)
	if err != nil {
		resolved = binaryPath
	}

	// Homebrew (macOS/Linux)
	if isHomebrewPath(resolved) {
		return SourceBrew
	}

	// Go install
	if isGoInstallPath(resolved) {
		return SourceGo
	}

	// User-local paths are likely script installs
	if isUserLocalPath(resolved) {
		return SourceScript
	}

	// For system paths (/usr/bin, /usr/local/bin, etc.) without
	// Homebrew/Go indicators, we can't reliably distinguish between
	// apt/yum/manual install. Return SourceUnknown rather than guessing.
	return SourceUnknown
}

// isHomebrewPath checks if the path is within a Homebrew installation.
func isHomebrewPath(path string) bool {
	// Check HOMEBREW_PREFIX environment variable first
	if prefix := os.Getenv("HOMEBREW_PREFIX"); prefix != "" {
		if hasPathPrefix(path, prefix) {
			return true
		}
	}

	// Common Homebrew prefixes (without trailing slash for proper boundary check)
	homebrewPrefixes := []string{
		"/opt/homebrew",              // macOS Apple Silicon
		"/usr/local/Cellar",          // macOS Intel
		"/usr/local/opt",             // macOS Intel (symlinks)
		"/home/linuxbrew/.linuxbrew", // Linuxbrew
	}

	for _, prefix := range homebrewPrefixes {
		if hasPathPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// isGoInstallPath checks if the path is within a Go bin directory.
func isGoInstallPath(path string) bool {
	// Check GOBIN first
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		if hasPathPrefix(path, gobin) {
			return true
		}
	}

	// Check GOPATH/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Default GOPATH is ~/go
		if home, err := os.UserHomeDir(); err == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	if gopath != "" {
		goBin := filepath.Join(gopath, "bin")
		if hasPathPrefix(path, goBin) {
			return true
		}
	}

	return false
}

// isUserLocalPath checks if the path is in a user-local directory.
// These are typically used by script installs.
func isUserLocalPath(path string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// Common user-local bin directories
	userLocalPaths := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
	}

	for _, prefix := range userLocalPaths {
		if hasPathPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// hasPathPrefix checks if path starts with prefix as a proper path component.
// This avoids false positives like "/opt/homebrew2" matching "/opt/homebrew".
// It also handles trailing slashes in prefix (e.g., HOMEBREW_PREFIX=/opt/homebrew/).
func hasPathPrefix(path, prefix string) bool {
	if prefix == "" {
		return false
	}
	// Normalize both paths and remove trailing separators from prefix
	path = filepath.Clean(path)
	prefix = filepath.Clean(strings.TrimRight(prefix, string(filepath.Separator)))

	if !strings.HasPrefix(path, prefix) {
		return false
	}
	// Check that the prefix ends at a path boundary
	if len(path) == len(prefix) {
		return true
	}
	return path[len(prefix)] == filepath.Separator
}
