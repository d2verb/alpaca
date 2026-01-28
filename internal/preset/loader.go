package preset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles loading presets from disk.
type Loader struct {
	presetsDir string
}

// NewLoader creates a new preset loader.
func NewLoader(presetsDir string) *Loader {
	return &Loader{presetsDir: presetsDir}
}

// Load loads a preset by name.
func (l *Loader) Load(name string) (*Preset, error) {
	path := filepath.Join(l.presetsDir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load preset %s: %w", name, err)
	}

	var preset Preset
	if err := yaml.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("parse preset %s: %w", name, err)
	}
	preset.Name = name

	// Expand ~ in model path
	if strings.HasPrefix(preset.Model, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("expand home dir: %w", err)
		}
		preset.Model = filepath.Join(home, preset.Model[2:])
	}

	return &preset, nil
}

// List returns all available preset names.
func (l *Loader) List() ([]string, error) {
	entries, err := os.ReadDir(l.presetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("list presets: %w", err)
	}

	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") {
			names = append(names, strings.TrimSuffix(name, ".yaml"))
		}
	}
	return names, nil
}
