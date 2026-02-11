package preset

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/d2verb/alpaca/internal/pathutil"
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
	_, p, err := l.findByName(name)
	return p, err
}

// FindPath returns the file path of a preset by name.
func (l *Loader) FindPath(name string) (string, error) {
	path, _, err := l.findByName(name)
	return path, err
}

// iteratePresets reads all preset YAML files and calls fn for each successfully parsed preset.
// fn receives the file path and parsed preset; return true to stop iteration early.
// Returns accumulated parse errors for files that failed to parse.
func (l *Loader) iteratePresets(fn func(path string, p *Preset) bool) []*ParseError {
	entries, err := os.ReadDir(l.presetsDir)
	if err != nil {
		return nil
	}
	var parseErrors []*ParseError
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(l.presetsDir, entry.Name())
		p, err := loadFromPath(path)
		if err != nil {
			parseErrors = append(parseErrors, &ParseError{File: entry.Name(), Err: err})
			continue
		}
		if fn(path, p) {
			return parseErrors
		}
	}
	return parseErrors
}

// List returns all available preset names.
// If some preset files fail to parse, they are skipped but a warning is included
// in the error (the list is still returned).
func (l *Loader) List() ([]string, error) {
	entries, err := os.ReadDir(l.presetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read presets dir: %w", err)
	}
	_ = entries // directory exists; proceed with iteration

	var names []string
	parseErrors := l.iteratePresets(func(_ string, p *Preset) bool {
		names = append(names, p.Name)
		return false
	})

	if len(parseErrors) > 0 {
		return names, fmt.Errorf("%d preset file(s) had parse errors (first: %v)", len(parseErrors), parseErrors[0])
	}
	return names, nil
}

// Exists checks if a preset with the given name exists.
func (l *Loader) Exists(name string) (bool, error) {
	_, err := l.Load(name)
	if err != nil {
		if IsNotFound(err) {
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
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid preset: %w", err)
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
		return &AlreadyExistsError{Name: p.Name}
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
	path, _, err := l.findByName(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove preset: %w", err)
	}
	return nil
}

// findByName searches for a preset by name and returns its path and preset.
// If multiple presets have the same name, the first one found (in directory order)
// is returned. This is intentional - users should avoid duplicate names.
// If not found, returns informative error including any parse failures.
func (l *Loader) findByName(name string) (string, *Preset, error) {
	entries, err := os.ReadDir(l.presetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, &NotFoundError{Name: name}
		}
		return "", nil, fmt.Errorf("read presets dir: %w", err)
	}
	_ = entries // directory exists; proceed with iteration

	var foundPath string
	var foundPreset *Preset
	parseErrors := l.iteratePresets(func(path string, p *Preset) bool {
		if p.Name == name {
			foundPath = path
			foundPreset = p
			return true
		}
		return false
	})

	if foundPreset != nil {
		return foundPath, foundPreset, nil
	}

	if len(parseErrors) > 0 {
		return "", nil, fmt.Errorf("preset '%s' not found; %d file(s) had parse errors (first: %v)", name, len(parseErrors), parseErrors[0])
	}
	return "", nil, &NotFoundError{Name: name}
}

// generateFilename generates a random filename (16 hex characters).
func generateFilename() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// LoadFile loads a preset from an explicit file path.
// Relative paths in the model field (f:./ or f:../) are resolved
// relative to the preset file's directory.
func LoadFile(filePath string) (*Preset, error) {
	// Resolve tilde and relative paths
	resolvedPath, err := pathutil.ResolvePath(filePath, "")
	if err != nil {
		return nil, fmt.Errorf("resolve preset path: %w", err)
	}

	// Get absolute path for the preset file
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("resolve preset path: %w", err)
	}

	return loadFromPath(absPath)
}

// loadFromPath loads a preset from an absolute file path.
func loadFromPath(absPath string) (*Preset, error) {
	data, err := os.ReadFile(absPath)
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

	if err := preset.Validate(); err != nil {
		return nil, fmt.Errorf("invalid preset: %w", err)
	}

	baseDir := filepath.Dir(absPath)

	if preset.IsRouter() {
		if err := resolveRouterModelPaths(&preset, baseDir); err != nil {
			return nil, err
		}
	} else {
		if err := resolveSingleModelPaths(&preset, baseDir); err != nil {
			return nil, err
		}
	}

	return &preset, nil
}

// resolveSingleModelPaths resolves model paths for single mode presets.
func resolveSingleModelPaths(preset *Preset, baseDir string) error {
	resolvedModel, err := resolveModelPath(preset.Model, baseDir)
	if err != nil {
		return fmt.Errorf("resolve model path: %w", err)
	}
	preset.Model = resolvedModel

	if preset.DraftModel != "" {
		resolvedDraft, err := resolveModelPath(preset.DraftModel, baseDir)
		if err != nil {
			return fmt.Errorf("resolve draft model path: %w", err)
		}
		preset.DraftModel = resolvedDraft
	}

	resolvedMmproj, err := resolveMmprojPath(preset.Mmproj, baseDir)
	if err != nil {
		return fmt.Errorf("resolve mmproj path: %w", err)
	}
	preset.Mmproj = resolvedMmproj

	return nil
}

// resolveRouterModelPaths resolves model paths for all models in router mode.
func resolveRouterModelPaths(preset *Preset, baseDir string) error {
	for i := range preset.Models {
		m := &preset.Models[i]

		resolvedModel, err := resolveModelPath(m.Model, baseDir)
		if err != nil {
			return fmt.Errorf("resolve model path for '%s': %w", m.Name, err)
		}
		m.Model = resolvedModel

		if m.DraftModel != "" {
			resolvedDraft, err := resolveModelPath(m.DraftModel, baseDir)
			if err != nil {
				return fmt.Errorf("resolve draft model path for '%s': %w", m.Name, err)
			}
			m.DraftModel = resolvedDraft
		}

		resolvedMmproj, err := resolveMmprojPath(m.Mmproj, baseDir)
		if err != nil {
			return fmt.Errorf("resolve mmproj path for '%s': %w", m.Name, err)
		}
		m.Mmproj = resolvedMmproj
	}

	return nil
}

// WriteFile writes a preset to the specified file path.
func WriteFile(path string, p *Preset) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// resolveMmprojPath resolves the mmproj path in a preset.
// - Empty string → return empty (auto-resolve happens in daemon)
// - "none" → return "none" as-is
// - "f:" prefix → expand path using pathutil.ResolvePath
// - Other values should have been caught by validation, but return error as safety net
func resolveMmprojPath(mmproj, baseDir string) (string, error) {
	if mmproj == "" {
		return "", nil
	}
	if mmproj == "none" {
		return "none", nil
	}
	if strings.HasPrefix(mmproj, "f:") {
		path := mmproj[2:]
		resolved, err := pathutil.ResolvePath(path, baseDir)
		if err != nil {
			return "", err
		}
		return "f:" + resolved, nil
	}
	return "", fmt.Errorf("invalid mmproj value: %q", mmproj)
}

// resolveModelPath resolves the model path in a preset.
// - h: prefixed paths are returned as-is (HuggingFace identifiers)
// - f: prefixed paths have relative paths resolved from baseDir
func resolveModelPath(model, baseDir string) (string, error) {
	if model == "" {
		return "", fmt.Errorf("model field is required")
	}

	// HuggingFace identifiers are returned as-is
	if strings.HasPrefix(model, "h:") {
		return model, nil
	}

	// File paths must have f: prefix
	if !strings.HasPrefix(model, "f:") {
		return "", fmt.Errorf("model must have h: or f: prefix, got %q", model)
	}

	// Extract path after f: prefix
	path := model[2:]

	// Resolve path (handles tilde expansion and relative paths)
	resolved, err := pathutil.ResolvePath(path, baseDir)
	if err != nil {
		return "", err
	}

	return "f:" + resolved, nil
}
