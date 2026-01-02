package setup

import (
	"net/http"

	"github.com/imroc/req/v3"
)

// APIClient is an HTTP client configured for the COps API server.
type APIClient struct {
	*req.Client
}

// InitAPIClient creates a new HTTP client for the COps API server.
func InitAPIClient(cfg *Config) *APIClient {
	client := req.C().
		SetBaseURL(cfg.API.URL).
		SetTimeout(cfg.API.Timeout)

	return &APIClient{Client: client}
}

// StandardHTTPClient returns an http.Client that can be used with libraries
// expecting the standard http.Client interface.
func (c *APIClient) StandardHTTPClient() *http.Client {
	return c.Client.GetClient()
}

// WithAuth returns a cloned client with auth header set.
func (c *APIClient) WithAuth(accessToken string) *req.Client {
	return c.Client.Clone().SetCommonBearerAuthToken(accessToken)
}
