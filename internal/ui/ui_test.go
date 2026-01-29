package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestStatusBadge(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		state    string
		contains string
	}{
		{
			name:     "running state",
			state:    "running",
			contains: "● Running",
		},
		{
			name:     "loading state",
			state:    "loading",
			contains: "◐ Loading",
		},
		{
			name:     "idle state",
			state:    "idle",
			contains: "○ Idle",
		},
		{
			name:     "not running state",
			state:    "stopped",
			contains: "○ Not Running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StatusBadge(tt.state)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("StatusBadge(%q) = %q, want to contain %q", tt.state, result, tt.contains)
			}
		})
	}
}

func TestPrintStatus(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintStatus("running", "h:test/model:Q4_K_M", "http://localhost:8080", "/path/to/logs")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Status:") {
		t.Error("Output should contain 'Status:'")
	}
	if !strings.Contains(output, "● Running") {
		t.Error("Output should contain running badge")
	}
	if !strings.Contains(output, "Preset:") {
		t.Error("Output should contain 'Preset:'")
	}
	if !strings.Contains(output, "h:test/model:Q4_K_M") {
		t.Error("Output should contain preset name")
	}
	if !strings.Contains(output, "http://localhost:8080") {
		t.Error("Output should contain endpoint")
	}
	if !strings.Contains(output, "/path/to/logs") {
		t.Error("Output should contain log path")
	}
}

func TestPrintStatus_NoPreset(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintStatus("idle", "", "", "/path/to/logs")

	// Assert
	output := buf.String()
	if strings.Contains(output, "Preset:") {
		t.Error("Output should not contain 'Preset:' when empty")
	}
	if strings.Contains(output, "Endpoint:") {
		t.Error("Output should not contain 'Endpoint:' when empty")
	}
}

func TestPrintModelList(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	models := []ModelInfo{
		{Repo: "org/model1", Quant: "Q4_K_M", SizeString: "2.5 GB"},
		{Repo: "org/model2", Quant: "Q8_0", SizeString: "5.0 GB"},
	}

	// Act
	PrintModelList(models)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Downloaded models:") {
		t.Error("Output should contain header")
	}
	if !strings.Contains(output, "org/model1") {
		t.Error("Output should contain first model")
	}
	if !strings.Contains(output, "Q4_K_M") {
		t.Error("Output should contain first quant")
	}
	if !strings.Contains(output, "2.5 GB") {
		t.Error("Output should contain first size")
	}
	if !strings.Contains(output, "org/model2") {
		t.Error("Output should contain second model")
	}
}

func TestPrintModelList_Empty(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintModelList([]ModelInfo{})

	// Assert
	output := buf.String()
	if !strings.Contains(output, "No models downloaded") {
		t.Error("Output should contain empty message")
	}
}

func TestPrintPresetList(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	presets := []string{"preset1", "preset2"}

	// Act
	PrintPresetList(presets)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Available presets:") {
		t.Error("Output should contain header")
	}
	if !strings.Contains(output, "preset1") {
		t.Error("Output should contain first preset")
	}
	if !strings.Contains(output, "preset2") {
		t.Error("Output should contain second preset")
	}
}

func TestPrintPresetList_Empty(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintPresetList([]string{})

	// Assert
	output := buf.String()
	if !strings.Contains(output, "No presets available") {
		t.Error("Output should contain empty message")
	}
}

func TestPrintSuccess(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintSuccess("Operation completed")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Error("Output should contain checkmark")
	}
	if !strings.Contains(output, "Operation completed") {
		t.Error("Output should contain message")
	}
}

func TestPrintError(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintError("Something went wrong")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "✗") {
		t.Error("Output should contain X mark")
	}
	if !strings.Contains(output, "Something went wrong") {
		t.Error("Output should contain message")
	}
}

func TestPrintWarning(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintWarning("Be careful")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "⚠") {
		t.Error("Output should contain warning symbol")
	}
	if !strings.Contains(output, "Be careful") {
		t.Error("Output should contain message")
	}
}

func TestPrintInfo(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	// Act
	PrintInfo("Information")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "•") {
		t.Error("Output should contain bullet")
	}
	if !strings.Contains(output, "Information") {
		t.Error("Output should contain message")
	}
}

func TestPrintPresetDetails(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

	preset := PresetDetails{
		Name:        "my-preset",
		Model:       "h:org/model:Q4_K_M",
		ContextSize: 4096,
		GPULayers:   32,
		Threads:     8,
		Host:        "127.0.0.1",
		Port:        8080,
		ExtraArgs:   []string{"--flash-attn", "--verbose"},
	}

	// Act
	PrintPresetDetails(preset)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Name:") {
		t.Error("Output should contain 'Name:'")
	}
	if !strings.Contains(output, "my-preset") {
		t.Error("Output should contain preset name")
	}
	if !strings.Contains(output, "Model:") {
		t.Error("Output should contain 'Model:'")
	}
	if !strings.Contains(output, "h:org/model:Q4_K_M") {
		t.Error("Output should contain model")
	}
	if !strings.Contains(output, "Context Size:") {
		t.Error("Output should contain 'Context Size:'")
	}
	if !strings.Contains(output, "4096") {
		t.Error("Output should contain context size value")
	}
	if !strings.Contains(output, "GPU Layers:") {
		t.Error("Output should contain 'GPU Layers:'")
	}
	if !strings.Contains(output, "32") {
		t.Error("Output should contain gpu layers value")
	}
	if !strings.Contains(output, "Threads:") {
		t.Error("Output should contain 'Threads:'")
	}
	if !strings.Contains(output, "Endpoint:") {
		t.Error("Output should contain 'Endpoint:'")
	}
	if !strings.Contains(output, "127.0.0.1:8080") {
		t.Error("Output should contain endpoint")
	}
	if !strings.Contains(output, "Extra Args:") {
		t.Error("Output should contain 'Extra Args:'")
	}
	if !strings.Contains(output, "--flash-attn") {
		t.Error("Output should contain extra args")
	}
}

func TestPrintPresetDetails_Minimal(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

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
	if !strings.Contains(output, "Name:") {
		t.Error("Output should contain 'Name:'")
	}
	if !strings.Contains(output, "Model:") {
		t.Error("Output should contain 'Model:'")
	}
	if strings.Contains(output, "Context Size:") {
		t.Error("Output should not contain 'Context Size:' when zero")
	}
	if strings.Contains(output, "GPU Layers:") {
		t.Error("Output should not contain 'GPU Layers:' when zero")
	}
	if strings.Contains(output, "Threads:") {
		t.Error("Output should not contain 'Threads:' when zero")
	}
	if strings.Contains(output, "Extra Args:") {
		t.Error("Output should not contain 'Extra Args:' when empty")
	}
}
