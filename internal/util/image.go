package util

import (
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	_ "golang.org/x/image/webp"
)

func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func DetectMimeType(data []byte) string {
	mime := http.DetectContentType(data)
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	return mime
}

func ImageDimensions(r io.Reader) (int, int, error) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// SanitizeFolder cleans a folder path (allows a/b/c)
func SanitizeFolder(folder string) string {
	folder = strings.TrimSpace(folder)
	folder = strings.Trim(folder, "/")
	folder = strings.ReplaceAll(folder, "..", "")
	folder = strings.ReplaceAll(folder, "\\", "/")

	var parts []string
	for _, p := range strings.Split(folder, "/") {
		cleaned := sanitizeSegment(p)
		if cleaned != "" {
			parts = append(parts, cleaned)
		}
	}
	return strings.Join(parts, "/")
}

// ToSlug converts a filename to a URL-safe slug
// "my image (2).png" → "my-image-2"
// "한글파일.jpg" → "한글파일"
// Preserves unicode letters/digits, replaces spaces/special with hyphens
func ToSlug(name string) string {
	// Remove extension
	if idx := strings.LastIndex(name, "."); idx > 0 {
		name = name[:idx]
	}

	name = strings.TrimSpace(name)
	name = strings.ToLower(name)

	var result []rune
	prevHyphen := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, r)
			prevHyphen = false
		} else if !prevHyphen && len(result) > 0 {
			result = append(result, '-')
			prevHyphen = true
		}
	}

	s := strings.Trim(string(result), "-")
	if s == "" {
		return "image"
	}
	return s
}

// SanitizeSlug ensures a slug is valid for URL use
func SanitizeSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	slug = strings.ToLower(slug)

	var result []rune
	prevHyphen := false
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			result = append(result, r)
			prevHyphen = false
		} else if r == '-' || r == ' ' {
			if !prevHyphen && len(result) > 0 {
				result = append(result, '-')
				prevHyphen = true
			}
		}
	}

	s := strings.Trim(string(result), "-")
	if s == "" {
		return "image"
	}
	return s
}

var segmentRe = regexp.MustCompile(`[^a-zA-Z0-9가-힣ぁ-んァ-ヶ一-龯\-_]`)

func sanitizeSegment(s string) string {
	s = strings.TrimSpace(s)
	s = segmentRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}
