package llama

import (
	"context"
	"net/http"
	"time"
)

const (
	// HealthCheckInterval is the interval between health checks.
	HealthCheckInterval = 500 * time.Millisecond
	// HealthCheckTimeout is the timeout for a single health check.
	HealthCheckTimeout = 5 * time.Second
)

// WaitForReady waits until the llama-server is ready to accept requests.
func WaitForReady(ctx context.Context, endpoint string) error {
	healthURL := endpoint + "/health"
	client := &http.Client{Timeout: HealthCheckTimeout}

	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
