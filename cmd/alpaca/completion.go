package main

import (
	"context"
	"strings"

	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/posener/complete"
)

// newShowIdentifierPredictor returns a predictor for 'show' command.
// Supports: p:preset-name, h:org/repo:quant
func newShowIdentifierPredictor() complete.Predictor {
	return newIdentifierPredictor([]string{"p:", "h:"})
}

// newRmIdentifierPredictor returns a predictor for 'rm' command.
// Supports: p:preset-name, h:org/repo:quant
func newRmIdentifierPredictor() complete.Predictor {
	return newIdentifierPredictor([]string{"p:", "h:"})
}

// newLoadIdentifierPredictor returns a predictor for 'load' command.
// Supports: p:preset-name, h:org/repo:quant, f:/path/to/file
func newLoadIdentifierPredictor() complete.Predictor {
	return newIdentifierPredictor([]string{"p:", "h:", "f:"})
}

// identifierPredictor implements complete.Predictor for identifier completion.
type identifierPredictor struct {
	validPrefixes []string
}

// newIdentifierPredictor returns a predictor that completes identifiers based on prefix.
// validPrefixes determines which prefixes to suggest when no input is provided.
func newIdentifierPredictor(validPrefixes []string) complete.Predictor {
	return &identifierPredictor{validPrefixes: validPrefixes}
}

// Predict implements complete.Predictor interface.
func (p *identifierPredictor) Predict(args complete.Args) []string {
	// Get the current value being completed
	value := args.Last

	// Get paths early to avoid errors during completion
	paths, err := getPaths()
	if err != nil {
		return nil
	}

	// Note: Using context.Background() here because complete.Predictor interface
	// doesn't provide context. This is acceptable for completion use case where
	// operations are expected to be fast (<100ms).
	ctx := context.Background()

	// Determine completion based on prefix
	switch {
	case value == "":
		// No input yet - suggest actual items from all valid prefixes
		var results []string
		for _, prefix := range p.validPrefixes {
			switch prefix {
			case "p:":
				results = append(results, completePresets(ctx, paths.Presets, "p:")...)
			case "h:":
				results = append(results, completeModels(ctx, paths.Models, "h:")...)
			case "f:":
				// f: prefix doesn't have completion
			}
		}
		return results

	case strings.HasPrefix(value, "p:"):
		// Preset completion: p:name
		return completePresets(ctx, paths.Presets, value)

	case strings.HasPrefix(value, "h:"):
		// HuggingFace model completion: h:org/repo:quant
		return completeModels(ctx, paths.Models, value)

	case strings.HasPrefix(value, "f:"):
		// File path completion - no completion support
		// Users can manually type the full path
		return nil

	default:
		// Invalid input - suggest actual items from valid prefixes
		var results []string
		for _, prefix := range p.validPrefixes {
			switch prefix {
			case "p:":
				results = append(results, completePresets(ctx, paths.Presets, prefix)...)
			case "h:":
				results = append(results, completeModels(ctx, paths.Models, prefix)...)
			}
		}
		return results
	}
}

// completePresets returns preset name completions.
func completePresets(ctx context.Context, presetsDir, partial string) []string {
	loader := preset.NewLoader(presetsDir)
	names, err := loader.List()
	if err != nil {
		return nil
	}

	// Add "p:" prefix to each name
	results := make([]string, 0, len(names))
	for _, name := range names {
		completion := "p:" + name
		if strings.HasPrefix(completion, partial) {
			results = append(results, completion)
		}
	}
	return results
}

// completeModels returns downloaded model identifier completions.
func completeModels(ctx context.Context, modelsDir, partial string) []string {
	modelMgr := model.NewManager(modelsDir)
	entries, err := modelMgr.List(ctx)
	if err != nil {
		return nil
	}

	// Build h:org/repo:quant format
	results := make([]string, 0, len(entries))
	for _, entry := range entries {
		completion := "h:" + entry.Repo + ":" + entry.Quant
		if strings.HasPrefix(completion, partial) {
			results = append(results, completion)
		}
	}
	return results
}
