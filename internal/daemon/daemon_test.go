package daemon

import (
	"context"
	"fmt"
	"testing"

	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
)

type stubPresetLoader struct {
	presets map[string]*preset.Preset
	names   []string
}

func (s *stubPresetLoader) Load(name string) (*preset.Preset, error) {
	p, ok := s.presets[name]
	if !ok {
		return nil, fmt.Errorf("preset %s not found", name)
	}
	return p, nil
}

func (s *stubPresetLoader) List() ([]string, error) {
	return s.names, nil
}

type stubModelManager struct {
	entries  []metadata.ModelEntry
	filePath string
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

func TestResolveHFPresetSuccess(t *testing.T) {
	models := &stubModelManager{filePath: "/path/to/model.gguf"}
	cfg := &config.Config{
		DefaultHost:      "127.0.0.1",
		DefaultPort:      8080,
		DefaultCtxSize:   4096,
		DefaultGPULayers: -1,
	}

	p, err := resolveHFPreset(context.Background(), models, cfg, "TheBloke/CodeLlama-7B-GGUF", "Q4_K_M")
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
	models := &stubModelManager{err: fmt.Errorf("model not found")}
	cfg := config.DefaultConfig()

	_, err := resolveHFPreset(context.Background(), models, cfg, "unknown/repo", "Q4_K_M")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewDaemonStartsIdle(t *testing.T) {
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	cfg := &Config{LlamaServerPath: "llama-server", SocketPath: "/tmp/test.sock"}

	d := New(cfg, presets, models, config.DefaultConfig())

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

	d := New(cfg, presets, models, config.DefaultConfig())

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

	d := New(cfg, presets, models, config.DefaultConfig())

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

