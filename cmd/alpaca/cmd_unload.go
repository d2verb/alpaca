package main

import (
	"fmt"

	"github.com/d2verb/alpaca/internal/ui"
)

type UnloadCmd struct{}

func (c *UnloadCmd) Run() error {
	cl, err := newClient()
	if err != nil {
		return err
	}

	resp, err := cl.Unload()
	if err != nil {
		return errDaemonNotRunning()
	}

	if resp.Status == "error" {
		return fmt.Errorf("%s", resp.Error)
	}

	ui.PrintSuccess("Model stopped")
	return nil
}
