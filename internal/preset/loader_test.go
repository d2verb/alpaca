package preset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid preset file with random filename
	validPreset := `name: valid-preset
model: "f:/path/to/model.gguf"
context_size: 4096
threads: 8
port: 9090
host: "0.0.0.0"
extra_args:
  - "--verbose"
  - "--mlock"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "abc123.yaml"), []byte(validPreset), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a preset without name field (should be skipped)
	noNamePreset := `model: "f:/path/to/noname.gguf"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "def456.yaml"), []byte(noNamePreset), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a preset with home directory expansion
	homePreset := `name: home-preset
model: "f:~/.alpaca/models/test.gguf"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ghi789.yaml"), []byte(homePreset), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an invalid YAML file
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)

	t.Run("loads preset by name", func(t *testing.T) {
		p, err := loader.Load("valid-preset")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if p.Name != "valid-preset" {
			t.Errorf("Name = %q, want %q", p.Name, "valid-preset")
		}
		if p.Model != "f:/path/to/model.gguf" {
			t.Errorf("Model = %q, want %q", p.Model, "f:/path/to/model.gguf")
		}
		if p.ContextSize != 4096 {
			t.Errorf("ContextSize = %d, want %d", p.ContextSize, 4096)
		}
		if p.Threads != 8 {
			t.Errorf("Threads = %d, want %d", p.Threads, 8)
		}
		if p.Port != 9090 {
			t.Errorf("Port = %d, want %d", p.Port, 9090)
		}
		if p.Host != "0.0.0.0" {
			t.Errorf("Host = %q, want %q", p.Host, "0.0.0.0")
		}
		if len(p.ExtraArgs) != 2 || p.ExtraArgs[0] != "--verbose" || p.ExtraArgs[1] != "--mlock" {
			t.Errorf("ExtraArgs = %v, want [--verbose --mlock]", p.ExtraArgs)
		}
	})

	t.Run("expands home directory", func(t *testing.T) {
		p, err := loader.Load("home-preset")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := "f:" + filepath.Join(home, ".alpaca/models/test.gguf")
		if p.Model != expected {
			t.Errorf("Model = %q, want %q", p.Model, expected)
		}
	})

	t.Run("returns error for non-existent preset", func(t *testing.T) {
		_, err := loader.Load("nonexistent")
		if err == nil {
			t.Error("Load() expected error for non-existent preset")
		}
	})

	t.Run("skips invalid files when searching", func(t *testing.T) {
		// Should still be able to load valid presets even with invalid files in directory
		p, err := loader.Load("valid-preset")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if p.Name != "valid-preset" {
			t.Errorf("Name = %q, want %q", p.Name, "valid-preset")
		}
	})
}

func TestLoader_FindPath(t *testing.T) {
	tmpDir := t.TempDir()

	preset := `name: test-preset
model: "f:/path/to/model.gguf"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "abc123.yaml"), []byte(preset), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)

	t.Run("returns path for existing preset", func(t *testing.T) {
		path, err := loader.FindPath("test-preset")
		if err != nil {
			t.Fatalf("FindPath() error = %v", err)
		}

		expected := filepath.Join(tmpDir, "abc123.yaml")
		if path != expected {
			t.Errorf("FindPath() = %q, want %q", path, expected)
		}
	})

	t.Run("returns error for non-existent preset", func(t *testing.T) {
		_, err := loader.FindPath("nonexistent")
		if err == nil {
			t.Error("FindPath() expected error for non-existent preset")
		}
		if !IsNotFound(err) {
			t.Errorf("FindPath() error should be NotFound, got: %v", err)
		}
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		loader := NewLoader("/nonexistent/path")

		_, err := loader.FindPath("test-preset")
		if err == nil {
			t.Error("FindPath() expected error for non-existent directory")
		}
		if !IsNotFound(err) {
			t.Errorf("FindPath() error should be NotFound, got: %v", err)
		}
	})
}

func TestLoader_List(t *testing.T) {
	t.Run("lists preset names", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create preset files with random filenames
		presets := []struct {
			filename string
			content  string
		}{
			{"abc123.yaml", "name: alpha\nmodel: \"f:/path/to/test.gguf\""},
			{"def456.yaml", "name: beta\nmodel: \"f:/path/to/test.gguf\""},
			{"ghi789.yaml", "name: gamma\nmodel: \"f:/path/to/test.gguf\""},
		}
		for _, p := range presets {
			if err := os.WriteFile(filepath.Join(tmpDir, p.filename), []byte(p.content), 0644); err != nil {
				t.Fatal(err)
			}
		}

		// Create a non-yaml file (should be ignored)
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not a preset"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create a directory (should be ignored)
		if err := os.Mkdir(filepath.Join(tmpDir, "subdir.yaml"), 0755); err != nil {
			t.Fatal(err)
		}

		// Create an invalid preset (should be skipped but reported as warning)
		if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte("{{invalid"), 0644); err != nil {
			t.Fatal(err)
		}

		loader := NewLoader(tmpDir)
		names, err := loader.List()
		// Should return valid names with a warning about parse errors
		if err == nil {
			t.Fatal("List() expected warning error for invalid files, got nil")
		}
		if !strings.Contains(err.Error(), "parse errors") {
			t.Errorf("List() error should mention parse errors, got: %v", err)
		}

		if len(names) != 3 {
			t.Errorf("List() returned %d items, want 3", len(names))
		}

		// Check that all expected names are present
		expected := map[string]bool{"alpha": true, "beta": true, "gamma": true}
		for _, name := range names {
			if !expected[name] {
				t.Errorf("List() returned unexpected name %q", name)
			}
			delete(expected, name)
		}
		if len(expected) > 0 {
			t.Errorf("List() missing names: %v", expected)
		}
	})

	t.Run("returns empty slice for non-existent directory", func(t *testing.T) {
		loader := NewLoader("/nonexistent/path")
		names, err := loader.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(names) != 0 {
			t.Errorf("List() returned %d items, want 0", len(names))
		}
	})

	t.Run("returns empty slice for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)
		names, err := loader.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(names) != 0 {
			t.Errorf("List() returned %d items, want 0", len(names))
		}
	})
}

func TestLoader_Exists(t *testing.T) {
	tmpDir := t.TempDir()

	preset := `name: test-preset
model: "f:/path/to/model.gguf"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "abc123.yaml"), []byte(preset), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)

	t.Run("returns true for existing preset", func(t *testing.T) {
		exists, err := loader.Exists("test-preset")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() = false, want true")
		}
	})

	t.Run("returns false for non-existent preset", func(t *testing.T) {
		exists, err := loader.Exists("nonexistent")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() = true, want false")
		}
	})
}

func TestLoader_Create(t *testing.T) {
	t.Run("creates preset with random filename", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)

		p := &Preset{
			Name:  "my-preset",
			Model: "f:/path/to/model.gguf",
		}

		err := loader.Create(p)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Verify preset can be loaded
		loaded, err := loader.Load("my-preset")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Name != "my-preset" {
			t.Errorf("Name = %q, want %q", loaded.Name, "my-preset")
		}
	})

	t.Run("rejects duplicate name", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)

		p := &Preset{
			Name:  "my-preset",
			Model: "f:/path/to/model.gguf",
		}

		if err := loader.Create(p); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Try to create another with same name
		err := loader.Create(p)
		if err == nil {
			t.Error("Create() expected error for duplicate name")
		}
	})

	t.Run("rejects invalid name", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)

		p := &Preset{
			Name:  "invalid name!", // spaces and special chars
			Model: "f:/path/to/model.gguf",
		}

		err := loader.Create(p)
		if err == nil {
			t.Error("Create() expected error for invalid name")
		}
	})
}

func TestLoadFile(t *testing.T) {
	t.Run("loads preset from file path", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: local-preset
model: f:/abs/path/model.gguf
context_size: 4096
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.Name != "local-preset" {
			t.Errorf("Name = %q, want %q", p.Name, "local-preset")
		}
		if p.Model != "f:/abs/path/model.gguf" {
			t.Errorf("Model = %q, want %q", p.Model, "f:/abs/path/model.gguf")
		}
		if p.ContextSize != 4096 {
			t.Errorf("ContextSize = %d, want %d", p.ContextSize, 4096)
		}
	})

	t.Run("resolves dot-relative model path from preset directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: relative-model
model: f:./models/local.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		expected := "f:" + filepath.Join(tmpDir, "models/local.gguf")
		if p.Model != expected {
			t.Errorf("Model = %q, want %q", p.Model, expected)
		}
	})

	t.Run("resolves bare relative model path from preset directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: bare-relative-model
model: f:models/local.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		expected := "f:" + filepath.Join(tmpDir, "models/local.gguf")
		if p.Model != expected {
			t.Errorf("Model = %q, want %q", p.Model, expected)
		}
	})

	t.Run("resolves parent-relative model path", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "project")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		preset := `name: parent-model
model: f:../shared/model.gguf
`
		presetPath := filepath.Join(subDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		expected := "f:" + filepath.Join(tmpDir, "shared/model.gguf")
		if p.Model != expected {
			t.Errorf("Model = %q, want %q", p.Model, expected)
		}
	})

	t.Run("expands tilde in model path", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: tilde-model
model: f:~/models/model.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := "f:" + filepath.Join(home, "models/model.gguf")
		if p.Model != expected {
			t.Errorf("Model = %q, want %q", p.Model, expected)
		}
	})

	t.Run("preserves HuggingFace model identifier", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: hf-model
model: h:org/repo:Q4_K_M
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.Model != "h:org/repo:Q4_K_M" {
			t.Errorf("Model = %q, want %q", p.Model, "h:org/repo:Q4_K_M")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := LoadFile("/nonexistent/path/.alpaca.yaml")
		if err == nil {
			t.Error("LoadFile() expected error for non-existent file")
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte("{{invalid yaml"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFile(presetPath)
		if err == nil {
			t.Error("LoadFile() expected error for invalid YAML")
		}
	})

	t.Run("returns error for missing name", func(t *testing.T) {
		tmpDir := t.TempDir()
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte("model: f:/path/model.gguf"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFile(presetPath)
		if err == nil {
			t.Error("LoadFile() expected error for missing name")
		}
	})

	t.Run("returns error for model without prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte("name: test\nmodel: /path/model.gguf"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFile(presetPath)
		if err == nil {
			t.Error("LoadFile() expected error for model without prefix")
		}
	})

	t.Run("loads preset with draft model", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: speculative-preset
model: f:/path/to/model.gguf
draft_model: f:/path/to/draft.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.DraftModel != "f:/path/to/draft.gguf" {
			t.Errorf("DraftModel = %q, want %q", p.DraftModel, "f:/path/to/draft.gguf")
		}
	})

	t.Run("resolves relative draft model path from preset directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: relative-draft
model: f:/path/to/model.gguf
draft_model: f:./models/draft.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		expected := "f:" + filepath.Join(tmpDir, "models/draft.gguf")
		if p.DraftModel != expected {
			t.Errorf("DraftModel = %q, want %q", p.DraftModel, expected)
		}
	})

	t.Run("expands tilde in draft model path", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: tilde-draft
model: f:/path/to/model.gguf
draft_model: f:~/models/draft.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := "f:" + filepath.Join(home, "models/draft.gguf")
		if p.DraftModel != expected {
			t.Errorf("DraftModel = %q, want %q", p.DraftModel, expected)
		}
	})

	t.Run("preserves HuggingFace draft model identifier", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: hf-draft
model: f:/path/to/model.gguf
draft_model: h:org/draft-repo:Q4_K_M
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.DraftModel != "h:org/draft-repo:Q4_K_M" {
			t.Errorf("DraftModel = %q, want %q", p.DraftModel, "h:org/draft-repo:Q4_K_M")
		}
	})

	t.Run("returns error for draft model without prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		content := "name: test\nmodel: f:/path/model.gguf\ndraft_model: /path/draft.gguf"
		if err := os.WriteFile(presetPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFile(presetPath)
		if err == nil {
			t.Error("LoadFile() expected error for draft model without prefix")
		}
	})

	t.Run("omits draft model when not specified", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: no-draft
model: f:/path/to/model.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.DraftModel != "" {
			t.Errorf("DraftModel = %q, want empty string", p.DraftModel)
		}
	})
}

func TestLoader_Remove(t *testing.T) {
	t.Run("removes preset by name", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: test-preset
model: "f:/path/to/model.gguf"
`
		if err := os.WriteFile(filepath.Join(tmpDir, "abc123.yaml"), []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		loader := NewLoader(tmpDir)

		err := loader.Remove("test-preset")
		if err != nil {
			t.Fatalf("Remove() error = %v", err)
		}

		// Verify preset is gone
		exists, _ := loader.Exists("test-preset")
		if exists {
			t.Error("Preset still exists after Remove()")
		}
	})

	t.Run("returns error for non-existent preset", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)

		err := loader.Remove("nonexistent")
		if err == nil {
			t.Error("Remove() expected error for non-existent preset")
		}
	})
}
