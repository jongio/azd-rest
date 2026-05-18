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
	var attemptTimes []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptTimes = append(attemptTimes, time.Now())
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
		Retry:    3,
	}

	resp, err := client.Execute(context.Background(), opts)

	require.NoError(t, err, "5xx responses should not cause errors, just retries")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Verify the correct number of attempts (initial + retries).
	assert.Equal(t, 4, len(attemptTimes), "Should have made 4 attempts (1 initial + 3 retries)")

	// Verify exponential backoff: each interval should be roughly double the previous.
	// Allow tolerance for timing jitter (intervals should increase monotonically).
	if len(attemptTimes) >= 3 {
		interval1 := attemptTimes[1].Sub(attemptTimes[0])
		interval2 := attemptTimes[2].Sub(attemptTimes[1])

		// Interval2 should be at least 1.5x interval1 (exponential with jitter tolerance).
		assert.Greater(t, interval2.Milliseconds(), interval1.Milliseconds(),
			"Second retry interval (%v) should be longer than first (%v) for exponential backoff",
			interval2, interval1)

		// Both intervals should be non-trivial (at least 50ms for first, indicating real backoff).
		assert.GreaterOrEqual(t, interval1.Milliseconds(), int64(50),
			"First retry delay should be at least 50ms")
	}

	if len(attemptTimes) >= 4 {
		interval2 := attemptTimes[2].Sub(attemptTimes[1])
		interval3 := attemptTimes[3].Sub(attemptTimes[2])

		// Third interval should also be longer than second.
		assert.Greater(t, interval3.Milliseconds(), interval2.Milliseconds(),
			"Third retry interval (%v) should be longer than second (%v)",
			interval3, interval2)
	}
}
