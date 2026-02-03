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

type NewCmd struct {
	Local bool `flag:"" help:"Create .alpaca.yaml in current directory"`
}

func (c *NewCmd) Run() error {
	if c.Local {
		return c.runLocal()
	}
	return c.runGlobal()
}

// runGlobal creates a preset in ~/.alpaca/presets/
func (c *NewCmd) runGlobal() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

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

	// Collect preset inputs
	p, err := c.collectInputs(name)
	if err != nil {
		return err
	}

	if err := loader.Create(p); err != nil {
		return fmt.Errorf("create preset: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Created '%s'", name))
	fmt.Fprintf(ui.Output, "%s %s\n", ui.Info("💡"), ui.Info(fmt.Sprintf("alpaca load p:%s", name)))
	return nil
}

// runLocal creates .alpaca.yaml in the current directory
func (c *NewCmd) runLocal() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	path := filepath.Join(cwd, LocalPresetFile)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", LocalPresetFile)
	}

	fmt.Fprintf(ui.Output, "📦 %s\n", ui.Heading("Create Local Preset"))

	// Default name from directory name
	defaultName := preset.SanitizeName(filepath.Base(cwd))
	if defaultName == "" {
		defaultName = "local"
	}

	// Prompt for name
	name, err := promptLine("Name", defaultName)
	if err != nil {
		return err
	}
	if err := preset.ValidateName(name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Collect preset inputs
	p, err := c.collectInputs(name)
	if err != nil {
		return err
	}

	if err := preset.WriteFile(path, p); err != nil {
		return err
	}

	ui.PrintSuccess(fmt.Sprintf("Created '%s'", LocalPresetFile))
	fmt.Fprintf(ui.Output, "%s %s\n", ui.Info("💡"), ui.Info("alpaca load"))
	return nil
}

// collectInputs prompts for model and optional fields, returns a preset.
func (c *NewCmd) collectInputs(name string) (*preset.Preset, error) {
	model, err := promptLine("Model", "")
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if !strings.HasPrefix(model, "h:") && !strings.HasPrefix(model, "f:") {
		return nil, fmt.Errorf("model must have h: or f: prefix\nExamples: h:unsloth/gemma3:Q4_K_M, f:~/models/model.gguf")
	}

	hostStr, err := promptLine("Host", preset.DefaultHost)
	if err != nil {
		return nil, err
	}
	portStr, err := promptLine("Port", strconv.Itoa(preset.DefaultPort))
	if err != nil {
		return nil, err
	}
	ctxStr, err := promptLine("Context", strconv.Itoa(preset.DefaultContextSize))
	if err != nil {
		return nil, err
	}

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

	return p, nil
}
