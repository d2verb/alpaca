package receipt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	SchemaVersion = 1
	FileName      = "install.json"
)

// InstallSource represents how alpaca was installed.
type InstallSource string

const (
	SourceScript  InstallSource = "script"
	SourceBrew    InstallSource = "brew"
	SourceApt     InstallSource = "apt"
	SourceGo      InstallSource = "go"
	SourceUnknown InstallSource = "unknown"
)

// Receipt contains installation metadata.
type Receipt struct {
	SchemaVersion     int           `json:"schema_version"`
	InstallSource     InstallSource `json:"install_source"`
	InstalledAt       time.Time     `json:"installed_at"`
	BinaryPath        string        `json:"binary_path"`
	BinaryFingerprint string        `json:"binary_fingerprint"`
}

// Errors
var (
	ErrNotFound            = errors.New("receipt not found")
	ErrFingerprintMismatch = errors.New("binary fingerprint does not match")
	ErrPathMismatch        = errors.New("binary path does not match")
)

// DefaultPath returns the default receipt file path (~/.alpaca/install.json).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".alpaca", FileName), nil
}

// Load reads the receipt from the default location.
func Load() (*Receipt, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads the receipt from the specified path.
func LoadFrom(path string) (*Receipt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read receipt file: %w", err)
	}

	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse receipt: %w", err)
	}

	return &r, nil
}

// Save writes the receipt to the default location.
func (r *Receipt) Save() error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return r.SaveTo(path)
}

// SaveTo writes the receipt to the specified path.
func (r *Receipt) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create receipt directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal receipt: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write receipt file: %w", err)
	}

	return nil
}

// New creates a new receipt for a fresh installation.
func New(source InstallSource, binaryPath string) (*Receipt, error) {
	fingerprint, err := ComputeFingerprint(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("compute fingerprint: %w", err)
	}

	return &Receipt{
		SchemaVersion:     SchemaVersion,
		InstallSource:     source,
		InstalledAt:       time.Now().UTC(),
		BinaryPath:        binaryPath,
		BinaryFingerprint: fingerprint,
	}, nil
}

// ComputeFingerprint calculates the SHA256 hash of a file.
func ComputeFingerprint(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyResult contains the result of receipt verification.
type VerifyResult struct {
	PathMismatch        bool
	FingerprintMismatch bool
}

// IsValid returns true if the fingerprint matches (path mismatch is just a warning).
func (v VerifyResult) IsValid() bool {
	return !v.FingerprintMismatch
}

// Verify checks if the receipt matches the current binary.
// Returns a VerifyResult indicating what matched/mismatched.
// Path mismatch alone is not a failure if fingerprint matches (e.g., symlinks).
func (r *Receipt) Verify(currentBinaryPath string) (VerifyResult, error) {
	result := VerifyResult{}

	// Normalize paths to handle symlinks
	normalizedCurrent, err := filepath.EvalSymlinks(currentBinaryPath)
	if err != nil {
		// If we can't resolve symlinks, use the original path
		normalizedCurrent = currentBinaryPath
	}
	normalizedReceipt, err := filepath.EvalSymlinks(r.BinaryPath)
	if err != nil {
		// If receipt path doesn't exist or can't be resolved, use original
		normalizedReceipt = r.BinaryPath
	}

	// Check if the binary path matches
	if normalizedReceipt != normalizedCurrent {
		result.PathMismatch = true
	}

	// Compute current fingerprint and compare
	currentFingerprint, err := ComputeFingerprint(currentBinaryPath)
	if err != nil {
		return result, fmt.Errorf("compute current fingerprint: %w", err)
	}

	if r.BinaryFingerprint != currentFingerprint {
		result.FingerprintMismatch = true
	}

	return result, nil
}

// UpdateFingerprint updates the receipt with a new binary's fingerprint.
func (r *Receipt) UpdateFingerprint(binaryPath string) error {
	fingerprint, err := ComputeFingerprint(binaryPath)
	if err != nil {
		return fmt.Errorf("compute fingerprint: %w", err)
	}

	r.BinaryPath = binaryPath
	r.BinaryFingerprint = fingerprint
	r.InstalledAt = time.Now().UTC()

	return nil
}
