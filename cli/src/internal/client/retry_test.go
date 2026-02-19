package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Execute_RetryOn5xx(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal server error"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true}`))
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

func TestClient_Execute_NoRetryOn4xx(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
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

func TestClient_Execute_ResponseSizeLimit(t *testing.T) {
	largeBody := make([]byte, 101*1024*1024)
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(largeBody)
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:          "GET",
		URL:             server.URL + "/test",
		SkipAuth:        true,
		MaxResponseSize: 100 * 1024 * 1024,
	}

	_, err := client.Execute(context.Background(), opts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestClient_Execute_RetryExponentialBackoff(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL + "/test",
		SkipAuth: true,
		Retry:    2,
	}

	start := time.Now()
	resp, err := client.Execute(context.Background(), opts)
	duration := time.Since(start)

	require.NoError(t, err, "5xx responses should not cause errors, just retries")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.GreaterOrEqual(t, attemptCount, 1, "Should have made at least one attempt")
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond, "Should have taken some time for retries")
}
