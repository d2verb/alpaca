package preset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid preset file with random filename (v2 format)
	validPreset := `name: valid-preset
model: "f:/path/to/model.gguf"
port: 9090
host: "0.0.0.0"
options:
  ctx-size: 4096
  threads: 8
  mlock: true
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
		if p.Options["ctx-size"] != "4096" {
			t.Errorf("Options[ctx-size] = %q, want %q", p.Options["ctx-size"], "4096")
		}
		if p.Options["threads"] != "8" {
			t.Errorf("Options[threads] = %q, want %q", p.Options["threads"], "8")
		}
		if p.Options["mlock"] != "true" {
			t.Errorf("Options[mlock] = %q, want %q", p.Options["mlock"], "true")
		}
		if p.Port != 9090 {
			t.Errorf("Port = %d, want %d", p.Port, 9090)
		}
		if p.Host != "0.0.0.0" {
			t.Errorf("Host = %q, want %q", p.Host, "0.0.0.0")
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

	t.Run("rejects invalid preset configuration", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)

		p := &Preset{
			Name: "invalid-preset",
			// Missing model field
		}

		err := loader.Create(p)
		if err == nil {
			t.Error("Create() expected error for invalid preset")
		}
		if !strings.Contains(err.Error(), "model field is required") {
			t.Errorf("Create() error = %q, want to contain 'model field is required'", err.Error())
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
