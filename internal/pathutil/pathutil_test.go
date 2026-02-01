package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	baseDir := "/base/dir"

	tests := []struct {
		name    string
		path    string
		baseDir string
		want    string
		wantErr bool
	}{
		{
			name:    "resolves dot-relative path",
			path:    "./model.gguf",
			baseDir: baseDir,
			want:    "/base/dir/model.gguf",
		},
		{
			name:    "resolves parent-relative path",
			path:    "../shared/model.gguf",
			baseDir: baseDir,
			want:    "/base/shared/model.gguf",
		},
		{
			name:    "resolves nested relative path",
			path:    "./subdir/model.gguf",
			baseDir: baseDir,
			want:    "/base/dir/subdir/model.gguf",
		},
		{
			name:    "expands tilde path",
			path:    "~/models/model.gguf",
			baseDir: baseDir,
			want:    filepath.Join(home, "models/model.gguf"),
		},
		{
			name:    "returns absolute path unchanged",
			path:    "/abs/path/model.gguf",
			baseDir: baseDir,
			want:    "/abs/path/model.gguf",
		},
		{
			name:    "resolves bare filename from baseDir",
			path:    "model.gguf",
			baseDir: baseDir,
			want:    "/base/dir/model.gguf",
		},
		{
			name:    "resolves bare path from baseDir",
			path:    "models/model.gguf",
			baseDir: baseDir,
			want:    "/base/dir/models/model.gguf",
		},
		{
			name:    "handles empty base dir with relative",
			path:    "./model.gguf",
			baseDir: "",
			want:    "model.gguf",
		},
		{
			name:    "tilde in middle of path is not expanded",
			path:    "/path/to/~user/file",
			baseDir: baseDir,
			want:    "/path/to/~user/file",
		},
		{
			name:    "empty path returns error",
			path:    "",
			baseDir: baseDir,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePath(tt.path, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolvePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
