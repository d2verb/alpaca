package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestPrintRouterStatus(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	models := []RouterModelInfo{
		{ID: "qwen3", Status: "loaded"},
		{ID: "nomic-embed", Status: "loaded"},
		{ID: "gemma3", Status: "unloaded"},
	}

	// Act
	PrintRouterStatus("running", "p:my-workspace", "http://127.0.0.1:8080", "~/.alpaca/logs/llama.log", models)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "🚀 Status") {
		t.Error("Output should contain status header")
	}
	if !strings.Contains(output, "● Running") {
		t.Error("Output should contain running badge")
	}
	if !strings.Contains(output, "p:my-workspace") {
		t.Error("Output should contain preset name")
	}
	if !strings.Contains(output, "Mode") {
		t.Error("Output should contain 'Mode' label")
	}
	if !strings.Contains(output, "router") {
		t.Error("Output should contain 'router' mode")
	}
	if !strings.Contains(output, "http://127.0.0.1:8080") {
		t.Error("Output should contain endpoint")
	}
	if !strings.Contains(output, "~/.alpaca/logs/llama.log") {
		t.Error("Output should contain log path")
	}
	if !strings.Contains(output, "Models (3)") {
		t.Error("Output should contain model count")
	}
	if !strings.Contains(output, "qwen3") {
		t.Error("Output should contain qwen3 model")
	}
	if !strings.Contains(output, "nomic-embed") {
		t.Error("Output should contain nomic-embed model")
	}
	if !strings.Contains(output, "gemma3") {
		t.Error("Output should contain gemma3 model")
	}
	if !strings.Contains(output, "● loaded") {
		t.Error("Output should contain loaded badge")
	}
	if !strings.Contains(output, "○ unloaded") {
		t.Error("Output should contain unloaded badge")
	}
}

func TestPrintRouterStatus_NoModels(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	// Act
	PrintRouterStatus("running", "p:test", "http://127.0.0.1:8080", "/log", nil)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "router") {
		t.Error("Output should contain 'router' mode")
	}
	if strings.Contains(output, "Models (") {
		t.Error("Output should not contain model section when no models")
	}
}

func TestPrintRouterStatus_WithMmproj(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	models := []RouterModelInfo{
		{ID: "gemma3-vision", Status: "loaded", Mmproj: "/path/to/mmproj.gguf"},
		{ID: "codellama", Status: "loaded"},
	}

	// Act
	PrintRouterStatus("running", "p:ws", "http://127.0.0.1:8080", "/log", models)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "gemma3-vision") {
		t.Error("Output should contain gemma3-vision model")
	}
	if !strings.Contains(output, "mmproj") {
		t.Error("Output should contain mmproj annotation for vision model")
	}
}

func TestPrintRouterPresetDetails(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	details := RouterPresetDetails{
		Name:        "my-workspace",
		Host:        "127.0.0.1",
		Port:        8080,
		MaxModels:   3,
		IdleTimeout: 300,
		Options: map[string]string{
			"flash-attn":   "on",
			"cache-type-k": "q8_0",
		},
		Models: []RouterModelDetail{
			{
				Name:       "qwen3",
				Model:      "h:Qwen/Qwen3-8B-GGUF",
				DraftModel: "h:Qwen/Qwen3-1B-GGUF",
				Options:    map[string]string{"ctx-size": "8192"},
			},
			{
				Name:  "nomic-embed",
				Model: "h:nomic-ai/nomic-embed-text-v2-moe-GGUF",
				Options: map[string]string{
					"ctx-size":   "2048",
					"embeddings": "true",
				},
			},
			{
				Name:  "gemma3",
				Model: "f:/path/to/gemma3.gguf",
				Options: map[string]string{
					"ctx-size": "4096",
					"threads":  "4",
				},
			},
		},
	}

	// Act
	PrintRouterPresetDetails(details)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "📦 Preset: p:my-workspace") {
		t.Error("Output should contain preset header")
	}
	if !strings.Contains(output, "router") {
		t.Error("Output should contain 'router' mode")
	}
	if !strings.Contains(output, "127.0.0.1:8080") {
		t.Error("Output should contain endpoint")
	}
	if !strings.Contains(output, "Max Models") {
		t.Error("Output should contain 'Max Models' label")
	}
	if !strings.Contains(output, "3") {
		t.Error("Output should contain max models value")
	}
	if !strings.Contains(output, "Idle Timeout") {
		t.Error("Output should contain 'Idle Timeout' label")
	}
	if !strings.Contains(output, "300s") {
		t.Error("Output should contain idle timeout value")
	}
	if !strings.Contains(output, "Options") {
		t.Error("Output should contain 'Options' label")
	}
	if !strings.Contains(output, "cache-type-k=q8_0") {
		t.Error("Output should contain cache-type-k option")
	}
	if !strings.Contains(output, "flash-attn=on") {
		t.Error("Output should contain flash-attn option")
	}
	if !strings.Contains(output, "Models (3)") {
		t.Error("Output should contain model count")
	}
	if !strings.Contains(output, "qwen3") {
		t.Error("Output should contain qwen3 model name")
	}
	if !strings.Contains(output, "h:Qwen/Qwen3-8B-GGUF") {
		t.Error("Output should contain qwen3 model path")
	}
	if !strings.Contains(output, "Draft Model") {
		t.Error("Output should contain 'Draft Model' label")
	}
	if !strings.Contains(output, "h:Qwen/Qwen3-1B-GGUF") {
		t.Error("Output should contain draft model path")
	}
	if !strings.Contains(output, "ctx-size=8192") {
		t.Error("Output should contain qwen3 ctx-size option")
	}
	if !strings.Contains(output, "nomic-embed") {
		t.Error("Output should contain nomic-embed model name")
	}
	if !strings.Contains(output, "embeddings=true") {
		t.Error("Output should contain per-model embeddings option")
	}
	if !strings.Contains(output, "gemma3") {
		t.Error("Output should contain gemma3 model name")
	}
	if !strings.Contains(output, "f:/path/to/gemma3.gguf") {
		t.Error("Output should contain gemma3 model path")
	}
	if !strings.Contains(output, "threads=4") {
		t.Error("Output should contain gemma3 threads option")
	}
}

func TestPrintRouterPresetDetails_Minimal(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	details := RouterPresetDetails{
		Name: "minimal",
		Host: "127.0.0.1",
		Port: 8080,
		Models: []RouterModelDetail{
			{
				Name:  "model1",
				Model: "h:org/model:Q4_K_M",
			},
		},
	}

	// Act
	PrintRouterPresetDetails(details)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "📦 Preset: p:minimal") {
		t.Error("Output should contain preset header")
	}
	if !strings.Contains(output, "router") {
		t.Error("Output should contain 'router' mode")
	}
	if strings.Contains(output, "Max Models") {
		t.Error("Output should not contain 'Max Models' when zero")
	}
	if strings.Contains(output, "Idle Timeout") {
		t.Error("Output should not contain 'Idle Timeout' when zero")
	}
	if strings.Contains(output, "Options") {
		t.Error("Output should not contain 'Options' when empty")
	}
	if strings.Contains(output, "Draft Model") {
		t.Error("Output should not contain 'Draft Model' when empty")
	}
}

func TestPrintRouterPresetDetails_WithMmproj(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	details := RouterPresetDetails{
		Name: "vision-workspace",
		Host: "127.0.0.1",
		Port: 8080,
		Models: []RouterModelDetail{
			{
				Name:   "gemma3-vision",
				Model:  "h:ggml-org/gemma-3-4b-it-GGUF:Q4_K_M",
				Mmproj: "f:/path/to/mmproj.gguf",
			},
			{
				Name:  "codellama",
				Model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M",
			},
		},
	}

	// Act
	PrintRouterPresetDetails(details)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Mmproj") {
		t.Error("Output should contain 'Mmproj' label for model with mmproj")
	}
	if !strings.Contains(output, "f:/path/to/mmproj.gguf") {
		t.Error("Output should contain mmproj value")
	}
}
