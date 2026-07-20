package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteScopeMappingsText(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, writeScopeMappings(&buf, "auto"))
	out := buf.String()
	assert.Contains(t, out, "SERVICE")
	assert.Contains(t, out, "Azure Resource Manager")
	assert.Contains(t, out, hostResourceManager)
	assert.Contains(t, out, serviceDataExplorer)
}

func TestWriteScopeMappingsJSON(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, writeScopeMappings(&buf, "json"))

	var decoded []scopeMapping
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	require.NotEmpty(t, decoded)

	services := map[string]bool{}
	for _, mapping := range decoded {
		services[mapping.Service] = true
	}
	assert.True(t, services[serviceResourceManager])
	assert.True(t, services[serviceMicrosoftGraph])
	assert.True(t, services[serviceKeyVault])
	assert.True(t, services[serviceStorage])
	assert.True(t, services[serviceCosmosDB])
	assert.True(t, services[serviceDataExplorer])
}

func TestNewScopesCommand_RunJSON(t *testing.T) {
	resetGlobalFlags()
	outputFormat = "json"
	defer resetGlobalFlags()

	cmd := NewScopesCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	require.NoError(t, cmd.Execute())
	assert.True(t, strings.Contains(buf.String(), serviceMicrosoftGraph))
}
