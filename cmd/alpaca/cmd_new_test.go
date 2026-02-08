package main

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/d2verb/alpaca/internal/ui"
)

// setStdinInput replaces the stdin reader with the given input for testing.
// Returns a cleanup function that restores the original reader.
func setStdinInput(t *testing.T, input string) {
	t.Helper()
	original := stdin
	stdin = bufio.NewReader(strings.NewReader(input))
	t.Cleanup(func() { stdin = original })
}

func TestCollectRouterInputs(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	// Simulate input: Host (default), Port (default), Model 1, Model 2, blank to finish
	input := strings.Join([]string{
		"",                                   // Host [127.0.0.1]
		"",                                   // Port [8080]
		"qwen3",                              // Name
		"h:Qwen/Qwen3-8B-GGUF:Q4_K_M",        // Model
		"8192",                               // Context
		"nomic-embed",                        // Name
		"h:nomic-ai/nomic-embed-text:Q4_K_M", // Model
		"",                                   // Context (default)
		"",                                   // blank name to finish
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectRouterInputs("test-workspace")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "test-workspace" {
		t.Errorf("Name = %q, want %q", p.Name, "test-workspace")
	}
	if p.Mode != "router" {
		t.Errorf("Mode = %q, want %q", p.Mode, "router")
	}
	if len(p.Models) != 2 {
		t.Fatalf("Models count = %d, want 2", len(p.Models))
	}
	if p.Models[0].Name != "qwen3" {
		t.Errorf("Models[0].Name = %q, want %q", p.Models[0].Name, "qwen3")
	}
	if p.Models[0].Model != "h:Qwen/Qwen3-8B-GGUF:Q4_K_M" {
		t.Errorf("Models[0].Model = %q, want %q", p.Models[0].Model, "h:Qwen/Qwen3-8B-GGUF:Q4_K_M")
	}
	if p.Models[0].ContextSize != 8192 {
		t.Errorf("Models[0].ContextSize = %d, want 8192", p.Models[0].ContextSize)
	}
	if p.Models[1].Name != "nomic-embed" {
		t.Errorf("Models[1].Name = %q, want %q", p.Models[1].Name, "nomic-embed")
	}
	// Default context size should not be stored
	if p.Models[1].ContextSize != 0 {
		t.Errorf("Models[1].ContextSize = %d, want 0 (default omitted)", p.Models[1].ContextSize)
	}
}

func TestCollectRouterInputs_CustomHostPort(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"0.0.0.0",            // Host
		"9090",               // Port
		"model1",             // Name
		"h:org/model:Q4_K_M", // Model
		"",                   // Context (default)
		"",                   // blank name to finish
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectRouterInputs("custom")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", p.Host, "0.0.0.0")
	}
	if p.Port != 9090 {
		t.Errorf("Port = %d, want 9090", p.Port)
	}
}

func TestCollectRouterInputs_NoModels(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"", // Host (default)
		"", // Port (default)
		"", // blank name immediately
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	_, err := cmd.collectRouterInputs("empty")

	// Assert
	if err == nil {
		t.Fatal("expected error for no models")
	}
	if !strings.Contains(err.Error(), "at least one model is required") {
		t.Errorf("expected 'at least one model' error, got: %v", err)
	}
}

func TestCollectRouterInputs_DuplicateModelNameRecovers(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"",                    // Host (default)
		"",                    // Port (default)
		"model1",              // Name 1
		"h:org/model1:Q4_K_M", // Model 1
		"",                    // Context (default)
		"model1",              // Duplicate name → warning
		"model2",              // Recovery with valid name
		"h:org/model2:Q4_K_M", // Model 2
		"",                    // Context (default)
		"",                    // blank name to finish
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectRouterInputs("dup")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Models) != 2 {
		t.Fatalf("Models count = %d, want 2", len(p.Models))
	}
	if !strings.Contains(buf.String(), "already added") {
		t.Error("expected warning about duplicate model name")
	}
}

func TestCollectRouterInputs_InvalidModelNameRecovers(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"",                   // Host (default)
		"",                   // Port (default)
		"invalid name!",      // Invalid name → warning
		"valid-name",         // Recovery with valid name
		"h:org/model:Q4_K_M", // Model
		"",                   // Context (default)
		"",                   // blank name to finish
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectRouterInputs("test")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Models) != 1 {
		t.Fatalf("Models count = %d, want 1", len(p.Models))
	}
	if p.Models[0].Name != "valid-name" {
		t.Errorf("Models[0].Name = %q, want %q", p.Models[0].Name, "valid-name")
	}
	if !strings.Contains(buf.String(), "invalid model name") {
		t.Error("expected warning about invalid model name")
	}
}

func TestCollectRouterInputs_MissingModelRefRecovers(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"",                   // Host (default)
		"",                   // Port (default)
		"model1",             // Name
		"",                   // Empty model ref → warning
		"model1",             // Re-enter name
		"h:org/model:Q4_K_M", // Valid model
		"",                   // Context (default)
		"",                   // blank name to finish
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectRouterInputs("test")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Models) != 1 {
		t.Fatalf("Models count = %d, want 1", len(p.Models))
	}
	if !strings.Contains(buf.String(), "model is required") {
		t.Error("expected warning about missing model")
	}
}

func TestCollectRouterInputs_InvalidModelPrefixRecovers(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"",                   // Host (default)
		"",                   // Port (default)
		"model1",             // Name
		"org/model:Q4_K_M",   // Missing prefix → warning
		"model1",             // Re-enter name
		"h:org/model:Q4_K_M", // Valid model with prefix
		"",                   // Context (default)
		"",                   // blank name to finish
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectRouterInputs("test")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Models) != 1 {
		t.Fatalf("Models count = %d, want 1", len(p.Models))
	}
	if !strings.Contains(buf.String(), "must have h: or f: prefix") {
		t.Error("expected warning about missing prefix")
	}
}

func TestCollectInputs_InvalidMode(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := "foobar\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	_, err := cmd.collectInputs("test")

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("expected 'invalid mode' error, got: %v", err)
	}
}

func TestCollectInputs_RouterModeBranch(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"router",             // Mode
		"",                   // Host (default)
		"",                   // Port (default)
		"model1",             // Name
		"h:org/model:Q4_K_M", // Model
		"",                   // Context (default)
		"",                   // blank name to finish
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectInputs("router-test")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Mode != "router" {
		t.Errorf("Mode = %q, want %q", p.Mode, "router")
	}
	if len(p.Models) != 1 {
		t.Errorf("Models count = %d, want 1", len(p.Models))
	}
}

func TestCollectInputs_SingleModeBranch(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	input := strings.Join([]string{
		"",                   // Mode (default single)
		"h:org/model:Q4_K_M", // Model
		"",                   // Host (default)
		"",                   // Port (default)
		"",                   // Context (default)
	}, "\n") + "\n"
	setStdinInput(t, input)

	cmd := &NewCmd{}

	// Act
	p, err := cmd.collectInputs("single-test")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Mode != "" {
		t.Errorf("Mode = %q, want empty (single)", p.Mode)
	}
	if p.Model != "h:org/model:Q4_K_M" {
		t.Errorf("Model = %q, want %q", p.Model, "h:org/model:Q4_K_M")
	}
}
