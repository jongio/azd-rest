package formatter

import (
	"strings"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	input := []byte(`{"name":"test","value":123}`)
	expected := `{
  "name": "test",
  "value": 123
}`
	
	result := formatJSON(input)
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"valid JSON object", []byte(`{"key":"value"}`), true},
		{"valid JSON array", []byte(`[1,2,3]`), true},
		{"invalid JSON", []byte(`not json`), false},
		{"empty", []byte(``), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for input: %s", tt.expected, result, string(tt.input))
			}
		})
	}
}

func TestFormatResponse(t *testing.T) {
	tests := []struct {
		name        string
		body        []byte
		contentType string
		shouldPretty bool
	}{
		{
			name:        "JSON content type",
			body:        []byte(`{"key":"value"}`),
			contentType: "application/json",
			shouldPretty: true,
		},
		{
			name:        "text content type",
			body:        []byte(`plain text`),
			contentType: "text/plain",
			shouldPretty: false,
		},
		{
			name:        "auto-detect JSON",
			body:        []byte(`{"auto":"detect"}`),
			contentType: "",
			shouldPretty: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatResponse(tt.body, tt.contentType)
			if tt.shouldPretty {
				// Should be prettified (contains newlines and indentation)
				if !strings.Contains(result, "\n") {
					t.Error("Expected prettified JSON output with newlines")
				}
			} else {
				// Should be unchanged
				if result != string(tt.body) {
					t.Errorf("Expected unchanged output, got different result")
				}
			}
		})
	}
}
