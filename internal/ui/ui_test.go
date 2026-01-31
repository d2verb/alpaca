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
	tests := []struct {
		name           string
		preset         string
		expectContains string
		expectLabel    string
	}{
		{
			name:           "preset without prefix",
			preset:         "test-preset",
			expectContains: "p:test-preset",
			expectLabel:    "Preset",
		},
		{
			name:           "preset with p: prefix",
			preset:         "p:codellama",
			expectContains: "p:codellama",
			expectLabel:    "Preset",
		},
		{
			name:           "HuggingFace identifier",
			preset:         "h:unsloth/gemma-3-4b-it-GGUF:Q4_K_M",
			expectContains: "h:unsloth/gemma-3-4b-it-GGUF:Q4_K_M",
			expectLabel:    "Model",
		},
		{
			name:           "file path identifier",
			preset:         "f:/path/to/model.gguf",
			expectContains: "f:/path/to/model.gguf",
			expectLabel:    "Model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable color for testing
			color.NoColor = true
			defer func() { color.NoColor = false }()

			// Arrange
			var buf bytes.Buffer
			Output = &buf
			defer func() { Output = nil }()

			// Act
			PrintStatus("running", tt.preset, "http://localhost:8080", "/path/to/llama.log")

			// Assert
			output := buf.String()
			if !strings.Contains(output, "🚀 Status") {
				t.Error("Output should contain '🚀 Status' header")
			}
			if !strings.Contains(output, "● Running") {
				t.Error("Output should contain running badge")
			}
			if !strings.Contains(output, "State") {
				t.Error("Output should contain 'State' label")
			}
			if !strings.Contains(output, tt.expectLabel) {
				t.Errorf("Output should contain '%s' label, got: %s", tt.expectLabel, output)
			}
			if !strings.Contains(output, tt.expectContains) {
				t.Errorf("Output should contain %q, got: %s", tt.expectContains, output)
			}
			if !strings.Contains(output, "http://localhost:8080") {
				t.Error("Output should contain endpoint")
			}
			if !strings.Contains(output, "Logs") {
				t.Error("Output should contain 'Logs' label")
			}
			if !strings.Contains(output, "/path/to/llama.log") {
				t.Error("Output should contain log path")
			}
		})
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
	PrintStatus("idle", "", "", "/path/to/llama.log")

	// Assert
	output := buf.String()
	if strings.Contains(output, "Preset") {
		t.Error("Output should not contain 'Preset' label when empty")
	}
	if strings.Contains(output, "Model") {
		t.Error("Output should not contain 'Model' label when empty")
	}
	if strings.Contains(output, "Endpoint") {
		t.Error("Output should not contain 'Endpoint' label when empty")
	}
	if !strings.Contains(output, "Logs") {
		t.Error("Output should contain 'Logs' label")
	}
	if !strings.Contains(output, "/path/to/llama.log") {
		t.Error("Output should contain log path")
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
		{Repo: "org/model1", Quant: "Q4_K_M", SizeString: "2.5 GB", DownloadedAt: "2024-01-15"},
		{Repo: "org/model2", Quant: "Q8_0", SizeString: "5.0 GB", DownloadedAt: "2024-01-16"},
	}

	// Act
	PrintModelList(models)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "🤖 Models") {
		t.Error("Output should contain header with icon")
	}
	if !strings.Contains(output, "h:org/model1:Q4_K_M") {
		t.Error("Output should contain first model with h: prefix and quant")
	}
	if !strings.Contains(output, "2.5 GB") {
		t.Error("Output should contain first size")
	}
	if !strings.Contains(output, "Downloaded 2024-01-15") {
		t.Error("Output should contain download date")
	}
	if !strings.Contains(output, "h:org/model2:Q8_0") {
		t.Error("Output should contain second model with h: prefix and quant")
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
	if !strings.Contains(output, "📦 Presets") {
		t.Error("Output should contain header with icon")
	}
	if !strings.Contains(output, "p:preset1") {
		t.Error("Output should contain first preset with p: prefix")
	}
	if !strings.Contains(output, "p:preset2") {
		t.Error("Output should contain second preset with p: prefix")
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
	if !strings.Contains(output, "ℹ") {
		t.Error("Output should contain info icon")
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
	if !strings.Contains(output, "📦 Preset: p:my-preset") {
		t.Error("Output should contain '📦 Preset: p:my-preset' header")
	}
	if !strings.Contains(output, "Model") {
		t.Error("Output should contain 'Model' label")
	}
	if !strings.Contains(output, "h:org/model:Q4_K_M") {
		t.Error("Output should contain model")
	}
	if !strings.Contains(output, "Context Size") {
		t.Error("Output should contain 'Context Size' label")
	}
	if !strings.Contains(output, "4096") {
		t.Error("Output should contain context size value")
	}
	if !strings.Contains(output, "GPU Layers") {
		t.Error("Output should contain 'GPU Layers' label")
	}
	if !strings.Contains(output, "32") {
		t.Error("Output should contain gpu layers value")
	}
	if !strings.Contains(output, "Threads") {
		t.Error("Output should contain 'Threads' label")
	}
	if !strings.Contains(output, "Endpoint") {
		t.Error("Output should contain 'Endpoint' label")
	}
	if !strings.Contains(output, "127.0.0.1:8080") {
		t.Error("Output should contain endpoint")
	}
	if !strings.Contains(output, "Extra Args") {
		t.Error("Output should contain 'Extra Args' label")
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
	if !strings.Contains(output, "📦 Preset: p:minimal") {
		t.Error("Output should contain preset header")
	}
	if !strings.Contains(output, "Model") {
		t.Error("Output should contain 'Model' label")
	}
	if strings.Contains(output, "Context Size") {
		t.Error("Output should not contain 'Context Size' label when zero")
	}
	if strings.Contains(output, "GPU Layers") {
		t.Error("Output should not contain 'GPU Layers' label when zero")
	}
	if strings.Contains(output, "Threads") {
		t.Error("Output should not contain 'Threads' label when zero")
	}
	if strings.Contains(output, "Extra Args") {
		t.Error("Output should not contain 'Extra Args' label when empty")
	}
}

func TestFormatEndpoint(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Act
	result := FormatEndpoint("http://localhost:8080")

	// Assert
	if !strings.Contains(result, "http://localhost:8080") {
		t.Errorf("FormatEndpoint should contain the endpoint, got: %q", result)
	}
}

func TestFormatPresetOrModel(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name         string
		input        string
		wantLabel    string
		wantContains []string
	}{
		{
			name:         "HuggingFace identifier with quant",
			input:        "h:org/repo:Q4_K_M",
			wantLabel:    "Model",
			wantContains: []string{"h:", "org/repo", "Q4_K_M"},
		},
		{
			name:         "HuggingFace identifier without quant",
			input:        "h:org/repo",
			wantLabel:    "Model",
			wantContains: []string{"h:", "org/repo"},
		},
		{
			name:         "HuggingFace identifier with multiple colons",
			input:        "h:org:special/repo:Q4_K_M",
			wantLabel:    "Model",
			wantContains: []string{"h:", "org:special/repo", "Q4_K_M"},
		},
		{
			name:         "preset identifier",
			input:        "p:my-preset",
			wantLabel:    "Preset",
			wantContains: []string{"p:", "my-preset"},
		},
		{
			name:         "file path identifier",
			input:        "f:/path/to/model.gguf",
			wantLabel:    "Model",
			wantContains: []string{"f:", "/path/to/model.gguf"},
		},
		{
			name:         "no prefix - treated as preset",
			input:        "just-a-name",
			wantLabel:    "Preset",
			wantContains: []string{"p:", "just-a-name"},
		},
		{
			name:         "unknown prefix - treated as preset",
			input:        "x:something",
			wantLabel:    "Preset",
			wantContains: []string{"p:", "x:something"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, formatted := formatPresetOrModel(tt.input)

			if label != tt.wantLabel {
				t.Errorf("formatPresetOrModel(%q) label = %q, want %q", tt.input, label, tt.wantLabel)
			}

			for _, expected := range tt.wantContains {
				if !strings.Contains(formatted, expected) {
					t.Errorf("formatPresetOrModel(%q) formatted = %q, should contain %q", tt.input, formatted, expected)
				}
			}
		})
	}
}

func TestPrintModelDetails(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = nil }()

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
}
