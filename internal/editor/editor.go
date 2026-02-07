package editor

import (
	"fmt"
	"os"
	"os/exec"
)

var fallbackEditors = []string{"nvim", "vim", "vi", "nano"}

// Find returns the editor command to use.
// It checks $EDITOR first, then falls back to nvim, vim, vi, nano.
func Find() (string, error) {
	if ed := os.Getenv("EDITOR"); ed != "" {
		return ed, nil
	}
	for _, ed := range fallbackEditors {
		if path, err := exec.LookPath(ed); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no editor found: set $EDITOR environment variable")
}

// Open opens the given file in the specified editor.
// The editor runs in the foreground with stdin/stdout/stderr connected.
func Open(editor, filePath string) error {
	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run editor %s: %w", editor, err)
	}
	return nil
}
