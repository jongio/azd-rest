package service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeBaseline writes content to a temp file and returns its path.
func writeBaseline(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "baseline.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestDiffAgainstBaseline_MatchIgnoresKeyOrderAndWhitespace(t *testing.T) {
	baseline := writeBaseline(t, `{"b": 2, "a": 1}`)
	body := []byte("{\n  \"a\": 1,\n  \"b\": 2\n}")

	var out bytes.Buffer
	err := diffAgainstBaseline(&out, body, baseline)
	require.NoError(t, err)
	assert.Empty(t, out.String(), "matching documents must print nothing")
}

func TestDiffAgainstBaseline_MatchNestedKeyOrder(t *testing.T) {
	baseline := writeBaseline(t, `{"outer":{"z":true,"a":[3,2,1]}}`)
	body := []byte(`{"outer":{"a":[3,2,1],"z":true}}`)

	var out bytes.Buffer
	err := diffAgainstBaseline(&out, body, baseline)
	require.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestDiffAgainstBaseline_DriftPrintsUnifiedDiffAndReturnsError(t *testing.T) {
	baseline := writeBaseline(t, `{"name":"old"}`)
	body := []byte(`{"name":"new"}`)

	var out bytes.Buffer
	err := diffAgainstBaseline(&out, body, baseline)
	require.Error(t, err)

	// Drift is a plain error, not a usage error, so it exits 1 (no ExitCoder).
	var coder exitCoder
	assert.False(t, errors.As(err, &coder), "drift should not carry a usage exit code")

	printed := out.String()
	assert.Contains(t, printed, "--- baseline")
	assert.Contains(t, printed, "+++ response")
	assert.Contains(t, printed, `-  "name": "old"`)
	assert.Contains(t, printed, `+  "name": "new"`)
}

func TestDiffAgainstBaseline_MissingBaselineReturnsUsageError(t *testing.T) {
	var out bytes.Buffer
	err := diffAgainstBaseline(&out, []byte(`{"a":1}`), filepath.Join(t.TempDir(), "nope.json"))
	require.Error(t, err)

	var coder exitCoder
	require.True(t, errors.As(err, &coder), "missing baseline should implement ExitCoder")
	assert.Equal(t, 2, coder.ExitCode())
	assert.Empty(t, out.String())
}

func TestDiffAgainstBaseline_NonJSONResponseReturnsUsageError(t *testing.T) {
	baseline := writeBaseline(t, `{"a":1}`)

	var out bytes.Buffer
	err := diffAgainstBaseline(&out, []byte("plain text body"), baseline)
	require.Error(t, err)

	var coder exitCoder
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.Contains(t, err.Error(), "requires a JSON response")
}

func TestDiffAgainstBaseline_NonJSONBaselineReturnsUsageError(t *testing.T) {
	baseline := writeBaseline(t, "not json at all")

	var out bytes.Buffer
	err := diffAgainstBaseline(&out, []byte(`{"a":1}`), baseline)
	require.Error(t, err)

	var coder exitCoder
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.Contains(t, err.Error(), "not valid JSON")
}

func TestExecute_Diff_MatchReturnsNil(t *testing.T) {
	srv := failTestServer(t, http.StatusOK, `{"a":1,"b":2}`)
	baseline := writeBaseline(t, `{"b":2,"a":1}`)

	cfg := baseTestConfig(t)
	cfg.Diff = baseline

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/data")
	require.NoError(t, err)
}

func TestExecute_Diff_DriftReturnsError(t *testing.T) {
	srv := failTestServer(t, http.StatusOK, `{"a":1,"b":3}`)
	baseline := writeBaseline(t, `{"a":1,"b":2}`)

	cfg := baseTestConfig(t)
	cfg.Diff = baseline

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/data")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "differs from --diff baseline")
}
