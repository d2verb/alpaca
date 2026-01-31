package main

import "github.com/d2verb/alpaca/internal/ui"

type StatusCmd struct{}

func (c *StatusCmd) Run() error {
	cl, err := newClient()
	if err != nil {
		return err
	}

	resp, err := cl.Status()
	if err != nil {
		return errDaemonNotRunning()
	}

	paths, err := getPaths()
	if err != nil {
		return err
	}

	state, _ := resp.Data["state"].(string)
	preset, _ := resp.Data["preset"].(string)
	endpoint, _ := resp.Data["endpoint"].(string)

	ui.PrintStatus(state, preset, endpoint, paths.LlamaLog)

	return nil
}
