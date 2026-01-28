package main

import "fmt"

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
	fmt.Printf("Status: %s\n", state)

	if presetName, ok := resp.Data["preset"].(string); ok {
		fmt.Printf("Preset: %s\n", presetName)
	}
	if endpoint, ok := resp.Data["endpoint"].(string); ok {
		fmt.Printf("Endpoint: %s\n", endpoint)
	}
	fmt.Printf("Logs: %s\n", paths.DaemonLog)

	return nil
}
