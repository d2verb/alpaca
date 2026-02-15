package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

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
