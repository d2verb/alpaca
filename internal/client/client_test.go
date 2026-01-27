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

func TestClient_Run(t *testing.T) {
	t.Run("successful run", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			if req.Command != protocol.CmdRun {
				t.Errorf("expected run command, got %q", req.Command)
			}
			preset, _ := req.Args["preset"].(string)
			if preset != "test-preset" {
				t.Errorf("preset = %q, want %q", preset, "test-preset")
			}
			return protocol.NewOKResponse(map[string]any{
				"endpoint": "http://localhost:8080",
			})
		})

		client := New(socketPath)
		resp, err := client.Run("test-preset")

		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if resp.Data["endpoint"] != "http://localhost:8080" {
			t.Errorf("endpoint = %v, want %q", resp.Data["endpoint"], "http://localhost:8080")
		}
	})

	t.Run("preset not found", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			return protocol.NewErrorResponse("preset 'unknown' not found")
		})

		client := New(socketPath)
		resp, err := client.Run("unknown")

		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if resp.Status != protocol.StatusError {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
		}
	})
}

func TestClient_Kill(t *testing.T) {
	t.Run("successful kill", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			if req.Command != protocol.CmdKill {
				t.Errorf("expected kill command, got %q", req.Command)
			}
			return protocol.NewOKResponse(nil)
		})

		client := New(socketPath)
		resp, err := client.Kill()

		if err != nil {
			t.Fatalf("Kill() error = %v", err)
		}
		if resp.Status != protocol.StatusOK {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
		}
	})
}

func TestClient_ListPresets(t *testing.T) {
	t.Run("returns preset list", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			if req.Command != protocol.CmdListPresets {
				t.Errorf("expected list_presets command, got %q", req.Command)
			}
			return protocol.NewOKResponse(map[string]any{
				"presets": []string{"preset-a", "preset-b", "preset-c"},
			})
		})

		client := New(socketPath)
		resp, err := client.ListPresets()

		if err != nil {
			t.Fatalf("ListPresets() error = %v", err)
		}
		presets, ok := resp.Data["presets"].([]string)
		if !ok {
			// JSON unmarshaling returns []any, not []string
			presetsAny, _ := resp.Data["presets"].([]any)
			if len(presetsAny) != 3 {
				t.Errorf("presets length = %d, want 3", len(presetsAny))
			}
		} else if len(presets) != 3 {
			t.Errorf("presets length = %d, want 3", len(presets))
		}
	})

	t.Run("returns empty list", func(t *testing.T) {
		socketPath := testServer(t, func(req *protocol.Request) *protocol.Response {
			return protocol.NewOKResponse(map[string]any{
				"presets": []string{},
			})
		})

		client := New(socketPath)
		resp, err := client.ListPresets()

		if err != nil {
			t.Fatalf("ListPresets() error = %v", err)
		}
		if resp.Status != protocol.StatusOK {
			t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
		}
	})
}
