package identifier

import (
	"fmt"
	"strings"
)

// Type represents the category of identifier.
type Type int

const (
	TypeUnknown Type = iota
	TypeFilePath
	TypeHuggingFace
	TypePresetName
)

// Identifier represents a parsed identifier.
type Identifier struct {
	Raw  string
	Type Type

	// For TypeFilePath
	FilePath string

	// For TypeHuggingFace
	Repo  string
	Quant string

	// For TypePresetName
	PresetName string
}

// Parse categorizes an identifier using explicit prefixes (h:, p:, f:).
func Parse(input string) (*Identifier, error) {
	if input == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	// Check for valid prefix format (minimum: "x:y")
	if len(input) < 3 || input[1] != ':' {
		return nil, fmt.Errorf("invalid identifier format '%s'\nExpected: h:org/repo:quant, p:preset-name, or f:/path/to/file", input)
	}

	prefix := input[0]
	value := input[2:] // Everything after "x:"

	if value == "" {
		return nil, fmt.Errorf("empty value after prefix '%c:'", prefix)
	}

	switch prefix {
	case 'h':
		// HuggingFace: h:org/repo:quant
		// Parse but don't validate (validation happens at download time)
		parts := strings.SplitN(value, ":", 2)
		repo := parts[0]
		quant := ""
		if len(parts) == 2 {
			quant = parts[1]
		}
		return &Identifier{
			Raw:   input,
			Type:  TypeHuggingFace,
			Repo:  repo,
			Quant: quant,
		}, nil

	case 'p':
		// Preset: p:preset-name
		return &Identifier{
			Raw:        input,
			Type:       TypePresetName,
			PresetName: value,
		}, nil

	case 'f':
		// File path: f:/path/to/file
		return &Identifier{
			Raw:      input,
			Type:     TypeFilePath,
			FilePath: value,
		}, nil

	default:
		return nil, fmt.Errorf("unknown prefix '%c:'\nExpected: h: (HuggingFace), p: (preset), or f: (file path)", prefix)
	}
}
