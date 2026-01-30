package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/d2verb/alpaca/internal/ui"
)

func TestListCmd_Run(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	cmd := &ListCmd{}

	// Act
	err := cmd.Run()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Output should contain either preset list or model list or both
	// At minimum it should not panic and should produce some output
	output := buf.String()
	if output == "" {
		t.Error("expected some output from ls command")
	}
}
