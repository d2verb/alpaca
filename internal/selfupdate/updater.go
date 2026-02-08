package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	repoOwner            = "d2verb"
	repoName             = "alpaca"
	apiTimeout           = 30 * time.Second
	defaultGitHubBaseURL = "https://api.github.com"

	// releasePublicKey is the hex-encoded Ed25519 public key used to verify release signatures.
	releasePublicKey = "394cf6c1b1ca12afda257e5338c7e6aa10d811c3054b5c16e341d0e97f3246e5"
)

// Updater handles self-update operations.
type Updater struct {
	currentVersion string
	client         *http.Client
	baseURL        string
	publicKey      ed25519.PublicKey // overridable for testing
}

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// New creates a new Updater.
func New(currentVersion string) *Updater {
	pubKey, err := hex.DecodeString(releasePublicKey)
	if err != nil {
		// This should never happen with a valid embedded key.
		panic(fmt.Sprintf("invalid embedded public key: %v", err))
	}

	return &Updater{
		currentVersion: currentVersion,
		client: &http.Client{
			Timeout: apiTimeout,
		},
		baseURL:   defaultGitHubBaseURL,
		publicKey: ed25519.PublicKey(pubKey),
	}
}

// CheckUpdate checks if a newer version is available.
// Returns the latest version, whether an update is available, and any error.
func (u *Updater) CheckUpdate() (string, bool, error) {
	release, err := u.getLatestRelease()
	if err != nil {
		return "", false, err
	}

	// Ensure versions have 'v' prefix for semver comparison
	latestVersion := ensureVPrefix(release.TagName)
	currentVersion := ensureVPrefix(u.currentVersion)

	// Use proper semver comparison
	// semver.Compare returns:
	//   -1 if current < latest
	//    0 if current == latest
	//   +1 if current > latest
	hasUpdate := semver.IsValid(latestVersion) &&
		semver.IsValid(currentVersion) &&
		semver.Compare(currentVersion, latestVersion) < 0

	return release.TagName, hasUpdate, nil
}

// ensureVPrefix ensures the version string has a 'v' prefix for semver.
func ensureVPrefix(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

// Update downloads and installs the latest version.
func (u *Updater) Update(currentBinaryPath string) error {
	release, err := u.getLatestRelease()
	if err != nil {
		return fmt.Errorf("get latest release: %w", err)
	}

	// Find the appropriate asset for this platform
	assetName := u.getAssetName(release.TagName)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download checksums (raw bytes for signature verification)
	checksumsRaw, err := u.downloadChecksumsRaw(release)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	// Download and verify signature (fail-closed: reject if signature is missing or invalid)
	sig, err := u.downloadSignature(release)
	if err != nil {
		return fmt.Errorf("download signature: %w", err)
	}
	if err := u.verifySignature(checksumsRaw, sig); err != nil {
		return fmt.Errorf("verify checksums signature: %w", err)
	}

	checksums := parseChecksums(checksumsRaw)
	expectedChecksum, ok := checksums[assetName]
	if !ok {
		return fmt.Errorf("checksum not found for %s", assetName)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "alpaca-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download the archive
	archivePath := filepath.Join(tmpDir, assetName)
	if err := u.downloadFile(downloadURL, archivePath); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	// Verify checksum
	if err := u.verifyChecksum(archivePath, expectedChecksum); err != nil {
		return fmt.Errorf("verify checksum: %w", err)
	}

	// Extract the binary
	newBinaryPath := filepath.Join(tmpDir, "alpaca")
	if err := u.extractBinary(archivePath, newBinaryPath); err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	// Replace the current binary atomically
	if err := u.replaceBinary(currentBinaryPath, newBinaryPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}

func (u *Updater) getLatestRelease() (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", u.baseURL, repoOwner, repoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("GitHub API rate limit exceeded. Try again later or use --force")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parse release: %w", err)
	}

	return &release, nil
}

func (u *Updater) getAssetName(tagName string) string {
	version := strings.TrimPrefix(tagName, "v")
	return fmt.Sprintf("alpaca_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
}

func (u *Updater) downloadChecksumsRaw(release *Release) ([]byte, error) {
	var checksumURL string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			checksumURL = asset.BrowserDownloadURL
			break
		}
	}
	if checksumURL == "" {
		return nil, fmt.Errorf("checksums.txt not found in release")
	}

	resp, err := u.client.Get(checksumURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksums: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func parseChecksums(data []byte) map[string]string {
	checksums := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			checksums[parts[1]] = parts[0]
		}
	}
	return checksums
}

func (u *Updater) downloadSignature(release *Release) ([]byte, error) {
	var sigURL string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt.sig" {
			sigURL = asset.BrowserDownloadURL
			break
		}
	}
	if sigURL == "" {
		return nil, fmt.Errorf("checksums.txt.sig not found in release")
	}

	resp, err := u.client.Get(sigURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download signature: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (u *Updater) verifySignature(data, signature []byte) error {
	if !ed25519.Verify(u.publicKey, data, signature) {
		return fmt.Errorf("Ed25519 signature verification failed")
	}
	return nil
}

func (u *Updater) downloadFile(url, destPath string) error {
	resp, err := u.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func (u *Updater) verifyChecksum(filePath, expectedChecksum string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(h.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func (u *Updater) extractBinary(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the alpaca binary
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "alpaca" {
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("alpaca binary not found in archive")
}

func (u *Updater) replaceBinary(currentPath, newPath string) error {
	// Get the directory of the current binary
	dir := filepath.Dir(currentPath)

	// Create a temp file in the same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, ".alpaca-new-*")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied. Try: sudo alpaca upgrade")
		}
		return err
	}
	tmpPath := tmpFile.Name()

	// Copy new binary to temp location
	src, err := os.Open(newPath)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	defer src.Close()

	if _, err := io.Copy(tmpFile, src); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	tmpFile.Close()

	// Set executable permissions
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpPath, currentPath); err != nil {
		os.Remove(tmpPath)
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied. Try: sudo alpaca upgrade")
		}
		return err
	}

	return nil
}
