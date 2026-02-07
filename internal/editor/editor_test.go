package editor

import (
	"os"
	"strings"
	"testing"
)

func TestFindReturnsEditorEnvVar(t *testing.T) {
	tests := []struct {
		name   string
		editor string
	}{
		{"absolute path", "/usr/bin/vim"},
		{"command name", "code"},
		{"editor with flags style name", "subl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editor)

			got, err := Find()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.editor {
				t.Errorf("got %q, want %q", got, tt.editor)
			}
		})
	}
}

func TestFindFallsBackWhenEditorUnset(t *testing.T) {
	t.Setenv("EDITOR", "")

	// Ensure PATH includes /usr/bin so at least "vi" or "nano" is found
	path := os.Getenv("PATH")
	if !strings.Contains(path, "/usr/bin") {
		t.Setenv("PATH", "/usr/bin:"+path)
	}

	got, err := Find()

	if err != nil {
		t.Skip("no fallback editor available in this environment")
	}
	if got == "" {
		t.Error("expected a fallback editor, got empty string")
	}
}

func TestFindReturnsErrorWhenNoEditorAvailable(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("PATH", t.TempDir())

	_, err := Find()

	if err == nil {
		t.Fatal("expected error when no editor is available")
	}
	if !strings.Contains(err.Error(), "no editor found") {
		t.Errorf("error message %q should contain %q", err.Error(), "no editor found")
	}
}

func TestOpenReturnsErrorForNonExistentEditor(t *testing.T) {
	err := Open("/nonexistent/editor", "file.txt")

	if err == nil {
		t.Fatal("expected error for non-existent editor")
	}
	if !strings.Contains(err.Error(), "run editor") {
		t.Errorf("error message %q should contain %q", err.Error(), "run editor")
	}
}

func TestOpenSucceedsWithTrueCommand(t *testing.T) {
	err := Open("/usr/bin/true", t.TempDir()+"/test.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenSplitsEditorWithArgs(t *testing.T) {
	// "/usr/bin/env true" should be split into ["/usr/bin/env", "true", filePath]
	err := Open("/usr/bin/env true", t.TempDir()+"/test.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
