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
