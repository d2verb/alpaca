package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/ui"
)

type LoadCmd struct {
	Identifier string `arg:"" help:"Identifier (p:preset, h:org/repo:quant, or f:/path/to/file)"`
}

func (c *LoadCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	cl, err := newClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Parse identifier to determine handling
	id, err := identifier.Parse(c.Identifier)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	// Handle HuggingFace auto-pull
	if id.Type == identifier.TypeHuggingFace {
		// Validate quant is provided
		if id.Quant == "" {
			return fmt.Errorf("missing quant specifier in HuggingFace identifier\nExpected format: h:org/repo:quant (e.g., h:unsloth/gemma3:Q4_K_M)")
		}

		modelMgr := model.NewManager(paths.Models)
		exists, err := modelMgr.Exists(ctx, id.Repo, id.Quant)
		if err != nil {
			return fmt.Errorf("check model: %w", err)
		}
		if !exists {
			ui.PrintInfo("Model not found. Downloading...")
			if err := pullModel(id.Repo, id.Quant, paths.Models); err != nil {
				return errDownloadFailed()
			}
		}
	}

	// Load model
	ui.PrintInfo(fmt.Sprintf("Loading %s...", c.Identifier))
	resp, err := cl.Load(c.Identifier)
	if err != nil {
		if strings.Contains(err.Error(), "connect") {
			return errDaemonNotRunning()
		}
		return fmt.Errorf("load model: %w", err)
	}

	if resp.Status == "error" {
		return parseLoadError(resp.Error, id)
	}

	endpoint, _ := resp.Data["endpoint"].(string)
	ui.PrintSuccess(fmt.Sprintf("Model ready at %s", ui.Blue(endpoint)))
	return nil
}

// parseLoadError converts daemon error messages into user-friendly errors.
func parseLoadError(errMsg string, id *identifier.Identifier) error {
	isNotFound := strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no such file")

	if id.Type == identifier.TypePresetName {
		// Preset file itself not found
		if strings.Contains(errMsg, "load preset") && strings.Contains(errMsg, "no such file") {
			return errPresetNotFound(id.PresetName)
		}
		// Model referenced by preset not found
		if isNotFound {
			return fmt.Errorf("model in preset '%s' not downloaded\nRun: alpaca model pull <model>", id.PresetName)
		}
	}

	if isNotFound {
		return errModelNotFound(id.Raw)
	}

	return fmt.Errorf("%s", errMsg)
}
