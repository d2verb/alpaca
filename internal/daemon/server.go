package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/logging"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/protocol"
)

// Server handles Unix socket communication.
type Server struct {
	daemon     *Daemon
	socketPath string
	listener   net.Listener
	logger     *slog.Logger
}

// NewServer creates a new daemon server.
func NewServer(daemon *Daemon, socketPath string, logWriter io.Writer) *Server {
	if logWriter == nil {
		panic("logWriter must not be nil")
	}
	return &Server{
		daemon:     daemon,
		socketPath: socketPath,
		logger:     logging.NewLogger(logWriter),
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

	// Set socket permissions to owner-only (0600)
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		listener.Close()
		return err
	}

	s.logger.Info("server started", "socket", s.socketPath)
	go s.acceptLoop(ctx)
	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	if s.listener != nil {
		err := s.listener.Close()
		if err == nil {
			s.logger.Info("server stopped")
		}
		return err
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
				if errors.Is(err, net.ErrClosed) {
					return
				}
				time.Sleep(100 * time.Millisecond)
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
		// EOF is normal when client closes connection without sending data
		if err != io.EOF {
			s.logger.Warn("request read failed", "error", err)
		}
		return
	}

	var req protocol.Request
	if err := json.Unmarshal(line, &req); err != nil {
		s.logger.Warn("invalid request", "error", err)
		s.writeResponse(conn, protocol.NewErrorResponse("invalid request"))
		return
	}

	resp := s.handleRequest(ctx, &req)
	s.writeResponse(conn, resp)
}

func (s *Server) handleRequest(ctx context.Context, req *protocol.Request) *protocol.Response {
	s.logger.Debug("request received", "command", req.Command)

	var resp *protocol.Response
	switch req.Command {
	case protocol.CmdStatus:
		resp = s.handleStatus(ctx)
	case protocol.CmdLoad:
		resp = s.handleLoad(ctx, req)
	case protocol.CmdUnload:
		resp = s.handleUnload(ctx)
	case protocol.CmdListPresets:
		resp = s.handleListPresets()
	case protocol.CmdListModels:
		resp = s.handleListModels(ctx)
	default:
		resp = protocol.NewErrorResponse("unknown command")
	}

	if resp.Status == protocol.StatusError {
		s.logger.Error("request failed", "command", req.Command, "error", resp.Error)
	}
	return resp
}

func (s *Server) handleStatus(ctx context.Context) *protocol.Response {
	state := s.daemon.State()
	data := map[string]any{
		"state": string(state),
	}
	if p := s.daemon.CurrentPreset(); p != nil {
		data["preset"] = p.Name
		data["endpoint"] = p.Endpoint()

		// Add mmproj path for single mode
		if preset.IsMmprojActive(p.Mmproj) {
			data["mmproj"] = strings.TrimPrefix(p.Mmproj, "f:")
		}

		if p.IsRouter() {
			data["mode"] = "router"

			// Build mmproj map from preset models
			mmprojMap := map[string]string{}
			for _, m := range p.Models {
				if preset.IsMmprojActive(m.Mmproj) {
					mmprojMap[m.Name] = strings.TrimPrefix(m.Mmproj, "f:")
				}
			}

			if statuses := s.daemon.FetchModelStatuses(ctx); statuses != nil {
				models := []map[string]any{}
				for _, m := range statuses {
					modelData := map[string]any{
						"id":     m.ID,
						"status": m.Status.Value,
					}
					if mmprojPath, ok := mmprojMap[m.ID]; ok {
						modelData["mmproj"] = mmprojPath
					}
					models = append(models, modelData)
				}
				data["models"] = models
			}
		}
	}
	return protocol.NewOKResponse(data)
}

func (s *Server) handleLoad(ctx context.Context, req *protocol.Request) *protocol.Response {
	identifier, ok := req.Args["identifier"].(string)
	if !ok {
		return protocol.NewErrorResponse("identifier required")
	}

	if err := s.daemon.Run(ctx, identifier); err != nil {
		code, msg := classifyLoadError(err)
		return protocol.NewErrorResponseWithCode(code, msg)
	}

	preset := s.daemon.CurrentPreset()
	return protocol.NewOKResponse(map[string]any{
		"endpoint": preset.Endpoint(),
	})
}

// classifyLoadError determines the error code based on the error type.
func classifyLoadError(err error) (code, message string) {
	msg := err.Error()

	if preset.IsNotFound(err) {
		return protocol.ErrCodePresetNotFound, msg
	}

	var modelNotFound *metadata.NotFoundError
	if errors.As(err, &modelNotFound) {
		return protocol.ErrCodeModelNotFound, msg
	}

	if llama.IsProcessError(err) {
		return protocol.ErrCodeServerFailed, msg
	}

	return "", msg
}

func (s *Server) handleUnload(ctx context.Context) *protocol.Response {
	if err := s.daemon.Kill(ctx); err != nil {
		return protocol.NewErrorResponse(err.Error())
	}
	return protocol.NewOKResponse(nil)
}

func (s *Server) handleListPresets() *protocol.Response {
	presets, err := s.daemon.ListPresets()
	if err != nil && len(presets) == 0 {
		return protocol.NewErrorResponse(err.Error())
	}
	data := map[string]any{
		"presets": presets,
	}
	if err != nil {
		data["warning"] = err.Error()
	}
	return protocol.NewOKResponse(data)
}

func (s *Server) handleListModels(ctx context.Context) *protocol.Response {
	models, err := s.daemon.ListModels(ctx)
	if err != nil {
		return protocol.NewErrorResponse(err.Error())
	}

	return protocol.NewOKResponse(map[string]any{
		"models": models,
	})
}

func (s *Server) writeResponse(conn net.Conn, resp *protocol.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("marshal response failed", "error", err)
		return
	}
	data = append(data, '\n')
	_, _ = conn.Write(data)
}
