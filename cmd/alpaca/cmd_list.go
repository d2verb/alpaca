package main

import (
	"context"
	"fmt"

	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/ui"
)

type ListCmd struct{}

func (c *ListCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	// Load presets
	loader := preset.NewLoader(paths.Presets)
	presetNames, err := loader.List()
	if err != nil {
		return fmt.Errorf("list presets: %w", err)
	}

	// Load models
	modelMgr := model.NewManager(paths.Models)
	ctx := context.Background()
	entries, err := modelMgr.List(ctx)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}

	// Convert to UI model format
	models := make([]ui.ModelInfo, len(entries))
	for i, entry := range entries {
		models[i] = ui.ModelInfo{
			Repo:         entry.Repo,
			Quant:        entry.Quant,
			SizeString:   formatSize(entry.Size),
			DownloadedAt: entry.DownloadedAt.Format("2006-01-02"),
		}
	}

	// Print both lists
	ui.PrintPresetList(presetNames)
	if len(presetNames) > 0 && len(models) > 0 {
		fmt.Fprintln(ui.Output) // Single blank line between sections
	}
	ui.PrintModelList(models)

	// Print help message if both are empty
	if len(presetNames) == 0 && len(models) == 0 {
		ui.PrintInfo("Create preset: alpaca new")
		ui.PrintInfo("Download model: alpaca pull h:org/repo:quant")
	}

	return nil
}
