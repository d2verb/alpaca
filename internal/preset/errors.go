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

// IsNotFound reports whether err indicates a preset was not found.
func IsNotFound(err error) bool {
	var notFound *NotFoundError
	return errors.As(err, &notFound)
}

// AlreadyExistsError indicates a preset already exists.
type AlreadyExistsError struct {
	Name string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("preset '%s' already exists", e.Name)
}

// ParseError indicates a preset file failed to parse.
type ParseError struct {
	File string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse %s: %v", e.File, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}
