package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/ui"
	"github.com/fatih/color"
)

func TestShowCmd_InvalidIdentifierType(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantErr    string
	}{
		{
			"file path identifier",
			"f:/path/to/model.gguf",
			"cannot show file details",
		},
		{
			"invalid identifier",
			"invalid",
			"invalid identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cmd := &ShowCmd{Identifier: tt.identifier}

			// Act
			err := cmd.Run()

			// Assert
			if err == nil {
				t.Fatal("expected error for invalid identifier")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestShowCmd_RouterPreset(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange: create a temp preset dir with a router preset
	tmpDir := t.TempDir()
	p := &preset.Preset{
		Name:        "my-workspace",
		Mode:        "router",
		Port:        8080,
		MaxModels:   3,
		IdleTimeout: 300,
		Options: preset.Options{
			"flash-attn": "on",
		},
		Models: []preset.ModelEntry{
			{
				Name:       "qwen3",
				Model:      "h:Qwen/Qwen3-8B-GGUF",
				DraftModel: "h:Qwen/Qwen3-1B-GGUF",
				Options:    preset.Options{"ctx-size": "8192"},
			},
			{
				Name:  "nomic-embed",
				Model: "h:nomic-ai/nomic-embed-text-v2-moe-GGUF",
				Options: preset.Options{
					"ctx-size":   "2048",
					"embeddings": "true",
				},
			},
		},
	}

	presetPath := filepath.Join(tmpDir, "test.yaml")
	if err := preset.WriteFile(presetPath, p); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	// Act
	cmd := &ShowCmd{}
	err := cmd.showPreset("my-workspace", tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "p:my-workspace") {
		t.Error("Output should contain preset name")
	}
	if !strings.Contains(output, "router") {
		t.Error("Output should contain 'router' mode")
	}
	if !strings.Contains(output, "Max Models") {
		t.Error("Output should contain 'Max Models'")
	}
	if !strings.Contains(output, "3") {
		t.Error("Output should contain max models value")
	}
	if !strings.Contains(output, "300s") {
		t.Error("Output should contain idle timeout value")
	}
	if !strings.Contains(output, "flash-attn=on") {
		t.Error("Output should contain server options")
	}
	if !strings.Contains(output, "qwen3") {
		t.Error("Output should contain qwen3 model")
	}
	if !strings.Contains(output, "nomic-embed") {
		t.Error("Output should contain nomic-embed model")
	}
	if !strings.Contains(output, "Draft Model") {
		t.Error("Output should contain 'Draft Model' for qwen3")
	}
}

func TestShowCmd_SinglePresetWithDraftModel(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange: create a single mode preset with draft-model
	tmpDir := t.TempDir()
	p := &preset.Preset{
		Name:       "with-draft",
		Model:      "h:org/model:Q4_K_M",
		DraftModel: "h:org/draft:Q4_K_M",
	}

	presetPath := filepath.Join(tmpDir, "draft.yaml")
	if err := preset.WriteFile(presetPath, p); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	// Act
	cmd := &ShowCmd{}
	err := cmd.showPreset("with-draft", tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Draft Model") {
		t.Error("Output should contain 'Draft Model' label")
	}
	if !strings.Contains(output, "h:org/draft:Q4_K_M") {
		t.Error("Output should contain draft model path")
	}
}

func TestShowCmd_SinglePresetStillWorks(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange: create a single mode preset
	tmpDir := t.TempDir()
	p := &preset.Preset{
		Name:  "my-single",
		Model: "h:org/model:Q4_K_M",
	}

	presetPath := filepath.Join(tmpDir, "single.yaml")
	if err := preset.WriteFile(presetPath, p); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	// Act
	cmd := &ShowCmd{}
	err := cmd.showPreset("my-single", tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "p:my-single") {
		t.Error("Output should contain preset name")
	}
	if !strings.Contains(output, "h:org/model:Q4_K_M") {
		t.Error("Output should contain model")
	}
	if strings.Contains(output, "router") {
		t.Error("Output should not contain 'router' for single mode preset")
	}
}

func TestFormatMmprojDetail(t *testing.T) {
	tests := []struct {
		name   string
		mmproj *metadata.MmprojEntry
		want   string
	}{
		{
			name:   "nil mmproj returns empty",
			mmproj: nil,
			want:   "",
		},
		{
			name:   "mmproj with filename and size",
			mmproj: &metadata.MmprojEntry{Filename: "org_model_mmproj-f16.gguf", Size: 892403712},
			want:   "org_model_mmproj-f16.gguf (851.1 MB)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMmprojDetail(tt.mmproj)
			if got != tt.want {
				t.Errorf("formatMmprojDetail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShowCmd_SinglePresetWithMmproj(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	tmpDir := t.TempDir()
	p := &preset.Preset{
		Name:   "with-mmproj",
		Model:  "h:org/model:Q4_K_M",
		Mmproj: "f:/path/to/mmproj.gguf",
	}

	presetPath := filepath.Join(tmpDir, "mmproj.yaml")
	if err := preset.WriteFile(presetPath, p); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	// Act
	cmd := &ShowCmd{}
	err := cmd.showPreset("with-mmproj", tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Mmproj") {
		t.Error("Output should contain 'Mmproj' label")
	}
	if !strings.Contains(output, "f:/path/to/mmproj.gguf") {
		t.Error("Output should contain mmproj value")
	}
}

func TestShowCmd_RouterPresetWithMmproj(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	tmpDir := t.TempDir()
	p := &preset.Preset{
		Name: "router-mmproj",
		Mode: "router",
		Port: 8080,
		Models: []preset.ModelEntry{
			{
				Name:   "gemma3-vision",
				Model:  "h:ggml-org/gemma-3-4b-it-GGUF",
				Mmproj: "f:/path/to/mmproj.gguf",
			},
			{
				Name:  "codellama",
				Model: "h:TheBloke/CodeLlama-7B-GGUF",
			},
		},
	}

	presetPath := filepath.Join(tmpDir, "router-mmproj.yaml")
	if err := preset.WriteFile(presetPath, p); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	// Act
	cmd := &ShowCmd{}
	err := cmd.showPreset("router-mmproj", tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Mmproj") {
		t.Error("Output should contain 'Mmproj' label for model with mmproj")
	}
	if !strings.Contains(output, "f:/path/to/mmproj.gguf") {
		t.Error("Output should contain mmproj value")
	}
}
