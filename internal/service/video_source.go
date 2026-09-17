package service

import (
	"net/url"
	"strings"

	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/util"
)

// ParseResourceSourceType maps a request value to upload/external.
// Missing or blank values are treated as upload for historical rows and old clients.
// Any other value is rejected.
func ParseResourceSourceType(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", model.ResourceSourceUpload:
		return model.ResourceSourceUpload, nil
	case model.ResourceSourceExternal:
		return model.ResourceSourceExternal, nil
	default:
		return "", util.ErrVideoSourceInvalid
	}
}

func stringFromMap(m map[string]interface{}, keys ...string) (string, bool) {
	if m == nil {
		return "", false
	}
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed, true
		}
	}
	return "", false
}

// ValidateVideoSource checks upload/external URLs locally.
// It never fetches or probes the remote resource.
func ValidateVideoSource(sourceType, rawURL string) (string, string, error) {
	st, err := ParseResourceSourceType(sourceType)
	if err != nil {
		return "", "", err
	}

	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", "", util.ErrVideoURLEmpty
	}
	if len(trimmed) > model.MaxVideoURLLength || strings.ContainsAny(trimmed, "\x00\r\n\t") {
		return "", "", util.ErrVideoURLInvalid
	}
	if strings.HasPrefix(trimmed, "//") {
		return "", "", util.ErrVideoURLUnsupportedProtocol
	}

	parsed, parseErr := url.Parse(trimmed)
	hasScheme := parseErr == nil && parsed != nil && parsed.Scheme != ""
	if hasScheme {
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return "", "", util.ErrVideoURLUnsupportedProtocol
		}
		if parsed.Host == "" {
			return "", "", util.ErrVideoURLInvalid
		}
		return st, trimmed, nil
	}
	if st == model.ResourceSourceExternal {
		if parseErr != nil {
			return "", "", util.ErrVideoURLInvalid
		}
		return "", "", util.ErrVideoURLUnsupportedProtocol
	}
	return st, trimmed, nil
}

func NormalizeVideoPayload(m map[string]interface{}) error {
	source, hasSource := stringFromMap(m, "sourceType", "source_type", "SourceType")
	rawURL, hasURL := stringFromMap(m, "url", "URL", "video_url", "videoUrl")
	if !hasSource && !hasURL {
		return nil
	}
	if !hasSource {
		source = model.ResourceSourceUpload
	}
	st, normalizedURL, err := ValidateVideoSource(source, rawURL)
	if err != nil {
		return err
	}
	m["sourceType"] = st
	m["url"] = normalizedURL
	return nil
}
