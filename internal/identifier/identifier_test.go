package identifier

import (
	"testing"
)

func TestParse_FilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType Type
		wantPath string
	}{
		{
			name:     "absolute path",
			input:    "f:/abs/path/model.gguf",
			wantType: TypeFilePath,
			wantPath: "/abs/path/model.gguf",
		},
		{
			name:     "home path",
			input:    "f:~/models/test.gguf",
			wantType: TypeFilePath,
			wantPath: "~/models/test.gguf",
		},
		{
			name:     "current dir relative path",
			input:    "f:./model.gguf",
			wantType: TypeFilePath,
			wantPath: "./model.gguf",
		},
		{
			name:     "parent dir relative path",
			input:    "f:../models/test.gguf",
			wantType: TypeFilePath,
			wantPath: "../models/test.gguf",
		},
		{
			name:     "simple filename",
			input:    "f:model.gguf",
			wantType: TypeFilePath,
			wantPath: "model.gguf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if id.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", id.Type, tt.wantType)
			}
			if id.FilePath != tt.wantPath {
				t.Errorf("FilePath = %v, want %v", id.FilePath, tt.wantPath)
			}
			if id.Raw != tt.input {
				t.Errorf("Raw = %v, want %v", id.Raw, tt.input)
			}
		})
	}
}

func TestParse_HuggingFace(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  Type
		wantRepo  string
		wantQuant string
		wantErr   bool
	}{
		{
			name:      "valid HF format with quant",
			input:     "h:unsloth/gemma3:Q4_K_M",
			wantType:  TypeHuggingFace,
			wantRepo:  "unsloth/gemma3",
			wantQuant: "Q4_K_M",
		},
		{
			name:      "HF with GGUF suffix",
			input:     "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M",
			wantType:  TypeHuggingFace,
			wantRepo:  "TheBloke/CodeLlama-7B-GGUF",
			wantQuant: "Q4_K_M",
		},
		{
			name:      "HF without quant (parser allows, runtime validates)",
			input:     "h:unsloth/gemma3",
			wantType:  TypeHuggingFace,
			wantRepo:  "unsloth/gemma3",
			wantQuant: "",
		},
		{
			name:      "HF with multiple colons",
			input:     "h:org/repo:Q4:extra",
			wantType:  TypeHuggingFace,
			wantRepo:  "org/repo",
			wantQuant: "Q4:extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Parse() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if id.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", id.Type, tt.wantType)
			}
			if id.Repo != tt.wantRepo {
				t.Errorf("Repo = %v, want %v", id.Repo, tt.wantRepo)
			}
			if id.Quant != tt.wantQuant {
				t.Errorf("Quant = %v, want %v", id.Quant, tt.wantQuant)
			}
			if id.Raw != tt.input {
				t.Errorf("Raw = %v, want %v", id.Raw, tt.input)
			}
		})
	}
}

func TestParse_PresetNames(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   Type
		wantPreset string
	}{
		{
			name:       "simple name",
			input:      "p:my-preset",
			wantType:   TypePresetName,
			wantPreset: "my-preset",
		},
		{
			name:       "name with underscores",
			input:      "p:code_llama_7b",
			wantType:   TypePresetName,
			wantPreset: "code_llama_7b",
		},
		{
			name:       "name with numbers",
			input:      "p:llama2-7b",
			wantType:   TypePresetName,
			wantPreset: "llama2-7b",
		},
		{
			name:       "single word",
			input:      "p:default",
			wantType:   TypePresetName,
			wantPreset: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if id.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", id.Type, tt.wantType)
			}
			if id.PresetName != tt.wantPreset {
				t.Errorf("PresetName = %v, want %v", id.PresetName, tt.wantPreset)
			}
			if id.Raw != tt.input {
				t.Errorf("Raw = %v, want %v", id.Raw, tt.input)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: "identifier cannot be empty",
		},
		{
			name:    "no prefix",
			input:   "my-preset",
			wantErr: "invalid identifier format",
		},
		{
			name:    "no colon",
			input:   "pmy-preset",
			wantErr: "invalid identifier format",
		},
		{
			name:    "unknown prefix",
			input:   "x:something",
			wantErr: "unknown prefix",
		},
		{
			name:    "empty value after prefix",
			input:   "p:",
			wantErr: "invalid identifier format",
		},
		{
			name:    "old format without prefix",
			input:   "org/repo:quant",
			wantErr: "invalid identifier format",
		},
		{
			name:    "file path without prefix",
			input:   "./model.gguf",
			wantErr: "invalid identifier format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatal("Parse() expected error, got nil")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
