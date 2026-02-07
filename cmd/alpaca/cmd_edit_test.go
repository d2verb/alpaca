package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d2verb/alpaca/internal/identifier"
)

func TestResolveLocalPreset(t *testing.T) {
	t.Run("returns identifier when provided", func(t *testing.T) {
		got, err := resolveLocalPreset("p:my-preset")
		if err != nil {
			t.Fatalf("resolveLocalPreset() error = %v", err)
		}

		if got != "p:my-preset" {
			t.Errorf("resolveLocalPreset() = %q, want %q", got, "p:my-preset")
		}
	})

	t.Run("defaults to local preset when no identifier", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .alpaca.yaml in tmpDir
		presetPath := filepath.Join(tmpDir, LocalPresetFile)
		if err := os.WriteFile(presetPath, []byte("name: test\nmodel: f:/path.gguf"), 0644); err != nil {
			t.Fatal(err)
		}

		// Change to tmpDir
		origDir, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chdir(origDir) })

		got, err := resolveLocalPreset("")
		if err != nil {
			t.Fatalf("resolveLocalPreset() error = %v", err)
		}

		// Use Getwd to get the actual working directory (handles macOS /var → /private/var symlink)
		actualDir, _ := os.Getwd()
		expected := "f:" + filepath.Join(actualDir, LocalPresetFile)
		if got != expected {
			t.Errorf("resolveLocalPreset() = %q, want %q", got, expected)
		}
	})

	t.Run("returns error when no local preset exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		origDir, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chdir(origDir) })

		_, err := resolveLocalPreset("")
		if err == nil {
			t.Fatal("expected error when no .alpaca.yaml")
		}
		if !strings.Contains(err.Error(), "no .alpaca.yaml found") {
			t.Errorf("expected error about missing .alpaca.yaml, got: %v", err)
		}
	})
}

func TestEditCmd_ResolveFilePath(t *testing.T) {
	tests := []struct {
		name    string
		id      *identifier.Identifier
		wantErr string
	}{
		{
			"rejects HuggingFace identifier",
			&identifier.Identifier{Type: identifier.TypeHuggingFace, Raw: "h:org/repo:Q4"},
			"cannot edit model files",
		},
		{
			"rejects model file path",
			&identifier.Identifier{Type: identifier.TypeModelFilePath, FilePath: "/path/model.gguf"},
			"cannot edit model files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &EditCmd{}

			_, err := cmd.resolveFilePath(tt.id)

			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}

	t.Run("resolves preset file path to absolute", func(t *testing.T) {
		cmd := &EditCmd{}
		id := &identifier.Identifier{
			Type:     identifier.TypePresetFilePath,
			FilePath: "/tmp/test-preset.yaml",
		}

		got, err := cmd.resolveFilePath(id)
		if err != nil {
			t.Fatalf("resolveFilePath() error = %v", err)
		}

		if got != "/tmp/test-preset.yaml" {
			t.Errorf("resolveFilePath() = %q, want %q", got, "/tmp/test-preset.yaml")
		}
	})
}

func TestEditCmd_RunRejectsModelIdentifiers(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantErr    string
	}{
		{
			"HuggingFace identifier",
			"h:org/repo:Q4_K_M",
			"cannot edit model files",
		},
		{
			"model file path",
			"f:/path/to/model.gguf",
			"cannot edit model files",
		},
		{
			"invalid identifier",
			"invalid",
			"invalid identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &EditCmd{Identifier: tt.identifier}

			err := cmd.Run()

			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestEditCmd_RunOpensPresetInEditor(t *testing.T) {
	// Arrange: create a temp preset file
	tmpDir := t.TempDir()
	presetPath := filepath.Join(tmpDir, "test-preset.yaml")
	if err := os.WriteFile(presetPath, []byte("name: test\nmodel: h:org/repo:Q4"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set editor to /usr/bin/true (exits 0 silently)
	t.Setenv("EDITOR", "/usr/bin/true")

	cmd := &EditCmd{Identifier: "f:" + presetPath}

	// Act
	err := cmd.Run()

	// Assert
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
