package main

import (
	"fmt"
	"strconv"

	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/ui"
)

type NewCmd struct{}

func (c *NewCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	// Print header
	fmt.Fprintf(ui.Output, "📦 %s\n", ui.Heading("Create Preset"))

	// Prompt for name
	name, err := promptLine("Name", "")
	if err != nil {
		return err
	}
	if err := preset.ValidateName(name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Check if preset already exists
	loader := preset.NewLoader(paths.Presets)
	exists, err := loader.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("preset '%s' already exists", name)
	}

	// Prompt for model
	model, err := promptLine("Model", "")
	if err != nil {
		return err
	}
	if model == "" {
		return fmt.Errorf("model is required")
	}

	// Prompt for optional fields with defaults
	hostStr, _ := promptLine("Host", preset.DefaultHost)
	portStr, _ := promptLine("Port", strconv.Itoa(preset.DefaultPort))
	ctxStr, _ := promptLine("Context", strconv.Itoa(preset.DefaultContextSize))
	gpuStr, _ := promptLine("GPU Layers", strconv.Itoa(preset.DefaultGPULayers))

	// Build preset
	p := &preset.Preset{
		Name:  name,
		Model: model,
	}

	// Only set non-default values
	if hostStr != "" && hostStr != preset.DefaultHost {
		p.Host = hostStr
	}
	if port, err := strconv.Atoi(portStr); err == nil && port != preset.DefaultPort {
		p.Port = port
	}
	if ctx, err := strconv.Atoi(ctxStr); err == nil && ctx != preset.DefaultContextSize {
		p.ContextSize = ctx
	}
	if gpu, err := strconv.Atoi(gpuStr); err == nil && gpu != preset.DefaultGPULayers {
		p.GPULayers = gpu
	}

	// Create preset
	if err := loader.Create(p); err != nil {
		return fmt.Errorf("create preset: %w", err)
	}

	// Success message with next steps
	ui.PrintSuccess(fmt.Sprintf("Created '%s'", name))
	fmt.Fprintf(ui.Output, "%s %s\n", ui.Info("💡"), ui.Info(fmt.Sprintf("alpaca load p:%s", name)))
	return nil
}
