package logging

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// Arrange
	path := "/var/log/test.log"

	// Act
	cfg := DefaultConfig(path)

	// Assert
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
	if cfg.MaxSizeMB != 50 {
		t.Errorf("MaxSizeMB = %d, want 50", cfg.MaxSizeMB)
	}
	if cfg.MaxBackups != 3 {
		t.Errorf("MaxBackups = %d, want 3", cfg.MaxBackups)
	}
	if cfg.MaxAgeDays != 7 {
		t.Errorf("MaxAgeDays = %d, want 7", cfg.MaxAgeDays)
	}
	if !cfg.Compress {
		t.Error("Compress = false, want true")
	}
}

func TestNewRotatingWriter(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	cfg := Config{
		Path:       logPath,
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
		Compress:   false,
	}

	// Act
	writer := NewRotatingWriter(cfg)
	defer writer.Close()
	_, err := writer.Write([]byte("test log message\n"))

	// Assert
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
}

func TestNewLogger(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := NewLogger(&buf)

	// Act
	logger.Info("test message", "key", "value")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain 'test message': %q", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Log output should contain 'key=value': %q", output)
	}
	if !strings.Contains(output, "level=INFO") {
		t.Errorf("Log output should contain 'level=INFO': %q", output)
	}
}
