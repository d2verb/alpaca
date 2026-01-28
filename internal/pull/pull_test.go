package pull

import (
	"testing"
)

func TestParseModelSpec(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantRepo  string
		wantQuant string
		wantErr   bool
	}{
		{
			name:      "valid spec",
			spec:      "TheBloke/CodeLlama-7B-GGUF:Q4_K_M",
			wantRepo:  "TheBloke/CodeLlama-7B-GGUF",
			wantQuant: "Q4_K_M",
			wantErr:   false,
		},
		{
			name:      "valid spec with lowercase quant",
			spec:      "TheBloke/Mistral-7B-GGUF:q5_k_m",
			wantRepo:  "TheBloke/Mistral-7B-GGUF",
			wantQuant: "q5_k_m",
			wantErr:   false,
		},
		{
			name:    "missing quant",
			spec:    "TheBloke/CodeLlama-7B-GGUF",
			wantErr: true,
		},
		{
			name:    "empty string",
			spec:    "",
			wantErr: true,
		},
		{
			name:    "only colon",
			spec:    ":",
			wantErr: false, // splits to ["", ""], both empty but valid split
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, quant, err := ParseModelSpec(tt.spec)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseModelSpec(%q) expected error, got nil", tt.spec)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseModelSpec(%q) unexpected error: %v", tt.spec, err)
				return
			}

			if repo != tt.wantRepo {
				t.Errorf("ParseModelSpec(%q) repo = %q, want %q", tt.spec, repo, tt.wantRepo)
			}
			if quant != tt.wantQuant {
				t.Errorf("ParseModelSpec(%q) quant = %q, want %q", tt.spec, quant, tt.wantQuant)
			}
		})
	}
}

func TestExtractQuants(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{
			name: "multiple quants",
			files: []string{
				"codellama-7b.Q4_K_M.gguf",
				"codellama-7b.Q5_K_M.gguf",
				"codellama-7b.Q8_0.gguf",
			},
			want: []string{"Q4_K_M", "Q5_K_M", "Q8_0"},
		},
		{
			name: "lowercase filenames",
			files: []string{
				"model.q4_k_m.gguf",
				"model.q5_k_s.gguf",
			},
			want: []string{"Q4_K_M", "Q5_K_S"},
		},
		{
			name: "duplicates removed",
			files: []string{
				"model-part1.Q4_K_M.gguf",
				"model-part2.Q4_K_M.gguf",
			},
			want: []string{"Q4_K_M"},
		},
		{
			name:  "no matching quants",
			files: []string{"readme.md", "config.json"},
			want:  nil,
		},
		{
			name:  "empty list",
			files: []string{},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractQuants(tt.files)

			if len(got) != len(tt.want) {
				t.Errorf("extractQuants() = %v, want %v", got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractQuants()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPuller_SetProgressFunc(t *testing.T) {
	puller := NewPuller("/tmp/models")

	var called bool
	var gotDownloaded, gotTotal int64

	puller.SetProgressFunc(func(downloaded, total int64) {
		called = true
		gotDownloaded = downloaded
		gotTotal = total
	})

	// Simulate progress callback
	if puller.onProgress != nil {
		puller.onProgress(100, 1000)
	}

	if !called {
		t.Error("progress function was not called")
	}
	if gotDownloaded != 100 {
		t.Errorf("downloaded = %d, want 100", gotDownloaded)
	}
	if gotTotal != 1000 {
		t.Errorf("total = %d, want 1000", gotTotal)
	}
}

func TestNewPuller(t *testing.T) {
	modelsDir := "/path/to/models"
	puller := NewPuller(modelsDir)

	if puller.modelsDir != modelsDir {
		t.Errorf("modelsDir = %q, want %q", puller.modelsDir, modelsDir)
	}
	if puller.client == nil {
		t.Error("client should not be nil")
	}
	if puller.onProgress != nil {
		t.Error("onProgress should be nil by default")
	}
}
