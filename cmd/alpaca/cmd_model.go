package main

import (
	"context"
	"fmt"

	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/pull"
)

type ModelCmd struct {
	List   ModelListCmd `cmd:"" name:"ls" help:"List downloaded models"`
	Pull   ModelPullCmd `cmd:"" help:"Download a model"`
	Remove ModelRmCmd   `cmd:"" name:"rm" help:"Remove a model"`
}

type ModelListCmd struct{}

func (c *ModelListCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	modelMgr := model.NewManager(paths.Models)
	ctx := context.Background()

	entries, err := modelMgr.List(ctx)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No models downloaded.")
		fmt.Println("Run: alpaca model pull <repo>:<quant>")
		return nil
	}

	fmt.Println("Downloaded models:")
	for _, entry := range entries {
		fmt.Printf("  - %s:%s (%s)\n", entry.Repo, entry.Quant, formatSize(entry.Size))
	}
	return nil
}

type ModelPullCmd struct {
	Model string `arg:"" help:"Model to download (format: repo:quant)"`
}

func (c *ModelPullCmd) Run() error {
	repo, quant, err := pull.ParseModelSpec(c.Model)
	if err != nil {
		return fmt.Errorf("invalid model spec: %w\nFormat: alpaca model pull <org>/<repo>:<quant>\nExample: alpaca model pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M", err)
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	if err := pullModel(repo, quant, paths.Models); err != nil {
		return errDownloadFailed()
	}
	return nil
}

type ModelRmCmd struct {
	Model string `arg:"" help:"Model to remove (format: repo:quant)"`
}

func (c *ModelRmCmd) Run() error {
	repo, quant, err := pull.ParseModelSpec(c.Model)
	if err != nil {
		return fmt.Errorf("invalid model spec: %w", err)
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	modelMgr := model.NewManager(paths.Models)
	ctx := context.Background()

	// Check if model exists
	exists, err := modelMgr.Exists(ctx, repo, quant)
	if err != nil {
		return fmt.Errorf("check model: %w", err)
	}
	if !exists {
		return errModelNotFound(fmt.Sprintf("%s:%s", repo, quant))
	}

	// Confirmation prompt
	fmt.Printf("Delete model '%s:%s'? (y/N): ", repo, quant)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Remove model
	if err := modelMgr.Remove(ctx, repo, quant); err != nil {
		return fmt.Errorf("remove model: %w", err)
	}

	fmt.Printf("Model '%s:%s' removed.\n", repo, quant)
	return nil
}
