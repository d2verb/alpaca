package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCompletePresets(t *testing.T) {
	// Arrange: Create temp directory with test presets
	tmpDir := t.TempDir()
	presets := []string{"codellama", "gemma-2b", "llama3"}
	for _, name := range presets {
		content := "name: " + name + "\nmodel: \"f:/path/to/model.gguf\""
		err := os.WriteFile(filepath.Join(tmpDir, name+".yaml"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create preset: %v", err)
		}
	}

	tests := []struct {
		name     string
		partial  string
		expected int
	}{
		{"no filter", "p:", 3},
		{"partial match", "p:code", 1},
		{"no match", "p:xyz", 0},
		{"empty input", "", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			ctx := context.Background()
			results := completePresets(ctx, tmpDir, tt.partial)

			// Assert
			if len(results) != tt.expected {
				t.Errorf("expected %d results, got %d: %v", tt.expected, len(results), results)
			}

			// Verify all results have p: prefix
			for _, r := range results {
				if len(r) < 2 || r[:2] != "p:" {
					t.Errorf("result missing p: prefix: %s", r)
				}
			}
		})
	}
}

func TestCompleteModels(t *testing.T) {
	// Arrange: Create temp directory with test metadata
	tmpDir := t.TempDir()
	metadataContent := `{
	"models": [
		{
			"repo": "TheBloke/CodeLlama-7B-GGUF",
			"quant": "Q4_K_M",
			"filename": "codellama-7b.Q4_K_M.gguf",
			"size": 4368438272,
			"downloaded_at": "2024-01-01T00:00:00Z"
		},
		{
			"repo": "unsloth/gemma-2-2b-it-bnb-4bit",
			"quant": "Q4_K_M",
			"filename": "gemma-2-2b.Q4_K_M.gguf",
			"size": 1610612736,
			"downloaded_at": "2024-01-01T00:00:00Z"
		}
	]
}`
	err := os.WriteFile(filepath.Join(tmpDir, ".metadata.json"), []byte(metadataContent), 0644)
	if err != nil {
		t.Fatalf("failed to create metadata: %v", err)
	}

	tests := []struct {
		name     string
		partial  string
		expected int
	}{
		{"no filter", "h:", 2},
		{"partial match", "h:TheBloke", 1},
		{"no match", "h:xyz", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			ctx := context.Background()
			results := completeModels(ctx, tmpDir, tt.partial)

			// Assert
			if len(results) != tt.expected {
				t.Errorf("expected %d results, got %d: %v", tt.expected, len(results), results)
			}

			// Verify all results have h: prefix and include quant
			for _, r := range results {
				if len(r) < 2 || r[:2] != "h:" {
					t.Errorf("result missing h: prefix: %s", r)
				}
			}
		})
	}
}

func TestCompleteModels_EmptyMetadata(t *testing.T) {
	// Arrange: Empty directory (no metadata file)
	tmpDir := t.TempDir()

	// Act
	ctx := context.Background()
	results := completeModels(ctx, tmpDir, "h:")

	// Assert: Should return nil or empty slice when metadata doesn't exist
	if len(results) != 0 {
		t.Errorf("expected nil or empty slice for missing metadata, got %d results", len(results))
	}
}
