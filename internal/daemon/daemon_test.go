package daemon

import (
	"context"
	"fmt"
	"io"
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
	return New(presets, models, io.Discard, io.Discard)
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
