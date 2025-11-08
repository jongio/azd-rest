package formatter

import (
	"encoding/json"
	"strings"
)

// FormatResponse formats the response body based on content type
func FormatResponse(body []byte, contentType string) string {
	// If it's JSON, pretty print it
	if strings.Contains(contentType, "application/json") || isJSON(body) {
		return formatJSON(body)
	}
	
	// Otherwise return as-is
	return string(body)
}

// isJSON checks if the body is valid JSON
func isJSON(body []byte) bool {
	var js interface{}
	return json.Unmarshal(body, &js) == nil
}

// formatJSON pretty-prints JSON
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
