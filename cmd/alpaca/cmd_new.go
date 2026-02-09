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

// collectInputs prompts for mode, then delegates to single or router flow.
func (c *NewCmd) collectInputs(name string) (*preset.Preset, error) {
	modeStr, err := promptLine("Mode (single/router)", "single")
	if err != nil {
		return nil, err
	}

	switch modeStr {
	case "single":
		return c.collectSingleInputs(name)
	case "router":
		return c.collectRouterInputs(name)
	default:
		return nil, fmt.Errorf("invalid mode '%s': must be 'single' or 'router'", modeStr)
	}
}

// collectSingleInputs prompts for model and optional fields for single mode.
func (c *NewCmd) collectSingleInputs(name string) (*preset.Preset, error) {
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

	return p, nil
}

// collectRouterInputs prompts for host, port, and models for router mode.
func (c *NewCmd) collectRouterInputs(name string) (*preset.Preset, error) {
	hostStr, err := promptLine("Host", preset.DefaultHost)
	if err != nil {
		return nil, err
	}
	portStr, err := promptLine("Port", strconv.Itoa(preset.DefaultPort))
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(ui.Output, "\n🤖 %s\n", ui.Heading("Add Models (enter blank name to finish)"))

	var models []preset.ModelEntry
	for i := 1; ; {
		fmt.Fprintf(ui.Output, "  Model %d:\n", i)

		modelName, err := promptLine("    Name", "")
		if err != nil {
			return nil, err
		}
		if modelName == "" {
			break
		}
		if err := preset.ValidateName(modelName); err != nil {
			ui.PrintWarning(fmt.Sprintf("invalid model name: %v", err))
			continue
		}
		isDuplicate := false
		for _, existing := range models {
			if existing.Name == modelName {
				ui.PrintWarning(fmt.Sprintf("model name '%s' already added", modelName))
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}

		modelRef, err := promptLine("    Model", "")
		if err != nil {
			return nil, err
		}
		if modelRef == "" {
			ui.PrintWarning(fmt.Sprintf("model is required for '%s'", modelName))
			continue
		}
		if !strings.HasPrefix(modelRef, "h:") && !strings.HasPrefix(modelRef, "f:") {
			ui.PrintWarning(fmt.Sprintf("model for '%s' must have h: or f: prefix\nExamples: h:unsloth/gemma3:Q4_K_M, f:~/models/model.gguf", modelName))
			continue
		}

		entry := preset.ModelEntry{
			Name:  modelName,
			Model: modelRef,
		}

		models = append(models, entry)
		i++
		fmt.Fprintln(ui.Output)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("at least one model is required for router mode")
	}

	fmt.Fprintf(ui.Output, "  %d model(s) added.\n\n", len(models))

	p := &preset.Preset{
		Name:   name,
		Mode:   "router",
		Models: models,
	}
	if hostStr != "" && hostStr != preset.DefaultHost {
		p.Host = hostStr
	}
	if port, err := strconv.Atoi(portStr); err == nil && port != preset.DefaultPort {
		p.Port = port
	}

	return p, nil
}
