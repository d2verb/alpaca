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
	if name == "" {
		return fmt.Errorf("preset name is required")
	}

	// Check if preset already exists
	presetPath := filepath.Join(paths.Presets, name+".yaml")
	if _, err := os.Stat(presetPath); err == nil {
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

	// Build preset content
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("model: %s\n", model))

	// Only write host if different from default
	if hostStr != "" && hostStr != preset.DefaultHost {
		sb.WriteString(fmt.Sprintf("host: %s\n", hostStr))
	}

	// Only write port if different from default
	if port, err := strconv.Atoi(portStr); err == nil && port != preset.DefaultPort {
		sb.WriteString(fmt.Sprintf("port: %d\n", port))
	}

	// Only write context_size if different from default
	if ctx, err := strconv.Atoi(ctxStr); err == nil && ctx != preset.DefaultContextSize {
		sb.WriteString(fmt.Sprintf("context_size: %d\n", ctx))
	}

	// Only write gpu_layers if different from default
	if gpu, err := strconv.Atoi(gpuStr); err == nil && gpu != preset.DefaultGPULayers {
		sb.WriteString(fmt.Sprintf("gpu_layers: %d\n", gpu))
	}

	// Write file
	if err := os.WriteFile(presetPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write preset: %w", err)
	}

	// Success message with next steps
	ui.PrintSuccess(fmt.Sprintf("Created '%s'", name))
	fmt.Fprintf(ui.Output, "%s %s\n", ui.Info("💡"), ui.Info(fmt.Sprintf("alpaca load p:%s", name)))
	return nil
}
