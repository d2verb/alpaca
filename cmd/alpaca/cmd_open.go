package main

import (
	"os/exec"

	"github.com/d2verb/alpaca/internal/ui"
)

// openBrowser opens a URL in the default browser. Can be replaced for testing.
var openBrowser = func(url string) error {
	return exec.Command("open", url).Start()
}

type OpenCmd struct{}

func (c *OpenCmd) Run() error {
	cl, err := newClient()
	if err != nil {
		return err
	}

	resp, err := cl.Status()
	if err != nil {
		return errDaemonNotRunning()
	}

	state, _ := resp.Data["state"].(string)
	endpoint, _ := resp.Data["endpoint"].(string)

	if state != "running" || endpoint == "" {
		return errServerNotRunning()
	}

	ui.PrintInfo("Opening " + endpoint + " in browser...")
	return openBrowser(endpoint)
}
