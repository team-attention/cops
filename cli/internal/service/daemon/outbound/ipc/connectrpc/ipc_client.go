package connectrpc

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"

	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/ipc"
	daemonv1 "github.com/team-attention/cops/shared/gen/grpcstub/daemon/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/daemon/v1/daemonv1connect"
)

// IPCClient implements IPC communication with Daemon over Unix socket.
type IPCClient struct {
	logger     *slog.Logger
	client     daemonv1connect.DaemonServiceClient
	sockPath   string
	httpClient *http.Client
}

// NewIPCClient creates a new IPC client for Unix socket communication.
func NewIPCClient(l *slog.Logger, socketPath string) *IPCClient {
	sockPath := expandSocketPath(socketPath)

	// Create HTTP/2 transport for Unix socket
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			dialer := net.Dialer{
				Timeout: 5 * time.Second,
			}
			return dialer.DialContext(ctx, "unix", sockPath)
		},
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Create ConnectRPC client
	// Base URL is required but ignored for Unix socket - use localhost
	client := daemonv1connect.NewDaemonServiceClient(
		httpClient,
		"http://localhost",
		connect.WithGRPC(),
	)

	return &IPCClient{
		logger:     l.With(slog.String("name", "daemon.ipc.connectrpc")),
		client:     client,
		sockPath:   sockPath,
		httpClient: httpClient,
	}
}

// expandSocketPath expands ~ to the home directory in the socket path.
func expandSocketPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// ScanLogs implements ipc.IPCPort.
func (c *IPCClient) ScanLogs(ctx context.Context, params ipc.ScanLogsParams) (*ipc.ScanLogsResult, error) {
	req := connect.NewRequest(&daemonv1.ScanLogsReq{
		ProjectId:      params.ProjectID,
		ProjectPath:    params.ProjectPath,
		OrganizationId: params.OrganizationID,
	})

	resp, err := c.client.ScanLogs(ctx, req)
	if err != nil {
		c.logger.Error("ScanLogs RPC failed", slog.Any("error", err))
		return nil, err
	}

	return &ipc.ScanLogsResult{
		Success: resp.Msg.Success,
		Message: resp.Msg.Message,
	}, nil
}

// IsAvailable implements ipc.IPCPort.
func (c *IPCClient) IsAvailable(ctx context.Context) bool {
	// Check if socket file exists
	if _, err := os.Stat(c.sockPath); os.IsNotExist(err) {
		return false
	}

	// Try to dial the socket with short timeout
	conn, err := net.DialTimeout("unix", c.sockPath, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}
