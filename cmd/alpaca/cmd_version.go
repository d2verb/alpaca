package main

import (
	"fmt"

	"github.com/d2verb/alpaca/internal/ui"
)

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Fprintf(ui.Output, "alpaca version %s (%s)\n", version, commit)
	return nil
}
