package preset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	t.Run("loads preset from file path", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: local-preset
model: f:/abs/path/model.gguf
options:
  ctx-size: 4096
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
		if p.Options["ctx-size"] != "4096" {
			t.Errorf("Options[ctx-size] = %q, want %q", p.Options["ctx-size"], "4096")
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
draft-model: f:/path/to/draft.gguf
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
draft-model: f:./models/draft.gguf
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
draft-model: f:~/models/draft.gguf
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
draft-model: h:org/draft-repo:Q4_K_M
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
		content := "name: test\nmodel: f:/path/model.gguf\ndraft-model: /path/draft.gguf"
		if err := os.WriteFile(presetPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFile(presetPath)
		if err == nil {
			t.Error("LoadFile() expected error for draft model without prefix")
		}
	})

	t.Run("resolves mmproj with empty value to empty", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: no-mmproj
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

		if p.Mmproj != "" {
			t.Errorf("Mmproj = %q, want empty string", p.Mmproj)
		}
	})

	t.Run("resolves mmproj none to none", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: mmproj-none
model: f:/path/to/model.gguf
mmproj: none
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.Mmproj != "none" {
			t.Errorf("Mmproj = %q, want %q", p.Mmproj, "none")
		}
	})

	t.Run("expands tilde in mmproj path", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: mmproj-tilde
model: f:/path/to/model.gguf
mmproj: "f:~/models/mmproj.gguf"
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
		expected := "f:" + filepath.Join(home, "models/mmproj.gguf")
		if p.Mmproj != expected {
			t.Errorf("Mmproj = %q, want %q", p.Mmproj, expected)
		}
	})

	t.Run("resolves relative mmproj path from preset directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: mmproj-relative
model: f:/path/to/model.gguf
mmproj: f:./models/mmproj.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		expected := "f:" + filepath.Join(tmpDir, "models/mmproj.gguf")
		if p.Mmproj != expected {
			t.Errorf("Mmproj = %q, want %q", p.Mmproj, expected)
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

func TestLoadFile_RouterMode(t *testing.T) {
	t.Run("resolves relative model paths in router mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: router-relative
mode: router
models:
  - name: model1
    model: f:./models/model1.gguf
  - name: model2
    model: f:./models/model2.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if len(p.Models) != 2 {
			t.Fatalf("Models count = %d, want 2", len(p.Models))
		}

		expected1 := "f:" + filepath.Join(tmpDir, "models/model1.gguf")
		if p.Models[0].Model != expected1 {
			t.Errorf("Models[0].Model = %q, want %q", p.Models[0].Model, expected1)
		}

		expected2 := "f:" + filepath.Join(tmpDir, "models/model2.gguf")
		if p.Models[1].Model != expected2 {
			t.Errorf("Models[1].Model = %q, want %q", p.Models[1].Model, expected2)
		}
	})

	t.Run("expands tilde in router model paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: router-tilde
mode: router
models:
  - name: model1
    model: f:~/models/model1.gguf
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
		expected := "f:" + filepath.Join(home, "models/model1.gguf")
		if p.Models[0].Model != expected {
			t.Errorf("Models[0].Model = %q, want %q", p.Models[0].Model, expected)
		}
	})

	t.Run("preserves HuggingFace identifiers in router mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: router-hf
mode: router
models:
  - name: qwen3
    model: h:Qwen/Qwen3-8B-GGUF:Q4_K_M
  - name: nomic
    model: h:nomic-ai/nomic-embed-text:Q4_K_M
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.Models[0].Model != "h:Qwen/Qwen3-8B-GGUF:Q4_K_M" {
			t.Errorf("Models[0].Model = %q, want %q", p.Models[0].Model, "h:Qwen/Qwen3-8B-GGUF:Q4_K_M")
		}
		if p.Models[1].Model != "h:nomic-ai/nomic-embed-text:Q4_K_M" {
			t.Errorf("Models[1].Model = %q, want %q", p.Models[1].Model, "h:nomic-ai/nomic-embed-text:Q4_K_M")
		}
	})

	t.Run("resolves draft model paths in router mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: router-draft
mode: router
models:
  - name: qwen3
    model: f:/abs/path/model.gguf
    draft-model: f:./drafts/draft.gguf
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.Models[0].Model != "f:/abs/path/model.gguf" {
			t.Errorf("Models[0].Model = %q, want %q", p.Models[0].Model, "f:/abs/path/model.gguf")
		}

		expectedDraft := "f:" + filepath.Join(tmpDir, "drafts/draft.gguf")
		if p.Models[0].DraftModel != expectedDraft {
			t.Errorf("Models[0].DraftModel = %q, want %q", p.Models[0].DraftModel, expectedDraft)
		}
	})

	t.Run("resolves mmproj paths in router mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: router-mmproj
mode: router
models:
  - name: vision
    model: f:/path/to/model.gguf
    mmproj: "f:~/models/mmproj.gguf"
  - name: text-only
    model: f:/path/to/text.gguf
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
		expectedMmproj := "f:" + filepath.Join(home, "models/mmproj.gguf")
		if p.Models[0].Mmproj != expectedMmproj {
			t.Errorf("Models[0].Mmproj = %q, want %q", p.Models[0].Mmproj, expectedMmproj)
		}

		if p.Models[1].Mmproj != "" {
			t.Errorf("Models[1].Mmproj = %q, want empty string", p.Models[1].Mmproj)
		}
	})

	t.Run("preserves HuggingFace draft model in router mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		preset := `name: router-hf-draft
mode: router
models:
  - name: qwen3
    model: h:Qwen/Qwen3-8B-GGUF:Q4_K_M
    draft-model: h:Qwen/Qwen3-1B-GGUF:Q4_K_M
`
		presetPath := filepath.Join(tmpDir, ".alpaca.yaml")
		if err := os.WriteFile(presetPath, []byte(preset), 0644); err != nil {
			t.Fatal(err)
		}

		p, err := LoadFile(presetPath)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}

		if p.Models[0].DraftModel != "h:Qwen/Qwen3-1B-GGUF:Q4_K_M" {
			t.Errorf("Models[0].DraftModel = %q, want %q", p.Models[0].DraftModel, "h:Qwen/Qwen3-1B-GGUF:Q4_K_M")
		}
	})
}
