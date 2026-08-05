package service

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/require"
)

func captureDryRunOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := stdoutWriter
	stdoutWriter = &buf
	t.Cleanup(func() { stdoutWriter = previous })
	return &buf
}

func TestExecute_DryRunDoesNotCallHTTPClient(t *testing.T) {
	out := captureDryRunOutput(t)
	httpCalled := false
	tokenCalled := false
	svc := NewRequestService(
		func() (client.TokenProvider, error) {
			tokenCalled = true
			return nil, nil
		},
		func(tp client.TokenProvider, insecure bool, timeout time.Duration) *client.Client {
			httpCalled = true
			return DefaultHTTPClientFactory(tp, insecure, timeout)
		},
	)

	cfg := config.Defaults()
	cfg.DryRun = true
	cfg.NoAuth = true
	cfg.Headers = []string{"Authorization: secret", "X-Test: ok"}
	cfg.Data = `{"secret":"not printed"}`

	err := svc.Execute(context.Background(), cfg, "POST", "https://example.com/resource?x=1")
	require.NoError(t, err)
	require.False(t, httpCalled)
	require.False(t, tokenCalled)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, "POST", got["method"])
	require.Equal(t, "none", got["authMode"])
	require.NotContains(t, out.String(), "not printed")
	require.NotContains(t, out.String(), "secret\"")
	require.Contains(t, out.String(), "REDACTED")
}

func TestWriteDryRunIncludesBodySourceAndSize(t *testing.T) {
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.Data = "hello"
	opts, cleanup, err := newTestService().BuildRequestOptions(cfg, "PATCH", "https://example.com")
	require.NoError(t, err)
	defer cleanup()

	var out bytes.Buffer
	require.NoError(t, writeDryRun(&out, cfg, opts))

	var got dryRunOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, "PATCH", got.Method)
	require.NotNil(t, got.Body)
	require.Equal(t, "data", got.Body.Source)
	require.EqualValues(t, 5, got.Body.Bytes)
}
