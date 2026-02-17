package client

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name           string
		verbose        bool
		format         string
		expectedFormat OutputFormat
	}{
		{
			name:           "Auto format",
			verbose:        false,
			format:         "",
			expectedFormat: FormatAuto,
		},
		{
			name:           "JSON format",
			verbose:        true,
			format:         "json",
			expectedFormat: FormatJSON,
		},
		{
			name:           "Raw format",
			verbose:        false,
			format:         "raw",
			expectedFormat: FormatRaw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.verbose, tt.format)
			assert.NotNil(t, formatter)
			assert.Equal(t, tt.verbose, formatter.verbose)
			assert.Equal(t, tt.expectedFormat, formatter.format)
		})
	}
}

func TestFormatter_Format_JSON(t *testing.T) {
	formatter := NewFormatter(false, "json")

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"name":"test","value":123}`),
		Duration:   100 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.Contains(t, output, `"name": "test"`)
	assert.Contains(t, output, `"value": 123`)
	// Should be pretty-printed with indentation
	assert.Contains(t, output, "\n")
}

func TestFormatter_Format_Raw(t *testing.T) {
	formatter := NewFormatter(false, "raw")

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("Hello, world!"),
		Duration:   50 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.Equal(t, "Hello, world!", output)
}

func TestFormatter_Format_Auto_JSON(t *testing.T) {
	formatter := NewFormatter(false, "auto")

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"key":"value"}`),
		Duration:   100 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	// Should auto-detect JSON and pretty-print
	assert.Contains(t, output, `"key": "value"`)
}

func TestFormatter_Format_Auto_Raw(t *testing.T) {
	formatter := NewFormatter(false, "auto")

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("Plain text"),
		Duration:   50 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.Equal(t, "Plain text", output)
}

func TestFormatter_Format_Verbose(t *testing.T) {
	formatter := NewFormatter(true, "json")

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"X-Request-ID":  []string{"abc-123"},
		"Authorization": []string{"Bearer secret-token-12345"},
	}

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    headers,
		Body:       []byte(`{"result":"ok"}`),
		Duration:   100 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.Contains(t, output, "< 200 OK")
	assert.Contains(t, output, "Duration:")
	assert.Contains(t, output, "Response Headers:")
	assert.Contains(t, output, "Content-Type: application/json")
	assert.Contains(t, output, "X-Request-ID: abc-123")
	
	// Authorization header should be redacted
	assert.Contains(t, output, "Authorization:")
	assert.Contains(t, output, "Bearer")
	assert.Contains(t, output, "...")
	assert.NotContains(t, output, "secret-token-12345")
}

func TestFormatter_Format_InvalidJSON(t *testing.T) {
	formatter := NewFormatter(false, "json")

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{},
		Body:       []byte("not valid json {"),
		Duration:   50 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	// Should fall back to raw output
	assert.Equal(t, "not valid json {", output)
}

func TestFormatter_WriteOutput_Stdout(t *testing.T) {
	formatter := NewFormatter(false, "raw")
	
	err := formatter.WriteOutput("test output", "")
	
	require.NoError(t, err)
	// Output goes to stdout, no error
}

func TestFormatter_WriteOutput_File(t *testing.T) {
	formatter := NewFormatter(false, "raw")
	
	// Create temp file
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	err := formatter.WriteOutput("test output", outputFile)
	
	require.NoError(t, err)

	// Verify file contents
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, "test output", string(content))
}

func TestFormatter_WriteRawOutput_File(t *testing.T) {
	formatter := NewFormatter(false, "raw")
	
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.bin")

	data := []byte{0x00, 0x01, 0x02, 0x03}
	err := formatter.WriteRawOutput(data, outputFile)
	
	require.NoError(t, err)

	// Verify file contents
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestRedactToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "Long token",
			token:    "Bearer secret-token-12345",
			expected: "Bearer...-12345",
		},
		{
			name:     "Short token",
			token:    "short",
			expected: "***REDACTED***",
		},
		{
			name:     "Exact 8 chars",
			token:    "12345678",
			expected: "***REDACTED***",
		},
		{
			name:     "9 chars",
			token:    "123456789",
			expected: "***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Valid JSON object",
			data:     []byte(`{"key":"value"}`),
			expected: true,
		},
		{
			name:     "Valid JSON array",
			data:     []byte(`[1,2,3]`),
			expected: true,
		},
		{
			name:     "Valid JSON string",
			data:     []byte(`"hello"`),
			expected: true,
		},
		{
			name:     "Valid JSON number",
			data:     []byte(`123`),
			expected: true,
		},
		{
			name:     "Invalid JSON",
			data:     []byte(`not json`),
			expected: false,
		},
		{
			name:     "Empty string",
			data:     []byte(``),
			expected: false,
		},
		{
			name:     "Malformed JSON",
			data:     []byte(`{"key":}`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJSON(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatter_Format_EmptyBody(t *testing.T) {
	formatter := NewFormatter(false, "auto")

	resp := &Response{
		StatusCode: 204,
		Status:     "204 No Content",
		Headers:    http.Header{},
		Body:       []byte{},
		Duration:   10 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.Empty(t, output)
}

func TestFormatter_Format_LargeJSON(t *testing.T) {
	formatter := NewFormatter(false, "json")

	// Create large JSON
	largeJSON := `{"items":[`
	for i := 0; i < 100; i++ {
		if i > 0 {
			largeJSON += ","
		}
		largeJSON += `{"id":` + string(rune(i)) + `,"name":"item"}`
	}
	largeJSON += `]}`

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(largeJSON),
		Duration:   100 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.NotEmpty(t, output)
	// Should be formatted
	assert.Contains(t, output, "\n")
}
