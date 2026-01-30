package main

import (
	"strings"
	"testing"
)

func TestShowCmd_InvalidIdentifierType(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantErr    string
	}{
		{
			"file path identifier",
			"f:/path/to/model.gguf",
			"cannot show file details",
		},
		{
			"invalid identifier",
			"invalid",
			"invalid identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cmd := &ShowCmd{Identifier: tt.identifier}

			// Act
			err := cmd.Run()

			// Assert
			if err == nil {
				t.Fatal("expected error for invalid identifier")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}
