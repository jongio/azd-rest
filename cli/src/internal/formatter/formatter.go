package formatter

import (
	"encoding/json"
	"strings"
)

// FormatResponse formats the response body based on content type
func FormatResponse(body []byte, contentType string) string {
	// If it's JSON by content type or appears to be JSON, pretty print it
	if strings.Contains(contentType, "application/json") || looksLikeJSON(body) {
		return formatJSON(body)
	}

	// Otherwise return as-is
	return string(body)
}

// looksLikeJSON checks if the body starts with '{' or '[' after skipping whitespace
// This is more efficient than unmarshaling for large bodies
func looksLikeJSON(body []byte) bool {
	// Skip leading whitespace
	for _, b := range body {
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			continue
		}
		// Check if first non-whitespace character is { or [
		return b == '{' || b == '['
	}
	return false
}

// formatJSON pretty-prints JSON (only unmarshals once)
func formatJSON(body []byte) string {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return string(body)
	}

	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return string(body)
	}

	return string(formatted)
}
