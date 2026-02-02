// Package client provides a client for communicating with the daemon.
package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"

	"github.com/d2verb/alpaca/internal/protocol"
)

// Client communicates with the daemon via Unix socket.
type Client struct {
	socketPath string
}

// New creates a new daemon client.
func New(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// Send sends a request to the daemon and returns the response.
func (c *Client) Send(req *protocol.Request) (*protocol.Response, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w", err)
	}
	defer conn.Close()

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp protocol.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}

// Status sends a status request to the daemon.
func (c *Client) Status() (*protocol.Response, error) {
	return c.Send(protocol.NewRequest(protocol.CmdStatus, nil))
}

// Load sends a load request to the daemon.
func (c *Client) Load(identifier string) (*protocol.Response, error) {
	return c.Send(protocol.NewRequest(protocol.CmdLoad, map[string]any{
		"identifier": identifier,
	}))
}

// Unload sends an unload request to the daemon.
func (c *Client) Unload() (*protocol.Response, error) {
	return c.Send(protocol.NewRequest(protocol.CmdUnload, nil))
}
