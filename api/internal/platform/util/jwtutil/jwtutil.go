package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config holds JWT configuration.
// This struct is populated from api/internal/platform/setup/config/config.go
type Config struct {
	SecretKey            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

// TokenPair contains access and refresh tokens.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// GenerateTokenPair creates new access and refresh tokens.
// Uses only jwt.RegisteredClaims with UserID stored in Subject field.
func GenerateTokenPair(cfg *Config, userID string) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(cfg.AccessTokenDuration)

	accessClaims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    cfg.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(accessExpiry),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshExpiry := now.Add(cfg.RefreshTokenDuration)
	refreshClaims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    cfg.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(refreshExpiry),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}

// ValidateAccessToken parses and validates an access token.
// Returns the userID from the Subject claim.
func ValidateAccessToken(cfg *Config, tokenString string) (string, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))

	token, err := parser.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.SecretKey), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.Subject, nil
}

// ValidateRefreshToken parses and validates a refresh token.
// Returns the userID from the Subject claim.
func ValidateRefreshToken(cfg *Config, tokenString string) (string, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))

	token, err := parser.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.SecretKey), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.Subject, nil
}
