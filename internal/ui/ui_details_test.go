package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestPrintPresetDetails(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	preset := PresetDetails{
		Name:  "my-preset",
		Model: "h:org/model:Q4_K_M",
		Host:  "127.0.0.1",
		Port:  8080,
		Options: map[string]string{
			"ctx-size":   "4096",
			"flash-attn": "on",
		},
	}

	// Act
	PrintPresetDetails(preset)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "📦 Preset: p:my-preset") {
		t.Error("Output should contain '📦 Preset: p:my-preset' header")
	}
	if !strings.Contains(output, "Model") {
		t.Error("Output should contain 'Model' label")
	}
	if !strings.Contains(output, "h:org/model:Q4_K_M") {
		t.Error("Output should contain model")
	}
	if !strings.Contains(output, "Options") {
		t.Error("Output should contain 'Options' label")
	}
	if !strings.Contains(output, "ctx-size=4096") {
		t.Error("Output should contain ctx-size option")
	}
	if !strings.Contains(output, "flash-attn=on") {
		t.Error("Output should contain flash-attn option")
	}
	if !strings.Contains(output, "Endpoint") {
		t.Error("Output should contain 'Endpoint' label")
	}
	if !strings.Contains(output, "127.0.0.1:8080") {
		t.Error("Output should contain endpoint")
	}
}

func TestPrintPresetDetails_Minimal(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	preset := PresetDetails{
		Name:  "minimal",
		Model: "f:/path/to/model.gguf",
		Host:  "localhost",
		Port:  9000,
	}

	// Act
	PrintPresetDetails(preset)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "📦 Preset: p:minimal") {
		t.Error("Output should contain preset header")
	}
	if !strings.Contains(output, "Model") {
		t.Error("Output should contain 'Model' label")
	}
	if strings.Contains(output, "Options") {
		t.Error("Output should not contain 'Options' label when empty")
	}
}

func TestPrintPresetDetails_WithMmproj(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	preset := PresetDetails{
		Name:   "vision-preset",
		Model:  "h:org/model:Q4_K_M",
		Mmproj: "f:/path/to/mmproj.gguf",
		Host:   "127.0.0.1",
		Port:   8080,
	}

	// Act
	PrintPresetDetails(preset)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Mmproj") {
		t.Error("Output should contain 'Mmproj' label")
	}
	if !strings.Contains(output, "f:/path/to/mmproj.gguf") {
		t.Error("Output should contain mmproj value")
	}
}

func TestPrintModelDetails(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	model := ModelDetails{
		Repo:         "org/model",
		Quant:        "Q4_K_M",
		Filename:     "model.Q4_K_M.gguf",
		Path:         "/path/to/model.gguf",
		Size:         "4.2 GB",
		DownloadedAt: "2024-01-15 10:30:00",
	}

	// Act
	PrintModelDetails(model)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "🤖 Model: h:org/model:Q4_K_M") {
		t.Error("Output should contain '🤖 Model: h:org/model:Q4_K_M' header")
	}
	if !strings.Contains(output, "Filename") {
		t.Error("Output should contain 'Filename' label")
	}
	if !strings.Contains(output, "model.Q4_K_M.gguf") {
		t.Error("Output should contain filename")
	}
	if !strings.Contains(output, "Path") {
		t.Error("Output should contain 'Path' label")
	}
	if !strings.Contains(output, "/path/to/model.gguf") {
		t.Error("Output should contain path")
	}
	if !strings.Contains(output, "Size") {
		t.Error("Output should contain 'Size' label")
	}
	if !strings.Contains(output, "4.2 GB") {
		t.Error("Output should contain size")
	}
	if !strings.Contains(output, "Downloaded") {
		t.Error("Output should contain 'Downloaded' label")
	}
	if !strings.Contains(output, "2024-01-15 10:30:00") {
		t.Error("Output should contain download time")
	}
	if !strings.Contains(output, "Status") {
		t.Error("Output should contain 'Status' label")
	}
	if !strings.Contains(output, "✓ Ready") {
		t.Error("Output should contain '✓ Ready' status")
	}
	if strings.Contains(output, "Mmproj") {
		t.Error("Output should not contain 'Mmproj' when empty")
	}
}

func TestPrintModelDetails_WithMmproj(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	model := ModelDetails{
		Repo:         "org/vision-model",
		Quant:        "Q4_K_M",
		Filename:     "model.Q4_K_M.gguf",
		Path:         "/path/to/model.gguf",
		Size:         "2.5 GB",
		DownloadedAt: "2024-01-15 10:30:00",
		Mmproj:       "org_vision-model_mmproj-model-f16.gguf (851.0 MB)",
	}

	// Act
	PrintModelDetails(model)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Mmproj") {
		t.Error("Output should contain 'Mmproj' label")
	}
	if !strings.Contains(output, "org_vision-model_mmproj-model-f16.gguf (851.0 MB)") {
		t.Error("Output should contain mmproj info")
	}
}
