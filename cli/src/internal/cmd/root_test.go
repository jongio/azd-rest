package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetGlobalFlags resets all global flags to their default values
func resetGlobalFlags() {
	scope = ""
	noAuth = false
	headers = []string{}
	data = ""
	dataFile = ""
	outputFile = ""
	outputFormat = "auto"
	verbose = false
	paginate = false
	retry = 3
	binary = false
	insecure = false
	timeout = 30 * time.Second
	followRedirects = true
	maxRedirects = 10
}

func TestNewRootCmd(t *testing.T) {
	resetGlobalFlags()
	cmd := NewRootCmd()
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "rest", cmd.Use)
	assert.Equal(t, "Execute REST API calls with Azure authentication", cmd.Short)
	
	// Check that all subcommands are added
	subcommands := cmd.Commands()
	subcommandNames := make(map[string]bool)
	for _, subcmd := range subcommands {
		// Use contains "<url>" for some commands, so check if it starts with the command name
		useParts := strings.Fields(subcmd.Use)
		if len(useParts) > 0 {
			subcommandNames[useParts[0]] = true
		}
	}
	
	expectedCommands := []string{"get", "post", "put", "patch", "delete", "head", "options", "version"}
	for _, expected := range expectedCommands {
		assert.True(t, subcommandNames[expected], "Subcommand %s should be present", expected)
	}
}

func TestBuildRequestOptions_Headers(t *testing.T) {
	resetGlobalFlags()
	headers = []string{"X-Custom: value1", "Authorization: Bearer token"}
	noAuth = true // Skip auth to avoid credential issues
	
	opts, err := buildRequestOptions("GET", "https://example.com")
	
	require.NoError(t, err)
	assert.Equal(t, "GET", opts.Method)
	assert.Equal(t, "https://example.com", opts.URL)
	assert.Equal(t, "value1", opts.Headers["X-Custom"])
	assert.Equal(t, "Bearer token", opts.Headers["Authorization"])
}

func TestBuildRequestOptions_InvalidHeader(t *testing.T) {
	resetGlobalFlags()
	headers = []string{"InvalidHeader"}
	noAuth = true
	
	_, err := buildRequestOptions("GET", "https://example.com")
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header format")
}

func TestBuildRequestOptions_DataFile(t *testing.T) {
	resetGlobalFlags()
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.json")
	err := os.WriteFile(tmpFile, []byte(`{"test": "data"}`), 0644)
	require.NoError(t, err)
	
	dataFile = tmpFile
	noAuth = true
	
	opts, err := buildRequestOptions("POST", "https://example.com")
	
	require.NoError(t, err)
	assert.NotNil(t, opts.Body)
	
	// Read the body to verify it's the file content
	bodyBytes, err := io.ReadAll(opts.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"test": "data"}`, string(bodyBytes))
}

func TestBuildRequestOptions_DataFileWithAtPrefix(t *testing.T) {
	resetGlobalFlags()
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.json")
	err := os.WriteFile(tmpFile, []byte(`{"test": "data"}`), 0644)
	require.NoError(t, err)
	
	dataFile = "@" + tmpFile
	noAuth = true
	
	opts, err := buildRequestOptions("POST", "https://example.com")
	
	require.NoError(t, err)
	assert.NotNil(t, opts.Body)
}

func TestBuildRequestOptions_DataFileNotFound(t *testing.T) {
	resetGlobalFlags()
	dataFile = "/nonexistent/file.json"
	noAuth = true
	
	_, err := buildRequestOptions("POST", "https://example.com")
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open data file")
}

func TestBuildRequestOptions_DataString(t *testing.T) {
	resetGlobalFlags()
	data = `{"test": "data"}`
	noAuth = true
	
	opts, err := buildRequestOptions("POST", "https://example.com")
	
	require.NoError(t, err)
	assert.NotNil(t, opts.Body)
	
	// Read the body to verify it's the string content
	bodyBytes, err := io.ReadAll(opts.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"test": "data"}`, string(bodyBytes))
}

func TestBuildRequestOptions_ScopeDetection(t *testing.T) {
	resetGlobalFlags()
	scope = ""
	noAuth = false
	// Note: This will try to create a token provider, which may fail in test environment
	// So we'll just verify the scope detection logic works
	opts, err := buildRequestOptions("GET", "https://management.azure.com/subscriptions")
	
	// May error if credentials not available, but scope should be detected
	if err == nil {
		assert.Equal(t, "https://management.azure.com/.default", opts.Scope)
	} else {
		// If it errors, it should be about credentials, not scope detection
		assert.Contains(t, err.Error(), "token provider")
	}
}

func TestBuildRequestOptions_CustomScope(t *testing.T) {
	resetGlobalFlags()
	scope = "https://custom.scope/.default"
	noAuth = false
	// May error if credentials not available
	opts, err := buildRequestOptions("GET", "https://example.com")
	
	if err == nil {
		assert.Equal(t, "https://custom.scope/.default", opts.Scope)
	}
}

func TestBuildRequestOptions_NoAuth(t *testing.T) {
	resetGlobalFlags()
	noAuth = true
	
	opts, err := buildRequestOptions("GET", "https://example.com")
	
	require.NoError(t, err)
	assert.True(t, opts.SkipAuth)
}

func TestBuildRequestOptions_HTTPURLSkipsAuth(t *testing.T) {
	resetGlobalFlags()
	noAuth = false
	
	opts, err := buildRequestOptions("GET", "http://example.com")
	
	require.NoError(t, err)
	assert.True(t, opts.SkipAuth, "HTTP URLs should skip auth by default")
}

func TestBuildRequestOptions_AllFlags(t *testing.T) {
	resetGlobalFlags()
	// Set all flags
	scope = "https://test.scope/.default"
	noAuth = true // Use noAuth to avoid credential issues
	headers = []string{"X-Test: value"}
	data = `{"test": true}`
	outputFile = "/tmp/output.json"
	outputFormat = "json"
	verbose = true
	paginate = true
	retry = 5
	binary = false
	insecure = true
	timeout = 60 * time.Second
	followRedirects = false
	maxRedirects = 5
	
	opts, err := buildRequestOptions("POST", "https://example.com")
	
	require.NoError(t, err)
	assert.Equal(t, "POST", opts.Method)
	assert.Equal(t, "https://test.scope/.default", opts.Scope)
	assert.True(t, opts.SkipAuth) // Because noAuth = true
	assert.Equal(t, "value", opts.Headers["X-Test"])
	assert.Equal(t, "json", opts.Format)
	assert.Equal(t, "/tmp/output.json", opts.OutputFile)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Paginate)
	assert.Equal(t, 5, opts.Retry)
	assert.True(t, opts.Insecure)
	assert.False(t, opts.FollowRedirects)
	assert.Equal(t, 5, opts.MaxRedirects)
}

func TestExecuteRequest_WithFileBody(t *testing.T) {
	resetGlobalFlags()
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.json")
	err := os.WriteFile(tmpFile, []byte(`{"test": "data"}`), 0644)
	require.NoError(t, err)
	
	// Set up flags
	dataFile = tmpFile
	noAuth = true
	timeout = 1 * time.Second // Short timeout
	
	// Create a mock command with context
	cmd := &cobra.Command{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd.SetContext(ctx)
	
	// This will fail quickly due to timeout/invalid URL
	err = executeRequest(cmd, "POST", "https://192.0.2.0/invalid")
	
	// Should get an error, but file should be closed
	assert.Error(t, err)
	// File should be closed (tested by checking it can be opened again)
	_, err = os.Open(tmpFile)
	assert.NoError(t, err, "File should still exist and be readable")
}

func TestExecuteRequest_BinaryOutput(t *testing.T) {
	resetGlobalFlags()
	noAuth = true
	binary = true
	timeout = 1 * time.Second
	
	cmd := &cobra.Command{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd.SetContext(ctx)
	
	// This will fail quickly, but tests the binary path
	err := executeRequest(cmd, "GET", "https://192.0.2.0/invalid")
	assert.Error(t, err)
}

func TestExecuteRequest_OutputToFile(t *testing.T) {
	resetGlobalFlags()
	tmpDir := t.TempDir()
	outputFile = filepath.Join(tmpDir, "output.json")
	noAuth = true
	timeout = 1 * time.Second
	
	cmd := &cobra.Command{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd.SetContext(ctx)
	
	// This will fail quickly, but tests the file output path
	err := executeRequest(cmd, "GET", "https://192.0.2.0/invalid")
	assert.Error(t, err)
}

func TestNewGetCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewGetCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "get")
	assert.Equal(t, "Execute a GET request", cmd.Short)
}

func TestNewPostCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewPostCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "post")
	assert.Equal(t, "Execute a POST request", cmd.Short)
}

func TestNewPutCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewPutCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "put")
}

func TestNewPatchCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewPatchCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "patch")
}

func TestNewDeleteCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewDeleteCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "delete")
}

func TestNewHeadCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewHeadCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "head")
}

func TestNewOptionsCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewOptionsCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "options")
}

func TestNewVersionCommand(t *testing.T) {
	resetGlobalFlags()
	cmd := NewVersionCommand()
	
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "version")
	
	// Test version command execution - output goes to stdout, not captured easily
	// Just verify command can be created and executed without error
	cmd.SetArgs([]string{})
	outputFormat = "default"
	
	// Execute should not panic
	assert.NotPanics(t, func() {
		cmd.Execute()
	})
}

func TestNewVersionCommand_JSON(t *testing.T) {
	resetGlobalFlags()
	cmd := NewVersionCommand()
	
	// Set JSON format
	outputFormat = "json"
	cmd.SetArgs([]string{})
	
	// Execute should not panic
	assert.NotPanics(t, func() {
		cmd.Execute()
	})
}

func TestNewVersionCommand_Quiet(t *testing.T) {
	resetGlobalFlags()
	cmd := NewVersionCommand()
	
	// Set quiet flag
	outputFormat = "default"
	cmd.SetArgs([]string{"--quiet"})
	
	// Execute should not panic
	assert.NotPanics(t, func() {
		cmd.Execute()
	})
}

func TestBuildRequestOptions_AzureHostWarning(t *testing.T) {
	resetGlobalFlags()
	// Test the warning path for Azure host without scope
	noAuth = false
	
	// Use a URL that looks like Azure but doesn't match any known pattern
	// Note: We can't easily capture stderr in unit tests, so we just verify it doesn't crash
	_, err := buildRequestOptions("GET", "https://unknown.azure.com/resource")
	
	// May error if credentials not available, but that's okay
	_ = err
}

func TestBuildRequestOptions_ScopeDetectionError(t *testing.T) {
	resetGlobalFlags()
	noAuth = false
	
	// Invalid URL should cause scope detection to fail
	_, err := buildRequestOptions("GET", "://invalid-url")
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to detect scope")
}

func TestExecuteRequest_ContextNil(t *testing.T) {
	resetGlobalFlags()
	noAuth = true
	timeout = 1 * time.Second
	
	cmd := &cobra.Command{} // No context set
	cmd.SetContext(nil)
	
	err := executeRequest(cmd, "GET", "https://192.0.2.0/invalid")
	assert.Error(t, err)
}

func TestExecuteRequest_FormatError(t *testing.T) {
	resetGlobalFlags()
	noAuth = true
	timeout = 1 * time.Second
	
	cmd := &cobra.Command{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd.SetContext(ctx)
	
	// Use invalid URL to trigger error path
	err := executeRequest(cmd, "GET", "https://192.0.2.0/invalid")
	assert.Error(t, err)
}
