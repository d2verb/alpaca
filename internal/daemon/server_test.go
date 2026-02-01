package daemon

import (
	"context"
	"fmt"
	"testing"

	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/protocol"
)

func TestClassifyLoadError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantCode     string
		wantContains string
	}{
		{
			name:         "preset not found",
			err:          fmt.Errorf("load preset: %w", &preset.NotFoundError{Name: "test"}),
			wantCode:     protocol.ErrCodePresetNotFound,
			wantContains: "not found",
		},
		{
			name:         "model not found",
			err:          fmt.Errorf("resolve model: %w", &metadata.NotFoundError{Repo: "unknown", Quant: "Q4_K_M"}),
			wantCode:     protocol.ErrCodeModelNotFound,
			wantContains: "not found in metadata",
		},
		{
			name:         "server start failed",
			err:          &llama.ProcessError{Op: llama.ProcessOpStart, Err: fmt.Errorf("command not found")},
			wantCode:     protocol.ErrCodeServerFailed,
			wantContains: "start llama-server",
		},
		{
			name:         "server health check failed",
			err:          &llama.ProcessError{Op: llama.ProcessOpWait, Err: fmt.Errorf("timeout")},
			wantCode:     protocol.ErrCodeServerFailed,
			wantContains: "wait llama-server",
		},
		{
			name:         "unknown error",
			err:          fmt.Errorf("some other error"),
			wantCode:     "",
			wantContains: "some other error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			code, msg := classifyLoadError(tt.err)

			// Assert
			if code != tt.wantCode {
				t.Errorf("classifyLoadError() code = %q, want %q", code, tt.wantCode)
			}
			if msg != tt.err.Error() {
				t.Errorf("classifyLoadError() msg = %q, want %q", msg, tt.err.Error())
			}
		})
	}
}

func TestHandleStatus_Idle(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Act
	resp := server.handleStatus()

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
	if resp.Data["state"] != string(StateIdle) {
		t.Errorf("state = %v, want %q", resp.Data["state"], StateIdle)
	}
	if _, exists := resp.Data["preset"]; exists {
		t.Error("preset should not exist when idle")
	}
	if _, exists := resp.Data["endpoint"]; exists {
		t.Error("endpoint should not exist when idle")
	}
}

func TestHandleStatus_Running(t *testing.T) {
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

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Mock dependencies to allow Run to succeed
	mockProc := &mockProcess{}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	// Load a preset to make daemon running
	err := daemon.Run(context.Background(), "p:test-preset", false)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	resp := server.handleStatus()

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
	if resp.Data["state"] != string(StateRunning) {
		t.Errorf("state = %v, want %q", resp.Data["state"], StateRunning)
	}
	if resp.Data["preset"] != "test-preset" {
		t.Errorf("preset = %v, want %q", resp.Data["preset"], "test-preset")
	}
	if resp.Data["endpoint"] != "http://127.0.0.1:8080" {
		t.Errorf("endpoint = %v, want %q", resp.Data["endpoint"], "http://127.0.0.1:8080")
	}
}

func TestHandleLoad_Success(t *testing.T) {
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

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Mock dependencies
	mockProc := &mockProcess{}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "p:test-preset",
		},
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
	if resp.Data["endpoint"] != "http://127.0.0.1:8080" {
		t.Errorf("endpoint = %v, want %q", resp.Data["endpoint"], "http://127.0.0.1:8080")
	}
}

func TestHandleLoad_MissingIdentifier(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args:    map[string]any{}, // No identifier
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "identifier required" {
		t.Errorf("Error = %q, want %q", resp.Error, "identifier required")
	}
}

func TestHandleLoad_PresetNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{},
	}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "p:nonexistent",
		},
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.ErrorCode != protocol.ErrCodePresetNotFound {
		t.Errorf("ErrorCode = %q, want %q", resp.ErrorCode, protocol.ErrCodePresetNotFound)
	}
}

func TestHandleLoad_ModelNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		err: &metadata.NotFoundError{Repo: "unknown", Quant: "Q4_K_M"},
	}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "h:unknown/repo:Q4_K_M",
		},
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.ErrorCode != protocol.ErrCodeModelNotFound {
		t.Errorf("ErrorCode = %q, want %q", resp.ErrorCode, protocol.ErrCodeModelNotFound)
	}
}

func TestHandleLoad_ServerStartFailed(t *testing.T) {
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

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Mock process that fails to start
	mockProc := &mockProcess{
		startErr: fmt.Errorf("command not found"),
	}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "p:test-preset",
		},
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.ErrorCode != protocol.ErrCodeServerFailed {
		t.Errorf("ErrorCode = %q, want %q", resp.ErrorCode, protocol.ErrCodeServerFailed)
	}
}

func TestHandleUnload_Success(t *testing.T) {
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

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Mock dependencies and start a model first
	mockProc := &mockProcess{}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	err := daemon.Run(context.Background(), "p:test-preset", false)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	resp := server.handleUnload(context.Background())

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
	if daemon.State() != StateIdle {
		t.Errorf("daemon state = %q, want %q after unload", daemon.State(), StateIdle)
	}
}

func TestHandleUnload_WhenIdle(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Act
	resp := server.handleUnload(context.Background())

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleUnload_Error(t *testing.T) {
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

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Mock process that fails to stop
	mockProc := &mockProcess{
		stopErr: fmt.Errorf("failed to stop process"),
	}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	err := daemon.Run(context.Background(), "p:test-preset", false)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	resp := server.handleUnload(context.Background())

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "failed to stop process" {
		t.Errorf("Error = %q, want %q", resp.Error, "failed to stop process")
	}
}

func TestHandleListPresets_Success(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		names: []string{"codellama", "mistral", "llama3"},
	}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Act
	resp := server.handleListPresets()

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}

	presetList, ok := resp.Data["presets"].([]string)
	if !ok {
		t.Fatal("presets data should be []string")
	}

	if len(presetList) != 3 {
		t.Errorf("len(presets) = %d, want 3", len(presetList))
	}
	if presetList[0] != "codellama" {
		t.Errorf("presets[0] = %q, want %q", presetList[0], "codellama")
	}
}

func TestHandleListPresets_Error(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		listErr: fmt.Errorf("failed to read directory"),
	}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Act
	resp := server.handleListPresets()

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "failed to read directory" {
		t.Errorf("Error = %q, want %q", resp.Error, "failed to read directory")
	}
}

func TestHandleListModels_Success(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		entries: []metadata.ModelEntry{
			{Repo: "TheBloke/CodeLlama-7B-GGUF", Quant: "Q4_K_M", Size: 4096000},
			{Repo: "TheBloke/Mistral-7B-GGUF", Quant: "Q5_K_M", Size: 5242880},
		},
	}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Act
	resp := server.handleListModels(context.Background())

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}

	modelList, ok := resp.Data["models"].([]map[string]any)
	if !ok {
		t.Fatal("models data should be []map[string]any")
	}

	if len(modelList) != 2 {
		t.Errorf("len(models) = %d, want 2", len(modelList))
	}
	if modelList[0]["repo"] != "TheBloke/CodeLlama-7B-GGUF" {
		t.Errorf("models[0].repo = %v, want %q", modelList[0]["repo"], "TheBloke/CodeLlama-7B-GGUF")
	}
	if modelList[0]["quant"] != "Q4_K_M" {
		t.Errorf("models[0].quant = %v, want %q", modelList[0]["quant"], "Q4_K_M")
	}
	if modelList[0]["size"] != int64(4096000) {
		t.Errorf("models[0].size = %v, want %d", modelList[0]["size"], 4096000)
	}
}

func TestHandleListModels_Error(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		err: fmt.Errorf("failed to read metadata"),
	}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Act
	resp := server.handleListModels(context.Background())

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "failed to read metadata" {
		t.Errorf("Error = %q, want %q", resp.Error, "failed to read metadata")
	}
}

func TestHandleRequest_Status(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{Command: protocol.CmdStatus}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_Load(t *testing.T) {
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

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	// Mock dependencies
	mockProc := &mockProcess{}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "p:test-preset",
		},
	}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_Unload(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{Command: protocol.CmdUnload}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_ListPresets(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		names: []string{"test"},
	}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{Command: protocol.CmdListPresets}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_ListModels(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{Command: protocol.CmdListModels}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_UnknownCommand(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "/usr/local/bin/llama-server"}
	userCfg := config.DefaultConfig()

	daemon := New(cfg, presets, models, nil, userCfg)
	server := NewServer(daemon, "/tmp/test.sock")

	req := &protocol.Request{Command: "unknown_command"}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "unknown command" {
		t.Errorf("Error = %q, want %q", resp.Error, "unknown command")
	}
}
