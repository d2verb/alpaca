package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/semver"
)

func TestNew(t *testing.T) {
	// Act
	u := New("v1.0.0")

	// Assert
	if u == nil {
		t.Fatal("New returned nil")
	}
	if u.currentVersion != "v1.0.0" {
		t.Errorf("currentVersion = %s, want v1.0.0", u.currentVersion)
	}
	if u.client == nil {
		t.Error("client should not be nil")
	}
}

func TestGetAssetName(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		wantOS   string
		wantArch string
	}{
		{
			name:    "version with v prefix",
			tagName: "v1.2.3",
		},
		{
			name:    "version without v prefix",
			tagName: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := New("v1.0.0")

			// Act
			assetName := u.getAssetName(tt.tagName)

			// Assert
			if assetName == "" {
				t.Error("assetName should not be empty")
			}
			// Should contain .tar.gz
			if filepath.Ext(assetName) != ".gz" {
				t.Errorf("assetName should end with .gz, got %s", assetName)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")
	content := []byte("test content for checksum")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Calculate expected checksum
	h := sha256.New()
	h.Write(content)
	expectedChecksum := hex.EncodeToString(h.Sum(nil))

	u := New("v1.0.0")

	t.Run("valid checksum", func(t *testing.T) {
		// Act
		err := u.verifyChecksum(testFile, expectedChecksum)

		// Assert
		if err != nil {
			t.Errorf("verifyChecksum should succeed: %v", err)
		}
	})

	t.Run("invalid checksum", func(t *testing.T) {
		// Act
		err := u.verifyChecksum(testFile, "invalid-checksum")

		// Assert
		if err == nil {
			t.Error("verifyChecksum should fail with invalid checksum")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		// Act
		err := u.verifyChecksum("/nonexistent/path", expectedChecksum)

		// Assert
		if err == nil {
			t.Error("verifyChecksum should fail with nonexistent file")
		}
	})
}

func TestExtractBinary(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create a test tar.gz archive with an alpaca binary
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	binaryContent := []byte("#!/bin/sh\necho 'hello from alpaca'")

	if err := createTestArchive(archivePath, "alpaca", binaryContent); err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	u := New("v1.0.0")
	destPath := filepath.Join(tmpDir, "extracted-alpaca")

	// Act
	err := u.extractBinary(archivePath, destPath)

	// Assert
	if err != nil {
		t.Fatalf("extractBinary failed: %v", err)
	}

	extracted, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extracted) != string(binaryContent) {
		t.Errorf("extracted content mismatch")
	}
}

func TestExtractBinaryNoBinary(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create a tar.gz archive without alpaca binary
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	if err := createTestArchive(archivePath, "other-file", []byte("not alpaca")); err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	u := New("v1.0.0")
	destPath := filepath.Join(tmpDir, "extracted-alpaca")

	// Act
	err := u.extractBinary(archivePath, destPath)

	// Assert
	if err == nil {
		t.Error("extractBinary should fail when alpaca binary not found")
	}
}

func TestReplaceBinary(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	currentPath := filepath.Join(tmpDir, "alpaca")
	newPath := filepath.Join(tmpDir, "alpaca-new")

	// Create current binary
	if err := os.WriteFile(currentPath, []byte("old version"), 0755); err != nil {
		t.Fatalf("failed to create current binary: %v", err)
	}

	// Create new binary
	newContent := []byte("new version")
	if err := os.WriteFile(newPath, newContent, 0755); err != nil {
		t.Fatalf("failed to create new binary: %v", err)
	}

	u := New("v1.0.0")

	// Act
	err := u.replaceBinary(currentPath, newPath)

	// Assert
	if err != nil {
		t.Fatalf("replaceBinary failed: %v", err)
	}

	// Verify the binary was replaced
	content, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("failed to read replaced binary: %v", err)
	}

	if string(content) != string(newContent) {
		t.Errorf("binary content mismatch after replace")
	}
}

func TestDownloadFile(t *testing.T) {
	// Arrange
	expectedContent := "downloaded content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded")

	u := New("v1.0.0")

	// Act
	err := u.downloadFile(server.URL, destPath)

	// Assert
	if err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("content = %s, want %s", string(content), expectedContent)
	}
}

func TestDownloadFileServerError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded")

	u := New("v1.0.0")

	// Act
	err := u.downloadFile(server.URL, destPath)

	// Assert
	if err == nil {
		t.Error("downloadFile should fail on server error")
	}
}

func TestEnsureVPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"0.1.0", "v0.1.0"},
		{"v0.1.0", "v0.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ensureVPrefix(tt.input)
			if got != tt.want {
				t.Errorf("ensureVPrefix(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestVersionComparison(t *testing.T) {
	// This tests the version comparison logic indirectly
	// by testing various version pairs
	tests := []struct {
		name            string
		current         string
		latest          string
		expectHasUpdate bool
	}{
		{"same version", "v1.0.0", "v1.0.0", false},
		{"patch update", "v1.0.0", "v1.0.1", true},
		{"minor update", "v1.0.0", "v1.1.0", true},
		{"major update", "v1.0.0", "v2.0.0", true},
		{"current is newer", "v2.0.0", "v1.0.0", false},
		{"1.10 vs 1.9 - semver correct", "v1.9.0", "v1.10.0", true},
		{"1.2 vs 1.10 - semver correct", "v1.2.0", "v1.10.0", true},
		{"no v prefix", "1.0.0", "1.0.1", true},
		{"mixed prefix", "v1.0.0", "1.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := ensureVPrefix(tt.current)
			latest := ensureVPrefix(tt.latest)

			// Simulate the comparison logic from CheckUpdate using real semver
			hasUpdate := semver.IsValid(latest) &&
				semver.IsValid(current) &&
				semver.Compare(current, latest) < 0

			if hasUpdate != tt.expectHasUpdate {
				t.Errorf("hasUpdate = %v, want %v (current=%s, latest=%s)",
					hasUpdate, tt.expectHasUpdate, current, latest)
			}
		})
	}
}

// Helper function to create a test tar.gz archive
func createTestArchive(archivePath, fileName string, content []byte) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	header := &tar.Header{
		Name: fileName,
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("write content: %w", err)
	}

	return nil
}
