package preset

import (
	"errors"
	"fmt"
)

// NotFoundError indicates a preset was not found.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("preset %s not found", e.Name)
}

// storeMissingError indicates the presets directory doesn't exist.
// This is unexported; use IsStoreMissing() to check.
type storeMissingError struct {
	err error
}

func (e *storeMissingError) Error() string {
	return fmt.Sprintf("presets directory missing: %v", e.err)
}

func (e *storeMissingError) Unwrap() error {
	return e.err
}

// IsStoreMissing reports whether err indicates the presets directory is missing.
func IsStoreMissing(err error) bool {
	var sm *storeMissingError
	return errors.As(err, &sm)
}

// AlreadyExistsError indicates a preset already exists.
type AlreadyExistsError struct {
	Name string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("preset '%s' already exists", e.Name)
}
