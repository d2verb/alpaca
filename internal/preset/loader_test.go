package preset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	// Create temp directory for test presets
	tmpDir := t.TempDir()

	// Create a valid preset file
	validPreset := `model: /path/to/model.gguf
context_size: 4096
gpu_layers: 32
threads: 8
port: 9090
host: "0.0.0.0"
extra_args:
  - "--verbose"
  - "--mlock"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "valid.yaml"), []byte(validPreset), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a minimal preset file
	minimalPreset := `model: /path/to/minimal.gguf
`
	if err := os.WriteFile(filepath.Join(tmpDir, "minimal.yaml"), []byte(minimalPreset), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a preset with home directory expansion
	homePreset := `model: ~/.alpaca/models/test.gguf
`
	if err := os.WriteFile(filepath.Join(tmpDir, "home.yaml"), []byte(homePreset), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an invalid YAML file
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)

	t.Run("loads valid preset", func(t *testing.T) {
		p, err := loader.Load("valid")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if p.Name != "valid" {
			t.Errorf("Name = %q, want %q", p.Name, "valid")
		}
		if p.Model != "/path/to/model.gguf" {
			t.Errorf("Model = %q, want %q", p.Model, "/path/to/model.gguf")
		}
		if p.ContextSize != 4096 {
			t.Errorf("ContextSize = %d, want %d", p.ContextSize, 4096)
		}
		if p.GPULayers != 32 {
			t.Errorf("GPULayers = %d, want %d", p.GPULayers, 32)
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

	t.Run("loads minimal preset with defaults", func(t *testing.T) {
		p, err := loader.Load("minimal")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if p.Model != "/path/to/minimal.gguf" {
			t.Errorf("Model = %q, want %q", p.Model, "/path/to/minimal.gguf")
		}
		if p.GetPort() != DefaultPort {
			t.Errorf("GetPort() = %d, want %d", p.GetPort(), DefaultPort)
		}
		if p.GetHost() != DefaultHost {
			t.Errorf("GetHost() = %q, want %q", p.GetHost(), DefaultHost)
		}
	})

	t.Run("expands home directory", func(t *testing.T) {
		p, err := loader.Load("home")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".alpaca/models/test.gguf")
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

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		_, err := loader.Load("invalid")
		if err == nil {
			t.Error("Load() expected error for invalid YAML")
		}
	})
}

func TestLoader_List(t *testing.T) {
	t.Run("lists preset files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create some preset files
		files := []string{"alpha.yaml", "beta.yaml", "gamma.yaml"}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("model: test.gguf"), 0644); err != nil {
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

		loader := NewLoader(tmpDir)
		names, err := loader.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
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
