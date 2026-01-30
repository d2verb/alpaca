package main

import (
	"context"
	"fmt"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/ui"
)

type ShowCmd struct {
	Identifier string `arg:"" help:"Show details (p:name or h:org/repo:quant)"`
}

func (c *ShowCmd) Run() error {
	id, err := identifier.Parse(c.Identifier)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	switch id.Type {
	case identifier.TypePresetName:
		return c.showPreset(id.PresetName, paths.Presets)

	case identifier.TypeHuggingFace:
		return c.showModel(id, paths.Models)

	case identifier.TypeFilePath:
		return fmt.Errorf("cannot show file details\nUse: alpaca show p:name or alpaca show h:org/repo:quant")

	default:
		return fmt.Errorf("unknown identifier type")
	}
}

func (c *ShowCmd) showPreset(name, presetsDir string) error {
	loader := preset.NewLoader(presetsDir)
	p, err := loader.Load(name)
	if err != nil {
		return errPresetNotFound(name)
	}

	ui.PrintPresetDetails(ui.PresetDetails{
		Name:        p.Name,
		Model:       p.Model,
		ContextSize: p.ContextSize,
		GPULayers:   p.GPULayers,
		Threads:     p.Threads,
		Host:        p.GetHost(),
		Port:        p.GetPort(),
		ExtraArgs:   p.ExtraArgs,
	})

	return nil
}

func (c *ShowCmd) showModel(id *identifier.Identifier, modelsDir string) error {
	modelMgr := model.NewManager(modelsDir)
	ctx := context.Background()

	// Check if model exists
	exists, err := modelMgr.Exists(ctx, id.Repo, id.Quant)
	if err != nil {
		return fmt.Errorf("check model: %w", err)
	}

	if !exists {
		ui.PrintError(fmt.Sprintf("Model '%s' not downloaded", id.Raw))
		ui.PrintInfo(fmt.Sprintf("Run: alpaca pull %s", id.Raw))
		return errModelNotFound(id.Raw)
	}

	// Get model details
	entry, err := modelMgr.GetDetails(ctx, id.Repo, id.Quant)
	if err != nil {
		return fmt.Errorf("get model details: %w", err)
	}

	// Get file path
	filePath, err := modelMgr.GetFilePath(ctx, id.Repo, id.Quant)
	if err != nil {
		return fmt.Errorf("get file path: %w", err)
	}

	ui.PrintModelDetails(ui.ModelDetails{
		Repo:         entry.Repo,
		Quant:        entry.Quant,
		Filename:     entry.Filename,
		Path:         filePath,
		Size:         formatSize(entry.Size),
		DownloadedAt: entry.DownloadedAt.Format("2006-01-02 15:04:05"),
	})

	return nil
}
