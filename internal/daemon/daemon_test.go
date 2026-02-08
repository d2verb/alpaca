package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
)

type stubPresetLoader struct {
	presets map[string]*preset.Preset
	names   []string
	listErr error
}

func (s *stubPresetLoader) Load(name string) (*preset.Preset, error) {
	p, ok := s.presets[name]
	if !ok {
		return nil, &preset.NotFoundError{Name: name}
	}
	return p, nil
}

func (s *stubPresetLoader) List() ([]string, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.names, nil
}

type stubModelManager struct {
	entries  []metadata.ModelEntry
	filePath string
	exists   bool
	err      error
}

func (s *stubModelManager) List(ctx context.Context) ([]metadata.ModelEntry, error) {
	return s.entries, s.err
}

func (s *stubModelManager) GetFilePath(ctx context.Context, repo, quant string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if !s.exists {
		return "", &metadata.NotFoundError{Repo: repo, Quant: quant}
	}
	return s.filePath, nil
}

func newTestDaemon(presets presetLoader, models modelManager) *Daemon {
	return New(presets, models, "", io.Discard, io.Discard)
}

func newTestDaemonWithConfigPath(presets presetLoader, models modelManager, configPath string) *Daemon {
	return New(presets, models, configPath, io.Discard, io.Discard)
}

func TestResolveHFPresetSuccess(t *testing.T) {
	models := &stubModelManager{filePath: "/path/to/model.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p, err := d.resolveHFPreset(context.Background(), "TheBloke/CodeLlama-7B-GGUF", "Q4_K_M")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name != "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M" {
		t.Errorf("Name = %q, want %q", p.Name, "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}
	if p.Model != "f:/path/to/model.gguf" {
		t.Errorf("Model = %q, want %q", p.Model, "f:/path/to/model.gguf")
	}
	// Host, Port, ContextSize use preset defaults via GetXxx() methods
	if p.GetHost() != preset.DefaultHost {
		t.Errorf("GetHost() = %q, want %q", p.GetHost(), preset.DefaultHost)
	}
	if p.GetPort() != preset.DefaultPort {
		t.Errorf("GetPort() = %d, want %d", p.GetPort(), preset.DefaultPort)
	}
	if p.GetContextSize() != preset.DefaultContextSize {
		t.Errorf("GetContextSize() = %d, want %d", p.GetContextSize(), preset.DefaultContextSize)
	}
}

func TestResolveHFPresetModelNotFound(t *testing.T) {
	models := &stubModelManager{exists: false}
	d := newTestDaemon(&stubPresetLoader{}, models)

	_, err := d.resolveHFPreset(context.Background(), "unknown/repo", "Q4_K_M")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewDaemonStartsIdle(t *testing.T) {
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil")
	}
}

func TestListPresetsViaInterface(t *testing.T) {
	presets := &stubPresetLoader{names: []string{"codellama", "mistral"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	names, err := d.ListPresets()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("len(names) = %d, want 2", len(names))
	}
	if names[0] != "codellama" {
		t.Errorf("names[0] = %q, want %q", names[0], "codellama")
	}
	if names[1] != "mistral" {
		t.Errorf("names[1] = %q, want %q", names[1], "mistral")
	}
}

func TestListModelsViaInterface(t *testing.T) {
	entries := []metadata.ModelEntry{
		{Repo: "TheBloke/CodeLlama-7B-GGUF", Quant: "Q4_K_M", Size: 1024},
	}
	models := &stubModelManager{entries: entries}
	presets := &stubPresetLoader{}
	d := newTestDaemon(presets, models)

	infos, err := d.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("len(infos) = %d, want 1", len(infos))
	}
	if infos[0].Repo != "TheBloke/CodeLlama-7B-GGUF" {
		t.Errorf("Repo = %q, want %q", infos[0].Repo, "TheBloke/CodeLlama-7B-GGUF")
	}
}

func TestStateIsLockFree(t *testing.T) {
	// This test verifies that State() and CurrentPreset() can be called
	// concurrently without blocking, even when Run() holds the mutex.
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Manually acquire the mutex to simulate Run() holding it
	d.mu.Lock()

	// State() and CurrentPreset() should return immediately (lock-free)
	done := make(chan struct{})
	go func() {
		_ = d.State()
		_ = d.CurrentPreset()
		close(done)
	}()

	// Wait with timeout - if State()/CurrentPreset() were blocked by the mutex,
	// this would timeout
	select {
	case <-done:
		// Success: State() and CurrentPreset() returned without blocking
	case <-time.After(100 * time.Millisecond):
		t.Fatal("State() or CurrentPreset() blocked on mutex - they should be lock-free")
	}

	d.mu.Unlock()
}

func TestConcurrentStateAccess(t *testing.T) {
	// Test that multiple goroutines can safely read state concurrently.
	// The race detector (-race flag) will catch any data races.
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	const numReaders = 100
	var wg sync.WaitGroup
	wg.Add(numReaders)

	for range numReaders {
		go func() {
			defer wg.Done()
			for range 1000 {
				_ = d.State()
				_ = d.CurrentPreset()
			}
		}()
	}

	wg.Wait()
}

// mockProcess is a mock implementation of llamaProcess for testing.
type mockProcess struct {
	startErr     error
	stopErr      error
	startCalled  bool
	stopCalled   bool
	logWriter    io.Writer
	receivedArgs []string
}

func (m *mockProcess) Start(ctx context.Context, args []string) error {
	m.startCalled = true
	m.receivedArgs = args
	if m.startErr != nil {
		return &llama.ProcessError{Op: llama.ProcessOpStart, Err: m.startErr}
	}
	return nil
}

func (m *mockProcess) Stop(ctx context.Context) error {
	m.stopCalled = true
	return m.stopErr
}

func (m *mockProcess) SetLogWriter(w io.Writer) {
	m.logWriter = w
}

// mockHealthChecker returns a health checker function that can be configured to succeed or fail.
func mockHealthChecker(err error) healthChecker {
	return func(ctx context.Context, endpoint string) error {
		return err
	}
}

func TestDaemonRun_PresetNameSuccess(t *testing.T) {
	// Arrange
	testPreset := &preset.Preset{
		Name:        "test-preset",
		Model:       "f:/path/to/model.gguf",
		Host:        "127.0.0.1",
		Port:        8080,
		ContextSize: 4096,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil) // Success

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}
	if d.CurrentPreset() != testPreset {
		t.Error("CurrentPreset() should return loaded preset")
	}
	if !mockProc.startCalled {
		t.Error("Process.Start() should be called")
	}
	if mockProc.stopCalled {
		t.Error("Process.Stop() should not be called on success")
	}
}

func TestDaemonRun_FilePathSuccess(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "f:/path/to/custom.gguf")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}

	p := d.CurrentPreset()
	if p == nil {
		t.Fatal("CurrentPreset() should not be nil")
	}
	if p.Model != "f:/path/to/custom.gguf" {
		t.Errorf("Preset.Model = %q, want %q", p.Model, "f:/path/to/custom.gguf")
	}
	// Host, Port use preset defaults via GetXxx() methods
	if p.GetHost() != preset.DefaultHost {
		t.Errorf("GetHost() = %q, want %q", p.GetHost(), preset.DefaultHost)
	}
	if p.GetPort() != preset.DefaultPort {
		t.Errorf("GetPort() = %d, want %d", p.GetPort(), preset.DefaultPort)
	}
}

func TestDaemonRun_HuggingFaceSuccess(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		filePath: "/models/codellama-7b.Q4_K_M.gguf",
		exists:   true,
	}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}

	preset := d.CurrentPreset()
	if preset == nil {
		t.Fatal("CurrentPreset() should not be nil")
	}
	if preset.Name != "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M" {
		t.Errorf("Preset.Name = %q, want %q", preset.Name, "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}
	if preset.Model != "f:/models/codellama-7b.Q4_K_M.gguf" {
		t.Errorf("Preset.Model = %q, want %q", preset.Model, "f:/models/codellama-7b.Q4_K_M.gguf")
	}
}

func TestDaemonRun_PresetNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:nonexistent")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q on error", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil on error")
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called when preset not found")
	}
}

func TestDaemonRun_ModelNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		exists: false,
	}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "h:unknown/repo:Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q on error", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil on error")
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called when model not found")
	}
}

func TestDaemonRun_ProcessStartFailure(t *testing.T) {
	// Arrange
	testPreset := &preset.Preset{
		Name:  "test-preset",
		Model: "f:/path/to/model.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{
		startErr: fmt.Errorf("failed to start process"),
	}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after start failure", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after start failure")
	}
	if !mockProc.startCalled {
		t.Error("Process.Start() should be called")
	}
}

func TestDaemonRun_HealthCheckTimeout(t *testing.T) {
	// Arrange
	testPreset := &preset.Preset{
		Name:  "test-preset",
		Model: "f:/path/to/model.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(fmt.Errorf("health check timeout"))

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after health check failure", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after health check failure")
	}
	if !mockProc.startCalled {
		t.Error("Process.Start() should be called")
	}
	if !mockProc.stopCalled {
		t.Error("Process.Stop() should be called to cleanup after health check failure")
	}
}

func TestDaemonRun_StopsExistingModel(t *testing.T) {
	// Arrange
	firstPreset := &preset.Preset{
		Name:  "first-preset",
		Model: "f:/path/to/first.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}
	secondPreset := &preset.Preset{
		Name:  "second-preset",
		Model: "f:/path/to/second.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"first-preset":  firstPreset,
			"second-preset": secondPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	firstMockProc := &mockProcess{}
	secondMockProc := &mockProcess{}
	callCount := 0
	d.newProcess = func(path string) llamaProcess {
		callCount++
		if callCount == 1 {
			return firstMockProc
		}
		return secondMockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	// Load first model
	err := d.Run(context.Background(), "p:first-preset")
	if err != nil {
		t.Fatalf("first Run() failed: %v", err)
	}

	// Load second model (should stop first)
	err = d.Run(context.Background(), "p:second-preset")

	// Assert
	if err != nil {
		t.Fatalf("second Run() failed: %v", err)
	}
	if !firstMockProc.stopCalled {
		t.Error("first process should be stopped when loading second model")
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}
	if d.CurrentPreset() != secondPreset {
		t.Error("CurrentPreset() should be second preset")
	}
}

func TestDaemonRun_InvalidIdentifier(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "invalid:format:too:many:colons")

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid identifier, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called for invalid identifier")
	}
}

func TestDaemonKill_WhenRunning(t *testing.T) {
	// Arrange
	testPreset := &preset.Preset{
		Name:  "test-preset",
		Model: "f:/path/to/model.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Start a model first
	err := d.Run(context.Background(), "p:test-preset")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	err = d.Kill(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mockProc.stopCalled {
		t.Error("Process.Stop() should be called")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after Kill()", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after Kill()")
	}
}

func TestDaemonKill_WhenIdle(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Act
	err := d.Kill(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Kill() on idle daemon should not error: %v", err)
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
}

func TestDaemonKill_StopError(t *testing.T) {
	// Arrange
	testPreset := &preset.Preset{
		Name:  "test-preset",
		Model: "f:/path/to/model.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{
		stopErr: fmt.Errorf("failed to stop process"),
	}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Start a model first
	err := d.Run(context.Background(), "p:test-preset")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	err = d.Kill(context.Background())

	// Assert
	if err == nil {
		t.Fatal("expected error from Kill(), got nil")
	}
	if !mockProc.stopCalled {
		t.Error("Process.Stop() should be called even if it errors")
	}
}

func TestDaemonRun_PresetWithHFModel(t *testing.T) {
	// Arrange - Preset that references a HuggingFace model
	testPreset := &preset.Preset{
		Name:  "codellama-preset",
		Model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M", // HF format in preset
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"codellama-preset": testPreset,
		},
	}
	models := &stubModelManager{
		filePath: "/models/codellama.gguf",
		exists:   true,
	}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:codellama-preset")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}

	// Verify that the preset's HF model was resolved to a file path
	loadedPreset := d.CurrentPreset()
	if loadedPreset == nil {
		t.Fatal("CurrentPreset() should not be nil")
	}
	// The model field should be resolved from h: to f:
	if loadedPreset.Model != "f:/models/codellama.gguf" {
		t.Errorf("Preset.Model = %q, want %q", loadedPreset.Model, "f:/models/codellama.gguf")
	}
}

func TestDaemonRun_PresetWithHFModelNotFound(t *testing.T) {
	// Arrange - Preset with HF model that doesn't exist
	testPreset := &preset.Preset{
		Name:  "missing-preset",
		Model: "h:unknown/repo:Q4_K_M",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"missing-preset": testPreset,
		},
	}
	models := &stubModelManager{
		exists: false,
	}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:missing-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error when preset's HF model not found, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q on error", d.State(), StateIdle)
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called when model resolution fails")
	}
}

func TestDaemonRun_FailsToStopExistingModel(t *testing.T) {
	// Arrange
	firstPreset := &preset.Preset{
		Name:  "first-preset",
		Model: "f:/path/to/first.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}
	secondPreset := &preset.Preset{
		Name:  "second-preset",
		Model: "f:/path/to/second.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"first-preset":  firstPreset,
			"second-preset": secondPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	firstMockProc := &mockProcess{
		stopErr: fmt.Errorf("failed to stop"),
	}
	callCount := 0
	d.newProcess = func(path string) llamaProcess {
		callCount++
		if callCount == 1 {
			return firstMockProc
		}
		return &mockProcess{}
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	// Load first model
	err := d.Run(context.Background(), "p:first-preset")
	if err != nil {
		t.Fatalf("first Run() failed: %v", err)
	}

	// Try to load second model (should fail to stop first)
	err = d.Run(context.Background(), "p:second-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error when stopping existing model fails, got nil")
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q (should remain in previous state)", d.State(), StateRunning)
	}
	if d.CurrentPreset() != firstPreset {
		t.Error("CurrentPreset() should still be first preset after failed stop")
	}
}

func TestDaemonRun_ContextCancelledDuringHealthCheck(t *testing.T) {
	// Arrange
	testPreset := &preset.Preset{
		Name:  "test-preset",
		Model: "f:/path/to/model.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(context.Canceled)

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	err := d.Run(ctx, "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error when context is cancelled, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after context cancellation", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after context cancellation")
	}
	if mockProc.stopCalled {
		t.Log("Process.Stop() was called for cleanup (acceptable)")
	}
}

func TestResolveModel_FilePath(t *testing.T) {
	models := &stubModelManager{filePath: "/should/not/be/used"}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "f:/abs/path/model.gguf",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File path should remain unchanged
	if resolved.Model != "f:/abs/path/model.gguf" {
		t.Errorf("Model = %q, want %q", resolved.Model, "f:/abs/path/model.gguf")
	}
}

func TestResolveModel_HuggingFace(t *testing.T) {
	models := &stubModelManager{filePath: "/resolved/path/model.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "h:org/repo:Q4_K_M",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// HF format should be resolved to file path with f: prefix
	if resolved.Model != "f:/resolved/path/model.gguf" {
		t.Errorf("Model = %q, want %q", resolved.Model, "f:/resolved/path/model.gguf")
	}

	// Original preset should not be mutated
	if p.Model != "h:org/repo:Q4_K_M" {
		t.Errorf("Original preset mutated: Model = %q, want %q", p.Model, "h:org/repo:Q4_K_M")
	}
}

func TestResolveModel_HuggingFaceNotExists(t *testing.T) {
	models := &stubModelManager{exists: false}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "h:org/repo:Q4_K_M",
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveModel_InvalidIdentifier(t *testing.T) {
	models := &stubModelManager{}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "",
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error for empty model field, got nil")
	}
}

func TestResolveModel_OldFormatError(t *testing.T) {
	models := &stubModelManager{}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "org/repo:Q4_K_M", // Old format without h: prefix
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error for old format without prefix, got nil")
	}
}

func TestResolveModel_DraftModelFilePath(t *testing.T) {
	models := &stubModelManager{filePath: "/should/not/be/used"}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:       "test",
		Model:      "f:/abs/path/model.gguf",
		DraftModel: "f:/abs/path/draft.gguf",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.DraftModel != "f:/abs/path/draft.gguf" {
		t.Errorf("DraftModel = %q, want %q", resolved.DraftModel, "f:/abs/path/draft.gguf")
	}
}

func TestResolveModel_DraftModelHuggingFace(t *testing.T) {
	models := &stubModelManager{filePath: "/resolved/path/draft.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:       "test",
		Model:      "f:/abs/path/model.gguf",
		DraftModel: "h:org/draft-repo:Q4_K_M",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.DraftModel != "f:/resolved/path/draft.gguf" {
		t.Errorf("DraftModel = %q, want %q", resolved.DraftModel, "f:/resolved/path/draft.gguf")
	}

	// Original preset should not be mutated
	if p.DraftModel != "h:org/draft-repo:Q4_K_M" {
		t.Errorf("Original preset mutated: DraftModel = %q, want %q", p.DraftModel, "h:org/draft-repo:Q4_K_M")
	}
}

func TestResolveModel_DraftModelHuggingFaceNotExists(t *testing.T) {
	models := &stubModelManager{exists: false}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:       "test",
		Model:      "f:/abs/path/model.gguf",
		DraftModel: "h:org/draft-repo:Q4_K_M",
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error when draft model not found, got nil")
	}
}

func TestResolveModel_NoDraftModel(t *testing.T) {
	models := &stubModelManager{filePath: "/resolved/path/model.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "f:/abs/path/model.gguf",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.DraftModel != "" {
		t.Errorf("DraftModel = %q, want empty string", resolved.DraftModel)
	}
}

// mapModelManager resolves different models based on repo+quant key.
type mapModelManager struct {
	paths map[string]string // key: "repo:quant", value: file path
}

func (m *mapModelManager) List(ctx context.Context) ([]metadata.ModelEntry, error) {
	return nil, nil
}

func (m *mapModelManager) GetFilePath(ctx context.Context, repo, quant string) (string, error) {
	key := repo + ":" + quant
	path, ok := m.paths[key]
	if !ok {
		return "", &metadata.NotFoundError{Repo: repo, Quant: quant}
	}
	return path, nil
}

func TestDaemonRun_RouterModeSuccess(t *testing.T) {
	// Arrange
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "router-config.ini")

	routerPreset := &preset.Preset{
		Name: "multi-model",
		Mode: "router",
		Host: "127.0.0.1",
		Port: 8080,
		Models: []preset.ModelEntry{
			{Name: "codellama", Model: "f:/models/codellama.gguf", ContextSize: 4096},
			{Name: "mistral", Model: "f:/models/mistral.gguf"},
		},
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"multi-model": routerPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemonWithConfigPath(presets, models, configPath)

	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:multi-model")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}

	// Verify config.ini was written
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.ini not written: %v", err)
	}
	if !strings.Contains(string(content), "[codellama]") {
		t.Errorf("config.ini missing [codellama] section")
	}
	if !strings.Contains(string(content), "[mistral]") {
		t.Errorf("config.ini missing [mistral] section")
	}

	// Verify BuildRouterArgs was used (should contain --models-preset)
	foundModelsPreset := false
	for _, arg := range mockProc.receivedArgs {
		if arg == "--models-preset" {
			foundModelsPreset = true
			break
		}
	}
	if !foundModelsPreset {
		t.Errorf("args should contain --models-preset, got %v", mockProc.receivedArgs)
	}
}

func TestDaemonRun_RouterModeResolvesModels(t *testing.T) {
	// Arrange
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "router-config.ini")

	routerPreset := &preset.Preset{
		Name: "multi-model-hf",
		Mode: "router",
		Host: "127.0.0.1",
		Port: 8080,
		Models: []preset.ModelEntry{
			{Name: "codellama", Model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M"},
			{Name: "mistral", Model: "f:/models/mistral.gguf"},
		},
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"multi-model-hf": routerPreset,
		},
	}
	models := &mapModelManager{
		paths: map[string]string{
			"TheBloke/CodeLlama-7B-GGUF:Q4_K_M": "/resolved/codellama.gguf",
		},
	}
	d := newTestDaemonWithConfigPath(presets, models, configPath)

	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:multi-model-hf")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config.ini contains resolved path
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.ini not written: %v", err)
	}
	if !strings.Contains(string(content), "/resolved/codellama.gguf") {
		t.Errorf("config.ini should contain resolved path, got:\n%s", string(content))
	}

	// Original preset should not be mutated
	if routerPreset.Models[0].Model != "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M" {
		t.Errorf("original preset mutated: Models[0].Model = %q", routerPreset.Models[0].Model)
	}
}

func TestDaemonRun_RouterModeWriteConfigFails(t *testing.T) {
	// Arrange - use a non-existent directory so write fails
	configPath := "/nonexistent-dir/router-config.ini"

	routerPreset := &preset.Preset{
		Name: "multi-model",
		Mode: "router",
		Host: "127.0.0.1",
		Port: 8080,
		Models: []preset.ModelEntry{
			{Name: "codellama", Model: "f:/models/codellama.gguf"},
		},
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"multi-model": routerPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemonWithConfigPath(presets, models, configPath)

	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:multi-model")

	// Assert
	if err == nil {
		t.Fatal("expected error when config write fails, got nil")
	}
	if !strings.Contains(err.Error(), "write router config") {
		t.Errorf("error should mention write router config, got: %v", err)
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after config write failure", d.State(), StateIdle)
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called when config write fails")
	}
}

func TestResolveModel_RouterMode(t *testing.T) {
	// Arrange
	models := &mapModelManager{
		paths: map[string]string{
			"org/model-a:Q4_K_M": "/resolved/model-a.gguf",
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name: "router-test",
		Mode: "router",
		Models: []preset.ModelEntry{
			{Name: "model-a", Model: "h:org/model-a:Q4_K_M"},
			{Name: "model-b", Model: "f:/path/to/model-b.gguf"},
		},
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Models[0].Model != "f:/resolved/model-a.gguf" {
		t.Errorf("Models[0].Model = %q, want %q", resolved.Models[0].Model, "f:/resolved/model-a.gguf")
	}
	if resolved.Models[1].Model != "f:/path/to/model-b.gguf" {
		t.Errorf("Models[1].Model = %q, want %q", resolved.Models[1].Model, "f:/path/to/model-b.gguf")
	}

	// Original should not be mutated
	if p.Models[0].Model != "h:org/model-a:Q4_K_M" {
		t.Errorf("original preset mutated: Models[0].Model = %q", p.Models[0].Model)
	}
}

func TestResolveModel_RouterModeDraftModel(t *testing.T) {
	// Arrange
	models := &mapModelManager{
		paths: map[string]string{
			"org/draft-repo:Q4_K_M": "/resolved/draft.gguf",
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name: "router-draft-test",
		Mode: "router",
		Models: []preset.ModelEntry{
			{
				Name:       "model-a",
				Model:      "f:/path/to/model.gguf",
				DraftModel: "h:org/draft-repo:Q4_K_M",
			},
		},
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Models[0].DraftModel != "f:/resolved/draft.gguf" {
		t.Errorf("Models[0].DraftModel = %q, want %q", resolved.Models[0].DraftModel, "f:/resolved/draft.gguf")
	}

	// Original should not be mutated
	if p.Models[0].DraftModel != "h:org/draft-repo:Q4_K_M" {
		t.Errorf("original preset mutated: Models[0].DraftModel = %q", p.Models[0].DraftModel)
	}
}

func TestAtomicWriteFile(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.ini")
	content := "[codellama]\nmodel = /path/to/model.gguf\n"

	// Act
	err := atomicWriteFile(path, content)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(got) != content {
		t.Errorf("file content = %q, want %q", string(got), content)
	}
}

func TestDaemonKill_CleansUpConfigFile(t *testing.T) {
	// Arrange
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "router-config.ini")

	routerPreset := &preset.Preset{
		Name: "multi-model",
		Mode: "router",
		Host: "127.0.0.1",
		Port: 8080,
		Models: []preset.ModelEntry{
			{Name: "codellama", Model: "f:/models/codellama.gguf"},
		},
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"multi-model": routerPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemonWithConfigPath(presets, models, configPath)

	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Start router mode
	err := d.Run(context.Background(), "p:multi-model")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify config.ini exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config.ini should exist after Run()")
	}

	// Act
	err = d.Kill(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Kill() failed: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("config.ini should be cleaned up after Kill()")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after Kill()", d.State(), StateIdle)
	}
}

func TestFetchModelStatuses_Success(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "qwen3", "status": "loaded"},
				{"id": "gemma3", "status": "unloaded"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Parse srv.URL to get host and port
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, _ := strconv.Atoi(u.Port())

	d := newTestDaemon(&stubPresetLoader{}, &stubModelManager{})
	d.httpClient = srv.Client()
	// Set a router preset pointing to the mock server
	d.preset.Store(&preset.Preset{
		Mode: "router",
		Host: u.Hostname(),
		Port: port,
	})
	d.state.Store(StateRunning)

	// Act
	statuses := d.FetchModelStatuses(context.Background())

	// Assert
	if len(statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(statuses))
	}
	if statuses[0].ID != "qwen3" || statuses[0].Status != "loaded" {
		t.Errorf("statuses[0] = %+v, want {ID:qwen3, Status:loaded}", statuses[0])
	}
	if statuses[1].ID != "gemma3" || statuses[1].Status != "unloaded" {
		t.Errorf("statuses[1] = %+v, want {ID:gemma3, Status:unloaded}", statuses[1])
	}
}

func TestFetchModelStatuses_NonRouterReturnsNil(t *testing.T) {
	// Arrange
	d := newTestDaemon(&stubPresetLoader{}, &stubModelManager{})
	d.preset.Store(&preset.Preset{
		Mode:  "single",
		Model: "f:/path/to/model.gguf",
	})
	d.state.Store(StateRunning)

	// Act
	statuses := d.FetchModelStatuses(context.Background())

	// Assert
	if statuses != nil {
		t.Errorf("expected nil for non-router preset, got %v", statuses)
	}
}

func TestFetchModelStatuses_NoPresetReturnsNil(t *testing.T) {
	// Arrange
	d := newTestDaemon(&stubPresetLoader{}, &stubModelManager{})

	// Act
	statuses := d.FetchModelStatuses(context.Background())

	// Assert
	if statuses != nil {
		t.Errorf("expected nil when no preset loaded, got %v", statuses)
	}
}

func TestFetchModelStatuses_ServerErrorReturnsNil(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, _ := strconv.Atoi(u.Port())

	d := newTestDaemon(&stubPresetLoader{}, &stubModelManager{})
	d.httpClient = srv.Client()
	d.preset.Store(&preset.Preset{
		Mode: "router",
		Host: u.Hostname(),
		Port: port,
	})
	d.state.Store(StateRunning)

	// Act
	statuses := d.FetchModelStatuses(context.Background())

	// Assert
	if statuses != nil {
		t.Errorf("expected nil on server error, got %v", statuses)
	}
}
