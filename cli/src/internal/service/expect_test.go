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

func expectTestServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSplitExpectArg(t *testing.T) {
	cases := []struct {
		name         string
		raw          string
		wantExpr     string
		wantExpected string
		wantEquality bool
	}{
		{"truthy only", "value", "value", "", false},
		{"equality", "properties.state=Succeeded", "properties.state", "Succeeded", true},
		{"equality trims expr", "  a.b  =x", "a.b", "x", true},
		{"double equals stays truthy", "a=='b'", "a=='b'", "", false},
		{"not equals stays truthy", "a!='b'", "a!='b'", "", false},
		{"greater equals stays truthy", "length(value)>=3", "length(value)>=3", "", false},
		{"less equals stays truthy", "n<=3", "n<=3", "", false},
		{"first standalone equals wins", "a=b=c", "a", "b=c", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, expected, hasEq := splitExpectArg(tc.raw)
			assert.Equal(t, tc.wantExpr, expr)
			assert.Equal(t, tc.wantExpected, expected)
			assert.Equal(t, tc.wantEquality, hasEq)
		})
	}
}

func TestIsExpectTruthy(t *testing.T) {
	assert.False(t, isExpectTruthy(nil))
	assert.False(t, isExpectTruthy(false))
	assert.False(t, isExpectTruthy(""))
	assert.False(t, isExpectTruthy([]any{}))
	assert.False(t, isExpectTruthy(map[string]any{}))
	assert.True(t, isExpectTruthy(true))
	assert.True(t, isExpectTruthy("x"))
	assert.True(t, isExpectTruthy([]any{1}))
	assert.True(t, isExpectTruthy(map[string]any{"a": 1}))
	assert.True(t, isExpectTruthy(0.0))
}

func TestEvaluateExpectations_Truthy(t *testing.T) {
	body := []byte(`{"value":[{"id":1}],"enabled":true,"name":""}`)

	require.NoError(t, evaluateExpectations(body, "application/json", []string{"value"}))
	require.NoError(t, evaluateExpectations(body, "application/json", []string{"enabled"}))

	// An empty string is falsy under JMESPath rules, so the assertion fails.
	err := evaluateExpectations(body, "application/json", []string{"name"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not truthy")
}

func TestEvaluateExpectations_Equality(t *testing.T) {
	body := []byte(`{"properties":{"provisioningState":"Succeeded"},"count":3,"active":true}`)

	require.NoError(t, evaluateExpectations(body, "application/json",
		[]string{"properties.provisioningState=Succeeded"}))
	require.NoError(t, evaluateExpectations(body, "application/json", []string{"count=3"}))
	require.NoError(t, evaluateExpectations(body, "application/json", []string{"active=true"}))

	// A missing field resolves to null and can be asserted with "=null".
	require.NoError(t, evaluateExpectations(body, "application/json", []string{"missing=null"}))

	err := evaluateExpectations(body, "application/json",
		[]string{"properties.provisioningState=Failed"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `expected "Failed"`)
	assert.Contains(t, err.Error(), `got "Succeeded"`)
}

func TestEvaluateExpectations_MultipleStopAtFirstFailure(t *testing.T) {
	body := []byte(`{"a":1,"b":2}`)
	err := evaluateExpectations(body, "application/json", []string{"a=1", "b=99", "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `--expect "b=99" failed`)
}

func TestEvaluateExpectations_UsageErrorsExit2(t *testing.T) {
	var coder exitCoder

	// Non-JSON body.
	err := evaluateExpectations([]byte("plain text"), "text/plain", []string{"a"})
	require.Error(t, err)
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.Contains(t, err.Error(), "requires a JSON response")

	// Invalid JMESPath expression.
	err = evaluateExpectations([]byte(`{"a":1}`), "application/json", []string{"a[["})
	require.Error(t, err)
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())

	// Empty expression.
	err = evaluateExpectations([]byte(`{"a":1}`), "application/json", []string{"=x"})
	require.Error(t, err)
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
}

func TestEvaluateExpectations_NoExpectsIsNoOp(t *testing.T) {
	require.NoError(t, evaluateExpectations([]byte("not json"), "text/plain", nil))
}

func TestExecute_Expect_PassPrintsBody(t *testing.T) {
	srv := expectTestServer(t, `{"status":"Succeeded"}`)

	cfg := baseTestConfig(t)
	cfg.Expect = []string{"status=Succeeded"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL)
	require.NoError(t, err)

	out, readErr := os.ReadFile(cfg.OutputFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "Succeeded")
}

func TestExecute_Expect_FailureReturnsExit1AndStillPrints(t *testing.T) {
	srv := expectTestServer(t, `{"status":"Failed"}`)

	cfg := baseTestConfig(t)
	cfg.Expect = []string{"status=Succeeded"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL)
	require.Error(t, err)

	// An assertion failure is a plain error, so it exits 1, not 2 (usage).
	var coder exitCoder
	assert.False(t, errors.As(err, &coder), "assertion failure should not be an ExitCoder")

	// The body is written before the assertion is checked.
	out, readErr := os.ReadFile(cfg.OutputFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "Failed")
}

func TestExecute_Expect_IndependentOfQuery(t *testing.T) {
	// --query narrows the printed output, but --expect still asserts against the
	// full response body.
	srv := expectTestServer(t, `{"name":"vm1","properties":{"state":"Succeeded"}}`)

	cfg := baseTestConfig(t)
	cfg.Query = "name"
	cfg.Expect = []string{"properties.state=Succeeded"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL)
	require.NoError(t, err)

	out, readErr := os.ReadFile(cfg.OutputFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "vm1")
}
