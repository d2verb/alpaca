// Package daemon implements the Alpaca daemon.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/logging"
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

// llamaProcess manages llama-server process lifecycle.
type llamaProcess interface {
	Start(ctx context.Context, args []string) error
	Stop(ctx context.Context) error
	SetLogWriter(w io.Writer)
}

// healthChecker waits for llama-server to become ready.
type healthChecker func(ctx context.Context, endpoint string) error

// State represents the daemon state.
type State string

const (
	StateIdle    State = "idle"
	StateLoading State = "loading"
	StateRunning State = "running"
)

// Daemon manages llama-server lifecycle.
type Daemon struct {
	// mu protects the process field and serializes Run/Kill operations.
	// Note: Changed from RWMutex to Mutex because state and preset are now
	// accessed atomically, eliminating the need for concurrent read locks.
	mu sync.Mutex

	// state and preset are accessed atomically for lock-free reads.
	// This allows State() and CurrentPreset() to return immediately
	// even while Run() is waiting for llama-server to become ready.
	state  atomic.Value                  // holds State
	preset atomic.Pointer[preset.Preset] // holds *preset.Preset

	process llamaProcess // protected by mu

	presets        presetLoader
	models         modelManager
	configPath     string // path for router mode config.ini
	logger         *slog.Logger
	llamaLogWriter io.Writer

	// Test hooks (optional, defaults to real implementations)
	newProcess   func(path string) llamaProcess
	waitForReady healthChecker
	httpClient   *http.Client // for FetchModelStatuses
}

// llamaServerCommand is the command to run llama-server.
// It relies on PATH resolution to find the binary.
const llamaServerCommand = "llama-server"

// New creates a new daemon instance.
func New(presets presetLoader, models modelManager, configPath string, daemonLogWriter io.Writer, llamaLogWriter io.Writer) *Daemon {
	if daemonLogWriter == nil {
		panic("daemonLogWriter must not be nil")
	}
	if llamaLogWriter == nil {
		panic("llamaLogWriter must not be nil")
	}
	logger := logging.NewLogger(daemonLogWriter)

	d := &Daemon{
		presets:        presets,
		models:         models,
		configPath:     configPath,
		logger:         logger,
		llamaLogWriter: llamaLogWriter,
		// Default implementations (can be overridden in tests)
		newProcess: func(path string) llamaProcess {
			return llama.NewProcess(path)
		},
		waitForReady: llama.WaitForReady,
		httpClient:   &http.Client{},
	}
	d.state.Store(StateIdle)
	return d
}

// State returns the current daemon state.
// This method is lock-free and returns immediately.
func (d *Daemon) State() State {
	return d.state.Load().(State)
}

// CurrentPreset returns the currently loaded preset, if any.
// This method is lock-free and returns immediately.
func (d *Daemon) CurrentPreset() *preset.Preset {
	return d.preset.Load()
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

// newDefaultPreset creates a preset with default settings.
func newDefaultPreset(name, model string) *preset.Preset {
	return &preset.Preset{
		Name:  name,
		Model: model,
		// Host, Port, ContextSize use preset package defaults via GetXxx() methods
	}
}

// resolveHFPreset creates a preset from HuggingFace format (h:repo:quant).
// Returns error if model is not downloaded.
func (d *Daemon) resolveHFPreset(ctx context.Context, repo, quant string) (*preset.Preset, error) {
	modelPath, err := d.models.GetFilePath(ctx, repo, quant)
	if err != nil {
		return nil, err
	}
	return newDefaultPreset(fmt.Sprintf("h:%s:%s", repo, quant), "f:"+modelPath), nil
}

// resolveModel resolves the model and draft_model fields in a preset if they use HuggingFace format.
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
			return nil, fmt.Errorf("invalid draft_model field in preset: %w", err)
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

	if id.Type == identifier.TypeHuggingFace {
		modelPath, err := d.models.GetFilePath(ctx, id.Repo, id.Quant)
		if err != nil {
			return nil, fmt.Errorf("resolve model %s:%s: %w", id.Repo, id.Quant, err)
		}
		resolved.Model = "f:" + modelPath
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
	needsResolve := false
	for _, m := range p.Models {
		if isHFIdentifier(m.Model) {
			needsResolve = true
			break
		}
		if m.DraftModel != "" && isHFIdentifier(m.DraftModel) {
			needsResolve = true
			break
		}
	}

	if !needsResolve {
		return p, nil
	}

	// Deep copy: copy the preset, Models slice, and ServerOptions maps
	resolved := *p
	resolved.ServerOptions = maps.Clone(p.ServerOptions)
	resolved.Models = make([]preset.ModelEntry, len(p.Models))
	copy(resolved.Models, p.Models)
	for i, m := range resolved.Models {
		resolved.Models[i].ServerOptions = maps.Clone(m.ServerOptions)
	}

	for i, m := range resolved.Models {
		if isHFIdentifier(m.Model) {
			id, err := identifier.Parse(m.Model)
			if err != nil {
				return nil, fmt.Errorf("invalid model field in models[%d]: %w", i, err)
			}
			modelPath, err := d.models.GetFilePath(ctx, id.Repo, id.Quant)
			if err != nil {
				return nil, fmt.Errorf("resolve model %s:%s in models[%d]: %w", id.Repo, id.Quant, i, err)
			}
			resolved.Models[i].Model = "f:" + modelPath
		}

		if m.DraftModel != "" && isHFIdentifier(m.DraftModel) {
			id, err := identifier.Parse(m.DraftModel)
			if err != nil {
				return nil, fmt.Errorf("invalid draft_model field in models[%d]: %w", i, err)
			}
			draftPath, err := d.models.GetFilePath(ctx, id.Repo, id.Quant)
			if err != nil {
				return nil, fmt.Errorf("resolve draft model %s:%s in models[%d]: %w", id.Repo, id.Quant, i, err)
			}
			resolved.Models[i].DraftModel = "f:" + draftPath
		}
	}

	return &resolved, nil
}

// isHFIdentifier returns true if the identifier string uses HuggingFace format (h: prefix).
func isHFIdentifier(s string) bool {
	id, err := identifier.Parse(s)
	if err != nil {
		return false
	}
	return id.Type == identifier.TypeHuggingFace
}

// Run loads and runs a model (preset name, file path, or HuggingFace format).
// Returns error if HuggingFace model is not downloaded (use CLI to pull first).
func (d *Daemon) Run(ctx context.Context, input string) error {
	d.logger.Info("run requested", "input", input)
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop current model if running
	if d.process != nil {
		d.logger.Info("stopping current model")
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
		p, err = d.presets.Load(id.PresetName)
		if err != nil {
			return fmt.Errorf("load preset: %w", err)
		}

	case identifier.TypePresetFilePath:
		p, err = preset.LoadFile(id.FilePath)
		if err != nil {
			return fmt.Errorf("load preset file: %w", err)
		}

	case identifier.TypeModelFilePath:
		p = newDefaultPreset(id.FilePath, input)

	case identifier.TypeHuggingFace:
		p, err = d.resolveHFPreset(ctx, id.Repo, id.Quant)
		if err != nil {
			return fmt.Errorf("resolve HuggingFace model: %w", err)
		}

	default:
		return fmt.Errorf("unknown identifier type")
	}

	// Resolve HuggingFace model reference if present
	p, err = d.resolveModel(ctx, p)
	if err != nil {
		return err
	}

	d.state.Store(StateLoading)
	d.preset.Store(p)

	// Build args depending on mode
	var args []string
	if p.IsRouter() {
		d.logger.Info("loading router preset", "preset", p.Name, "models", len(p.Models))

		// Write config.ini for router mode
		content := p.GenerateConfigINI()
		if err := atomicWriteFile(d.configPath, content); err != nil {
			d.resetState()
			return fmt.Errorf("write router config: %w", err)
		}

		args = p.BuildRouterArgs(d.configPath)
	} else {
		d.logger.Info("loading model", "preset", p.Name, "model", p.Model)
		args = p.BuildArgs()
	}

	// Start llama-server
	proc := d.newProcess(llamaServerCommand)
	proc.SetLogWriter(d.llamaLogWriter)
	if err := proc.Start(ctx, args); err != nil {
		d.resetState()
		if p.IsRouter() && !errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w (requires llama-server b7350 or later)", err)
		}
		return err
	}
	d.process = proc

	// Wait for llama-server to become ready
	if err := d.waitForReady(ctx, p.Endpoint()); err != nil {
		d.process.Stop(ctx)
		d.process = nil
		d.resetState()
		processErr := &llama.ProcessError{Op: llama.ProcessOpWait, Err: err}
		if p.IsRouter() {
			return fmt.Errorf("%w (requires llama-server b7350 or later)", processErr)
		}
		return processErr
	}

	d.state.Store(StateRunning)
	d.logger.Info("model ready", "endpoint", p.Endpoint())
	return nil
}

// Kill stops the currently running model.
func (d *Daemon) Kill(ctx context.Context) error {
	d.logger.Info("kill requested")
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

	// Check if router mode before clearing state
	isRouter := false
	if p := d.preset.Load(); p != nil {
		isRouter = p.IsRouter()
	}

	d.process = nil
	d.resetState()

	// Best-effort cleanup of router config.ini
	if isRouter && d.configPath != "" {
		os.Remove(d.configPath) // ignore error
	}

	d.logger.Info("model stopped")
	return nil
}

// resetState clears state and preset to idle state.
func (d *Daemon) resetState() {
	d.preset.Store(nil)
	d.state.Store(StateIdle)
}

// RouterModelStatus represents the status of a single model in router mode.
type RouterModelStatus struct {
	ID     string            `json:"id"`
	Status routerModelStatus `json:"status"`
}

// routerModelStatus wraps the status object from llama-server's /models API.
// The API returns {"status": {"value": "loaded", ...}} not a plain string.
type routerModelStatus struct {
	Value string `json:"value"` // "loaded", "loading", "unloaded"
}

// FetchModelStatuses queries the running llama-server's /models endpoint
// to get the status of each model in router mode.
// Returns nil for non-router presets or on any error (graceful degradation).
func (d *Daemon) FetchModelStatuses(ctx context.Context) []RouterModelStatus {
	p := d.CurrentPreset()
	if p == nil || !p.IsRouter() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Endpoint()+"/models", nil)
	if err != nil {
		return nil
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Parse the response: {"data": [{"id": "...", "status": "..."}]}
	// Limit response body to 1MB to prevent excessive memory usage
	limitedBody := http.MaxBytesReader(nil, resp.Body, 1<<20)
	var body struct {
		Data []RouterModelStatus `json:"data"`
	}
	if err := json.NewDecoder(limitedBody).Decode(&body); err != nil {
		return nil
	}

	return body.Data
}

// atomicWriteFile writes content to path atomically using a temp file + rename.
func atomicWriteFile(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".alpaca-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}
