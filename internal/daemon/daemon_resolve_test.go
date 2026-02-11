package daemon

import (
	"context"
	"strings"
	"testing"

	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
)

func TestResolveHFPresetSuccess(t *testing.T) {
	models := &stubModelManager{filePath: "/path/to/model.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p, err := d.resolveHFPreset(context.Background(), "TheBloke/CodeLlama-7B-GGUF", "Q4_K_M")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name != "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M" {
		t.Errorf("Name = %q, want %q", p.Name, "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}
	if p.Model != "f:/path/to/model.gguf" {
		t.Errorf("Model = %q, want %q", p.Model, "f:/path/to/model.gguf")
	}
	// Host, Port use preset defaults via GetXxx() methods
	if p.GetHost() != preset.DefaultHost {
		t.Errorf("GetHost() = %q, want %q", p.GetHost(), preset.DefaultHost)
	}
	if p.GetPort() != preset.DefaultPort {
		t.Errorf("GetPort() = %d, want %d", p.GetPort(), preset.DefaultPort)
	}
}

func TestResolveHFPresetModelNotFound(t *testing.T) {
	models := &stubModelManager{exists: false}
	d := newTestDaemon(&stubPresetLoader{}, models)

	_, err := d.resolveHFPreset(context.Background(), "unknown/repo", "Q4_K_M")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveModel_FilePath(t *testing.T) {
	models := &stubModelManager{filePath: "/should/not/be/used"}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "f:/abs/path/model.gguf",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File path should remain unchanged
	if resolved.Model != "f:/abs/path/model.gguf" {
		t.Errorf("Model = %q, want %q", resolved.Model, "f:/abs/path/model.gguf")
	}
}

func TestResolveModel_HuggingFace(t *testing.T) {
	models := &stubModelManager{filePath: "/resolved/path/model.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "h:org/repo:Q4_K_M",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// HF format should be resolved to file path with f: prefix
	if resolved.Model != "f:/resolved/path/model.gguf" {
		t.Errorf("Model = %q, want %q", resolved.Model, "f:/resolved/path/model.gguf")
	}

	// Original preset should not be mutated
	if p.Model != "h:org/repo:Q4_K_M" {
		t.Errorf("Original preset mutated: Model = %q, want %q", p.Model, "h:org/repo:Q4_K_M")
	}
}

func TestResolveModel_HuggingFaceNotExists(t *testing.T) {
	models := &stubModelManager{exists: false}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "h:org/repo:Q4_K_M",
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveModel_InvalidIdentifier(t *testing.T) {
	models := &stubModelManager{}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "",
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error for empty model field, got nil")
	}
}

func TestResolveModel_OldFormatError(t *testing.T) {
	models := &stubModelManager{}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "org/repo:Q4_K_M", // Old format without h: prefix
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error for old format without prefix, got nil")
	}
}

func TestResolveModel_DraftModelFilePath(t *testing.T) {
	models := &stubModelManager{filePath: "/should/not/be/used"}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:       "test",
		Model:      "f:/abs/path/model.gguf",
		DraftModel: "f:/abs/path/draft.gguf",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.DraftModel != "f:/abs/path/draft.gguf" {
		t.Errorf("DraftModel = %q, want %q", resolved.DraftModel, "f:/abs/path/draft.gguf")
	}
}

func TestResolveModel_DraftModelHuggingFace(t *testing.T) {
	models := &stubModelManager{filePath: "/resolved/path/draft.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:       "test",
		Model:      "f:/abs/path/model.gguf",
		DraftModel: "h:org/draft-repo:Q4_K_M",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.DraftModel != "f:/resolved/path/draft.gguf" {
		t.Errorf("DraftModel = %q, want %q", resolved.DraftModel, "f:/resolved/path/draft.gguf")
	}

	// Original preset should not be mutated
	if p.DraftModel != "h:org/draft-repo:Q4_K_M" {
		t.Errorf("Original preset mutated: DraftModel = %q, want %q", p.DraftModel, "h:org/draft-repo:Q4_K_M")
	}
}

func TestResolveModel_DraftModelHuggingFaceNotExists(t *testing.T) {
	models := &stubModelManager{exists: false}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:       "test",
		Model:      "f:/abs/path/model.gguf",
		DraftModel: "h:org/draft-repo:Q4_K_M",
	}

	_, err := d.resolveModel(context.Background(), p)
	if err == nil {
		t.Fatal("expected error when draft model not found, got nil")
	}
}

func TestResolveModel_NoDraftModel(t *testing.T) {
	models := &stubModelManager{filePath: "/resolved/path/model.gguf", exists: true}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "f:/abs/path/model.gguf",
	}

	resolved, err := d.resolveModel(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.DraftModel != "" {
		t.Errorf("DraftModel = %q, want empty string", resolved.DraftModel)
	}
}

func TestResolveHFPresetWithMmproj(t *testing.T) {
	// Arrange
	models := &stubModelManager{
		filePath: "/models/vision-model.gguf",
		exists:   true,
		entries: []metadata.ModelEntry{
			{
				Repo:  "org/vision-model-GGUF",
				Quant: "Q4_K_M",
				Mmproj: &metadata.MmprojEntry{
					Filename: "mmproj-vision.gguf",
					Size:     1024,
				},
			},
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	// Act
	p, err := d.resolveHFPreset(context.Background(), "org/vision-model-GGUF", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Model != "f:/models/vision-model.gguf" {
		t.Errorf("Model = %q, want %q", p.Model, "f:/models/vision-model.gguf")
	}
	wantMmproj := "f:/models/mmproj-vision.gguf"
	if p.Mmproj != wantMmproj {
		t.Errorf("Mmproj = %q, want %q", p.Mmproj, wantMmproj)
	}
}

func TestResolveHFPresetWithoutMmproj(t *testing.T) {
	// Arrange - model has no mmproj
	models := &stubModelManager{
		filePath: "/models/text-model.gguf",
		exists:   true,
		entries: []metadata.ModelEntry{
			{
				Repo:  "org/text-model-GGUF",
				Quant: "Q4_K_M",
			},
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	// Act
	p, err := d.resolveHFPreset(context.Background(), "org/text-model-GGUF", "Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Mmproj != "" {
		t.Errorf("Mmproj = %q, want empty", p.Mmproj)
	}
}

func TestResolveModel_HuggingFaceWithMmproj(t *testing.T) {
	// Arrange - preset with HF model that has mmproj in metadata
	models := &mapModelManager{
		paths: map[string]string{
			"org/vision:Q4_K_M": "/models/vision.gguf",
		},
		entries: map[string]*metadata.ModelEntry{
			"org/vision:Q4_K_M": {
				Repo:  "org/vision",
				Quant: "Q4_K_M",
				Mmproj: &metadata.MmprojEntry{
					Filename: "mmproj-vision.gguf",
					Size:     2048,
				},
			},
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:  "test",
		Model: "h:org/vision:Q4_K_M",
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Model != "f:/models/vision.gguf" {
		t.Errorf("Model = %q, want %q", resolved.Model, "f:/models/vision.gguf")
	}
	wantMmproj := "f:/models/mmproj-vision.gguf"
	if resolved.Mmproj != wantMmproj {
		t.Errorf("Mmproj = %q, want %q", resolved.Mmproj, wantMmproj)
	}

	// Original should not be mutated
	if p.Mmproj != "" {
		t.Errorf("original preset mutated: Mmproj = %q, want empty", p.Mmproj)
	}
}

func TestResolveModel_MmprojNonePreserved(t *testing.T) {
	// Arrange - preset explicitly sets mmproj to "none"
	models := &mapModelManager{
		paths: map[string]string{
			"org/vision:Q4_K_M": "/models/vision.gguf",
		},
		entries: map[string]*metadata.ModelEntry{
			"org/vision:Q4_K_M": {
				Repo:  "org/vision",
				Quant: "Q4_K_M",
				Mmproj: &metadata.MmprojEntry{
					Filename: "mmproj-vision.gguf",
					Size:     2048,
				},
			},
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:   "test",
		Model:  "h:org/vision:Q4_K_M",
		Mmproj: "none",
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Mmproj != "none" {
		t.Errorf("Mmproj = %q, want %q (should preserve 'none')", resolved.Mmproj, "none")
	}
}

func TestResolveModel_MmprojExplicitPreserved(t *testing.T) {
	// Arrange - preset explicitly sets mmproj to f: path
	models := &mapModelManager{
		paths: map[string]string{
			"org/vision:Q4_K_M": "/models/vision.gguf",
		},
		entries: map[string]*metadata.ModelEntry{
			"org/vision:Q4_K_M": {
				Repo:  "org/vision",
				Quant: "Q4_K_M",
				Mmproj: &metadata.MmprojEntry{
					Filename: "mmproj-vision.gguf",
					Size:     2048,
				},
			},
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name:   "test",
		Model:  "h:org/vision:Q4_K_M",
		Mmproj: "f:/custom/mmproj.gguf",
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Mmproj != "f:/custom/mmproj.gguf" {
		t.Errorf("Mmproj = %q, want %q (should preserve explicit path)", resolved.Mmproj, "f:/custom/mmproj.gguf")
	}
}

func TestResolveModel_RouterMode(t *testing.T) {
	// Arrange
	models := &mapModelManager{
		paths: map[string]string{
			"org/model-a:Q4_K_M": "/resolved/model-a.gguf",
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name: "router-test",
		Mode: "router",
		Models: []preset.ModelEntry{
			{Name: "model-a", Model: "h:org/model-a:Q4_K_M"},
			{Name: "model-b", Model: "f:/path/to/model-b.gguf"},
		},
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Models[0].Model != "f:/resolved/model-a.gguf" {
		t.Errorf("Models[0].Model = %q, want %q", resolved.Models[0].Model, "f:/resolved/model-a.gguf")
	}
	if resolved.Models[1].Model != "f:/path/to/model-b.gguf" {
		t.Errorf("Models[1].Model = %q, want %q", resolved.Models[1].Model, "f:/path/to/model-b.gguf")
	}

	// Original should not be mutated
	if p.Models[0].Model != "h:org/model-a:Q4_K_M" {
		t.Errorf("original preset mutated: Models[0].Model = %q", p.Models[0].Model)
	}
}

func TestResolveModel_RouterModeDraftModel(t *testing.T) {
	// Arrange
	models := &mapModelManager{
		paths: map[string]string{
			"org/draft-repo:Q4_K_M": "/resolved/draft.gguf",
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name: "router-draft-test",
		Mode: "router",
		Models: []preset.ModelEntry{
			{
				Name:       "model-a",
				Model:      "f:/path/to/model.gguf",
				DraftModel: "h:org/draft-repo:Q4_K_M",
			},
		},
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Models[0].DraftModel != "f:/resolved/draft.gguf" {
		t.Errorf("Models[0].DraftModel = %q, want %q", resolved.Models[0].DraftModel, "f:/resolved/draft.gguf")
	}

	// Original should not be mutated
	if p.Models[0].DraftModel != "h:org/draft-repo:Q4_K_M" {
		t.Errorf("original preset mutated: Models[0].DraftModel = %q", p.Models[0].DraftModel)
	}
}

func TestResolveModel_RouterModeInvalidModel(t *testing.T) {
	// Arrange
	d := newTestDaemon(&stubPresetLoader{}, &stubModelManager{})

	p := &preset.Preset{
		Name: "router-invalid-test",
		Mode: "router",
		Models: []preset.ModelEntry{
			{Name: "good", Model: "f:/path/to/model.gguf"},
			{Name: "bad", Model: "no-prefix"},
		},
	}

	// Act
	_, err := d.resolveModel(context.Background(), p)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid identifier, got nil")
	}
	if !strings.Contains(err.Error(), "models[1]") {
		t.Errorf("error should reference models[1], got: %v", err)
	}
}

func TestResolveModel_RouterModeInvalidDraftModel(t *testing.T) {
	// Arrange
	d := newTestDaemon(&stubPresetLoader{}, &stubModelManager{})

	p := &preset.Preset{
		Name: "router-invalid-draft-test",
		Mode: "router",
		Models: []preset.ModelEntry{
			{
				Name:       "model-a",
				Model:      "f:/path/to/model.gguf",
				DraftModel: "no-prefix",
			},
		},
	}

	// Act
	_, err := d.resolveModel(context.Background(), p)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid draft-model identifier, got nil")
	}
	if !strings.Contains(err.Error(), "draft-model") {
		t.Errorf("error should mention draft-model, got: %v", err)
	}
}

func TestResolveModel_RouterModeWithMmproj(t *testing.T) {
	// Arrange - router with HF model that has mmproj in metadata
	models := &mapModelManager{
		paths: map[string]string{
			"org/vision:Q4_K_M": "/models/vision.gguf",
		},
		entries: map[string]*metadata.ModelEntry{
			"org/vision:Q4_K_M": {
				Repo:  "org/vision",
				Quant: "Q4_K_M",
				Mmproj: &metadata.MmprojEntry{
					Filename: "mmproj-vision.gguf",
					Size:     2048,
				},
			},
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name: "router-mmproj-test",
		Mode: "router",
		Models: []preset.ModelEntry{
			{Name: "vision", Model: "h:org/vision:Q4_K_M"},
			{Name: "text", Model: "f:/models/text.gguf"},
		},
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Models[0].Model != "f:/models/vision.gguf" {
		t.Errorf("Models[0].Model = %q, want %q", resolved.Models[0].Model, "f:/models/vision.gguf")
	}
	wantMmproj := "f:/models/mmproj-vision.gguf"
	if resolved.Models[0].Mmproj != wantMmproj {
		t.Errorf("Models[0].Mmproj = %q, want %q", resolved.Models[0].Mmproj, wantMmproj)
	}
	// Text model should have no mmproj
	if resolved.Models[1].Mmproj != "" {
		t.Errorf("Models[1].Mmproj = %q, want empty", resolved.Models[1].Mmproj)
	}

	// Original should not be mutated
	if p.Models[0].Mmproj != "" {
		t.Errorf("original preset mutated: Models[0].Mmproj = %q, want empty", p.Models[0].Mmproj)
	}
}

func TestResolveModel_RouterModeWithExplicitMmproj(t *testing.T) {
	// Arrange - router model with explicit mmproj should not be overridden
	models := &mapModelManager{
		paths: map[string]string{
			"org/vision:Q4_K_M": "/models/vision.gguf",
		},
		entries: map[string]*metadata.ModelEntry{
			"org/vision:Q4_K_M": {
				Repo:  "org/vision",
				Quant: "Q4_K_M",
				Mmproj: &metadata.MmprojEntry{
					Filename: "mmproj-auto.gguf",
					Size:     2048,
				},
			},
		},
	}
	d := newTestDaemon(&stubPresetLoader{}, models)

	p := &preset.Preset{
		Name: "router-explicit-mmproj",
		Mode: "router",
		Models: []preset.ModelEntry{
			{Name: "vision", Model: "h:org/vision:Q4_K_M", Mmproj: "f:/custom/mmproj.gguf"},
		},
	}

	// Act
	resolved, err := d.resolveModel(context.Background(), p)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Models[0].Mmproj != "f:/custom/mmproj.gguf" {
		t.Errorf("Models[0].Mmproj = %q, want %q (should preserve explicit)", resolved.Models[0].Mmproj, "f:/custom/mmproj.gguf")
	}
}
