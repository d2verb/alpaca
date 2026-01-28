package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/pull"
)

type LoadCmd struct {
	Identifier string `arg:"" help:"Preset name or HuggingFace format (repo:quant)"`
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

	// If HF format, check if downloaded
	if strings.Contains(c.Identifier, ":") {
		repo, quant, err := pull.ParseModelSpec(c.Identifier)
		if err != nil {
			return fmt.Errorf("invalid model spec: %w", err)
		}

		// Check if model exists
		modelMgr := model.NewManager(paths.Models)
		exists, err := modelMgr.Exists(ctx, repo, quant)
		if err != nil {
			return fmt.Errorf("check model: %w", err)
		}

		// Auto-pull if not downloaded
		if !exists {
			fmt.Println("Model not found. Downloading...")
			if err := pullModel(repo, quant, paths.Models); err != nil {
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
