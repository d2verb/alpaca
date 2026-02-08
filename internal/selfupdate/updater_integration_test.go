package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// newTestUpdater creates an Updater configured for testing with a custom base URL.
func newTestUpdater(version, baseURL string) *Updater {
	u := New(version)
	u.baseURL = baseURL
	return u
}

// createTestTarGz creates a tar.gz archive containing a single file named "alpaca"
func createTestTarGz(content []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	header := &tar.Header{
		Name: "alpaca",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gzw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestUpdate_EndToEnd(t *testing.T) {
	// Arrange
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	newBinaryContent := []byte("#!/bin/sh\necho 'new version'")
	archiveBytes, err := createTestTarGz(newBinaryContent)
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	// Compute checksum and sign
	h := sha256.New()
	h.Write(archiveBytes)
	checksum := hex.EncodeToString(h.Sum(nil))
	assetName := fmt.Sprintf("alpaca_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	checksumsContent := fmt.Sprintf("%s  %s\n", checksum, assetName)
	sig := ed25519.Sign(priv, []byte(checksumsContent))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/latest"):
			release := Release{
				TagName: "v1.2.3",
				Assets: []Asset{
					{Name: assetName, BrowserDownloadURL: "http://" + r.Host + "/" + assetName},
					{Name: "checksums.txt", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt"},
					{Name: "checksums.txt.sig", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt.sig"},
				},
			}
			json.NewEncoder(w).Encode(release)

		case r.URL.Path == "/checksums.txt":
			w.Write([]byte(checksumsContent))

		case r.URL.Path == "/checksums.txt.sig":
			w.Write(sig)

		case r.URL.Path == "/"+assetName:
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(archiveBytes)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	// Create a fake current binary
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(currentBinary, []byte("old version"), 0755); err != nil {
		t.Fatalf("failed to create current binary: %v", err)
	}

	updater := newTestUpdater("v1.0.0", srv.URL)
	updater.publicKey = pub

	// Act
	err = updater.Update(currentBinary)

	// Assert
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify the binary was replaced
	content, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("failed to read updated binary: %v", err)
	}
	if string(content) != string(newBinaryContent) {
		t.Errorf("binary content = %q, want %q", string(content), string(newBinaryContent))
	}

	// Verify executable permissions
	info, err := os.Stat(currentBinary)
	if err != nil {
		t.Fatalf("failed to stat binary: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("binary should be executable")
	}
}

func TestUpdate_WithValidSignature(t *testing.T) {
	// Arrange
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	newBinaryContent := []byte("#!/bin/sh\necho 'signed version'")
	archiveBytes, err := createTestTarGz(newBinaryContent)
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	h := sha256.New()
	h.Write(archiveBytes)
	checksum := hex.EncodeToString(h.Sum(nil))
	assetName := fmt.Sprintf("alpaca_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	checksumsContent := fmt.Sprintf("%s  %s\n", checksum, assetName)
	sig := ed25519.Sign(priv, []byte(checksumsContent))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/latest"):
			release := Release{
				TagName: "v1.2.3",
				Assets: []Asset{
					{Name: assetName, BrowserDownloadURL: "http://" + r.Host + "/" + assetName},
					{Name: "checksums.txt", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt"},
					{Name: "checksums.txt.sig", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt.sig"},
				},
			}
			json.NewEncoder(w).Encode(release)

		case r.URL.Path == "/checksums.txt":
			w.Write([]byte(checksumsContent))

		case r.URL.Path == "/checksums.txt.sig":
			w.Write(sig)

		case r.URL.Path == "/"+assetName:
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(archiveBytes)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(currentBinary, []byte("old version"), 0755); err != nil {
		t.Fatalf("failed to create current binary: %v", err)
	}

	updater := newTestUpdater("v1.0.0", srv.URL)
	updater.publicKey = pub

	// Act
	err = updater.Update(currentBinary)

	// Assert
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	content, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("failed to read updated binary: %v", err)
	}
	if string(content) != string(newBinaryContent) {
		t.Errorf("binary content = %q, want %q", string(content), string(newBinaryContent))
	}
}

func TestUpdate_WithInvalidSignature(t *testing.T) {
	// Arrange
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	_, attackerPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate attacker key: %v", err)
	}

	newBinaryContent := []byte("malicious binary")
	archiveBytes, err := createTestTarGz(newBinaryContent)
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	h := sha256.New()
	h.Write(archiveBytes)
	checksum := hex.EncodeToString(h.Sum(nil))
	assetName := fmt.Sprintf("alpaca_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	checksumsContent := fmt.Sprintf("%s  %s\n", checksum, assetName)
	// Sign with attacker's key (not the expected key)
	badSig := ed25519.Sign(attackerPriv, []byte(checksumsContent))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/latest"):
			release := Release{
				TagName: "v1.2.3",
				Assets: []Asset{
					{Name: assetName, BrowserDownloadURL: "http://" + r.Host + "/" + assetName},
					{Name: "checksums.txt", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt"},
					{Name: "checksums.txt.sig", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt.sig"},
				},
			}
			json.NewEncoder(w).Encode(release)

		case r.URL.Path == "/checksums.txt":
			w.Write([]byte(checksumsContent))

		case r.URL.Path == "/checksums.txt.sig":
			w.Write(badSig)

		case r.URL.Path == "/"+assetName:
			w.Write(archiveBytes)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(currentBinary, []byte("old version"), 0755); err != nil {
		t.Fatalf("failed to create current binary: %v", err)
	}

	updater := newTestUpdater("v1.0.0", srv.URL)
	updater.publicKey = pub

	// Act
	err = updater.Update(currentBinary)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
	if !strings.Contains(err.Error(), "signature") {
		t.Errorf("error = %q, want to contain 'signature'", err.Error())
	}

	// Verify original binary is untouched
	content, _ := os.ReadFile(currentBinary)
	if string(content) != "old version" {
		t.Error("original binary should be untouched after failed update")
	}
}

func TestUpdate_WithoutSignatureFile(t *testing.T) {
	// Arrange - release without checksums.txt.sig
	newBinaryContent := []byte("#!/bin/sh\necho 'unsigned version'")
	archiveBytes, err := createTestTarGz(newBinaryContent)
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	h := sha256.New()
	h.Write(archiveBytes)
	checksum := hex.EncodeToString(h.Sum(nil))
	assetName := fmt.Sprintf("alpaca_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/latest"):
			release := Release{
				TagName: "v1.2.3",
				Assets: []Asset{
					{Name: assetName, BrowserDownloadURL: "http://" + r.Host + "/" + assetName},
					{Name: "checksums.txt", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt"},
					// No checksums.txt.sig asset
				},
			}
			json.NewEncoder(w).Encode(release)

		case r.URL.Path == "/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", checksum, assetName)

		case r.URL.Path == "/"+assetName:
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(archiveBytes)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(currentBinary, []byte("old version"), 0755); err != nil {
		t.Fatalf("failed to create current binary: %v", err)
	}

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act - should fail because signature file is missing (fail-closed)
	err = updater.Update(currentBinary)

	// Assert
	if err == nil {
		t.Fatal("Update() error = nil, want error for missing signature file")
	}
	if !strings.Contains(err.Error(), "download signature") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "download signature")
	}

	// Verify binary was NOT replaced
	content, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("failed to read binary: %v", err)
	}
	if string(content) != "old version" {
		t.Error("binary was replaced despite missing signature")
	}
}

func TestUpdate_ChecksumMismatch(t *testing.T) {
	// Arrange
	newBinaryContent := []byte("new version")
	archiveBytes, err := createTestTarGz(newBinaryContent)
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	assetName := fmt.Sprintf("alpaca_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/latest"):
			release := Release{
				TagName: "v1.2.3",
				Assets: []Asset{
					{Name: assetName, BrowserDownloadURL: "http://" + r.Host + "/" + assetName},
					{Name: "checksums.txt", BrowserDownloadURL: "http://" + r.Host + "/checksums.txt"},
				},
			}
			json.NewEncoder(w).Encode(release)

		case r.URL.Path == "/checksums.txt":
			// Return wrong checksum
			fmt.Fprintf(w, "%s  %s\n", "0000000000000000000000000000000000000000000000000000000000000000", assetName)

		case r.URL.Path == "/"+assetName:
			w.Write(archiveBytes)
		}
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "alpaca")
	if err := os.WriteFile(currentBinary, []byte("old"), 0755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act
	err = updater.Update(currentBinary)

	// Assert
	if err == nil {
		t.Fatal("expected error for checksum mismatch")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("error = %q, want to contain 'checksum'", err.Error())
	}

	// Verify original binary is untouched
	content, _ := os.ReadFile(currentBinary)
	if string(content) != "old" {
		t.Error("original binary should be untouched after failed update")
	}
}

func TestUpdate_AssetNotFound(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return release with no matching asset for this platform
		release := Release{
			TagName: "v1.2.3",
			Assets: []Asset{
				{Name: "alpaca_1.2.3_windows_amd64.tar.gz", BrowserDownloadURL: "http://example.com/windows"},
			},
		}
		json.NewEncoder(w).Encode(release)
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "alpaca")
	os.WriteFile(currentBinary, []byte("old"), 0755)

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act
	err := updater.Update(currentBinary)

	// Assert
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
	if !strings.Contains(err.Error(), "no asset found") {
		t.Errorf("error = %q, want to contain 'no asset found'", err.Error())
	}
}

func TestUpdate_ChecksumFileNotFound(t *testing.T) {
	// Arrange
	assetName := fmt.Sprintf("alpaca_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return release without checksums.txt
		release := Release{
			TagName: "v1.2.3",
			Assets: []Asset{
				{Name: assetName, BrowserDownloadURL: "http://example.com/asset"},
			},
		}
		json.NewEncoder(w).Encode(release)
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "alpaca")
	os.WriteFile(currentBinary, []byte("old"), 0755)

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act
	err := updater.Update(currentBinary)

	// Assert
	if err == nil {
		t.Fatal("expected error for missing checksums")
	}
	if !strings.Contains(err.Error(), "checksums") {
		t.Errorf("error = %q, want to contain 'checksums'", err.Error())
	}
}

func TestCheckUpdate_NewVersionAvailable(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := Release{
			TagName: "v2.0.0",
			Assets:  []Asset{},
		}
		json.NewEncoder(w).Encode(release)
	}))
	t.Cleanup(srv.Close)

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act
	version, hasUpdate, err := updater.CheckUpdate()

	// Assert
	if err != nil {
		t.Fatalf("CheckUpdate() error = %v", err)
	}
	if !hasUpdate {
		t.Error("hasUpdate = false, want true")
	}
	if version != "v2.0.0" {
		t.Errorf("version = %q, want %q", version, "v2.0.0")
	}
}

func TestCheckUpdate_AlreadyUpToDate(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := Release{
			TagName: "v1.0.0",
			Assets:  []Asset{},
		}
		json.NewEncoder(w).Encode(release)
	}))
	t.Cleanup(srv.Close)

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act
	_, hasUpdate, err := updater.CheckUpdate()

	// Assert
	if err != nil {
		t.Fatalf("CheckUpdate() error = %v", err)
	}
	if hasUpdate {
		t.Error("hasUpdate = true, want false (already up to date)")
	}
}

func TestCheckUpdate_RateLimitError(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act
	_, _, err := updater.CheckUpdate()

	// Assert
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error = %q, want to contain 'rate limit'", err.Error())
	}
}

func TestCheckUpdate_ServerError(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	updater := newTestUpdater("v1.0.0", srv.URL)

	// Act
	_, _, err := updater.CheckUpdate()

	// Assert
	if err == nil {
		t.Fatal("expected error for server error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want to contain '500'", err.Error())
	}
}
