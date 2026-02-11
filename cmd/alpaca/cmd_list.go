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
	presetNames, presetErr := loader.List()
	if presetErr != nil && len(presetNames) == 0 {
		return fmt.Errorf("list presets: %w", presetErr)
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
		sizeStr := formatSize(entry.Size)
		if entry.Mmproj != nil {
			sizeStr += " + mmproj " + formatSize(entry.Mmproj.Size)
		}
		models[i] = ui.ModelInfo{
			Repo:         entry.Repo,
			Quant:        entry.Quant,
			SizeString:   sizeStr,
			DownloadedAt: entry.DownloadedAt.Format("2006-01-02"),
		}
	}

	// Print both lists
	ui.PrintPresetList(presetNames)
	if presetErr != nil {
		ui.PrintWarning(presetErr.Error())
	}
	fmt.Fprintln(ui.Output) // Single blank line between sections
	ui.PrintModelList(models)

	return nil
}
