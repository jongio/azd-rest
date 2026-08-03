package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exitCoder mirrors the cmd.ExitCoder contract so the service tests can assert
// the process exit code without importing the cmd package.
type exitCoder interface {
	error
	ExitCode() int
}

func failTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestExecute_Fail_ErrorStatusReturnsExit22(t *testing.T) {
	srv := failTestServer(t, http.StatusNotFound, `{"error":"not found"}`)

	cfg := baseTestConfig(t)
	cfg.Fail = true

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/missing")
	require.Error(t, err)

	var coder exitCoder
	require.True(t, errors.As(err, &coder), "fail error should implement ExitCoder")
	assert.Equal(t, 22, coder.ExitCode())

	// The response body is still written before the failure is returned.
	out, readErr := os.ReadFile(cfg.OutputFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "not found")
}

func TestExecute_Fail_SuccessStatusReturnsNil(t *testing.T) {
	srv := failTestServer(t, http.StatusOK, `{"ok":true}`)

	cfg := baseTestConfig(t)
	cfg.Fail = true

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/ok")
	require.NoError(t, err)
}

func TestExecute_Fail_RedirectStatusReturnsNil(t *testing.T) {
	// A 3xx status is below 400 and must not trigger --fail. Redirect following
	// is disabled so the 302 is the final response the service sees.
	srv := failTestServer(t, http.StatusFound, `{"moved":true}`)

	cfg := baseTestConfig(t)
	cfg.Fail = true
	cfg.FollowRedirects = false

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/moved")
	require.NoError(t, err)
}

func TestExecute_Fail_DisabledLeavesExitUnchanged(t *testing.T) {
	srv := failTestServer(t, http.StatusInternalServerError, `{"error":"boom"}`)

	cfg := baseTestConfig(t)
	cfg.Fail = false

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/boom")
	require.NoError(t, err)
}
