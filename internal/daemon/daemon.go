// Package daemon implements the Alpaca daemon.
package daemon

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/pull"
)

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

	presetLoader   *preset.Loader
	modelManager   *model.Manager
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
func New(cfg *Config, presetLoader *preset.Loader, modelManager *model.Manager, userConfig *config.Config) *Daemon {
	return &Daemon{
		state:          StateIdle,
		presetLoader:   presetLoader,
		modelManager:   modelManager,
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
	return d.presetLoader.List()
}

// ModelInfo represents information about a downloaded model.
type ModelInfo struct {
	Repo  string `json:"repo"`
	Quant string `json:"quant"`
	Size  int64  `json:"size"`
}

// ListModels returns all downloaded models.
func (d *Daemon) ListModels(ctx context.Context) ([]ModelInfo, error) {
	entries, err := d.modelManager.List(ctx)
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

// createPresetFromHF creates a preset from HuggingFace format (repo:quant).
func (d *Daemon) createPresetFromHF(ctx context.Context, repo, quant string) (*preset.Preset, error) {
	// Get model file path from metadata
	modelPath, err := d.modelManager.GetFilePath(ctx, repo, quant)
	if err != nil {
		return nil, err
	}

	// Create preset with defaults from userConfig
	return &preset.Preset{
		Name:        fmt.Sprintf("%s:%s", repo, quant),
		Model:       modelPath,
		Host:        d.userConfig.DefaultHost,
		Port:        d.userConfig.DefaultPort,
		ContextSize: d.userConfig.DefaultCtxSize,
		GPULayers:   d.userConfig.DefaultGPULayers,
	}, nil
}

// Run loads and runs a model (preset name or HuggingFace format).
func (d *Daemon) Run(ctx context.Context, identifier string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop current model if running
	if d.process != nil {
		if err := d.stopLocked(ctx); err != nil {
			return fmt.Errorf("stop current model: %w", err)
		}
	}

	// Determine if identifier is HuggingFace format or preset name
	var p *preset.Preset
	var err error

	if strings.Contains(identifier, ":") {
		// HuggingFace format (repo:quant)
		repo, quant, parseErr := pull.ParseModelSpec(identifier)
		if parseErr != nil {
			return parseErr
		}
		p, err = d.createPresetFromHF(ctx, repo, quant)
	} else {
		// Preset name
		p, err = d.presetLoader.Load(identifier)
	}

	if err != nil {
		return err
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
