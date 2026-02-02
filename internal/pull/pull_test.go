package pull

import (
	"path/filepath"
	"testing"
)

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
	if puller.metadata == nil {
		t.Error("metadata should not be nil")
	}
}

// TestFilepathIsLocalBehavior documents the expected behavior of filepath.IsLocal
// which is used to validate filenames before passing to os.Root.Create.
func TestFilepathIsLocalBehavior(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantOK   bool
	}{
		// Valid filenames
		{name: "simple filename", filename: "model.gguf", wantOK: true},
		{name: "filename with dashes and dots", filename: "codellama-7b.Q4_K_M.gguf", wantOK: true},
		{name: "hidden file", filename: ".hidden.gguf", wantOK: true},
		{name: "double dots in name", filename: "model..v2.gguf", wantOK: true},
		{name: "filename starting with double dots", filename: "..model.gguf", wantOK: true},

		// Invalid filenames (path traversal attempts)
		{name: "path traversal", filename: "../../../.bashrc", wantOK: false},
		{name: "path traversal mid-path", filename: "foo/../../../.bashrc", wantOK: false},
		{name: "absolute path", filename: "/etc/passwd", wantOK: false},
		{name: "double dot only", filename: "..", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filepath.IsLocal(tt.filename)
			if got != tt.wantOK {
				t.Errorf("filepath.IsLocal(%q) = %v, want %v", tt.filename, got, tt.wantOK)
			}
		})
	}
}
