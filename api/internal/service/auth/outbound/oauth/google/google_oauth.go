package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	oauthport "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
)

type GoogleOAuthAdapter struct {
	logger       *slog.Logger
	config       *oauth2.Config
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

func NewGoogleOAuthAdapter(l *slog.Logger, cfg *config.Config) *GoogleOAuthAdapter {
	return &GoogleOAuthAdapter{
		logger: l.With(slog.String("name", "auth.oauth.google")),
		config: &oauth2.Config{
			ClientID:     cfg.OAuth.GoogleClientID,
			ClientSecret: cfg.OAuth.GoogleClientSecret,
			Scopes:       cfg.OAuth.GoogleScopes,
			Endpoint:     google.Endpoint,
		},
		clientID:     cfg.OAuth.GoogleClientID,
		clientSecret: cfg.OAuth.GoogleClientSecret,
		httpClient:   http.DefaultClient,
	}
}

func (a *GoogleOAuthAdapter) ExchangeCode(ctx context.Context, code, redirectURI string) (*oauthport.TokenResponse, error) {
	cfg := *a.config
	cfg.RedirectURL = redirectURI

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		a.logger.Error("failed to exchange code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	return &oauthport.TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    int(time.Until(token.Expiry).Seconds()),
	}, nil
}

func (a *GoogleOAuthAdapter) GetUserInfo(ctx context.Context, accessToken string) (*oauthport.GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("failed to get user info",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("google API returned error",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return nil, fmt.Errorf("google API error: status %d", resp.StatusCode)
	}

	var userInfo struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &oauthport.GoogleUserInfo{
		ID:            userInfo.ID,
		Email:         userInfo.Email,
		Name:          userInfo.Name,
		Picture:       userInfo.Picture,
		EmailVerified: userInfo.VerifiedEmail,
	}, nil
}

func (a *GoogleOAuthAdapter) InitiateDeviceFlow(ctx context.Context) (*oauthport.DeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("scope", strings.Join(a.config.Scopes, " "))

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/device/code", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("failed to initiate device flow",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to initiate device flow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("google device flow API error",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return nil, fmt.Errorf("device flow API error: status %d", resp.StatusCode)
	}

	var result struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURL string `json:"verification_url"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}

	return &oauthport.DeviceCodeResponse{
		DeviceCode:      result.DeviceCode,
		UserCode:        result.UserCode,
		VerificationURL: result.VerificationURL,
		ExpiresIn:       result.ExpiresIn,
		Interval:        result.Interval,
	}, nil
}

func (a *GoogleOAuthAdapter) PollDeviceCode(ctx context.Context, deviceCode string) (*oauthport.TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("failed to poll device code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to poll device code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error string `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errorResp.Error == "authorization_pending" || errorResp.Error == "slow_down" {
				return nil, nil
			}

			if errorResp.Error == "access_denied" || errorResp.Error == "expired_token" {
				return nil, fmt.Errorf("device code error: %s", errorResp.Error)
			}
		}

		a.logger.Error("google token API error",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return nil, fmt.Errorf("token API error: status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &oauthport.TokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	}, nil
}

var _ oauthport.GoogleOAuthPort = (*GoogleOAuthAdapter)(nil)
