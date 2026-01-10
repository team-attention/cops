package apikeyutil_test

import (
	"strings"
	"testing"

	"github.com/team-attention/cops/api/internal/platform/util/apikeyutil"
)

func TestGenerateAPIKey(t *testing.T) {
	t.Run("produces key starting with cops_ prefix", func(t *testing.T) {
		key, err := apikeyutil.GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		if !strings.HasPrefix(key, apikeyutil.KeyPrefix) {
			t.Errorf("GenerateAPIKey() = %q, want prefix %q", key, apikeyutil.KeyPrefix)
		}
	})

	t.Run("produces key with correct total length", func(t *testing.T) {
		key, err := apikeyutil.GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		expectedLen := len(apikeyutil.KeyPrefix) + apikeyutil.KeyRandomLength
		if len(key) != expectedLen {
			t.Errorf("GenerateAPIKey() len = %d, want %d", len(key), expectedLen)
		}
	})

	t.Run("uses only Base62 characters", func(t *testing.T) {
		key, err := apikeyutil.GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}

		randomPart := strings.TrimPrefix(key, apikeyutil.KeyPrefix)
		for _, c := range randomPart {
			if !isBase62(c) {
				t.Errorf("GenerateAPIKey() contains non-Base62 character: %q", c)
			}
		}
	})

	t.Run("produces unique keys", func(t *testing.T) {
		keys := make(map[string]bool)
		for i := 0; i < 100; i++ {
			key, err := apikeyutil.GenerateAPIKey()
			if err != nil {
				t.Fatalf("GenerateAPIKey() error = %v", err)
			}
			if keys[key] {
				t.Errorf("GenerateAPIKey() produced duplicate key: %q", key)
			}
			keys[key] = true
		}
	})
}

func TestHashAPIKey(t *testing.T) {
	t.Run("produces consistent hash for same key", func(t *testing.T) {
		key := "cops_abc12345678901234567890123456"
		hash1 := apikeyutil.HashAPIKey(key)
		hash2 := apikeyutil.HashAPIKey(key)
		if hash1 != hash2 {
			t.Errorf("HashAPIKey() not consistent: %q != %q", hash1, hash2)
		}
	})

	t.Run("produces different hash for different keys", func(t *testing.T) {
		key1 := "cops_abc12345678901234567890123456"
		key2 := "cops_xyz12345678901234567890123456"
		hash1 := apikeyutil.HashAPIKey(key1)
		hash2 := apikeyutil.HashAPIKey(key2)
		if hash1 == hash2 {
			t.Errorf("HashAPIKey() produced same hash for different keys")
		}
	})

	t.Run("produces 64-character hex hash (SHA-256)", func(t *testing.T) {
		key := "cops_abc12345678901234567890123456"
		hash := apikeyutil.HashAPIKey(key)
		if len(hash) != 64 {
			t.Errorf("HashAPIKey() len = %d, want 64", len(hash))
		}
		// Verify it's valid hex
		for _, c := range hash {
			if !isHex(c) {
				t.Errorf("HashAPIKey() contains non-hex character: %q", c)
			}
		}
	})
}

func TestExtractPrefix(t *testing.T) {
	t.Run("extracts first 8 chars after prefix", func(t *testing.T) {
		key := "cops_abc12345xyz90123456789012345"
		prefix := apikeyutil.ExtractPrefix(key)
		if prefix != "abc12345" {
			t.Errorf("ExtractPrefix() = %q, want %q", prefix, "abc12345")
		}
	})

	t.Run("returns empty for key without cops_ prefix", func(t *testing.T) {
		key := "invalid_abc12345xyz"
		prefix := apikeyutil.ExtractPrefix(key)
		if prefix != "" {
			t.Errorf("ExtractPrefix() = %q, want empty string", prefix)
		}
	})

	t.Run("returns short prefix if random part is less than 8 chars", func(t *testing.T) {
		key := "cops_abc"
		prefix := apikeyutil.ExtractPrefix(key)
		if prefix != "abc" {
			t.Errorf("ExtractPrefix() = %q, want %q", prefix, "abc")
		}
	})

	t.Run("returns empty for just prefix", func(t *testing.T) {
		key := "cops_"
		prefix := apikeyutil.ExtractPrefix(key)
		if prefix != "" {
			t.Errorf("ExtractPrefix() = %q, want empty string", prefix)
		}
	})
}

// isBase62 checks if a character is in the Base62 alphabet.
func isBase62(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// isHex checks if a character is a valid hexadecimal digit.
func isHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
