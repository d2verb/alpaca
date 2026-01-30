package main

import (
	"fmt"

	"github.com/d2verb/alpaca/internal/identifier"
)

type PullCmd struct {
	Identifier string `arg:"" help:"Model to download (format: h:org/repo:quant)"`
}

func (c *PullCmd) Run() error {
	id, err := identifier.Parse(c.Identifier)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	if id.Type != identifier.TypeHuggingFace {
		return fmt.Errorf("pull only supports HuggingFace models\nFormat: alpaca pull h:org/repo:quant\nExample: alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}

	if id.Quant == "" {
		return fmt.Errorf("missing quant specifier\nFormat: alpaca pull h:org/repo:quant\nExample: alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	if err := pullModel(id.Repo, id.Quant, paths.Models); err != nil {
		return errDownloadFailed()
	}
	return nil
}
