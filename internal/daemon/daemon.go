// Package daemon implements the Alpaca daemon.
package daemon

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
)

// presetLoader loads and lists presets.
type presetLoader interface {
	Load(name string) (*preset.Preset, error)
	List() ([]string, error)
}

// modelManager manages downloaded models.
type modelManager interface {
	List(ctx context.Context) ([]metadata.ModelEntry, error)
	GetFilePath(ctx context.Context, repo, quant string) (string, error)
}

// State represents the daemon state.
type State string

const (
	StateIdle    State = "idle"
	StateLoading State = "loading"
	StateRunning State = "running"
)

// Daemon manages llama-server lifecycle.
type Daemon struct {
	mu      sync.RWMutex
	state   State
	preset  *preset.Preset
	process *llama.Process

	presets        presetLoader
	models         modelManager
	userConfig     *config.Config
	config         *Config
	llamaLogWriter io.Writer
}

// Config holds daemon configuration.
type Config struct {
	LlamaServerPath string
	SocketPath      string
	LlamaLogWriter  io.Writer
}

// New creates a new daemon instance.
func New(cfg *Config, presets presetLoader, models modelManager, userConfig *config.Config) *Daemon {
	return &Daemon{
		state:          StateIdle,
		presets:        presets,
		models:         models,
		userConfig:     userConfig,
		config:         cfg,
		llamaLogWriter: cfg.LlamaLogWriter,
	}
}

// State returns the current daemon state.
func (d *Daemon) State() State {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

// CurrentPreset returns the currently loaded preset, if any.
func (d *Daemon) CurrentPreset() *preset.Preset {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.preset
}

// ListPresets returns all available preset names.
func (d *Daemon) ListPresets() ([]string, error) {
	return d.presets.List()
}

// ModelInfo represents information about a downloaded model.
type ModelInfo struct {
	Repo  string `json:"repo"`
	Quant string `json:"quant"`
	Size  int64  `json:"size"`
}

// ListModels returns all downloaded models.
func (d *Daemon) ListModels(ctx context.Context) ([]ModelInfo, error) {
	entries, err := d.models.List(ctx)
	if err != nil {
		return nil, err
	}

	models := []ModelInfo{}
	for _, e := range entries {
		models = append(models, ModelInfo{
			Repo:  e.Repo,
			Quant: e.Quant,
			Size:  e.Size,
		})
	}
	return models, nil
}

// resolveHFPreset creates a preset from HuggingFace format (h:repo:quant).
func resolveHFPreset(ctx context.Context, models modelManager, cfg *config.Config, repo, quant string) (*preset.Preset, error) {
	// Get model file path from metadata
	modelPath, err := models.GetFilePath(ctx, repo, quant)
	if err != nil {
		return nil, err
	}

	// Create preset with defaults from userConfig (with f: prefix)
	return &preset.Preset{
		Name:        fmt.Sprintf("h:%s:%s", repo, quant),
		Model:       "f:" + modelPath,
		Host:        cfg.DefaultHost,
		Port:        cfg.DefaultPort,
		ContextSize: cfg.DefaultCtxSize,
		GPULayers:   cfg.DefaultGPULayers,
	}, nil
}

// Run loads and runs a model (preset name, file path, or HuggingFace format).
func (d *Daemon) Run(ctx context.Context, input string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop current model if running
	if d.process != nil {
		if err := d.stopLocked(ctx); err != nil {
			return fmt.Errorf("stop current model: %w", err)
		}
	}

	// Parse identifier
	id, err := identifier.Parse(input)
	if err != nil {
		return fmt.Errorf("parse identifier: %w", err)
	}

	// Load preset based on identifier type
	var p *preset.Preset

	switch id.Type {
	case identifier.TypePresetName:
		// Load preset from presets directory
		p, err = d.presets.Load(id.PresetName)
		if err != nil {
			return fmt.Errorf("load preset: %w", err)
		}

		// Resolve model field if it's HF format
		p, err = preset.ResolveModel(ctx, p, d.models)
		if err != nil {
			return err
		}

	case identifier.TypeFilePath:
		// Create dynamic preset from file path with default settings
		p = &preset.Preset{
			Name:        id.FilePath,
			Model:       input, // Keep f: prefix for consistency
			Host:        d.userConfig.DefaultHost,
			Port:        d.userConfig.DefaultPort,
			ContextSize: d.userConfig.DefaultCtxSize,
			GPULayers:   d.userConfig.DefaultGPULayers,
		}

	case identifier.TypeHuggingFace:
		// Create preset from HF format
		p, err = resolveHFPreset(ctx, d.models, d.userConfig, id.Repo, id.Quant)
		if err != nil {
			return fmt.Errorf("resolve HuggingFace model: %w", err)
		}

	default:
		return fmt.Errorf("unknown identifier type")
	}

	d.state = StateLoading
	d.preset = p

	// Start llama-server
	proc := llama.NewProcess(d.config.LlamaServerPath)
	if d.llamaLogWriter != nil {
		proc.SetLogWriter(d.llamaLogWriter)
	}
	if err := proc.Start(ctx, p.BuildArgs()); err != nil {
		d.state = StateIdle
		d.preset = nil
		return fmt.Errorf("start llama-server: %w", err)
	}
	d.process = proc

	// Wait for llama-server to become ready
	if err := llama.WaitForReady(ctx, p.Endpoint()); err != nil {
		// Cleanup on failure
		d.process.Stop(ctx)
		d.process = nil
		d.preset = nil
		d.state = StateIdle
		return fmt.Errorf("wait for llama-server ready: %w", err)
	}

	d.state = StateRunning
	return nil
}

// Kill stops the currently running model.
func (d *Daemon) Kill(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stopLocked(ctx)
}

func (d *Daemon) stopLocked(ctx context.Context) error {
	if d.process == nil {
		return nil
	}

	if err := d.process.Stop(ctx); err != nil {
		return err
	}

	d.process = nil
	d.preset = nil
	d.state = StateIdle
	return nil
}
