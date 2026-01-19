package slugutil

import (
	"crypto/rand"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	// maxSlugBaseLength is the maximum length of the base slug before adding suffix.
	maxSlugBaseLength = 50

	// suffixLength is the length of the random alphanumeric suffix.
	suffixLength = 4

	// suffixChars contains characters used for random suffix generation.
	suffixChars = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// nonAlphanumericRegex matches any character that is not alphanumeric or hyphen.
var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]+`)

// multipleHyphenRegex matches multiple consecutive hyphens.
var multipleHyphenRegex = regexp.MustCompile(`-+`)

// GenerateSlug generates a URL-safe slug from the given name with a random suffix.
// The slug format is: {slugified-name}-{random-suffix}
// Example: "Jayce Kim" -> "jayce-kim-a3f9"
func GenerateSlug(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "user"
	}

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, err := transform.String(t, name)
	if err != nil {
		normalized = name
	}

	base := strings.ToLower(normalized)
	base = strings.ReplaceAll(base, " ", "-")
	base = nonAlphanumericRegex.ReplaceAllString(base, "")
	base = multipleHyphenRegex.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")

	if base == "" {
		base = "user"
	}

	if len(base) > maxSlugBaseLength {
		base = base[:maxSlugBaseLength]
		base = strings.TrimSuffix(base, "-")
	}

	suffix, err := generateRandomSuffix()
	if err != nil {
		return "", err
	}

	return base + "-" + suffix, nil
}

// generateRandomSuffix generates a random alphanumeric suffix of suffixLength characters.
func generateRandomSuffix() (string, error) {
	bytes := make([]byte, suffixLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = suffixChars[int(b)%len(suffixChars)]
	}

	return string(bytes), nil
}
