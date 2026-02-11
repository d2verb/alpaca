package ui

import (
	"bytes"
	"os"
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
			defer func() { Output = os.Stdout }()

			// Act
			PrintStatus("running", tt.preset, "http://localhost:8080", "/path/to/llama.log", "")

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
	defer func() { Output = os.Stdout }()

	// Act
	PrintStatus("idle", "", "", "/path/to/llama.log", "")

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
	defer func() { Output = os.Stdout }()

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
	defer func() { Output = os.Stdout }()

	// Act
	PrintModelList([]ModelInfo{})

	// Assert
	output := buf.String()
	if !strings.Contains(output, "🤖 Models") {
		t.Error("Output should contain header even when empty")
	}
	if !strings.Contains(output, "(none)") {
		t.Error("Output should contain '(none)' when empty")
	}
}

func TestPrintPresetList(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

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
	defer func() { Output = os.Stdout }()

	// Act
	PrintPresetList([]string{})

	// Assert
	output := buf.String()
	if !strings.Contains(output, "📦 Presets") {
		t.Error("Output should contain header even when empty")
	}
	if !strings.Contains(output, "(none)") {
		t.Error("Output should contain '(none)' when empty")
	}
}

func TestPrintSuccess(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

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
	defer func() { Output = os.Stdout }()

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
	defer func() { Output = os.Stdout }()

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
	defer func() { Output = os.Stdout }()

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

func TestModelStatusBadge(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		status   string
		contains string
	}{
		{
			name:     "loaded status",
			status:   "loaded",
			contains: "● loaded",
		},
		{
			name:     "loading status",
			status:   "loading",
			contains: "◐ loading",
		},
		{
			name:     "unloaded status",
			status:   "unloaded",
			contains: "○ unloaded",
		},
		{
			name:     "unknown status shows as error",
			status:   "exit code 1",
			contains: "✗ exit code 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ModelStatusBadge(tt.status)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("ModelStatusBadge(%q) = %q, want to contain %q", tt.status, result, tt.contains)
			}
		})
	}
}

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

func TestFormatOptions(t *testing.T) {
	tests := []struct {
		name string
		opts map[string]string
		want string
	}{
		{
			name: "single option",
			opts: map[string]string{"flash-attn": "on"},
			want: "flash-attn=on",
		},
		{
			name: "multiple options sorted",
			opts: map[string]string{
				"flash-attn":   "on",
				"cache-type-k": "q8_0",
			},
			want: "cache-type-k=q8_0 flash-attn=on",
		},
		{
			name: "empty options",
			opts: map[string]string{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOptions(tt.opts)
			if got != tt.want {
				t.Errorf("formatOptions() = %q, want %q", got, tt.want)
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

func TestPrintStatus_WithMmproj(t *testing.T) {
	// Disable color for testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Arrange
	var buf bytes.Buffer
	Output = &buf
	defer func() { Output = os.Stdout }()

	// Act
	PrintStatus("running", "p:vision", "http://localhost:8080", "/path/to/llama.log", "/models/mmproj.gguf")

	// Assert
	output := buf.String()
	if !strings.Contains(output, "Mmproj") {
		t.Error("Output should contain 'Mmproj' label")
	}
	if !strings.Contains(output, "/models/mmproj.gguf") {
		t.Error("Output should contain mmproj path")
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
