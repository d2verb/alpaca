package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/d2verb/alpaca/internal/editor"
	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/pathutil"
	"github.com/d2verb/alpaca/internal/preset"
)

type EditCmd struct {
	Identifier string `arg:"" optional:"" help:"Preset to edit (p:name or f:path/to/preset.yaml)" predictor:"edit-identifier"`
}

func (c *EditCmd) Run() error {
	// Resolve identifier (handles empty input → .alpaca.yaml)
	idStr, err := resolveLocalPreset(c.Identifier)
	if err != nil {
		return err
	}

	// Parse identifier
	id, err := identifier.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	// Resolve to file path
	filePath, err := c.resolveFilePath(id)
	if err != nil {
		return err
	}

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("check file: %w", err)
	}

	// Find and open editor
	ed, err := editor.Find()
	if err != nil {
		return err
	}

	return editor.Open(ed, filePath)
}

// resolveFilePath resolves the identifier to an absolute file path.
func (c *EditCmd) resolveFilePath(id *identifier.Identifier) (string, error) {
	switch id.Type {
	case identifier.TypePresetName:
		paths, err := getPaths()
		if err != nil {
			return "", err
		}
		loader := preset.NewLoader(paths.Presets)
		path, err := loader.FindPath(id.PresetName)
		if err != nil {
			return "", mapPresetError(err, id.PresetName)
		}
		return path, nil

	case identifier.TypePresetFilePath:
		resolved, err := pathutil.ResolvePath(id.FilePath, "")
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
		absPath, err := filepath.Abs(resolved)
		if err != nil {
			return "", fmt.Errorf("make absolute path: %w", err)
		}
		return absPath, nil

	case identifier.TypeHuggingFace, identifier.TypeModelFilePath:
		return "", fmt.Errorf("cannot edit model files\nUse: alpaca edit p:name or alpaca edit f:path/to/preset.yaml")

	default:
		return "", fmt.Errorf("unknown identifier type")
	}
}
