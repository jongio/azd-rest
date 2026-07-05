package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveScope_AzureManagement(t *testing.T) {
	res, err := resolveScope("https://management.azure.com/subscriptions?api-version=2020-01-01", "", false, nil)
	require.NoError(t, err)
	assert.Equal(t, authModeBearer, res.AuthMode)
	assert.Equal(t, scopeResourceManager, res.Scope)
	assert.Equal(t, serviceResourceManager, res.Service)
	assert.Empty(t, res.Note)
}

func TestResolveScope_ScopeOverride(t *testing.T) {
	res, err := resolveScope("https://api.myservice.com/data", "https://myservice.com/.default", false, nil)
	require.NoError(t, err)
	assert.Equal(t, authModeBearer, res.AuthMode)
	assert.Equal(t, "https://myservice.com/.default", res.Scope)
	assert.Empty(t, res.Service)
}

func TestResolveScope_NoAuthFlag(t *testing.T) {
	res, err := resolveScope("https://api.github.com/repos/Azure/azure-dev", "", true, nil)
	require.NoError(t, err)
	assert.Equal(t, authModeNone, res.AuthMode)
	assert.Contains(t, res.Reason, "--no-auth")
	assert.Empty(t, res.Scope)
}

func TestResolveScope_NonHTTPS(t *testing.T) {
	res, err := resolveScope("http://localhost:8080/health", "", false, nil)
	require.NoError(t, err)
	assert.Equal(t, authModeNone, res.AuthMode)
	assert.Contains(t, res.Reason, "non-HTTPS")
}

func TestResolveScope_AuthorizationHeaderSkips(t *testing.T) {
	res, err := resolveScope("https://api.example.com/data", "", false, []string{"Authorization: Bearer abc"})
	require.NoError(t, err)
	assert.Equal(t, authModeNone, res.AuthMode)
	assert.Contains(t, res.Reason, "Authorization header")
}

func TestResolveScope_UnknownAzureHost(t *testing.T) {
	res, err := resolveScope("https://unknown.azure.com/thing", "", false, nil)
	require.NoError(t, err)
	assert.Equal(t, authModeBearer, res.AuthMode)
	assert.Empty(t, res.Scope)
	assert.Contains(t, res.Note, "Azure host detected")
}

func TestResolveScope_UnknownNonAzureHost(t *testing.T) {
	res, err := resolveScope("https://api.example.com/data", "", false, nil)
	require.NoError(t, err)
	assert.Equal(t, authModeBearer, res.AuthMode)
	assert.Empty(t, res.Scope)
	assert.Contains(t, res.Note, "No scope detected")
}

func TestResolveScope_InvalidURL(t *testing.T) {
	_, err := resolveScope("://invalid-url", "", false, nil)
	require.Error(t, err)
}

func TestResolveScope_InvalidHeader(t *testing.T) {
	_, err := resolveScope("https://management.azure.com/subscriptions", "", false, []string{"BadHeader"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header format")
}

func TestResolveScope_KustoSuffix(t *testing.T) {
	res, err := resolveScope("https://help.kusto.windows.net/v1/rest/query", "", false, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://help.kusto.windows.net/.default", res.Scope)
	assert.Equal(t, "Azure Data Explorer", res.Service)
}

func TestWriteScopeResult_Text(t *testing.T) {
	var buf bytes.Buffer
	res := scopeResult{
		URL:      "https://management.azure.com/subscriptions",
		AuthMode: authModeBearer,
		Scope:    scopeResourceManager,
		Service:  serviceResourceManager,
	}
	require.NoError(t, writeScopeResult(&buf, res, "auto"))
	out := buf.String()
	assert.Contains(t, out, "URL:      https://management.azure.com/subscriptions")
	assert.Contains(t, out, "Auth:     bearer")
	assert.Contains(t, out, "Scope:    https://management.azure.com/.default")
	assert.Contains(t, out, "Service:  Azure Resource Manager")
}

func TestWriteScopeResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	res := scopeResult{
		URL:      "https://management.azure.com/subscriptions",
		AuthMode: authModeBearer,
		Scope:    scopeResourceManager,
		Service:  serviceResourceManager,
	}
	require.NoError(t, writeScopeResult(&buf, res, "json"))

	var decoded scopeResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, res, decoded)
}

func TestNewScopeCommand_RunJSON(t *testing.T) {
	resetGlobalFlags()
	outputFormat = "json"
	defer resetGlobalFlags()

	cmd := NewScopeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"https://graph.microsoft.com/v1.0/me"})

	require.NoError(t, cmd.Execute())
	assert.True(t, strings.Contains(buf.String(), "graph.microsoft.com/.default"))

	var decoded scopeResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, "Microsoft Graph", decoded.Service)
}
