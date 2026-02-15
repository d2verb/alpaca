package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

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
