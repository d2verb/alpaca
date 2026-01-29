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
	List   PresetListCmd `cmd:"" name:"ls" help:"List available presets"`
	Show   PresetShowCmd `cmd:"" help:"Show preset details"`
	New    PresetNewCmd  `cmd:"" help:"Create a new preset interactively"`
	Remove PresetRmCmd   `cmd:"" name:"rm" help:"Remove a preset"`
}

type PresetListCmd struct{}

func (c *PresetListCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	loader := preset.NewLoader(paths.Presets)
	presetNames, err := loader.List()
	if err != nil {
		return fmt.Errorf("list presets: %w", err)
	}

	ui.PrintPresetList(presetNames)

	if len(presetNames) == 0 {
		ui.PrintInfo(fmt.Sprintf("Add presets to: %s", paths.Presets))
	}

	return nil
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

type PresetRmCmd struct {
	Name string `arg:"" help:"Preset name to remove"`
}

func (c *PresetRmCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	presetPath := filepath.Join(paths.Presets, c.Name+".yaml")

	// Check if preset exists
	if _, err := os.Stat(presetPath); os.IsNotExist(err) {
		return errPresetNotFound(c.Name)
	}

	// Confirmation prompt
	if !promptConfirm(fmt.Sprintf("Delete preset '%s'?", c.Name)) {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete file
	if err := os.Remove(presetPath); err != nil {
		return fmt.Errorf("remove preset: %w", err)
	}

	fmt.Printf("Preset '%s' removed.\n", c.Name)
	return nil
}
