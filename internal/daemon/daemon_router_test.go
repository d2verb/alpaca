package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/d2verb/alpaca/internal/preset"
)

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
			{Name: "codellama", Model: "f:/models/codellama.gguf", Options: preset.Options{"ctx-size": "4096"}},
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
				{"id": "qwen3", "status": map[string]any{"value": "loaded"}},
				{"id": "gemma3", "status": map[string]any{"value": "unloaded"}},
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
	if statuses[0].ID != "qwen3" || statuses[0].Status.Value != "loaded" {
		t.Errorf("statuses[0] = {ID:%s, Status:%s}, want {ID:qwen3, Status:loaded}", statuses[0].ID, statuses[0].Status.Value)
	}
	if statuses[1].ID != "gemma3" || statuses[1].Status.Value != "unloaded" {
		t.Errorf("statuses[1] = {ID:%s, Status:%s}, want {ID:gemma3, Status:unloaded}", statuses[1].ID, statuses[1].Status.Value)
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
