// Package config handles Alpaca path configuration.
package config

import (
	"os"
	"path/filepath"
)

// Paths holds common paths used by Alpaca.
type Paths struct {
	Home         string
	Socket       string
	PID          string
	Presets      string
	Models       string
	Logs         string
	DaemonLog    string
	LlamaLog     string
	RouterConfig string
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
		Home:         alpacaHome,
		Socket:       filepath.Join(alpacaHome, "alpaca.sock"),
		PID:          filepath.Join(alpacaHome, "alpaca.pid"),
		Presets:      filepath.Join(alpacaHome, "presets"),
		Models:       filepath.Join(alpacaHome, "models"),
		Logs:         logsDir,
		DaemonLog:    filepath.Join(logsDir, "daemon.log"),
		LlamaLog:     filepath.Join(logsDir, "llama.log"),
		RouterConfig: filepath.Join(alpacaHome, "router-config.ini"),
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
