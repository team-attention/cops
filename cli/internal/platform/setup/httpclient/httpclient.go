package httpclient

import (
	"net/http"

	"github.com/imroc/req/v3"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
)

// CollectorHTTPClient is an HTTP client configured for the Collector server.
type CollectorHTTPClient struct {
	*req.Client
}

// APIHTTPClient is an HTTP client configured for the API server.
type APIHTTPClient struct {
	*req.Client
}

// InitCollectorHTTPClient creates a new HTTP client for the Collector server.
func InitCollectorHTTPClient(cfg *config.Config) *CollectorHTTPClient {
	client := req.C().
		SetBaseURL(cfg.Collector.URL).
		SetTimeout(cfg.Collector.Timeout)

	return &CollectorHTTPClient{Client: client}
}

// InitAPIHTTPClient creates a new HTTP client for the API server.
func InitAPIHTTPClient(cfg *config.Config) *APIHTTPClient {
	client := req.C().
		SetBaseURL(cfg.API.URL).
		SetTimeout(cfg.API.Timeout)

	return &APIHTTPClient{Client: client}
}

// StandardHTTPClient returns an http.Client that can be used with libraries
// expecting the standard http.Client interface.
func (c *CollectorHTTPClient) StandardHTTPClient() *http.Client {
	return c.Client.GetClient()
}

// StandardHTTPClient returns an http.Client that can be used with libraries
// expecting the standard http.Client interface.
func (c *APIHTTPClient) StandardHTTPClient() *http.Client {
	return c.Client.GetClient()
}
