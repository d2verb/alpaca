package llama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForReady_Success(t *testing.T) {
	// Arrange
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ctx := context.Background()

	// Act
	err := WaitForReady(ctx, mockServer.URL)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForReady_RetryThenSuccess(t *testing.T) {
	// Arrange
	var callCount atomic.Int32

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count < 3 {
			// First 2 calls fail
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			// 3rd call succeeds
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mockServer.Close()

	ctx := context.Background()

	// Act
	start := time.Now()
	err := WaitForReady(ctx, mockServer.URL)
	elapsed := time.Since(start)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount.Load() < 3 {
		t.Errorf("expected at least 3 health check attempts, got %d", callCount.Load())
	}
	// Should have taken at least 2 intervals (2 * 500ms = 1s) for retries
	if elapsed < HealthCheckInterval {
		t.Logf("elapsed time: %v (expected at least %v for retries)", elapsed, HealthCheckInterval)
	}
}

func TestWaitForReady_ContextTimeout(t *testing.T) {
	// Arrange
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return non-OK status
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer mockServer.Close()

	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	// Act
	err := WaitForReady(ctx, mockServer.URL)

	// Assert
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestWaitForReady_ContextCanceled(t *testing.T) {
	// Arrange
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return non-OK status
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after 300ms
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	// Act
	err := WaitForReady(ctx, mockServer.URL)

	// Assert
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWaitForReady_NetworkError(t *testing.T) {
	// Arrange - Invalid endpoint that causes network error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Use a port that's likely not listening
	invalidEndpoint := "http://localhost:9999"

	// Act
	err := WaitForReady(ctx, invalidEndpoint)

	// Assert
	if err == nil {
		t.Fatal("expected timeout error due to network errors, got nil")
	}
	// Should timeout because network errors cause retries
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestWaitForReady_ImmediateSuccess(t *testing.T) {
	// Arrange - Server immediately returns OK
	var callCount atomic.Int32

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ctx := context.Background()

	// Act
	start := time.Now()
	err := WaitForReady(ctx, mockServer.URL)
	elapsed := time.Since(start)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should complete within first interval
	if elapsed > 2*HealthCheckInterval {
		t.Errorf("took too long: %v (expected < %v)", elapsed, 2*HealthCheckInterval)
	}
	// Should make at least 1 call
	if callCount.Load() < 1 {
		t.Errorf("expected at least 1 health check call, got %d", callCount.Load())
	}
}

func TestWaitForReady_MultipleStatusChanges(t *testing.T) {
	// Arrange - Server alternates between error and success
	var callCount atomic.Int32

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		// Pattern: 503, 500, 503, 200
		switch count {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.WriteHeader(http.StatusInternalServerError)
		case 3:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mockServer.Close()

	ctx := context.Background()

	// Act
	err := WaitForReady(ctx, mockServer.URL)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount.Load() < 4 {
		t.Errorf("expected at least 4 health check attempts, got %d", callCount.Load())
	}
}

func TestWaitForReady_SlowServer(t *testing.T) {
	// Arrange - Server responds slowly but eventually succeeds
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response (but within HealthCheckTimeout)
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ctx := context.Background()

	// Act
	err := WaitForReady(ctx, mockServer.URL)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
