// Package daemon implements the Alpaca daemon.
package daemon

import (
	"context"
	"fmt"
	"sync"

	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/preset"
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

	presetLoader *preset.Loader
	config       *Config
}

// Config holds daemon configuration.
type Config struct {
	LlamaServerPath string
	SocketPath      string
}

// New creates a new daemon instance.
func New(cfg *Config, presetLoader *preset.Loader) *Daemon {
	return &Daemon{
		state:        StateIdle,
		presetLoader: presetLoader,
		config:       cfg,
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

// Run loads and runs a preset.
func (d *Daemon) Run(ctx context.Context, presetName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop current model if running
	if d.process != nil {
		if err := d.stopLocked(ctx); err != nil {
			return fmt.Errorf("stop current model: %w", err)
		}
	}

	// Load preset
	p, err := d.presetLoader.Load(presetName)
	if err != nil {
		return err
	}

	d.state = StateLoading
	d.preset = p

	// Start llama-server
	proc := llama.NewProcess(d.config.LlamaServerPath)
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
