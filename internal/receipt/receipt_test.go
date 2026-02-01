package receipt

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeFingerprint(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-binary")
	content := []byte("test binary content")
	if err := os.WriteFile(testFile, content, 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	fingerprint, err := ComputeFingerprint(testFile)

	// Assert
	if err != nil {
		t.Fatalf("ComputeFingerprint failed: %v", err)
	}
	if fingerprint == "" {
		t.Error("fingerprint should not be empty")
	}
	if len(fingerprint) != 71 { // "sha256:" (7) + 64 hex chars
		t.Errorf("fingerprint length = %d, want 71", len(fingerprint))
	}
	if fingerprint[:7] != "sha256:" {
		t.Errorf("fingerprint should start with 'sha256:', got %s", fingerprint[:7])
	}
}

func TestComputeFingerprintDeterministic(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-binary")
	if err := os.WriteFile(testFile, []byte("same content"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	fp1, _ := ComputeFingerprint(testFile)
	fp2, _ := ComputeFingerprint(testFile)

	// Assert
	if fp1 != fp2 {
		t.Errorf("fingerprints should be identical for same content")
	}
}

func TestComputeFingerprintFileNotFound(t *testing.T) {
	// Act
	_, err := ComputeFingerprint("/nonexistent/path")

	// Assert
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestNew(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	r, err := New(SourceScript, binaryPath)

	// Assert
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if r.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", r.SchemaVersion, SchemaVersion)
	}
	if r.InstallSource != SourceScript {
		t.Errorf("InstallSource = %s, want %s", r.InstallSource, SourceScript)
	}
	if r.BinaryPath != binaryPath {
		t.Errorf("BinaryPath = %s, want %s", r.BinaryPath, binaryPath)
	}
	if r.BinaryFingerprint == "" {
		t.Error("BinaryFingerprint should not be empty")
	}
	if r.InstalledAt.IsZero() {
		t.Error("InstalledAt should be set")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r, err := New(SourceBrew, binaryPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	receiptPath := filepath.Join(tmpDir, "install.json")

	// Act
	if err := r.SaveTo(receiptPath); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(receiptPath)

	// Assert
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if loaded.SchemaVersion != r.SchemaVersion {
		t.Errorf("SchemaVersion mismatch")
	}
	if loaded.InstallSource != r.InstallSource {
		t.Errorf("InstallSource mismatch")
	}
	if loaded.BinaryPath != r.BinaryPath {
		t.Errorf("BinaryPath mismatch")
	}
	if loaded.BinaryFingerprint != r.BinaryFingerprint {
		t.Errorf("BinaryFingerprint mismatch")
	}
}

func TestLoadNotFound(t *testing.T) {
	// Act
	_, err := LoadFrom("/nonexistent/path/install.json")

	// Assert
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestVerifySuccess(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(binaryPath, []byte("binary content"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r, _ := New(SourceScript, binaryPath)

	// Act
	result, err := r.Verify(binaryPath)

	// Assert
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !result.IsValid() {
		t.Error("Verify should succeed")
	}
	if result.PathMismatch {
		t.Error("PathMismatch should be false")
	}
	if result.FingerprintMismatch {
		t.Error("FingerprintMismatch should be false")
	}
}

func TestVerifyPathMismatch(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpaca")
	otherPath := filepath.Join(tmpDir, "alpaca-other")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	// Create another file with same content (same fingerprint)
	if err := os.WriteFile(otherPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to create other file: %v", err)
	}

	r, _ := New(SourceScript, binaryPath)

	// Act
	result, err := r.Verify(otherPath)

	// Assert
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.PathMismatch != true {
		t.Error("PathMismatch should be true")
	}
	if result.FingerprintMismatch != false {
		t.Error("FingerprintMismatch should be false (same content)")
	}
	// IsValid should be true because fingerprint matches
	if !result.IsValid() {
		t.Error("IsValid should be true when only path mismatches")
	}
}

func TestVerifyFingerprintMismatch(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(binaryPath, []byte("original"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r, _ := New(SourceScript, binaryPath)

	// Modify the binary
	if err := os.WriteFile(binaryPath, []byte("modified"), 0755); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Act
	result, err := r.Verify(binaryPath)

	// Assert
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.FingerprintMismatch != true {
		t.Error("FingerprintMismatch should be true")
	}
	if result.IsValid() {
		t.Error("IsValid should be false when fingerprint mismatches")
	}
}

func TestVerifySymlink(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpaca")
	symlinkPath := filepath.Join(tmpDir, "alpaca-link")

	if err := os.WriteFile(binaryPath, []byte("binary content"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.Symlink(binaryPath, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create receipt with symlink path
	r, _ := New(SourceScript, symlinkPath)

	// Act - verify with original path (symlinks should resolve to same)
	result, err := r.Verify(binaryPath)

	// Assert
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !result.IsValid() {
		t.Error("Verify should succeed for symlinked paths")
	}
}

func TestUpdateFingerprint(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(binaryPath, []byte("original"), 0755); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r, _ := New(SourceScript, binaryPath)
	originalFingerprint := r.BinaryFingerprint
	originalTime := r.InstalledAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Modify the binary
	newBinaryPath := filepath.Join(tmpDir, "alpaca-new")
	if err := os.WriteFile(newBinaryPath, []byte("new version"), 0755); err != nil {
		t.Fatalf("failed to create new binary: %v", err)
	}

	// Act
	err := r.UpdateFingerprint(newBinaryPath)

	// Assert
	if err != nil {
		t.Fatalf("UpdateFingerprint failed: %v", err)
	}
	if r.BinaryFingerprint == originalFingerprint {
		t.Error("fingerprint should have changed")
	}
	if r.BinaryPath != newBinaryPath {
		t.Errorf("BinaryPath = %s, want %s", r.BinaryPath, newBinaryPath)
	}
	if !r.InstalledAt.After(originalTime) {
		t.Error("InstalledAt should be updated")
	}
}

func TestVerifyResultIsValid(t *testing.T) {
	tests := []struct {
		name                string
		pathMismatch        bool
		fingerprintMismatch bool
		wantValid           bool
	}{
		{"both match", false, false, true},
		{"path mismatch only", true, false, true},
		{"fingerprint mismatch only", false, true, false},
		{"both mismatch", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyResult{
				PathMismatch:        tt.pathMismatch,
				FingerprintMismatch: tt.fingerprintMismatch,
			}

			if got := result.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}
