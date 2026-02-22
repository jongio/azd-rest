package cmd

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to build a CallToolRequest with the given arguments map.
func newCallToolRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func TestParseHeaders_Valid(t *testing.T) {
	req := newCallToolRequest(map[string]any{
		"headers": map[string]any{
			"Content-Type": "application/json",
			"X-Custom":     "value",
		},
	})

	headers, err := parseHeaders(req)

	require.NoError(t, err)
	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "value", headers["X-Custom"])
}

func TestParseHeaders_BlockedHeaders(t *testing.T) {
	blocked := []string{"Authorization", "Host", "Cookie", "Proxy-Authorization"}
	for _, h := range blocked {
		req := newCallToolRequest(map[string]any{
			"headers": map[string]any{
				h: "some-value",
			},
		})

		_, err := parseHeaders(req)
		require.Error(t, err, "header %q should be blocked", h)
		assert.Contains(t, err.Error(), "not allowed")
	}
}

func TestIsBlockedURL(t *testing.T) {
	tests := []struct {
		url     string
		blocked bool
	}{
		{"http://169.254.169.254/latest/meta-data/", true},
		{"http://metadata.google.internal/computeMetadata/v1/", true},
		{"https://management.azure.com/subscriptions", false},
		{"https://api.github.com/repos", false},
	}
	for _, tc := range tests {
		got := isBlockedURL(tc.url)
		assert.Equal(t, tc.blocked, got, "isBlockedURL(%q)", tc.url)
	}
}

func TestValidateScopeURLMatch(t *testing.T) {
	// Matching domain — should succeed
	err := validateScopeURLMatch(
		"https://management.azure.com/.default",
		"https://management.azure.com/subscriptions",
	)
	assert.NoError(t, err)

	// Mismatched domain — should fail
	err = validateScopeURLMatch(
		"https://management.azure.com/.default",
		"https://attacker.com/exfil",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func TestFormatResponse(t *testing.T) {
	resp := &mcpResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"id":1}`,
	}

	out := formatResponse(resp)

	var parsed mcpResponse
	err := json.Unmarshal([]byte(out), &parsed)
	require.NoError(t, err)
	assert.Equal(t, 200, parsed.StatusCode)
	assert.Equal(t, "application/json", parsed.Headers["Content-Type"])
	assert.Equal(t, `{"id":1}`, parsed.Body)
}
