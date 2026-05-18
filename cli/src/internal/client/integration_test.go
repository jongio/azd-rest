package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullRequestLifecycle verifies the complete request lifecycle:
// authentication, headers, body, response parsing, and timing.
func TestIntegration_FullRequestLifecycle(t *testing.T) {
	var (
		receivedMethod string
		receivedPath   string
		receivedAuth   string
		receivedBody   string
		receivedUA     string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")
		receivedUA = r.Header.Get("User-Agent")
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","id":"resource-1"}`))
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "integration-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method: "POST",
		URL:    server.URL + "/api/v1/resources",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "test-value",
		},
		Body:     strings.NewReader(`{"name":"my-resource"}`),
		SkipAuth: false,
		Scope:    "https://management.azure.com/.default",
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)

	// Verify request was correctly formed.
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/resources", receivedPath)
	assert.Equal(t, "Bearer integration-token", receivedAuth)
	assert.Equal(t, `{"name":"my-resource"}`, receivedBody)
	assert.Contains(t, receivedUA, "azd-rest")

	// Verify response parsing.
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Greater(t, resp.Duration, time.Duration(0))

	var body map[string]interface{}
	err = json.Unmarshal(resp.Body, &body)
	require.NoError(t, err)
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, "resource-1", body["id"])
}

// TestIntegration_ErrorStatusCodes verifies handling of various HTTP error codes.
func TestIntegration_ErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "400 Bad Request", statusCode: 400, body: `{"error":{"code":"BadRequest","message":"Invalid input"}}`},
		{name: "401 Unauthorized", statusCode: 401, body: `{"error":{"code":"Unauthorized","message":"Token expired"}}`},
		{name: "403 Forbidden", statusCode: 403, body: `{"error":{"code":"Forbidden","message":"Insufficient permissions"}}`},
		{name: "404 Not Found", statusCode: 404, body: `{"error":{"code":"NotFound","message":"Resource not found"}}`},
		{name: "409 Conflict", statusCode: 409, body: `{"error":{"code":"Conflict","message":"Resource already exists"}}`},
		{name: "429 Too Many Requests", statusCode: 429, body: `{"error":{"code":"TooManyRequests","message":"Rate limited"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			provider := &auth.MockTokenProvider{Token: "test-token"}
			client := NewClient(provider, false, 30*time.Second)

			opts := RequestOptions{
				Method:   "GET",
				URL:      server.URL + "/test",
				SkipAuth: true,
				Retry:    0, // No retries for these tests
			}

			resp, err := client.Execute(context.Background(), opts)
			require.NoError(t, err, "HTTP error responses should not return Go errors")
			assert.Equal(t, tt.statusCode, resp.StatusCode)
			assert.Equal(t, tt.body, string(resp.Body))
		})
	}
}

// TestIntegration_LargeResponseBody verifies handling of large JSON responses
// typical of Azure list operations.
func TestIntegration_LargeResponseBody(t *testing.T) {
	// Build a response with 100 items.
	items := make([]map[string]interface{}, 100)
	for i := range items {
		items[i] = map[string]interface{}{
			"id":       i,
			"name":     "resource-" + strings.Repeat("x", 100),
			"location": "eastus",
		}
	}
	responseData, _ := json.Marshal(map[string]interface{}{"value": items})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseData)
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/list",
		SkipAuth: true,
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed map[string]interface{}
	err = json.Unmarshal(resp.Body, &parsed)
	require.NoError(t, err)
	valueArr, ok := parsed["value"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 100, len(valueArr))
}

// TestIntegration_RetryThenSucceed verifies retry succeeds after transient failures.
func TestIntegration_RetryThenSucceed(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"service unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recovered":true}`))
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/flaky",
		SkipAuth: true,
		Retry:    5,
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, attemptCount, "Should succeed on third attempt")

	var body map[string]interface{}
	err = json.Unmarshal(resp.Body, &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["recovered"])
}

// TestIntegration_CustomHeaders verifies custom headers are sent and
// standard headers coexist correctly.
func TestIntegration_CustomHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		checks  func(t *testing.T, r *http.Request)
	}{
		{
			name: "Content-Type override",
			headers: map[string]string{
				"Content-Type": "application/xml",
			},
			checks: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "application/xml", r.Header.Get("Content-Type"))
			},
		},
		{
			name: "Multiple custom headers",
			headers: map[string]string{
				"X-Correlation-Id": "abc-123",
				"X-Request-Source": "azd-rest-test",
				"Accept":           "application/json",
			},
			checks: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "abc-123", r.Header.Get("X-Correlation-Id"))
				assert.Equal(t, "azd-rest-test", r.Header.Get("X-Request-Source"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
			},
		},
		{
			name: "API version header",
			headers: map[string]string{
				"x-ms-version": "2024-01-01",
			},
			checks: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "2024-01-01", r.Header.Get("x-ms-version"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.checks(t, r)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			provider := &auth.MockTokenProvider{Token: "test-token"}
			client := NewClient(provider, false, 30*time.Second)

			opts := RequestOptions{
				Method:   "GET",
				URL:      server.URL + "/test",
				Headers:  tt.headers,
				SkipAuth: true,
			}

			resp, err := client.Execute(context.Background(), opts)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

// TestIntegration_SkipAuthForHTTPURLs verifies that when SkipAuth is true,
// no authorization header is sent regardless of the token provider.
func TestIntegration_SkipAuthForHTTPURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.Empty(t, authHeader, "SkipAuth=true should not send Authorization header")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"public":true}`))
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "should-not-be-sent"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/public",
		SkipAuth: true, // Explicitly skip auth (HTTP auto-skip is handled by CLI layer)
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestIntegration_ResponseHeaders verifies response headers are captured.
func TestIntegration_ResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Ms-Request-Id", "azure-req-456")
		w.Header().Set("X-Ms-Correlation-Request-Id", "corr-789")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/headers",
		SkipAuth: true,
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
	assert.Equal(t, "azure-req-456", resp.Headers.Get("X-Ms-Request-Id"))
	assert.Equal(t, "corr-789", resp.Headers.Get("X-Ms-Correlation-Request-Id"))
}

// TestIntegration_EmptyResponseBody verifies handling of responses with no body.
func TestIntegration_EmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "DELETE",
		URL:      server.URL + "/resource/123",
		SkipAuth: true,
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Empty(t, resp.Body)
}
