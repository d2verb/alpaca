package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"

	"github.com/d2verb/alpaca/internal/protocol"
)

// Server handles Unix socket communication.
type Server struct {
	daemon     *Daemon
	socketPath string
	listener   net.Listener
}

// NewServer creates a new daemon server.
func NewServer(daemon *Daemon, socketPath string) *Server {
	return &Server{
		daemon:     daemon,
		socketPath: socketPath,
	}
}

// Start starts listening on the Unix socket.
func (s *Server) Start(ctx context.Context) error {
	// Remove existing socket file
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = listener

	go s.acceptLoop(ctx)
	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		go s.handleConnection(ctx, conn)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return
	}

	var req protocol.Request
	if err := json.Unmarshal(line, &req); err != nil {
		s.writeResponse(conn, protocol.NewErrorResponse("invalid request"))
		return
	}

	resp := s.handleRequest(ctx, &req)
	s.writeResponse(conn, resp)
}

func (s *Server) handleRequest(ctx context.Context, req *protocol.Request) *protocol.Response {
	switch req.Command {
	case protocol.CmdStatus:
		return s.handleStatus()
	case protocol.CmdLoad:
		return s.handleLoad(ctx, req)
	case protocol.CmdUnload:
		return s.handleUnload(ctx)
	case protocol.CmdListPresets:
		return s.handleListPresets()
	case protocol.CmdListModels:
		return s.handleListModels(ctx)
	default:
		return protocol.NewErrorResponse("unknown command")
	}
}

func (s *Server) handleStatus() *protocol.Response {
	state := s.daemon.State()
	data := map[string]any{
		"state": string(state),
	}
	if preset := s.daemon.CurrentPreset(); preset != nil {
		data["preset"] = preset.Name
		data["endpoint"] = preset.Endpoint()
	}
	return protocol.NewOKResponse(data)
}

func (s *Server) handleLoad(ctx context.Context, req *protocol.Request) *protocol.Response {
	identifier, ok := req.Args["identifier"].(string)
	if !ok {
		return protocol.NewErrorResponse("identifier required")
	}

	if err := s.daemon.Run(ctx, identifier); err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	preset := s.daemon.CurrentPreset()
	return protocol.NewOKResponse(map[string]any{
		"endpoint": preset.Endpoint(),
	})
}

func (s *Server) handleUnload(ctx context.Context) *protocol.Response {
	if err := s.daemon.Kill(ctx); err != nil {
		return protocol.NewErrorResponse(err.Error())
	}
	return protocol.NewOKResponse(nil)
}

func (s *Server) handleListPresets() *protocol.Response {
	presets, err := s.daemon.ListPresets()
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}
	return protocol.NewOKResponse(map[string]any{
		"presets": presets,
	})
}

func (s *Server) handleListModels(ctx context.Context) *protocol.Response {
	models, err := s.daemon.ListModels(ctx)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	// Convert to map format for JSON
	modelList := []map[string]any{}
	for _, m := range models {
		modelList = append(modelList, map[string]any{
			"repo":  m.Repo,
			"quant": m.Quant,
			"size":  m.Size,
		})
	}

	return protocol.NewOKResponse(map[string]any{
		"models": modelList,
	})
}

func (s *Server) writeResponse(conn net.Conn, resp *protocol.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_, _ = conn.Write(data)
}
