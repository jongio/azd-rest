package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findVersionCommand returns the version subcommand from a fresh root.
func findVersionCommand(t *testing.T) (*cobra.Command, *cobra.Command) {
	t.Helper()
	root := NewRootCmd()
	for _, c := range root.Commands() {
		if c.Name() == "version" {
			return root, c
		}
	}
	t.Fatal("version subcommand should exist")
	return nil, nil
}

// TestVersionCommand_QuietHasNoShorthand guards the shorthand collision that
// crashed the extension: the root binds -q to --query, so the version command
// must register --quiet without a shorthand.
func TestVersionCommand_QuietHasNoShorthand(t *testing.T) {
	resetGlobalFlags()
	root, versionCmd := findVersionCommand(t)

	quiet := versionCmd.Flags().Lookup("quiet")
	require.NotNil(t, quiet, "version should expose --quiet")
	assert.Empty(t, quiet.Shorthand, "--quiet must not claim -q; the root uses it for --query")

	query := root.PersistentFlags().Lookup("query")
	require.NotNil(t, query, "root should still expose --query")
	assert.Equal(t, "q", query.Shorthand, "--query keeps -q")
}

// TestVersionCommand_Executes covers the parse path end to end, which is where
// a shorthand collision would panic.
func TestVersionCommand_Executes(t *testing.T) {
	resetGlobalFlags()
	root := NewRootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version"})

	assert.NotPanics(t, func() {
		assert.NoError(t, root.Execute())
	})
}

// TestVersionCommand_JSONIncludesBuildMetadata verifies the fields regained by
// moving off azdext.NewVersionCommand, which reports only the version string.
// This extension drives output from --format, not the SDK's --output.
func TestVersionCommand_JSONIncludesBuildMetadata(t *testing.T) {
	resetGlobalFlags()

	stdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	root := NewRootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version", "--format", "json"})
	execErr := root.Execute()

	require.NoError(t, w.Close())
	os.Stdout = stdout

	var buf bytes.Buffer
	_, copyErr := buf.ReadFrom(r)
	require.NoError(t, copyErr)
	require.NoError(t, execErr)

	var info map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &info), "version --output json should emit JSON, got %q", buf.String())

	for _, key := range []string{"version", "buildDate", "gitCommit", "extensionId", "name"} {
		assert.Contains(t, info, key, "version JSON should include %q", key)
	}
	assert.Equal(t, "jongio.azd.rest", info["extensionId"])
}

// TestVersionCommand_DoesNotConstrainOutputFlag guards the WithOutputFlag("")
// opt-out. azd-core declares allowed values for --output by default, but this
// extension reads --format instead, so constraining --output would reject
// values on a flag the command ignores.
func TestVersionCommand_DoesNotConstrainOutputFlag(t *testing.T) {
	resetGlobalFlags()
	_, versionCmd := findVersionCommand(t)

	for key := range versionCmd.Annotations {
		assert.False(t, strings.HasPrefix(key, "azdext.allowed-values/"),
			"version should not declare allowed values, found %q", key)
	}
}

// TestVersionCommand_FormatFlagStillAcceptsOtherValues confirms the version
// command did not narrow the root's --format vocabulary.
func TestVersionCommand_FormatFlagStillAcceptsOtherValues(t *testing.T) {
	resetGlobalFlags()
	root := NewRootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version", "--format", "table"})

	require.NoError(t, root.Execute(), "--format table should still parse")
}
