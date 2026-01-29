package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/model"
)

type LoadCmd struct {
	Identifier string `arg:"" help:"Model identifier (h:org/repo:quant, p:preset-name, or f:/path/to/file)"`
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
			fmt.Println("Model not found. Downloading...")
			if err := pullModel(id.Repo, id.Quant, paths.Models); err != nil {
				return errDownloadFailed()
			}
		}
	}

	// Load model
	fmt.Printf("Loading %s...\n", c.Identifier)
	resp, err := cl.Load(c.Identifier)
	if err != nil {
		if strings.Contains(err.Error(), "connect") {
			return errDaemonNotRunning()
		}
		return fmt.Errorf("load model: %w", err)
	}

	if resp.Status == "error" {
		if strings.Contains(resp.Error, "not found") {
			return errModelNotFound(c.Identifier)
		}
		return fmt.Errorf("%s", resp.Error)
	}

	endpoint, _ := resp.Data["endpoint"].(string)
	fmt.Printf("Model ready at %s\n", endpoint)
	return nil
}
