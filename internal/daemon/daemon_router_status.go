package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// RouterModelStatus represents the status of a single model in router mode.
type RouterModelStatus struct {
	ID     string            `json:"id"`
	Status routerModelStatus `json:"status"`
}

// routerModelStatus wraps the status object from llama-server's /models API.
// The API returns {"status": {"value": "loaded", ...}} not a plain string.
type routerModelStatus struct {
	Value string `json:"value"` // "loaded", "loading", "unloaded"
}

// FetchModelStatuses queries the running llama-server's /models endpoint
// to get the status of each model in router mode.
// Returns nil for non-router presets or on any error (graceful degradation).
func (d *Daemon) FetchModelStatuses(ctx context.Context) []RouterModelStatus {
	p := d.CurrentPreset()
	if p == nil || !p.IsRouter() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Endpoint()+"/models", nil)
	if err != nil {
		return nil
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Parse the response: {"data": [{"id": "...", "status": "..."}]}
	// Limit response body to 1MB to prevent excessive memory usage
	limitedBody := http.MaxBytesReader(nil, resp.Body, 1<<20)
	var body struct {
		Data []RouterModelStatus `json:"data"`
	}
	if err := json.NewDecoder(limitedBody).Decode(&body); err != nil {
		return nil
	}

	return body.Data
}
