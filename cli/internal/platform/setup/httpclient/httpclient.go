package httpclient

import (
	"net/http"

	"github.com/imroc/req/v3"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
)

// APIHTTPClient is an HTTP client configured for the API server.
type APIHTTPClient struct {
	*req.Client
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
func (c *APIHTTPClient) StandardHTTPClient() *http.Client {
	return c.Client.GetClient()
}

// WithAuth returns a cloned client with auth header set for authenticated requests.
// This should be called by service logic that needs authentication.
func (c *APIHTTPClient) WithAuth(accessToken string) *req.Client {
	return c.Client.Clone().SetCommonBearerAuthToken(accessToken)
}
