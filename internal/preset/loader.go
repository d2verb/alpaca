package preset

import (
	"crypto/rand"
	"encoding/hex"
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

// Load loads a preset by name (searches all YAML files for matching name field).
func (l *Loader) Load(name string) (*Preset, error) {
	entries, err := os.ReadDir(l.presetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("preset %s not found", name)
		}
		return nil, fmt.Errorf("read presets dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		preset, err := l.loadFile(filepath.Join(l.presetsDir, entry.Name()))
		if err != nil {
			continue // Skip invalid files
		}

		if preset.Name == name {
			return preset, nil
		}
	}

	return nil, fmt.Errorf("preset %s not found", name)
}

// loadFile loads a preset from a specific file path.
func (l *Loader) loadFile(path string) (*Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var preset Preset
	if err := yaml.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	if err := ValidateName(preset.Name); err != nil {
		return nil, fmt.Errorf("invalid preset: %w", err)
	}

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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		preset, err := l.loadFile(filepath.Join(l.presetsDir, entry.Name()))
		if err != nil {
			continue // Skip invalid files
		}

		names = append(names, preset.Name)
	}
	return names, nil
}

// Exists checks if a preset with the given name exists.
func (l *Loader) Exists(name string) (bool, error) {
	_, err := l.Load(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create creates a new preset file with a random filename.
func (l *Loader) Create(p *Preset) error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(l.presetsDir, 0755); err != nil {
		return fmt.Errorf("create presets dir: %w", err)
	}

	// Check if name already exists
	exists, err := l.Exists(p.Name)
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if exists {
		return fmt.Errorf("preset '%s' already exists", p.Name)
	}

	// Generate random filename
	filename, err := generateFilename()
	if err != nil {
		return fmt.Errorf("generate filename: %w", err)
	}

	path := filepath.Join(l.presetsDir, filename+".yaml")

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write preset: %w", err)
	}

	return nil
}

// Remove removes a preset by name.
func (l *Loader) Remove(name string) error {
	entries, err := os.ReadDir(l.presetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("preset %s not found", name)
		}
		return fmt.Errorf("read presets dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(l.presetsDir, entry.Name())
		preset, err := l.loadFile(path)
		if err != nil {
			continue
		}

		if preset.Name == name {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove preset: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("preset %s not found", name)
}

// generateFilename generates a random filename (16 hex characters).
func generateFilename() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
