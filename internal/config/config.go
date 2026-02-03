// Package config handles Alpaca configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/d2verb/alpaca/internal/preset"
	"gopkg.in/yaml.v3"
)

// Config represents the global Alpaca configuration.
type Config struct {
	LlamaServerPath string `yaml:"llama_server_path"`
	DefaultPort     int    `yaml:"default_port"`
	DefaultHost     string `yaml:"default_host"`
	DefaultCtxSize  int    `yaml:"default_ctx_size"`
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		LlamaServerPath: "llama-server",
		DefaultPort:     preset.DefaultPort,
		DefaultHost:     preset.DefaultHost,
		DefaultCtxSize:  preset.DefaultContextSize,
	}
}

// LoadConfig loads configuration from the specified path.
// If the file doesn't exist, returns DefaultConfig().
// If the file exists but is partially filled, missing fields use default values.
func LoadConfig(configPath string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Try to read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - use defaults
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Parse YAML, overlaying onto defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
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
