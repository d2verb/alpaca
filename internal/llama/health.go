package llama

import (
	"context"
	"fmt"
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := client.Get(healthURL)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

// CheckHealth checks if the llama-server is healthy.
func CheckHealth(endpoint string) error {
	healthURL := endpoint + "/health"
	client := &http.Client{Timeout: HealthCheckTimeout}

	resp, err := client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check: status %d", resp.StatusCode)
	}
	return nil
}
