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

const testSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["name", "age"],
  "properties": {
    "name": { "type": "string" },
    "age": { "type": "integer", "minimum": 0 }
  }
}`

// writeSchema writes content to a temp file and returns its path.
func writeSchema(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "schema.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestValidateResponseSchema_ConformingReturnsNil(t *testing.T) {
	schema := writeSchema(t, testSchema)

	var errOut bytes.Buffer
	err := validateResponseSchema(&errOut, []byte(`{"name":"rg1","age":3}`), schema)
	require.NoError(t, err)
	assert.Empty(t, errOut.String())
}

func TestValidateResponseSchema_NonConformingPrintsErrorsAndReturnsError(t *testing.T) {
	schema := writeSchema(t, testSchema)

	// Missing "age" and wrong type for "name".
	var errOut bytes.Buffer
	err := validateResponseSchema(&errOut, []byte(`{"name":123}`), schema)
	require.Error(t, err)

	// A conformance failure is a plain error (exit 1), not a usage error.
	var coder exitCoder
	assert.False(t, errors.As(err, &coder), "conformance failure should not carry a usage exit code")
	assert.Contains(t, err.Error(), "does not conform")

	printed := errOut.String()
	assert.NotEmpty(t, printed, "each validation error should be printed")
}

func TestValidateResponseSchema_MissingSchemaReturnsUsageError(t *testing.T) {
	var errOut bytes.Buffer
	err := validateResponseSchema(&errOut, []byte(`{"name":"rg1","age":3}`), filepath.Join(t.TempDir(), "nope.json"))
	require.Error(t, err)

	var coder exitCoder
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.Empty(t, errOut.String())
}

func TestValidateResponseSchema_InvalidSchemaJSONReturnsUsageError(t *testing.T) {
	schema := writeSchema(t, "this is not json")

	var errOut bytes.Buffer
	err := validateResponseSchema(&errOut, []byte(`{"name":"rg1","age":3}`), schema)
	require.Error(t, err)

	var coder exitCoder
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.Contains(t, err.Error(), "not valid JSON")
}

func TestValidateResponseSchema_NonJSONResponseReturnsUsageError(t *testing.T) {
	schema := writeSchema(t, testSchema)

	var errOut bytes.Buffer
	err := validateResponseSchema(&errOut, []byte("plain text body"), schema)
	require.Error(t, err)

	var coder exitCoder
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.Contains(t, err.Error(), "requires a JSON response")
}

func TestExecute_ValidateSchema_ConformingReturnsNilAndWritesBody(t *testing.T) {
	srv := failTestServer(t, http.StatusOK, `{"name":"rg1","age":3}`)
	schema := writeSchema(t, testSchema)

	cfg := baseTestConfig(t)
	cfg.ValidateSchema = schema

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/data")
	require.NoError(t, err)

	out, readErr := os.ReadFile(cfg.OutputFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "rg1")
}

func TestExecute_ValidateSchema_NonConformingReturnsErrorAndWritesBody(t *testing.T) {
	srv := failTestServer(t, http.StatusOK, `{"name":"rg1"}`)
	schema := writeSchema(t, testSchema)

	cfg := baseTestConfig(t)
	cfg.ValidateSchema = schema

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/data")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not conform")

	// The response body is still written before the failure is returned.
	out, readErr := os.ReadFile(cfg.OutputFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "rg1")
}
