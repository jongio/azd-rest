package service

import (
	"fmt"
	"net/url"
	"strings"
)

// contentTypeHeader is the canonical name of the Content-Type header.
const contentTypeHeader = "Content-Type"

// formURLEncoded is the media type set for --form-field bodies.
const formURLEncoded = "application/x-www-form-urlencoded"

// encodeFormFields turns repeatable key=value flags into an
// application/x-www-form-urlencoded body. Repeated keys become multi-valued
// fields. Keys and values are URL-encoded and keys are sorted for a stable
// result.
func encodeFormFields(fields []string) (string, error) {
	values := url.Values{}
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return "", fmt.Errorf("invalid --form-field format: %s (expected key=value)", field)
		}
		values.Add(parts[0], parts[1])
	}
	return values.Encode(), nil
}

// hasHeader reports whether headers already contains name, matched
// case-insensitively.
func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}
