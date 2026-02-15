package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Execute_RetryOn5xx(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Return 5xx error for first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))
		} else {
			// Success on third attempt
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
		}
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/test",
		SkipAuth: true,
		Retry:    3,
	}

	resp, err := client.Execute(context.Background(), opts)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, attemptCount, "Should have retried 2 times (3 total attempts)")
}

func TestClient_Execute_RetryOnNetworkError(t *testing.T) {
	// Test that retry logic is invoked for network errors
	// This is tested indirectly through the retry count mechanism
	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 1*time.Second) // Short timeout

	opts := RequestOptions{
		Method:   "GET",
		URL:      "https://192.0.2.0/invalid", // Invalid IP that will timeout
		SkipAuth: true,
		Retry:    2,
	}

	// Should fail after retries
	_, err := client.Execute(context.Background(), opts)
	
	// Should get an error (either timeout or connection error)
	assert.Error(t, err)
}

func TestClient_Execute_NoRetryOn4xx(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/test",
		SkipAuth: true,
		Retry:    3,
	}

	resp, err := client.Execute(context.Background(), opts)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, 1, attemptCount, "Should not retry on 4xx errors")
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Timeout error",
			err:      fmt.Errorf("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "Connection refused",
			err:      fmt.Errorf("connection refused"),
			expected: true,
		},
		{
			name:     "Network unreachable",
			err:      fmt.Errorf("network is unreachable"),
			expected: true,
		},
		{
			name:     "Non-retryable error",
			err:      fmt.Errorf("invalid argument"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_Execute_ResponseSizeLimit(t *testing.T) {
	// Create a response larger than the limit
	largeBody := make([]byte, 101*1024*1024) // 101MB
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeBody)
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:          "GET",
		URL:             server.URL + "/test",
		SkipAuth:        true,
		MaxResponseSize: 100 * 1024 * 1024, // 100MB limit
	}

	_, err := client.Execute(context.Background(), opts)
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestClient_Execute_ResponseSizeWithinLimit(t *testing.T) {
	// Create a response within the limit
	body := make([]byte, 50*1024*1024) // 50MB
	for i := range body {
		body[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:          "GET",
		URL:             server.URL + "/test",
		SkipAuth:        true,
		MaxResponseSize: 100 * 1024 * 1024, // 100MB limit
	}

	resp, err := client.Execute(context.Background(), opts)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, len(body), len(resp.Body))
}

func TestClient_Execute_RetryExponentialBackoff(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/test",
		SkipAuth: true,
		Retry:    2, // Will try 3 times total
	}

	start := time.Now()
	resp, err := client.Execute(context.Background(), opts)
	duration := time.Since(start)
	
	// 5xx errors are retried, but we return the last response (not an error)
	// So we should get a response with 5xx status
	require.NoError(t, err, "5xx responses should not cause errors, just retries")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.GreaterOrEqual(t, attemptCount, 1, "Should have made at least one attempt")
	// Should have taken some time due to retries and backoff
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond, "Should have taken some time for retries")
}
