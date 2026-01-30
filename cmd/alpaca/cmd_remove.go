package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/model"
)

type RemoveCmd struct {
	Identifier string `arg:"" help:"Identifier to remove (p:name or h:org/repo:quant)"`
}

func (c *RemoveCmd) Run() error {
	id, err := identifier.Parse(c.Identifier)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	switch id.Type {
	case identifier.TypePresetName:
		return c.removePreset(id.PresetName, paths.Presets)

	case identifier.TypeHuggingFace:
		return c.removeModel(id, paths.Models)

	case identifier.TypeFilePath:
		return fmt.Errorf("file paths (f:) cannot be removed\nUse: alpaca rm p:preset-name or alpaca rm h:org/repo:quant")

	default:
		return fmt.Errorf("unknown identifier type")
	}
}

func (c *RemoveCmd) removePreset(name, presetsDir string) error {
	presetPath := filepath.Join(presetsDir, name+".yaml")

	// Check if preset exists
	if _, err := os.Stat(presetPath); os.IsNotExist(err) {
		return errPresetNotFound(name)
	}

	// Confirmation prompt
	if !promptConfirm(fmt.Sprintf("Delete preset '%s'?", name)) {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete file
	if err := os.Remove(presetPath); err != nil {
		return fmt.Errorf("remove preset: %w", err)
	}

	fmt.Printf("Preset '%s' removed.\n", name)
	return nil
}

func (c *RemoveCmd) removeModel(id *identifier.Identifier, modelsDir string) error {
	modelMgr := model.NewManager(modelsDir)
	ctx := context.Background()

	// Check if model exists
	exists, err := modelMgr.Exists(ctx, id.Repo, id.Quant)
	if err != nil {
		return fmt.Errorf("check model: %w", err)
	}
	if !exists {
		return errModelNotFound(fmt.Sprintf("h:%s:%s", id.Repo, id.Quant))
	}

	// Confirmation prompt
	if !promptConfirm(fmt.Sprintf("Delete model 'h:%s:%s'?", id.Repo, id.Quant)) {
		fmt.Println("Cancelled.")
		return nil
	}

	// Remove model
	if err := modelMgr.Remove(ctx, id.Repo, id.Quant); err != nil {
		return fmt.Errorf("remove model: %w", err)
	}

	fmt.Printf("Model 'h:%s:%s' removed.\n", id.Repo, id.Quant)
	return nil
}
