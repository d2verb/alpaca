package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LlamaServerPath != "llama-server" {
		t.Errorf("LlamaServerPath = %q, want %q", cfg.LlamaServerPath, "llama-server")
	}
	if cfg.DefaultPort != DefaultPort {
		t.Errorf("DefaultPort = %d, want %d", cfg.DefaultPort, DefaultPort)
	}
	if cfg.DefaultHost != DefaultHost {
		t.Errorf("DefaultHost = %q, want %q", cfg.DefaultHost, DefaultHost)
	}
}

func TestGetPaths(t *testing.T) {
	paths, err := GetPaths()
	if err != nil {
		t.Fatalf("GetPaths() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	alpacaHome := filepath.Join(home, ".alpaca")
	logsDir := filepath.Join(alpacaHome, "logs")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Home", paths.Home, alpacaHome},
		{"Config", paths.Config, filepath.Join(alpacaHome, "config.yaml")},
		{"Socket", paths.Socket, filepath.Join(alpacaHome, "alpaca.sock")},
		{"PID", paths.PID, filepath.Join(alpacaHome, "alpaca.pid")},
		{"Presets", paths.Presets, filepath.Join(alpacaHome, "presets")},
		{"Models", paths.Models, filepath.Join(alpacaHome, "models")},
		{"Logs", paths.Logs, logsDir},
		{"DaemonLog", paths.DaemonLog, filepath.Join(logsDir, "daemon.log")},
		{"LlamaLog", paths.LlamaLog, filepath.Join(logsDir, "llama.log")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestGetPaths_ContainsAlpacaDir(t *testing.T) {
	paths, err := GetPaths()
	if err != nil {
		t.Fatalf("GetPaths() error = %v", err)
	}

	// All paths should be under .alpaca
	if !strings.Contains(paths.Home, ".alpaca") {
		t.Errorf("Home should contain .alpaca: %q", paths.Home)
	}
	if !strings.HasPrefix(paths.Config, paths.Home) {
		t.Errorf("Config should be under Home: %q", paths.Config)
	}
	if !strings.HasPrefix(paths.Socket, paths.Home) {
		t.Errorf("Socket should be under Home: %q", paths.Socket)
	}
	if !strings.HasPrefix(paths.Presets, paths.Home) {
		t.Errorf("Presets should be under Home: %q", paths.Presets)
	}
	if !strings.HasPrefix(paths.Models, paths.Home) {
		t.Errorf("Models should be under Home: %q", paths.Models)
	}
	if !strings.HasPrefix(paths.Logs, paths.Home) {
		t.Errorf("Logs should be under Home: %q", paths.Logs)
	}
	if !strings.HasPrefix(paths.DaemonLog, paths.Home) {
		t.Errorf("DaemonLog should be under Home: %q", paths.DaemonLog)
	}
	if !strings.HasPrefix(paths.LlamaLog, paths.Home) {
		t.Errorf("LlamaLog should be under Home: %q", paths.LlamaLog)
	}
}

func TestPaths_EnsureDirectories(t *testing.T) {
	// Use temp directory as base
	tmpDir := t.TempDir()
	alpacaHome := filepath.Join(tmpDir, ".alpaca")
	paths := &Paths{
		Home:    alpacaHome,
		Presets: filepath.Join(alpacaHome, "presets"),
		Models:  filepath.Join(alpacaHome, "models"),
		Logs:    filepath.Join(alpacaHome, "logs"),
	}

	// Directories should not exist yet
	if _, err := os.Stat(paths.Home); !os.IsNotExist(err) {
		t.Fatal("Home directory should not exist before EnsureDirectories")
	}

	// Create directories
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	// Verify directories exist
	dirs := []string{paths.Home, paths.Presets, paths.Models, paths.Logs}
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory %q should exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q should be a directory", dir)
		}
	}

	// Calling again should not error (idempotent)
	if err := paths.EnsureDirectories(); err != nil {
		t.Errorf("EnsureDirectories() second call error = %v", err)
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have expected values
	if DefaultPort != 8080 {
		t.Errorf("DefaultPort = %d, want 8080", DefaultPort)
	}
	if DefaultHost != "127.0.0.1" {
		t.Errorf("DefaultHost = %q, want 127.0.0.1", DefaultHost)
	}
}
