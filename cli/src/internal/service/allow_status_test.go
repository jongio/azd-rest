package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseAllowedStatuses(t *testing.T) {
	ranges, err := parseAllowedStatuses("200-204, 404")
	require.NoError(t, err)
	require.True(t, ranges.allows(200))
	require.True(t, ranges.allows(202))
	require.True(t, ranges.allows(404))
	require.False(t, ranges.allows(409))
}

func TestParseAllowedStatusesInvalid(t *testing.T) {
	_, err := parseAllowedStatuses("204-200")
	require.Error(t, err)

	var usage interface{ ExitCode() int }
	require.True(t, errors.As(err, &usage))
	require.Equal(t, 2, usage.ExitCode())
}

func TestExecute_FailAllowsConfiguredStatus(t *testing.T) {
	srv := failTestServer(t, 404, `{"error":"missing"}`)
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Fail = true
	cfg.AllowStatus = "404"

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/missing")
	require.NoError(t, err)
}

func TestExecute_FailRejectsDisallowedStatus(t *testing.T) {
	srv := failTestServer(t, 409, `{"error":"conflict"}`)
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Fail = true
	cfg.AllowStatus = "404"

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/conflict")
	require.Error(t, err)
	var failErr *httpFailError
	require.True(t, errors.As(err, &failErr))
}

func TestExecute_InvalidAllowedStatusFailsBeforeRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.Fail = true
	cfg.AllowStatus = "abc"

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/missing")
	require.Error(t, err)
	var usage interface{ ExitCode() int }
	require.True(t, errors.As(err, &usage))
	require.Equal(t, 2, usage.ExitCode())
	require.False(t, called)
}
