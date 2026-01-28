// Package config handles Alpaca configuration.
package config

import (
	"os"
	"path/filepath"
)

const (
	// DefaultPort is the default port for llama-server.
	DefaultPort = 8080
	// DefaultHost is the default host for llama-server.
	DefaultHost = "127.0.0.1"
)

// Config represents the global Alpaca configuration.
type Config struct {
	LlamaServerPath string `yaml:"llama_server_path"`
	DefaultPort     int    `yaml:"default_port"`
	DefaultHost     string `yaml:"default_host"`
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		LlamaServerPath: "llama-server",
		DefaultPort:     DefaultPort,
		DefaultHost:     DefaultHost,
	}
}

// Paths returns common paths used by Alpaca.
type Paths struct {
	Home      string
	Config    string
	Socket    string
	PID       string
	Presets   string
	Models    string
	Logs      string
	DaemonLog string
	LlamaLog  string
}

// GetPaths returns the paths for the current user.
func GetPaths() (*Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	alpacaHome := filepath.Join(home, ".alpaca")
	logsDir := filepath.Join(alpacaHome, "logs")
	return &Paths{
		Home:      alpacaHome,
		Config:    filepath.Join(alpacaHome, "config.yaml"),
		Socket:    filepath.Join(alpacaHome, "alpaca.sock"),
		PID:       filepath.Join(alpacaHome, "alpaca.pid"),
		Presets:   filepath.Join(alpacaHome, "presets"),
		Models:    filepath.Join(alpacaHome, "models"),
		Logs:      logsDir,
		DaemonLog: filepath.Join(logsDir, "daemon.log"),
		LlamaLog:  filepath.Join(logsDir, "llama.log"),
	}, nil
}

// EnsureDirectories creates the required directories if they don't exist.
func (p *Paths) EnsureDirectories() error {
	dirs := []string{p.Home, p.Presets, p.Models, p.Logs}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
