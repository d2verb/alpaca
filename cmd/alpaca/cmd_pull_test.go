package main

import (
	"strings"
	"testing"
)

func TestPullCmd_InvalidIdentifierType(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantErr    string
	}{
		{
			"preset identifier",
			"p:my-preset",
			"pull only supports HuggingFace models",
		},
		{
			"file path identifier",
			"f:/path/to/model.gguf",
			"pull only supports HuggingFace models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cmd := &PullCmd{Identifier: tt.identifier}

			// Act
			err := cmd.Run()

			// Assert
			if err == nil {
				t.Fatal("expected error for non-HF identifier")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestPullCmd_MissingQuant(t *testing.T) {
	// Arrange
	cmd := &PullCmd{Identifier: "h:org/repo"}

	// Act
	err := cmd.Run()

	// Assert
	if err == nil {
		t.Fatal("expected error for missing quant")
	}
	if !strings.Contains(err.Error(), "missing quant specifier") {
		t.Errorf("expected missing quant error, got: %v", err)
	}
}

func TestPullCmd_InvalidIdentifierFormat(t *testing.T) {
	// Arrange
	cmd := &PullCmd{Identifier: "not-a-valid-identifier"}

	// Act
	err := cmd.Run()

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid identifier")
	}
	if !strings.Contains(err.Error(), "invalid identifier") {
		t.Errorf("expected invalid identifier error, got: %v", err)
	}
}
