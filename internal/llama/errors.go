package llama

import (
	"errors"
	"fmt"
)

// ProcessOp represents a llama-server process operation.
type ProcessOp string

const (
	ProcessOpStart ProcessOp = "start"
	ProcessOpWait  ProcessOp = "wait"
)

// ProcessError indicates a llama-server process operation failed.
type ProcessError struct {
	Op  ProcessOp
	Err error
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf("%s llama-server: %v", e.Op, e.Err)
}

func (e *ProcessError) Unwrap() error {
	return e.Err
}

// IsProcessError reports whether err indicates a llama-server process failure.
func IsProcessError(err error) bool {
	var pe *ProcessError
	return errors.As(err, &pe)
}
