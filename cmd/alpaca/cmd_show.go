package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/ui"
)

type ShowCmd struct {
	Identifier string `arg:"" help:"Show details (p:name or h:org/repo:quant)" predictor:"show-identifier"`
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

	case identifier.TypeModelFilePath, identifier.TypePresetFilePath:
		return fmt.Errorf("cannot show file details\nUse: alpaca show p:name or alpaca show h:org/repo:quant")

	default:
		return fmt.Errorf("unknown identifier type")
	}
}

func (c *ShowCmd) showPreset(name, presetsDir string) error {
	loader := preset.NewLoader(presetsDir)
	p, err := loader.Load(name)
	if err != nil {
		return mapPresetError(err, name)
	}

	if p.IsRouter() {
		c.showRouterPreset(p)
	} else {
		ui.PrintPresetDetails(ui.PresetDetails{
			Name:       p.Name,
			Model:      p.Model,
			DraftModel: p.DraftModel,
			Host:       p.GetHost(),
			Port:       p.GetPort(),
			Options:    p.Options,
		})
	}

	return nil
}

func (c *ShowCmd) showRouterPreset(p *preset.Preset) {
	details := ui.RouterPresetDetails{
		Name:        p.Name,
		Host:        p.GetHost(),
		Port:        p.GetPort(),
		MaxModels:   p.MaxModels,
		IdleTimeout: p.IdleTimeout,
		Options:     p.Options,
	}
	for _, m := range p.Models {
		details.Models = append(details.Models, ui.RouterModelDetail{
			Name:       m.Name,
			Model:      m.Model,
			DraftModel: m.DraftModel,
			Options:    m.Options,
		})
	}
	ui.PrintRouterPresetDetails(details)
}

func (c *ShowCmd) showModel(id *identifier.Identifier, modelsDir string) error {
	modelMgr := model.NewManager(modelsDir)
	ctx := context.Background()

	// Get model details
	entry, err := modelMgr.GetDetails(ctx, id.Repo, id.Quant)
	if err != nil {
		var notFound *metadata.NotFoundError
		if errors.As(err, &notFound) {
			return &ExitError{
				Code:    exitModelNotFound,
				Kind:    ExitKindError,
				Message: fmt.Sprintf("Model '%s' not found.\nRun: alpaca pull %s", id.Raw, id.Raw),
			}
		}
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
