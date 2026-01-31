package metadata

import "fmt"

// NotFoundError indicates a model was not found in metadata.
type NotFoundError struct {
	Repo  string
	Quant string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("model %s:%s not found in metadata", e.Repo, e.Quant)
}
