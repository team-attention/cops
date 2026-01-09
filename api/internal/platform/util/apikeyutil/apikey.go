package apikeyutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	// KeyPrefix is the prefix for all API keys.
	KeyPrefix = "cops_"
	// KeyRandomLength is the length of the random part of the key.
	KeyRandomLength = 32
	// KeyPrefixDisplayLength is the length of the prefix displayed for identification.
	KeyPrefixDisplayLength = 8
)

// Base62 characters for random key generation.
const base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateAPIKey generates a new API key.
// Format: cops_{random_32_chars} (37 characters total).
// Uses crypto/rand for secure random generation with Base62 characters.
func GenerateAPIKey() (string, error) {
	randomBytes := make([]byte, KeyRandomLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	randomPart := make([]byte, KeyRandomLength)
	for i := 0; i < KeyRandomLength; i++ {
		randomPart[i] = base62Chars[int(randomBytes[i])%len(base62Chars)]
	}

	return KeyPrefix + string(randomPart), nil
}

// HashAPIKey hashes an API key using SHA-256.
// Returns hex-encoded hash.
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ExtractPrefix extracts the first 8 characters after "cops_" for identification.
// Returns the display prefix (e.g., "cops_abc12345" -> "abc12345").
func ExtractPrefix(key string) string {
	if !strings.HasPrefix(key, KeyPrefix) {
		return ""
	}

	randomPart := strings.TrimPrefix(key, KeyPrefix)
	if len(randomPart) < KeyPrefixDisplayLength {
		return randomPart
	}

	return randomPart[:KeyPrefixDisplayLength]
}
