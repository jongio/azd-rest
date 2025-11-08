package cmd

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-core/azdextutil"
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
	if testing.Short() {
		t.Skip("skipping DNS-dependent test in short mode")
	}
	tests := []struct {
		url     string
		blocked bool
	}{
		// Cloud metadata endpoints
		{"http://169.254.169.254/latest/meta-data/", true},
		{"http://metadata.google.internal/computeMetadata/v1/", true},
		// Loopback addresses
		{"http://127.0.0.1:8080/admin", true},
		{"http://[::1]:8080/admin", true},
		// Private ranges
		{"http://10.0.0.1/internal", true},
		{"http://192.168.1.1/admin", true},
		{"http://172.16.0.1/internal", true},
		// Valid external URLs
		{"https://management.azure.com/subscriptions", false},
		{"https://api.github.com/repos", false},
	}
	for _, tc := range tests {
		got := isBlockedURL(tc.url)
		assert.Equal(t, tc.blocked, got, "isBlockedURL(%q)", tc.url)
	}
}

func TestIsBlockedURL_Unit(t *testing.T) {
	// These tests work without DNS resolution (pure IP checks).
	tests := []struct {
		url     string
		blocked bool
	}{
		{"http://169.254.169.254/latest/meta-data/", true},
		{"http://127.0.0.1:8080/admin", true},
		{"http://[::1]:8080/admin", true},
		{"http://0.0.0.0/admin", true},
		{"http://0.0.0.1:8080/service", true},
		{"http://[::]:80/admin", true},
		{"http://10.0.0.1/internal", true},
		{"http://192.168.1.1/admin", true},
		{"http://172.16.0.1/internal", true},
		{"not-a-valid-url://\x00", true}, // parse error → blocked
	}
	for _, tc := range tests {
		got := isBlockedURL(tc.url)
		assert.Equal(t, tc.blocked, got, "isBlockedURL(%q)", tc.url)
	}
}

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"0.0.0.0", true},
		{"0.0.0.1", true},
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.0.1", true},
		{"169.254.169.254", true},
		{"8.8.8.8", false},
		{"20.0.0.1", false},
	}
	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		require.NotNil(t, ip, "failed to parse IP: %s", tc.ip)
		got := isBlockedIP(ip)
		assert.Equal(t, tc.blocked, got, "isBlockedIP(%q)", tc.ip)
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

// ---------------------------------------------------------------------------
// getOrCreateTokenProvider
// ---------------------------------------------------------------------------

func TestGetOrCreateTokenProvider_Caching(t *testing.T) {
	// Save and restore global state.
	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	tokenProviderMu.Unlock()
	defer func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	}()

	mock := &auth.MockTokenProvider{Token: "cached-test-token"}
	tokenProviderMu.Lock()
	cachedTokenProvider = mock
	tokenProviderMu.Unlock()

	// First call returns the cached provider.
	tp1, err := getOrCreateTokenProvider()
	require.NoError(t, err)
	assert.Equal(t, mock, tp1)

	// Second call returns the same cached instance.
	tp2, err := getOrCreateTokenProvider()
	require.NoError(t, err)
	assert.Equal(t, tp1, tp2)
}

func TestGetOrCreateTokenProvider_ReturnsSameInstance(t *testing.T) {
	// When a provider is already cached, repeated calls must return it.
	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	tokenProviderMu.Unlock()
	defer func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	}()

	mock := &auth.MockTokenProvider{Token: "same-instance"}
	tokenProviderMu.Lock()
	cachedTokenProvider = mock
	tokenProviderMu.Unlock()

	results := make([]auth.TokenProvider, 5)
	for i := range results {
		tp, err := getOrCreateTokenProvider()
		require.NoError(t, err)
		results[i] = tp
	}
	for _, tp := range results {
		assert.Equal(t, mock, tp)
	}
}

// ---------------------------------------------------------------------------
// validateScopeURLMatch — additional edge cases
// ---------------------------------------------------------------------------

func TestValidateScopeURLMatch_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		scope     string
		url       string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "empty scope host",
			scope:     "/.default",
			url:       "https://management.azure.com/test",
			wantErr:   true,
			errSubstr: "valid hosts",
		},
		{
			name:      "empty request host",
			scope:     "https://management.azure.com/.default",
			url:       "/.default",
			wantErr:   true,
			errSubstr: "valid hosts",
		},
		{
			name:      "both hosts empty",
			scope:     "not-a-url",
			url:       "also-not-a-url",
			wantErr:   true,
			errSubstr: "valid hosts",
		},
		{
			name:    "subdomain matches parent scope",
			scope:   "https://azure.com/.default",
			url:     "https://sub.azure.com/api",
			wantErr: false,
		},
		{
			name:    "deeply nested subdomain matches",
			scope:   "https://azure.com/.default",
			url:     "https://a.b.c.azure.com/api",
			wantErr: false,
		},
		{
			name:      "parent does not match subdomain scope",
			scope:     "https://sub.azure.com/.default",
			url:       "https://azure.com/api",
			wantErr:   true,
			errSubstr: "does not match",
		},
		{
			name:    "case insensitive match",
			scope:   "https://Management.Azure.COM/.default",
			url:     "https://management.azure.com/subscriptions",
			wantErr: false,
		},
		{
			name:    "IPv6 scope and URL match",
			scope:   "https://[2001:db8::1]/.default",
			url:     "https://[2001:db8::1]/api",
			wantErr: false,
		},
		{
			name:      "IPv6 scope and URL mismatch",
			scope:     "https://[2001:db8::1]/.default",
			url:       "https://[2001:db8::2]/api",
			wantErr:   true,
			errSubstr: "does not match",
		},
		{
			name:      "suffix not a real subdomain",
			scope:     "https://azure.com/.default",
			url:       "https://notazure.com/api",
			wantErr:   true,
			errSubstr: "does not match",
		},
		{
			name:    "exact match with port",
			scope:   "https://management.azure.com:443/.default",
			url:     "https://management.azure.com:443/subscriptions",
			wantErr: false,
		},
		{
			name:      "single-label scope (bare TLD) rejected",
			scope:     "https://com/.default",
			url:       "https://attacker.com/steal",
			wantErr:   true,
			errSubstr: "at least two labels",
		},
		{
			name:      "single-label scope (localhost) rejected",
			scope:     "https://localhost/.default",
			url:       "https://localhost/api",
			wantErr:   true,
			errSubstr: "at least two labels",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateScopeURLMatch(tc.scope, tc.url)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseHeaders — additional edge cases
// ---------------------------------------------------------------------------

func TestParseHeaders_EmptyHeadersMap(t *testing.T) {
	req := newCallToolRequest(map[string]any{
		"headers": map[string]any{},
	})
	headers, err := parseHeaders(req)
	require.NoError(t, err)
	assert.Empty(t, headers)
}

func TestParseHeaders_NoHeadersArgument(t *testing.T) {
	req := newCallToolRequest(map[string]any{})
	headers, err := parseHeaders(req)
	require.NoError(t, err)
	assert.Empty(t, headers)
}

func TestParseHeaders_NonStringValues(t *testing.T) {
	req := newCallToolRequest(map[string]any{
		"headers": map[string]any{
			"X-Valid":  "value",
			"X-Number": 123,
			"X-Bool":   true,
			"X-Nil":    nil,
		},
	})
	headers, err := parseHeaders(req)
	require.NoError(t, err)
	assert.Equal(t, "value", headers["X-Valid"])
	assert.Len(t, headers, 1, "only string-typed header values should be included")
}

func TestParseHeaders_MixedValidAndBlocked(t *testing.T) {
	req := newCallToolRequest(map[string]any{
		"headers": map[string]any{
			"X-Custom":      "value",
			"Authorization": "Bearer token",
		},
	})
	_, err := parseHeaders(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestParseHeaders_HeadersNotMap(t *testing.T) {
	req := newCallToolRequest(map[string]any{
		"headers": "not-a-map",
	})
	headers, err := parseHeaders(req)
	require.NoError(t, err)
	assert.Empty(t, headers, "non-map headers value should be silently ignored")
}

func TestParseHeaders_BlockedHeaderCaseInsensitive(t *testing.T) {
	variants := []string{"AUTHORIZATION", "authorization", "Authorization", "aUtHoRiZaTiOn"}
	for _, h := range variants {
		req := newCallToolRequest(map[string]any{
			"headers": map[string]any{h: "value"},
		})
		_, err := parseHeaders(req)
		require.Error(t, err, "header %q should be blocked", h)
	}
}

// ---------------------------------------------------------------------------
// formatResponse — additional edge cases
// ---------------------------------------------------------------------------

func TestFormatResponse_EmptyBody(t *testing.T) {
	resp := &mcpResponse{StatusCode: 204, Headers: map[string]string{}}
	out := formatResponse(resp)
	var parsed mcpResponse
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, 204, parsed.StatusCode)
	assert.Empty(t, parsed.Body)
}

func TestFormatResponse_NilHeaders(t *testing.T) {
	resp := &mcpResponse{StatusCode: 200, Body: "test"}
	out := formatResponse(resp)
	var parsed mcpResponse
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, 200, parsed.StatusCode)
	assert.Nil(t, parsed.Headers)
	assert.Equal(t, "test", parsed.Body)
}

func TestFormatResponse_LargeStatusCode(t *testing.T) {
	resp := &mcpResponse{StatusCode: 599, Body: "error"}
	out := formatResponse(resp)
	var parsed mcpResponse
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, 599, parsed.StatusCode)
}

// ---------------------------------------------------------------------------
// executeMCPRequest
// ---------------------------------------------------------------------------

func TestExecuteMCPRequest_BlockedURL(t *testing.T) {
	_, err := executeMCPRequest(context.Background(), "GET", "http://169.254.169.254/latest", "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestExecuteMCPRequest_BlockedLoopback(t *testing.T) {
	_, err := executeMCPRequest(context.Background(), "GET", "http://127.0.0.1:8080/admin", "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestExecuteMCPRequest_RateLimitExceeded(t *testing.T) {
	// Replace the global limiter with one that always rejects.
	origLimiter := limiter
	limiter = azdextutil.NewRateLimiter(0, 0) //nolint:staticcheck // test helper; deprecated API
	defer func() { limiter = origLimiter }()

	_, err := executeMCPRequest(context.Background(), "GET", "https://management.azure.com/test", "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
}

func TestExecuteMCPRequest_ScopeMismatch(t *testing.T) {
	// Scope override for a different domain should fail validation.
	_, err := executeMCPRequest(context.Background(), "GET",
		"https://management.azure.com/subscriptions", "", "https://evil.com/.default", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scope/URL mismatch")
}

func TestExecuteMCPRequest_CustomHeaders(t *testing.T) {
	// Test that custom headers are passed through (fails at auth, but covers header setup).
	_, err := executeMCPRequest(context.Background(), "POST",
		"https://management.azure.com/test", `{"key":"val"}`, "", map[string]string{"X-Custom": "value"})
	// Will fail at token provider, but that's fine — we're testing earlier paths.
	require.Error(t, err)
}

func TestExecuteMCPRequest_WithBody(t *testing.T) {
	// Test body path through executeMCPRequest.
	_, err := executeMCPRequest(context.Background(), "POST",
		"https://management.azure.com/test", `{"data":true}`, "", nil)
	require.Error(t, err) // Will fail at auth
}

func TestExecuteMCPRequest_InvalidScopeURL(t *testing.T) {
	// URL with no known scope and no override — scope detection returns empty.
	_, err := executeMCPRequest(context.Background(), "GET", "https://unknown-host-no-scope.example.com/path", "", "", nil)
	require.Error(t, err) // Will fail at auth since scope is empty
}

// resultText extracts the text from a CallToolResult's first content item.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, result.Content)
	tc, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected TextContent, got %T", result.Content[0])
	return tc.Text
}

func TestHandleBodyMethod_MissingURL(t *testing.T) {
	handler := handleBodyMethod("POST")
	result, err := handler(context.Background(), newCallToolRequest(map[string]any{}))
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "missing required argument: url")
}

func TestHandleBodyMethod_BlockedHeader(t *testing.T) {
	handler := handleBodyMethod("POST")
	req := newCallToolRequest(map[string]any{
		"url":     "https://management.azure.com/test",
		"headers": map[string]any{"Authorization": "Bearer token"},
	})
	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "not allowed")
}

func TestHandleBodyMethod_BlockedURL(t *testing.T) {
	handler := handleBodyMethod("PUT")
	req := newCallToolRequest(map[string]any{
		"url": "http://169.254.169.254/latest/meta-data/",
	})
	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "blocked")
}

func TestHandleNoBodyMethod_MissingURL(t *testing.T) {
	handler := handleNoBodyMethod("GET")
	result, err := handler(context.Background(), newCallToolRequest(map[string]any{}))
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "missing required argument: url")
}

func TestHandleNoBodyMethod_BlockedHeader(t *testing.T) {
	handler := handleNoBodyMethod("DELETE")
	req := newCallToolRequest(map[string]any{
		"url":     "https://management.azure.com/test",
		"headers": map[string]any{"Cookie": "session=abc"},
	})
	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "not allowed")
}

func TestHandleNoBodyMethod_BlockedURL(t *testing.T) {
	handler := handleNoBodyMethod("GET")
	req := newCallToolRequest(map[string]any{
		"url": "http://10.0.0.1/internal",
	})
	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "blocked")
}

func TestHandleHead_MissingURL(t *testing.T) {
	result, err := handleHead(context.Background(), newCallToolRequest(map[string]any{}))
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "missing required argument: url")
}

func TestHandleHead_BlockedURL(t *testing.T) {
	req := newCallToolRequest(map[string]any{
		"url": "http://127.0.0.1:8080/admin",
	})
	result, err := handleHead(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "blocked")
}

func TestHandleHead_BlockedHeader(t *testing.T) {
	req := newCallToolRequest(map[string]any{
		"url":     "https://management.azure.com/test",
		"headers": map[string]any{"Host": "evil.com"},
	})
	result, err := handleHead(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "not allowed")
}

// ---------------------------------------------------------------------------
// newMCPServer — tool registration
// ---------------------------------------------------------------------------

func TestNewMCPServer_RegistersAllTools(t *testing.T) {
	s := newMCPServer()
	tools := s.ListTools()

	expectedTools := []string{
		"rest_get", "rest_post", "rest_put",
		"rest_patch", "rest_delete", "rest_head",
	}

	assert.Len(t, tools, len(expectedTools))
	for _, name := range expectedTools {
		_, exists := tools[name]
		assert.True(t, exists, "tool %q should be registered", name)
	}
}

func TestNewMCPServer_ToolsHaveDescriptions(t *testing.T) {
	s := newMCPServer()
	tools := s.ListTools()

	for name, tool := range tools {
		desc := tool.Tool.Description
		assert.NotEmpty(t, desc, "tool %q should have a description", name)
	}
}

func TestNewMCPServer_ToolsRequireURL(t *testing.T) {
	s := newMCPServer()
	tools := s.ListTools()

	for name, tool := range tools {
		props := tool.Tool.InputSchema.Properties
		require.NotNil(t, props, "tool %q should have properties", name)
		_, hasURL := props["url"]
		assert.True(t, hasURL, "tool %q should have a url parameter", name)
	}
}

// ---------------------------------------------------------------------------
// isBlockedURL — additional IP format tests (no DNS, always safe in -short)
// ---------------------------------------------------------------------------

func TestIsBlockedIP_IPv6(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"::", true},           // IPv6 unspecified
		{"::1", true},          // IPv6 loopback
		{"fe80::1", true},      // IPv6 link-local
		{"2001:db8::1", false}, // documentation range, not blocked
	}
	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		require.NotNil(t, ip, "failed to parse IP: %s", tc.ip)
		assert.Equal(t, tc.blocked, isBlockedIP(ip), "isBlockedIP(%q)", tc.ip)
	}
}

func TestIsBlockedURL_InvalidURL(t *testing.T) {
	// Unparseable URL should be blocked.
	assert.True(t, isBlockedURL("://"))
	assert.True(t, isBlockedURL(""))
}

func TestIsBlockedURL_MetadataHosts(t *testing.T) {
	// Explicit blocklist hosts that aren't IP addresses.
	assert.True(t, isBlockedURL("http://metadata.google.internal/computeMetadata/v1/"))
	assert.True(t, isBlockedURL("http://fd00:ec2::254/latest"))
	assert.True(t, isBlockedURL("http://100.100.100.200/latest"))
}

// ---------------------------------------------------------------------------
// executeMCPRequest — full success path with httptest server
// ---------------------------------------------------------------------------

func TestExecuteMCPRequest_SuccessPath(t *testing.T) {
	// Stand up a test HTTP server that returns JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	}))
	defer server.Close()

	// Temporarily allow loopback for httptest.
	origCIDRs := blockedCIDRs
	origHosts := blockedHosts
	blockedCIDRs = nil
	blockedHosts = nil
	defer func() {
		blockedCIDRs = origCIDRs
		blockedHosts = origHosts
	}()

	// Pre-cache a mock token provider so we don't need Azure creds.
	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	cachedTokenProvider = &auth.MockTokenProvider{Token: "test-token"}
	tokenProviderMu.Unlock()
	defer func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	}()

	resp, err := executeMCPRequest(context.Background(), "GET", server.URL+"/api/test", "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Body, `"result":"ok"`)
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
}

func TestExecuteMCPRequest_PostWithBody(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()

	origCIDRs := blockedCIDRs
	origHosts := blockedHosts
	blockedCIDRs = nil
	blockedHosts = nil
	defer func() {
		blockedCIDRs = origCIDRs
		blockedHosts = origHosts
	}()

	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	cachedTokenProvider = &auth.MockTokenProvider{Token: "test-token"}
	tokenProviderMu.Unlock()
	defer func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	}()

	resp, err := executeMCPRequest(context.Background(), "POST", server.URL+"/api/resource", "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "POST", receivedMethod)
}

func TestExecuteMCPRequest_SkipAuthForHTTP(t *testing.T) {
	// HTTP (non-HTTPS) URLs should skip auth entirely.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no Authorization header is sent.
		assert.Empty(t, r.Header.Get("Authorization"), "HTTP requests should not send auth token")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"noauth":"ok"}`))
	}))
	defer server.Close()

	origCIDRs := blockedCIDRs
	origHosts := blockedHosts
	blockedCIDRs = nil
	blockedHosts = nil
	defer func() {
		blockedCIDRs = origCIDRs
		blockedHosts = origHosts
	}()

	// Clear cached token provider to verify no auth is attempted.
	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	cachedTokenProvider = nil
	tokenProviderMu.Unlock()
	defer func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	}()

	resp, err := executeMCPRequest(context.Background(), "GET", server.URL+"/api/test", "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
