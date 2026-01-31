package client

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/d2verb/alpaca/internal/protocol"
)

// testServer creates a Unix socket server for testing.
// Returns the socket path and a cleanup function.
func testServer(t *testing.T, handler func(req *protocol.Request) *protocol.Response) string {
	t.Helper()

	// Use /tmp directly to avoid long path issues on macOS
	// (Unix socket paths are limited to ~104 characters)
	socketPath := filepath.Join("/tmp", "alpaca-test-"+filepath.Base(t.TempDir())+".sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Server closed
			}
			go handleTestConnection(conn, handler)
		}
	}()

	t.Cleanup(func() {
		listener.Close()
		os.Remove(socketPath)
	})

	return socketPath
}

func handleTestConnection(conn net.Conn, handler func(req *protocol.Request) *protocol.Response) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return
	}

	var req protocol.Request
	if err := json.Unmarshal(line, &req); err != nil {
		return
	}

	resp := handler(&req)
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	conn.Write(data)
}

func TestNew(t *testing.T) {
	client := New("/tmp/test.sock")

	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.socketPath != "/tmp/test.sock" {
		t.Errorf("socketPath = %q, want %q", client.socketPath, "/tmp/test.sock")
	}
}

func TestClient_Send(t *testing.T) {
	t.Run("successful request/response", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			return protocol.NewOKResponse(map[string]any{"received": req.Command})
		})

		client := New(socketPath)
		resp, err := client.Send(protocol.NewRequest("test", nil))

		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		if resp.Status != protocol.StatusOK {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
		}
		if resp.Data["received"] != "test" {
			t.Errorf("Data[received] = %v, want %q", resp.Data["received"], "test")
		}
	})

	t.Run("connection refused when server not running", func(t *testing.T) {
		client := New("/tmp/nonexistent.sock")
		_, err := client.Send(protocol.NewRequest("test", nil))

		if err == nil {
			t.Error("Send() expected error when server not running")
		}
	})
}

func TestClient_Status(t *testing.T) {
	t.Run("returns idle state", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			if req.Command != protocol.CmdStatus {
				t.Errorf("expected status command, got %q", req.Command)
			}
			return protocol.NewOKResponse(map[string]any{"state": "idle"})
		})

		client := New(socketPath)
		resp, err := client.Status()

		if err != nil {
			t.Fatalf("Status() error = %v", err)
		}
		if resp.Data["state"] != "idle" {
			t.Errorf("state = %v, want %q", resp.Data["state"], "idle")
		}
	})

	t.Run("returns running state with preset info", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			return protocol.NewOKResponse(map[string]any{
				"state":    "running",
				"preset":   "codellama-7b",
				"endpoint": "http://localhost:8080",
			})
		})

		client := New(socketPath)
		resp, err := client.Status()

		if err != nil {
			t.Fatalf("Status() error = %v", err)
		}
		if resp.Data["state"] != "running" {
			t.Errorf("state = %v, want %q", resp.Data["state"], "running")
		}
		if resp.Data["preset"] != "codellama-7b" {
			t.Errorf("preset = %v, want %q", resp.Data["preset"], "codellama-7b")
		}
	})
}

func TestClient_Load(t *testing.T) {
	t.Run("sends load command with identifier", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			if req.Command != protocol.CmdLoad {
				t.Errorf("expected load command, got %q", req.Command)
			}
			if req.Args["identifier"] != "codellama-7b" {
				t.Errorf("identifier = %v, want %q", req.Args["identifier"], "codellama-7b")
			}
			return protocol.NewOKResponse(map[string]any{
				"state":    "running",
				"preset":   "codellama-7b",
				"endpoint": "http://localhost:8080",
			})
		})

		client := New(socketPath)
		resp, err := client.Load("codellama-7b")

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if resp.Status != protocol.StatusOK {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
		}
		if resp.Data["preset"] != "codellama-7b" {
			t.Errorf("preset = %v, want %q", resp.Data["preset"], "codellama-7b")
		}
	})

	t.Run("returns error for invalid preset", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			return protocol.NewErrorResponseWithCode(
				protocol.ErrCodePresetNotFound,
				"preset 'nonexistent' not found",
			)
		})

		client := New(socketPath)
		resp, err := client.Load("nonexistent")

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if resp.Status != protocol.StatusError {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
		}
		if resp.ErrorCode != protocol.ErrCodePresetNotFound {
			t.Errorf("ErrorCode = %q, want %q", resp.ErrorCode, protocol.ErrCodePresetNotFound)
		}
	})
}

func TestClient_Unload(t *testing.T) {
	t.Run("sends unload command", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			if req.Command != protocol.CmdUnload {
				t.Errorf("expected unload command, got %q", req.Command)
			}
			return protocol.NewOKResponse(map[string]any{"state": "idle"})
		})

		client := New(socketPath)
		resp, err := client.Unload()

		if err != nil {
			t.Fatalf("Unload() error = %v", err)
		}
		if resp.Status != protocol.StatusOK {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
		}
		if resp.Data["state"] != "idle" {
			t.Errorf("state = %v, want %q", resp.Data["state"], "idle")
		}
	})

	t.Run("unload when already idle", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			return protocol.NewOKResponse(map[string]any{"state": "idle"})
		})

		client := New(socketPath)
		resp, err := client.Unload()

		if err != nil {
			t.Fatalf("Unload() error = %v", err)
		}
		if resp.Status != protocol.StatusOK {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
		}
	})
}
