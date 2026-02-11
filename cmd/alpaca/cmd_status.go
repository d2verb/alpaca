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
	mode, _ := resp.Data["mode"].(string)

	if mode == "router" {
		var models []ui.RouterModelInfo
		if rawModels, ok := resp.Data["models"].([]any); ok {
			for _, rm := range rawModels {
				if m, ok := rm.(map[string]any); ok {
					models = append(models, ui.RouterModelInfo{
						ID:     stringVal(m, "id"),
						Status: stringVal(m, "status"),
						Mmproj: stringVal(m, "mmproj"),
					})
				}
			}
		}
		ui.PrintRouterStatus(state, preset, endpoint, paths.LlamaLog, models)
	} else {
		mmproj := stringVal(resp.Data, "mmproj")
		ui.PrintStatus(state, preset, endpoint, paths.LlamaLog, mmproj)
	}

	return nil
}

// stringVal extracts a string value from a map, returning empty string if not found.
func stringVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
