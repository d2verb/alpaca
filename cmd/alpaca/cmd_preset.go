package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/ui"
)

type PresetCmd struct {
	Show PresetShowCmd `cmd:"" help:"Show preset details"`
	New  PresetNewCmd  `cmd:"" help:"Create a new preset interactively"`
}

type PresetShowCmd struct {
	Name string `arg:"" help:"Preset name to show"`
}

func (c *PresetShowCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	loader := preset.NewLoader(paths.Presets)
	p, err := loader.Load(c.Name)
	if err != nil {
		return errPresetNotFound(c.Name)
	}

	ui.PrintPresetDetails(ui.PresetDetails{
		Name:        p.Name,
		Model:       p.Model,
		ContextSize: p.ContextSize,
		GPULayers:   p.GPULayers,
		Threads:     p.Threads,
		Host:        p.GetHost(),
		Port:        p.GetPort(),
		ExtraArgs:   p.ExtraArgs,
	})

	return nil
}

type PresetNewCmd struct{}

func (c *PresetNewCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	// Prompt for name
	name, err := promptLine("Preset name", "")
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("preset name is required")
	}

	// Check if preset already exists
	presetPath := filepath.Join(paths.Presets, name+".yaml")
	if _, err := os.Stat(presetPath); err == nil {
		return fmt.Errorf("preset '%s' already exists", name)
	}

	// Prompt for model
	model, err := promptLine("Model (e.g., h:org/repo:Q4_K_M or f:/path/to/model.gguf)", "")
	if err != nil {
		return err
	}
	if model == "" {
		return fmt.Errorf("model is required")
	}

	// Prompt for optional fields
	ctxStr, _ := promptLine("Context size (default: 2048)", "")
	gpuStr, _ := promptLine("GPU layers (default: -1, all)", "")

	// Build preset content
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("model: %s\n", model))
	if ctx, err := strconv.Atoi(ctxStr); err == nil && ctx > 0 {
		sb.WriteString(fmt.Sprintf("context_size: %d\n", ctx))
	}
	if gpu, err := strconv.Atoi(gpuStr); err == nil {
		sb.WriteString(fmt.Sprintf("gpu_layers: %d\n", gpu))
	}

	// Write file
	if err := os.WriteFile(presetPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write preset: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Preset '%s' created", name))
	return nil
}
