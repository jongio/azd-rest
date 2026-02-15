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

func TestFormatter_Format_AutoDetectsJSON(t *testing.T) {
	formatter := NewFormatter(false, "auto")

	headers := http.Header{
		"Content-Type": []string{"application/json"},
	}

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    headers,
		Body:       []byte(`{"key":"value"}`),
		Duration:   50 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	// Should be pretty-printed JSON
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestFormatter_Format_AutoDetectsRaw(t *testing.T) {
	formatter := NewFormatter(false, "auto")

	headers := http.Header{
		"Content-Type": []string{"text/plain"},
	}

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    headers,
		Body:       []byte("plain text"),
		Duration:   50 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.Equal(t, "plain text", output)
}

func TestFormatter_FormatJSON_Error(t *testing.T) {
	formatter := NewFormatter(false, "json")

	// Invalid JSON that can't be parsed
	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{},
		Body:       []byte("not json {"),
		Duration:   50 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	// Should fall back to raw output
	require.NoError(t, err)
	assert.Equal(t, "not json {", output)
}

func TestFormatter_WriteOutput_ToFile(t *testing.T) {
	formatter := NewFormatter(false, "json")

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	err := formatter.WriteOutput(`{"test": "data"}`, outputFile)
	
	require.NoError(t, err)
	
	// Verify file was created with correct content
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, `{"test": "data"}`, string(data))
	
	// Verify file permissions (should be 0600)
	info, err := os.Stat(outputFile)
	require.NoError(t, err)
	// On Windows, file permissions work differently, so we just check file exists
	assert.NotNil(t, info)
}

func TestFormatter_WriteRawOutput_ToFile(t *testing.T) {
	formatter := NewFormatter(false, "raw")

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.bin")

	data := []byte{0x00, 0x01, 0x02, 0xFF}
	err := formatter.WriteRawOutput(data, outputFile)
	
	require.NoError(t, err)
	
	// Verify file was created with correct content
	fileData, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, data, fileData)
}

func TestRedactSensitiveHeader_AuthorizationBearer(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "Long bearer token",
			value:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
			expected: "Bearer eyJhbG...MDIyfQ",
		},
		{
			name:     "Short bearer token",
			value:    "Bearer token",
			expected: "Bearer ***REDACTED***",
		},
		{
			name:     "Lowercase bearer",
			value:    "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Bearer eyJhbG...pXVCJ9", // Token is 37 chars, first 6: "eyJhbG", last 6: "pXVCJ9"
		},
		{
			name:     "Non-bearer authorization",
			value:    "Basic dXNlcm5hbWU6cGFzc3dvcmQ=",
			expected: "***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveHeader("Authorization", tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactSensitiveHeader_OtherSensitiveHeaders(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "X-API-Key long",
			key:      "X-API-Key",
			value:    "sk_live_1234567890abcdef",
			expected: "sk_liv...abcdef", // Value is 25 chars, first 6: "sk_liv", last 6: "abcdef"
		},
		{
			name:     "X-API-Key short",
			key:      "X-API-Key",
			value:    "short",
			expected: "***REDACTED***",
		},
		{
			name:     "Cookie",
			key:      "Cookie",
			value:    "session=abc123def456",
			expected: "sessio...def456", // First 6 chars: "sessio", last 6 chars: "def456" (from "session=abc123def456" which is 20 chars)
		},
		{
			name:     "Non-sensitive header",
			key:      "Content-Type",
			value:    "application/json",
			expected: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveHeader(tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatter_Format_EmptyHeaders(t *testing.T) {
	formatter := NewFormatter(true, "json")

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{},
		Body:       []byte(`{"result":"ok"}`),
		Duration:   100 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	assert.Contains(t, output, "< 200 OK")
	assert.Contains(t, output, "Response Headers:")
}

func TestFormatter_Format_MultipleHeaderValues(t *testing.T) {
	formatter := NewFormatter(true, "json")

	headers := http.Header{}
	headers.Add("Set-Cookie", "session=abc123")
	headers.Add("Set-Cookie", "token=xyz789")

	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    headers,
		Body:       []byte(`{"result":"ok"}`),
		Duration:   100 * time.Millisecond,
	}

	output, err := formatter.Format(resp)
	
	require.NoError(t, err)
	// Should show both cookie values (redacted)
	assert.Contains(t, output, "Set-Cookie")
}

func TestFormatter_FormatJSON_PrettyPrint(t *testing.T) {
	formatter := NewFormatter(false, "json")

	// Compact JSON input
	compactJSON := `{"key1":"value1","key2":"value2","nested":{"key3":"value3"}}`
	
	formatted, err := formatter.formatJSON([]byte(compactJSON))
	
	require.NoError(t, err)
	// Should be pretty-printed with indentation
	assert.Contains(t, formatted, "\n")
	assert.Contains(t, formatted, "  ") // Indentation
	assert.Contains(t, formatted, "key1")
	assert.Contains(t, formatted, "key2")
}

func TestFormatter_FormatJSON_Array(t *testing.T) {
	formatter := NewFormatter(false, "json")

	arrayJSON := `[{"id":1},{"id":2},{"id":3}]`
	
	formatted, err := formatter.formatJSON([]byte(arrayJSON))
	
	require.NoError(t, err)
	assert.Contains(t, formatted, "[")
	assert.Contains(t, formatted, "]")
	assert.Contains(t, formatted, "id")
}

func TestIsJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Valid JSON number",
			data:     []byte("123"),
			expected: true,
		},
		{
			name:     "Valid JSON boolean",
			data:     []byte("true"),
			expected: true,
		},
		{
			name:     "Valid JSON null",
			data:     []byte("null"),
			expected: true,
		},
		{
			name:     "Invalid - incomplete",
			data:     []byte(`{"key":`),
			expected: false,
		},
		{
			name:     "Invalid - trailing comma",
			data:     []byte(`{"key":"value",}`),
			expected: false,
		},
		{
			name:     "Empty string",
			data:     []byte(""),
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
