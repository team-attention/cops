package setup

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/fx"
)

// IPCServer manages the Unix socket server for CLI-Daemon IPC.
type IPCServer struct {
	logger   *slog.Logger
	listener net.Listener
	server   *http.Server
	sockPath string
	mux      *http.ServeMux
}

// NewIPCServer creates a new IPC server instance.
func NewIPCServer(l *slog.Logger, paths *ExpandedPaths) (*IPCServer, error) {
	sockPath := paths.SocketPath

	// Ensure parent directory exists
	parentDir := filepath.Dir(sockPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, err
	}

	// Remove existing socket file if present
	if _, err := os.Stat(sockPath); err == nil {
		if err := os.Remove(sockPath); err != nil {
			return nil, err
		}
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}

	return &IPCServer{
		logger:   l.With(slog.String("name", "platform.ipc_server")),
		listener: listener,
		sockPath: sockPath,
		mux:      http.NewServeMux(),
	}, nil
}

// RegisterHandler registers an HTTP handler with the server.
func (s *IPCServer) RegisterHandler(path string, handler http.Handler) {
	s.mux.Handle(path, handler)
}

// Start begins serving requests on the Unix socket.
func (s *IPCServer) Start(ctx context.Context) error {
	s.server = &http.Server{
		Handler: s.mux,
	}

	go func() {
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			s.logger.Error("IPC server error", slog.Any("error", err))
		}
	}()

	s.logger.Info("IPC server started", slog.String("socket", s.sockPath))
	return nil
}

// Stop gracefully shuts down the server.
func (s *IPCServer) Stop(ctx context.Context) error {
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Warn("IPC server shutdown error", slog.Any("error", err))
		}
	}

	if err := os.Remove(s.sockPath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("failed to remove socket file", slog.Any("error", err))
	}

	s.logger.Info("IPC server stopped")
	return nil
}

// RegisterIPCServerLifecycle registers the IPC server with fx lifecycle.
func RegisterIPCServerLifecycle(lc fx.Lifecycle, server *IPCServer) {
	lc.Append(fx.Hook{
		OnStart: server.Start,
		OnStop:  server.Stop,
	})
}
