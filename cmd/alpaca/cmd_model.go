package main

import (
	"context"
	"fmt"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/model"
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
		fmt.Println("Run: alpaca model pull h:org/repo:quant")
		return nil
	}

	fmt.Println("Downloaded models:")
	for _, entry := range entries {
		fmt.Printf("  - h:%s:%s (%s)\n", entry.Repo, entry.Quant, formatSize(entry.Size))
	}
	return nil
}

type ModelPullCmd struct {
	Model string `arg:"" help:"Model to download (format: h:org/repo:quant)"`
}

func (c *ModelPullCmd) Run() error {
	id, err := identifier.Parse(c.Model)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	if id.Type != identifier.TypeHuggingFace {
		return fmt.Errorf("model pull only supports HuggingFace models\nFormat: alpaca model pull h:org/repo:quant\nExample: alpaca model pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}

	if id.Quant == "" {
		return fmt.Errorf("missing quant specifier\nFormat: alpaca model pull h:org/repo:quant\nExample: alpaca model pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	if err := pullModel(id.Repo, id.Quant, paths.Models); err != nil {
		return errDownloadFailed()
	}
	return nil
}

type ModelRmCmd struct {
	Model string `arg:"" help:"Model to remove (format: h:org/repo:quant)"`
}

func (c *ModelRmCmd) Run() error {
	id, err := identifier.Parse(c.Model)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	if id.Type != identifier.TypeHuggingFace {
		return fmt.Errorf("model rm only supports HuggingFace models\nFormat: alpaca model rm h:org/repo:quant")
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	modelMgr := model.NewManager(paths.Models)
	ctx := context.Background()

	// Check if model exists
	exists, err := modelMgr.Exists(ctx, id.Repo, id.Quant)
	if err != nil {
		return fmt.Errorf("check model: %w", err)
	}
	if !exists {
		return errModelNotFound(fmt.Sprintf("h:%s:%s", id.Repo, id.Quant))
	}

	// Confirmation prompt
	fmt.Printf("Delete model 'h:%s:%s'? (y/N): ", id.Repo, id.Quant)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Remove model
	if err := modelMgr.Remove(ctx, id.Repo, id.Quant); err != nil {
		return fmt.Errorf("remove model: %w", err)
	}

	fmt.Printf("Model 'h:%s:%s' removed.\n", id.Repo, id.Quant)
	return nil
}
