package daemon

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/d2verb/alpaca/internal/config"
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
	return s.filePath, nil
}

func (s *stubModelManager) Exists(ctx context.Context, repo, quant string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.exists, nil
}

type stubPuller struct {
	pullCalled bool
	pullErr    error
}

func (s *stubPuller) Pull(ctx context.Context, repo, quant string) error {
	s.pullCalled = true
	return s.pullErr
}

func TestResolveHFPresetSuccess(t *testing.T) {
	models := &stubModelManager{filePath: "/path/to/model.gguf", exists: true}
	userCfg := &config.Config{
		DefaultHost:      "127.0.0.1",
		DefaultPort:      8080,
		DefaultCtxSize:   4096,
		DefaultGPULayers: -1,
	}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}

	d := New(cfg, &stubPresetLoader{}, models, nil, userCfg)

	p, err := d.resolveHFPreset(context.Background(), "TheBloke/CodeLlama-7B-GGUF", "Q4_K_M", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name != "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M" {
		t.Errorf("Name = %q, want %q", p.Name, "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}
	if p.Model != "f:/path/to/model.gguf" {
		t.Errorf("Model = %q, want %q", p.Model, "f:/path/to/model.gguf")
	}
	if p.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", p.Host, "127.0.0.1")
	}
	if p.Port != 8080 {
		t.Errorf("Port = %d, want %d", p.Port, 8080)
	}
	if p.ContextSize != 4096 {
		t.Errorf("ContextSize = %d, want %d", p.ContextSize, 4096)
	}
	if p.GPULayers != -1 {
		t.Errorf("GPULayers = %d, want %d", p.GPULayers, -1)
	}
}

func TestResolveHFPresetModelNotFound(t *testing.T) {
	models := &stubModelManager{exists: false}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	userCfg := config.DefaultConfig()

	d := New(cfg, &stubPresetLoader{}, models, nil, userCfg)

	_, err := d.resolveHFPreset(context.Background(), "unknown/repo", "Q4_K_M", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewDaemonStartsIdle(t *testing.T) {
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}

	d := New(cfg, presets, models, nil, config.DefaultConfig())

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
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}

	d := New(cfg, presets, models, nil, config.DefaultConfig())

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
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}

	d := New(cfg, presets, models, nil, config.DefaultConfig())

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
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}

	d := New(cfg, presets, models, nil, config.DefaultConfig())

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
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}

	d := New(cfg, presets, models, nil, config.DefaultConfig())

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
		GPULayers:   -1,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	cfg := &Config{
		LlamaServerPath: "/usr/local/bin/llama-server",
		SocketPath:      "/tmp/test.sock",
	}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil) // Success

	// Act
	err := d.Run(context.Background(), "p:test-preset", false)

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
	cfg := &Config{
		LlamaServerPath: "/usr/local/bin/llama-server",
		SocketPath:      "/tmp/test.sock",
	}
	userCfg := &config.Config{
		DefaultHost:      "0.0.0.0",
		DefaultPort:      9090,
		DefaultCtxSize:   8192,
		DefaultGPULayers: 35,
	}

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "f:/path/to/custom.gguf", false)

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
	if preset.Model != "f:/path/to/custom.gguf" {
		t.Errorf("Preset.Model = %q, want %q", preset.Model, "f:/path/to/custom.gguf")
	}
	if preset.Host != "0.0.0.0" {
		t.Errorf("Preset.Host = %q, want %q", preset.Host, "0.0.0.0")
	}
	if preset.Port != 9090 {
		t.Errorf("Preset.Port = %d, want %d", preset.Port, 9090)
	}
}

func TestDaemonRun_HuggingFaceSuccess(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		filePath: "/models/codellama-7b.Q4_K_M.gguf",
		exists:   true,
	}
	cfg := &Config{
		LlamaServerPath: "/usr/local/bin/llama-server",
		SocketPath:      "/tmp/test.sock",
	}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:nonexistent", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "h:unknown/repo:Q4_K_M", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{
		startErr: fmt.Errorf("failed to start process"),
	}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:test-preset", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(fmt.Errorf("health check timeout"))

	// Act
	err := d.Run(context.Background(), "p:test-preset", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

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
	err := d.Run(context.Background(), "p:first-preset", false)
	if err != nil {
		t.Fatalf("first Run() failed: %v", err)
	}

	// Load second model (should stop first)
	err = d.Run(context.Background(), "p:second-preset", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "invalid:format:too:many:colons", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Start a model first
	err := d.Run(context.Background(), "p:test-preset", false)
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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{
		stopErr: fmt.Errorf("failed to stop process"),
	}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Start a model first
	err := d.Run(context.Background(), "p:test-preset", false)
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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:codellama-preset", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:missing-preset", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

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
	err := d.Run(context.Background(), "p:first-preset", false)
	if err != nil {
		t.Fatalf("first Run() failed: %v", err)
	}

	// Try to load second model (should fail to stop first)
	err = d.Run(context.Background(), "p:second-preset", false)

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
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	d := New(cfg, presets, models, nil, userCfg)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(context.Canceled)

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	err := d.Run(ctx, "p:test-preset", false)

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
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, nil, config.DefaultConfig())

	p := &preset.Preset{
		Name:  "test",
		Model: "f:/abs/path/model.gguf",
	}

	resolved, err := d.resolveModel(context.Background(), p, false)
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
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, nil, config.DefaultConfig())

	p := &preset.Preset{
		Name:  "test",
		Model: "h:org/repo:Q4_K_M",
	}

	resolved, err := d.resolveModel(context.Background(), p, false)
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
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, nil, config.DefaultConfig())

	p := &preset.Preset{
		Name:  "test",
		Model: "h:org/repo:Q4_K_M",
	}

	_, err := d.resolveModel(context.Background(), p, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveModel_InvalidIdentifier(t *testing.T) {
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, nil, config.DefaultConfig())

	p := &preset.Preset{
		Name:  "test",
		Model: "",
	}

	_, err := d.resolveModel(context.Background(), p, false)
	if err == nil {
		t.Fatal("expected error for empty model field, got nil")
	}
}

func TestResolveModel_OldFormatError(t *testing.T) {
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, nil, config.DefaultConfig())

	p := &preset.Preset{
		Name:  "test",
		Model: "org/repo:Q4_K_M", // Old format without h: prefix
	}

	_, err := d.resolveModel(context.Background(), p, false)
	if err == nil {
		t.Fatal("expected error for old format without prefix, got nil")
	}
}

func TestEnsureModel_AutoPullWhenNotExists(t *testing.T) {
	models := &stubModelManager{exists: false, filePath: "/models/test.gguf"}
	puller := &stubPuller{}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, puller, config.DefaultConfig())

	err := d.ensureModel(context.Background(), "org/repo", "Q4_K_M", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !puller.pullCalled {
		t.Error("puller.Pull() should be called when autoPull=true and model not exists")
	}
}

func TestEnsureModel_NoPullWhenExists(t *testing.T) {
	models := &stubModelManager{exists: true, filePath: "/models/test.gguf"}
	puller := &stubPuller{}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, puller, config.DefaultConfig())

	err := d.ensureModel(context.Background(), "org/repo", "Q4_K_M", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if puller.pullCalled {
		t.Error("puller.Pull() should not be called when model exists")
	}
}

func TestEnsureModel_ErrorWhenNotExistsAndNoPull(t *testing.T) {
	models := &stubModelManager{exists: false}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}
	d := New(cfg, &stubPresetLoader{}, models, nil, config.DefaultConfig())

	err := d.ensureModel(context.Background(), "org/repo", "Q4_K_M", false)
	if err == nil {
		t.Fatal("expected error when model not exists and autoPull=false")
	}
}
