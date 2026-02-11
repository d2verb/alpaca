package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/pathutil"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/protocol"
	"github.com/d2verb/alpaca/internal/ui"
)

// LocalPresetFile is the filename for local presets.
const LocalPresetFile = ".alpaca.yaml"

type LoadCmd struct {
	Identifier string `arg:"" optional:"" help:"Identifier (p:preset, h:org/repo:quant, f:/path/to/file, or f:*.yaml)" predictor:"load-identifier"`
}

func (c *LoadCmd) Run() error {
	cl, err := newClient()
	if err != nil {
		return err
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	// Resolve identifier (handles empty input → .alpaca.yaml)
	idStr, err := resolveLocalPreset(c.Identifier)
	if err != nil {
		return err
	}

	// Parse and normalize identifier
	id, err := identifier.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	// Prepare load request (normalize paths, get display name)
	req, err := c.prepare(id)
	if err != nil {
		return err
	}

	// Ensure HuggingFace model is downloaded (with progress bar)
	// This handles direct HF identifiers and presets that reference HF models
	if err := c.ensureHFModel(paths, id); err != nil {
		return err
	}

	// Send to daemon
	ui.PrintInfo(fmt.Sprintf("Loading %s...", req.displayName))
	resp, err := cl.Load(req.identifier)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.ECONNREFUSED) {
			return errDaemonNotRunning()
		}
		return fmt.Errorf("load model: %w", err)
	}

	if resp.Status == "error" {
		return handleLoadError(resp.ErrorCode, resp.Error, id)
	}

	endpoint, _ := resp.Data["endpoint"].(string)
	readyMsg := "Model ready"
	if isRouterPreset(id, paths) {
		readyMsg = "Router ready"
	}
	ui.PrintSuccess(fmt.Sprintf("%s at %s", readyMsg, ui.FormatEndpoint(endpoint)))
	return nil
}

// ensureHFModel ensures HuggingFace models are downloaded before loading.
// Handles direct HF identifiers and presets that reference HF models.
func (c *LoadCmd) ensureHFModel(paths *config.Paths, id *identifier.Identifier) error {
	var repo, quant string

	switch id.Type {
	case identifier.TypeHuggingFace:
		// Direct HF identifier: h:org/repo:quant
		repo, quant = id.Repo, id.Quant

	case identifier.TypePresetName:
		// Preset name: p:name - load from presets directory
		loader := preset.NewLoader(paths.Presets)
		p, err := loader.Load(id.PresetName)
		if err == nil {
			if p.IsRouter() {
				return c.ensureRouterModels(paths, p)
			}
			repo, quant = extractHFModel(p.Model)
			if err := c.ensureDraftModel(paths, p.DraftModel); err != nil {
				return err
			}
			if err := c.ensureMmprojFile(p.Mmproj); err != nil {
				return err
			}
		}
		// If preset loading fails, daemon will provide consistent error message

	case identifier.TypePresetFilePath:
		// Preset file: f:*.yaml - load from file path
		p, err := preset.LoadFile(id.FilePath)
		if err == nil {
			if p.IsRouter() {
				return c.ensureRouterModels(paths, p)
			}
			repo, quant = extractHFModel(p.Model)
			if err := c.ensureDraftModel(paths, p.DraftModel); err != nil {
				return err
			}
			if err := c.ensureMmprojFile(p.Mmproj); err != nil {
				return err
			}
		}
		// If preset loading fails, daemon will provide consistent error message

	default:
		// Model file path or other types don't need pulling
		return nil
	}

	// No HF model to pull
	if repo == "" {
		return nil
	}

	if err := pullIfNeeded(context.Background(), paths.Models, repo, quant); err != nil {
		return fmt.Errorf("download model: %w", err)
	}
	return nil
}

// ensureRouterModels downloads all HF models in a router preset.
// Uses fail-fast: stops at the first download failure.
func (c *LoadCmd) ensureRouterModels(paths *config.Paths, p *preset.Preset) error {
	ctx := context.Background()

	for _, m := range p.Models {
		repo, quant := extractHFModel(m.Model)
		if repo != "" {
			if err := pullIfNeeded(ctx, paths.Models, repo, quant); err != nil {
				return fmt.Errorf("download model '%s': %w", m.Name, err)
			}
		}

		draftRepo, draftQuant := extractHFModel(m.DraftModel)
		if draftRepo != "" {
			if err := pullIfNeeded(ctx, paths.Models, draftRepo, draftQuant); err != nil {
				return fmt.Errorf("download draft model for '%s': %w", m.Name, err)
			}
		}

		if err := c.ensureMmprojFile(m.Mmproj); err != nil {
			return fmt.Errorf("model '%s': %w", m.Name, err)
		}
	}

	return nil
}

// ensureMmprojFile validates that an explicit mmproj file path exists.
func (c *LoadCmd) ensureMmprojFile(mmproj string) error {
	if !preset.IsMmprojActive(mmproj) {
		return nil
	}
	if !strings.HasPrefix(mmproj, "f:") {
		return nil
	}
	path := mmproj[2:]
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("mmproj file not found: %s", path)
		}
		return fmt.Errorf("check mmproj file: %w", err)
	}
	return nil
}

// ensureDraftModel downloads a draft model if it uses HuggingFace format.
func (c *LoadCmd) ensureDraftModel(paths *config.Paths, draftModel string) error {
	draftRepo, draftQuant := extractHFModel(draftModel)
	if draftRepo == "" {
		return nil
	}

	if err := pullIfNeeded(context.Background(), paths.Models, draftRepo, draftQuant); err != nil {
		return fmt.Errorf("download draft model: %w", err)
	}
	return nil
}

// pullIfNeeded downloads a model if not already present.
func pullIfNeeded(ctx context.Context, modelsDir, repo, quant string) error {
	modelMgr := model.NewManager(modelsDir)
	exists, err := modelMgr.Exists(ctx, repo, quant)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return pullModel(repo, quant, modelsDir)
}

// extractHFModel extracts repo and quant from an HF model reference (h:org/repo:quant).
// Returns empty strings if not an HF model.
func extractHFModel(modelField string) (repo, quant string) {
	id, err := identifier.Parse(modelField)
	if err != nil {
		return "", ""
	}
	if id.Type != identifier.TypeHuggingFace {
		return "", ""
	}
	return id.Repo, id.Quant
}

// loadRequest holds the prepared load request data.
type loadRequest struct {
	identifier  string
	displayName string
}

// prepare normalizes paths and determines display name.
func (c *LoadCmd) prepare(id *identifier.Identifier) (*loadRequest, error) {
	req := &loadRequest{
		identifier:  id.Raw,
		displayName: id.Raw,
	}

	// Handle file path types (both preset and model)
	if id.Type == identifier.TypePresetFilePath || id.Type == identifier.TypeModelFilePath {
		absID, err := toAbsFileID(id.FilePath)
		if err != nil {
			return nil, err
		}
		req.identifier = absID
		req.displayName = id.FilePath
	}

	return req, nil
}

// toAbsFileID converts a file path to absolute f: identifier.
func toAbsFileID(path string) (string, error) {
	// Resolve tilde expansion first
	resolved, err := pathutil.ResolvePath(path, "")
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	// Then make absolute (for relative paths like ./preset.yaml)
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("make absolute path: %w", err)
	}
	return "f:" + absPath, nil
}

// isRouterPreset checks if the identifier refers to a router mode preset.
func isRouterPreset(id *identifier.Identifier, paths *config.Paths) bool {
	switch id.Type {
	case identifier.TypePresetName:
		loader := preset.NewLoader(paths.Presets)
		p, err := loader.Load(id.PresetName)
		if err != nil {
			return false
		}
		return p.IsRouter()
	case identifier.TypePresetFilePath:
		p, err := preset.LoadFile(id.FilePath)
		if err != nil {
			return false
		}
		return p.IsRouter()
	default:
		return false
	}
}

// handleLoadError converts daemon error codes into user-friendly errors.
func handleLoadError(code, message string, id *identifier.Identifier) error {
	switch code {
	case protocol.ErrCodePresetNotFound:
		return errPresetNotFound(id.PresetName)

	case protocol.ErrCodeModelNotFound:
		if id.Type == identifier.TypePresetName {
			return fmt.Errorf("model in preset '%s' not downloaded\nRun: alpaca pull <model>", id.PresetName)
		}
		return errModelNotFound(id.Raw)

	case protocol.ErrCodeServerFailed:
		return fmt.Errorf("failed to start server: %s", message)

	default:
		return fmt.Errorf("%s", message)
	}
}
