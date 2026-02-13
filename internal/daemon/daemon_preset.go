package daemon

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/preset"
)

// newDefaultPreset creates a preset with default settings.
func newDefaultPreset(name, model string) *preset.Preset {
	return &preset.Preset{
		Name:  name,
		Model: model,
		// Host, Port use preset package defaults via GetXxx() methods
	}
}

// autoResolveMmproj resolves the mmproj file path from model metadata when the
// mmproj field is empty. The modelName parameter is used for router-mode logging;
// pass empty string for non-router cases.
func (d *Daemon) autoResolveMmproj(ctx context.Context, mmproj *string, modelPath, repo, quant, modelName string) {
	if *mmproj != "" {
		return
	}
	entry, err := d.models.GetDetails(ctx, repo, quant)
	if err != nil || entry.Mmproj == nil {
		return
	}
	mmprojPath := filepath.Join(filepath.Dir(modelPath), entry.Mmproj.Filename)
	*mmproj = "f:" + mmprojPath
	attrs := []any{"path", mmprojPath}
	if modelName != "" {
		attrs = append(attrs, "model", modelName)
	}
	attrs = append(attrs, "source", "auto-resolved from metadata")
	d.logger.Info("using mmproj", attrs...)
}

// resolveHFPreset creates a preset from HuggingFace format (h:repo:quant).
// Returns error if model is not downloaded.
func (d *Daemon) resolveHFPreset(ctx context.Context, repo, quant string) (*preset.Preset, error) {
	modelPath, err := d.models.GetFilePath(ctx, repo, quant)
	if err != nil {
		return nil, err
	}
	p := newDefaultPreset(fmt.Sprintf("h:%s:%s", repo, quant), "f:"+modelPath)

	d.autoResolveMmproj(ctx, &p.Mmproj, modelPath, repo, quant, "")

	return p, nil
}

// resolveModel resolves the model and draft-model fields in a preset if they use HuggingFace format.
// Returns a new preset with the resolved model paths without mutating the original.
// Returns the original preset as-is if no resolution is needed.
// Returns error if HuggingFace model is not downloaded.
func (d *Daemon) resolveModel(ctx context.Context, p *preset.Preset) (*preset.Preset, error) {
	if p.IsRouter() {
		return d.resolveRouterModels(ctx, p)
	}

	id, err := identifier.Parse(p.Model)
	if err != nil {
		return nil, fmt.Errorf("invalid model field in preset: %w", err)
	}

	needsResolve := id.Type == identifier.TypeHuggingFace

	var draftID *identifier.Identifier
	if p.DraftModel != "" {
		parsed, err := identifier.Parse(p.DraftModel)
		if err != nil {
			return nil, fmt.Errorf("invalid draft-model field in preset: %w", err)
		}
		draftID = parsed
		if parsed.Type == identifier.TypeHuggingFace {
			needsResolve = true
		}
	}

	if !needsResolve {
		return p, nil
	}

	// Create copy to avoid mutating the original
	resolved := *p
	resolved.Options = maps.Clone(p.Options)

	if id.Type == identifier.TypeHuggingFace {
		modelPath, err := d.models.GetFilePath(ctx, id.Repo, id.Quant)
		if err != nil {
			return nil, fmt.Errorf("resolve model %s:%s: %w", id.Repo, id.Quant, err)
		}
		resolved.Model = "f:" + modelPath

		d.autoResolveMmproj(ctx, &resolved.Mmproj, modelPath, id.Repo, id.Quant, "")
	}

	if draftID != nil && draftID.Type == identifier.TypeHuggingFace {
		draftPath, err := d.models.GetFilePath(ctx, draftID.Repo, draftID.Quant)
		if err != nil {
			return nil, fmt.Errorf("resolve draft model %s:%s: %w", draftID.Repo, draftID.Quant, err)
		}
		resolved.DraftModel = "f:" + draftPath
	}

	return &resolved, nil
}

// resolveRouterModels resolves HuggingFace model references in router mode Models[].
func (d *Daemon) resolveRouterModels(ctx context.Context, p *preset.Preset) (*preset.Preset, error) {
	// Validate all model identifiers and check if any need HF resolution.
	needsResolve := false
	for i, m := range p.Models {
		id, err := identifier.Parse(m.Model)
		if err != nil {
			return nil, fmt.Errorf("invalid model field in models[%d]: %w", i, err)
		}
		if id.Type == identifier.TypeHuggingFace {
			needsResolve = true
		}

		if m.DraftModel != "" {
			did, err := identifier.Parse(m.DraftModel)
			if err != nil {
				return nil, fmt.Errorf("invalid draft-model field in models[%d]: %w", i, err)
			}
			if did.Type == identifier.TypeHuggingFace {
				needsResolve = true
			}
		}
	}

	if !needsResolve {
		return p, nil
	}

	// Deep copy: copy the preset, Models slice, and Options maps
	resolved := *p
	resolved.Options = maps.Clone(p.Options)
	resolved.Models = make([]preset.ModelEntry, len(p.Models))
	copy(resolved.Models, p.Models)
	for i, m := range resolved.Models {
		resolved.Models[i].Options = maps.Clone(m.Options)
	}

	for i, m := range resolved.Models {
		// Parse already validated in the loop above; safe to ignore error.
		id, _ := identifier.Parse(m.Model)
		if id.Type == identifier.TypeHuggingFace {
			modelPath, err := d.models.GetFilePath(ctx, id.Repo, id.Quant)
			if err != nil {
				return nil, fmt.Errorf("resolve model %s:%s in models[%d]: %w", id.Repo, id.Quant, i, err)
			}
			resolved.Models[i].Model = "f:" + modelPath

			d.autoResolveMmproj(ctx, &resolved.Models[i].Mmproj, modelPath, id.Repo, id.Quant, m.Name)
		}

		if m.DraftModel != "" {
			did, _ := identifier.Parse(m.DraftModel)
			if did.Type == identifier.TypeHuggingFace {
				draftPath, err := d.models.GetFilePath(ctx, did.Repo, did.Quant)
				if err != nil {
					return nil, fmt.Errorf("resolve draft model %s:%s in models[%d]: %w", did.Repo, did.Quant, i, err)
				}
				resolved.Models[i].DraftModel = "f:" + draftPath
			}
		}
	}

	return &resolved, nil
}

// loadPreset parses the input identifier and loads the corresponding preset.
// It resolves HuggingFace model references to local file paths.
func (d *Daemon) loadPreset(ctx context.Context, input string) (*preset.Preset, error) {
	id, err := identifier.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("parse identifier: %w", err)
	}

	var p *preset.Preset

	switch id.Type {
	case identifier.TypePresetName:
		p, err = d.presets.Load(id.PresetName)
		if err != nil {
			return nil, fmt.Errorf("load preset: %w", err)
		}

	case identifier.TypePresetFilePath:
		p, err = preset.LoadFile(id.FilePath)
		if err != nil {
			return nil, fmt.Errorf("load preset file: %w", err)
		}

	case identifier.TypeModelFilePath:
		p = newDefaultPreset(id.FilePath, input)

	case identifier.TypeHuggingFace:
		p, err = d.resolveHFPreset(ctx, id.Repo, id.Quant)
		if err != nil {
			return nil, fmt.Errorf("resolve HuggingFace model: %w", err)
		}
		// resolveHFPreset already returns a fully resolved local-file preset.
		return p, nil

	default:
		return nil, fmt.Errorf("unknown identifier type")
	}

	p, err = d.resolveModel(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("resolve model: %w", err)
	}

	return p, nil
}

// prepareArgsAndConfig builds llama-server args and writes config.ini for router mode.
func (d *Daemon) prepareArgsAndConfig(p *preset.Preset) ([]string, error) {
	if p.IsRouter() {
		d.logger.Info("loading router preset", "preset", p.Name, "models", len(p.Models))

		content := p.GenerateConfigINI()
		if err := atomicWriteFile(d.configPath, content); err != nil {
			return nil, fmt.Errorf("write router config: %w", err)
		}

		return p.BuildRouterArgs(d.configPath), nil
	}

	d.logger.Info("loading model", "preset", p.Name, "model", p.Model)
	return p.BuildArgs(), nil
}
