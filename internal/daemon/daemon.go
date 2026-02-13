// Package daemon implements the Alpaca daemon.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

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

// ErrSuperseded indicates that a Run operation was superseded by a newer
// Run/Kill request (generation mismatch), not by caller context cancellation.
var ErrSuperseded = errors.New("operation superseded by newer request")

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

// RuntimeStatus is a consistent daemon runtime status view.
type RuntimeStatus struct {
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
func (d *Daemon) StatusSnapshot() RuntimeStatus {
	snap := d.snapshot.Load()
	if snap == nil {
		return RuntimeStatus{State: StateIdle}
	}
	return RuntimeStatus{
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

// Run loads and runs a model (preset name, file path, or HuggingFace format).
// Returns error if HuggingFace model is not downloaded (use CLI to pull first).
func (d *Daemon) Run(ctx context.Context, input string) error {
	d.logger.Info("run requested", "input", input)

	d.cancelExistingStartup()

	// Locking strategy:
	// 1) beginRun: short mu section to reserve generation and stop old process.
	// 2) prepare/start: heavy work outside mu, with generation-guarded state mutations.
	// 3) finalizeRun: short mu section to commit final state only if still current.
	myGen, err := d.beginRun(ctx)
	if err != nil {
		return err
	}

	// Heavy operations run outside mu for better Kill()/Run() responsiveness.
	p, err := d.loadPreset(ctx, input)
	if err != nil {
		return err
	}
	if !d.setLoadingIfCurrent(myGen, p) {
		return ErrSuperseded
	}

	args, err := d.prepareArgsAndConfig(p)
	if err != nil {
		d.resetIfCurrent(myGen)
		return err
	}

	start, err := d.startProcess(ctx, myGen, args)
	if !start.current {
		d.cleanupRouterConfig(p)
		return ErrSuperseded
	}
	if err != nil {
		d.cleanupRouterConfig(p)
		if p.IsRouter() && !errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w (requires llama-server b7350 or later)", err)
		}
		return err
	}
	defer start.startupCancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(start.startupCtx, d.startupTimeout)
	defer timeoutCancel()

	// Monitor process death → cancel health check
	go func() {
		select {
		case <-start.proc.Done():
			start.startupCancel()
		case <-timeoutCtx.Done():
		}
	}()

	// Wait for llama-server to become ready
	err = d.waitForReady(timeoutCtx, p.Endpoint())
	d.clearStartupCancel(myGen)

	return d.finalizeRun(ctx, myGen, start.proc, p, err)
}

func (d *Daemon) beginRun(ctx context.Context) (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.runGen++
	myGen := d.runGen

	if d.process != nil {
		d.logger.Info("stopping current model")
		if err := d.stopLocked(ctx); err != nil {
			return 0, fmt.Errorf("stop current model: %w", err)
		}
	}
	return myGen, nil
}

func (d *Daemon) setLoadingIfCurrent(gen uint64, p *preset.Preset) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.runGen != gen {
		return false
	}
	d.setSnapshot(StateLoading, p)
	return true
}

func (d *Daemon) resetIfCurrent(gen uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.runGen == gen {
		d.resetState()
	}
}

type startProcessResult struct {
	proc          llamaProcess
	startupCtx    context.Context
	startupCancel context.CancelFunc
	current       bool
}

func (d *Daemon) startProcess(ctx context.Context, gen uint64, args []string) (startProcessResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.runGen != gen {
		return startProcessResult{current: false}, nil
	}

	proc := d.newProcess(llamaServerCommand)
	proc.SetLogWriter(d.llamaLogWriter)
	if err := proc.Start(args); err != nil {
		d.resetState()
		return startProcessResult{current: true}, err
	}

	startupCtx, startupCancel := context.WithCancel(ctx)
	d.process = proc
	d.setStartupCancel(gen, startupCancel)
	return startProcessResult{
		proc:          proc,
		startupCtx:    startupCtx,
		startupCancel: startupCancel,
		current:       true,
	}, nil
}

func (d *Daemon) finalizeRun(ctx context.Context, gen uint64, proc llamaProcess, p *preset.Preset, waitErr error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Another Run/Kill superseded this operation.
	if d.runGen != gen || d.process != proc {
		return ErrSuperseded
	}

	if waitErr != nil {
		// Determine cause and build user-friendly error message
		select {
		case <-proc.Done():
			waitErr = fmt.Errorf("llama-server exited unexpectedly: %w", proc.ExitErr())
		default:
			if errors.Is(waitErr, context.DeadlineExceeded) {
				waitErr = fmt.Errorf("server did not become ready within %s", d.startupTimeout)
			}
		}

		if stopErr := d.process.Stop(ctx); stopErr != nil {
			d.logger.Warn("failed to stop process during cleanup", "error", stopErr)
		}
		d.process = nil
		d.resetState()
		d.cleanupRouterConfig(p)

		processErr := &llama.ProcessError{Op: llama.ProcessOpWait, Err: waitErr}
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
	hadProcess := d.process != nil

	if err := d.stopLocked(ctx); err != nil {
		return err
	}
	if !hadProcess {
		d.resetState()
	}
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
