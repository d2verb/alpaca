package main

import (
	"strings"
	"testing"
)

func TestRemoveCmd_FilePathIdentifierError(t *testing.T) {
	// Arrange
	cmd := &RemoveCmd{Identifier: "f:/path/to/file.gguf"}

	// Act
	err := cmd.Run()

	// Assert
	if err == nil {
		t.Fatal("expected error for file path identifier")
	}
	if !strings.Contains(err.Error(), "file paths (f:) cannot be removed") {
		t.Errorf("expected file path error, got: %v", err)
	}
}

func TestRemoveCmd_InvalidIdentifier(t *testing.T) {
	// Arrange
	cmd := &RemoveCmd{Identifier: "invalid-identifier"}

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
