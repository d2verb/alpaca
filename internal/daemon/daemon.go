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
	GetDetails(ctx context.Context, repo, quant string) (*metadata.ModelEntry, error)
}

// llamaProcess manages llama-server process lifecycle.
type llamaProcess interface {
	Start(args []string) error
	Stop(ctx context.Context) error
	SetLogWriter(w io.Writer)
	Done() <-chan struct{}
	ExitErr() error
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
	// The lock is intentionally scoped to process lifecycle updates and run
	// generation transitions; heavy operations (preset/model resolution, health
	// checks) run outside this lock.
	mu sync.Mutex

	// runGen is incremented on each Run/Kill request to invalidate older
	// in-flight Run operations once control is yielded outside mu.
	runGen uint64 // protected by mu

	// snapshot is atomically replaced so status readers observe a consistent
	// state+preset pair with a single load.
	snapshot atomic.Pointer[daemonSnapshot]

	process llamaProcess // protected by mu

	presets        presetLoader
	models         modelManager
	configPath     string // path for router mode config.ini
	logger         *slog.Logger
	llamaLogWriter io.Writer

	// startupMu protects cancelStartup.
	// Separate from mu so Kill() can cancel startup without acquiring mu.
	startupMu     sync.Mutex
	startupGen    uint64
	cancelStartup context.CancelFunc

	startupTimeout time.Duration

	// Test hooks (optional, defaults to real implementations)
	newProcess   func(path string) llamaProcess
	waitForReady healthChecker
	httpClient   *http.Client // for FetchModelStatuses
}

type daemonSnapshot struct {
	state  State
	preset *preset.Preset
}

// StatusSnapshot is a consistent daemon status view.
type StatusSnapshot struct {
	State  State
	Preset *preset.Preset
}

// llamaServerCommand is the command to run llama-server.
// It relies on PATH resolution to find the binary.
const llamaServerCommand = "llama-server"

// defaultStartupTimeout is the maximum time to wait for llama-server to become ready.
const defaultStartupTimeout = 60 * time.Second

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
		waitForReady:   llama.WaitForReady,
		httpClient:     &http.Client{},
		startupTimeout: defaultStartupTimeout,
	}
	d.snapshot.Store(&daemonSnapshot{state: StateIdle})
	return d
}

// State returns the current daemon state.
// This method is lock-free and returns immediately.
func (d *Daemon) State() State {
	return d.StatusSnapshot().State
}

// CurrentPreset returns the currently loaded preset, if any.
// This method is lock-free and returns immediately.
func (d *Daemon) CurrentPreset() *preset.Preset {
	return d.StatusSnapshot().Preset
}

// StatusSnapshot returns a consistent daemon status snapshot.
// This method is lock-free and returns immediately.
func (d *Daemon) StatusSnapshot() StatusSnapshot {
	snap := d.snapshot.Load()
	if snap == nil {
		return StatusSnapshot{State: StateIdle}
	}
	return StatusSnapshot{
		State:  snap.state,
		Preset: snap.preset,
	}
}

func (d *Daemon) setSnapshot(state State, p *preset.Preset) {
	d.snapshot.Store(&daemonSnapshot{
		state:  state,
		preset: p,
	})
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

	default:
		return nil, fmt.Errorf("unknown identifier type")
	}

	p, err = d.resolveModel(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("resolve model: %w", err)
	}

	return p, nil
}

// buildArgs builds the llama-server arguments for the preset.
// For router mode, it writes the config.ini file and returns router args.
func (d *Daemon) buildArgs(p *preset.Preset) ([]string, error) {
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

// Run loads and runs a model (preset name, file path, or HuggingFace format).
// Returns error if HuggingFace model is not downloaded (use CLI to pull first).
func (d *Daemon) Run(ctx context.Context, input string) error {
	d.logger.Info("run requested", "input", input)

	d.cancelExistingStartup()

	// Reserve generation and stop current process quickly under lock.
	d.mu.Lock()
	d.runGen++
	myGen := d.runGen

	if d.process != nil {
		d.logger.Info("stopping current model")
		if err := d.stopLocked(ctx); err != nil {
			d.mu.Unlock()
			return fmt.Errorf("stop current model: %w", err)
		}
	}
	d.mu.Unlock()

	// Heavy operations run outside mu for better Kill()/Run() responsiveness.
	p, err := d.loadPreset(ctx, input)
	if err != nil {
		return err
	}
	d.mu.Lock()
	if d.runGen != myGen {
		d.mu.Unlock()
		return context.Canceled
	}
	d.setSnapshot(StateLoading, p)
	d.mu.Unlock()

	args, err := d.buildArgs(p)
	if err != nil {
		d.mu.Lock()
		if d.runGen == myGen {
			d.resetState()
		}
		d.mu.Unlock()
		return err
	}
	d.mu.Lock()
	stale := d.runGen != myGen
	d.mu.Unlock()
	if stale {
		d.cleanupRouterConfig(p)
		return context.Canceled
	}

	// Install process and startup cancel function only if this run is still current.
	d.mu.Lock()
	if d.runGen != myGen {
		d.mu.Unlock()
		d.cleanupRouterConfig(p)
		return context.Canceled
	}

	// Start llama-server
	proc := d.newProcess(llamaServerCommand)
	proc.SetLogWriter(d.llamaLogWriter)
	if err := proc.Start(args); err != nil {
		d.resetState()
		d.mu.Unlock()
		d.cleanupRouterConfig(p)
		if p.IsRouter() && !errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w (requires llama-server b7350 or later)", err)
		}
		return err
	}
	d.process = proc

	// Build cancellable context with timeout for startup
	startupCtx, startupCancel := context.WithCancel(ctx)
	d.setStartupCancel(myGen, startupCancel)
	d.mu.Unlock()
	defer startupCancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(startupCtx, d.startupTimeout)
	defer timeoutCancel()

	// Monitor process death → cancel health check
	go func() {
		select {
		case <-proc.Done():
			startupCancel()
		case <-timeoutCtx.Done():
		}
	}()

	// Wait for llama-server to become ready
	err = d.waitForReady(timeoutCtx, p.Endpoint())
	d.clearStartupCancel(myGen)

	d.mu.Lock()
	defer d.mu.Unlock()

	// Another Run/Kill superseded this operation.
	if d.runGen != myGen || d.process != proc {
		return context.Canceled
	}

	if err != nil {
		// Determine cause and build user-friendly error message
		select {
		case <-proc.Done():
			err = fmt.Errorf("llama-server exited unexpectedly: %w", proc.ExitErr())
		default:
			if errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("server did not become ready within %s", d.startupTimeout)
			}
		}

		if stopErr := d.process.Stop(ctx); stopErr != nil {
			d.logger.Warn("failed to stop process during cleanup", "error", stopErr)
		}
		d.process = nil
		d.resetState()
		d.cleanupRouterConfig(p)

		processErr := &llama.ProcessError{Op: llama.ProcessOpWait, Err: err}
		if p.IsRouter() {
			return fmt.Errorf("%w (requires llama-server b7350 or later)", processErr)
		}
		return processErr
	}

	d.setSnapshot(StateRunning, p)
	d.logger.Info("model ready", "endpoint", p.Endpoint())
	return nil
}

// Kill stops the currently running model.
func (d *Daemon) Kill(ctx context.Context) error {
	d.logger.Info("kill requested")

	d.cancelExistingStartup()

	d.mu.Lock()
	defer d.mu.Unlock()
	d.runGen++

	if err := d.stopLocked(ctx); err != nil {
		return err
	}
	d.resetState()
	return nil
}

func (d *Daemon) stopLocked(ctx context.Context) error {
	if d.process == nil {
		return nil
	}

	if err := d.process.Stop(ctx); err != nil {
		return err
	}

	p := d.CurrentPreset()
	d.process = nil
	d.resetState()
	d.cleanupRouterConfig(p)

	d.logger.Info("model stopped")
	return nil
}

// cleanupRouterConfig removes the router config.ini file (best-effort).
func (d *Daemon) cleanupRouterConfig(p *preset.Preset) {
	if p != nil && p.IsRouter() && d.configPath != "" {
		os.Remove(d.configPath) // ignore error
	}
}

// resetState clears state and preset to idle state.
func (d *Daemon) resetState() {
	d.setSnapshot(StateIdle, nil)
}

func (d *Daemon) cancelExistingStartup() {
	d.startupMu.Lock()
	cancel := d.cancelStartup
	d.startupMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (d *Daemon) setStartupCancel(gen uint64, cancel context.CancelFunc) {
	d.startupMu.Lock()
	d.startupGen = gen
	d.cancelStartup = cancel
	d.startupMu.Unlock()
}

func (d *Daemon) clearStartupCancel(gen uint64) {
	d.startupMu.Lock()
	if d.startupGen == gen {
		d.cancelStartup = nil
	}
	d.startupMu.Unlock()
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
