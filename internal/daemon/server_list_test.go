package daemon

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/protocol"
)

func TestHandleListPresets_Success(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		names: []string{"codellama", "mistral", "llama3"},
	}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

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
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

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
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	// Act
	resp := server.handleListModels(context.Background())

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}

	modelList, ok := resp.Data["models"].([]ModelInfo)
	if !ok {
		t.Fatal("models data should be []ModelInfo")
	}

	if len(modelList) != 2 {
		t.Errorf("len(models) = %d, want 2", len(modelList))
	}
	if modelList[0].Repo != "TheBloke/CodeLlama-7B-GGUF" {
		t.Errorf("models[0].Repo = %v, want %q", modelList[0].Repo, "TheBloke/CodeLlama-7B-GGUF")
	}
	if modelList[0].Quant != "Q4_K_M" {
		t.Errorf("models[0].Quant = %v, want %q", modelList[0].Quant, "Q4_K_M")
	}
	if modelList[0].Size != 4096000 {
		t.Errorf("models[0].Size = %v, want %d", modelList[0].Size, 4096000)
	}
}

func TestHandleListModels_Error(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		err: fmt.Errorf("failed to read metadata"),
	}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

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
