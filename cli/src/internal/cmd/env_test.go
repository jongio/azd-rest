package cmd

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvVarName(t *testing.T) {
	cases := map[string]string{
		"retry":             "AZD_REST_RETRY",
		"api-version":       "AZD_REST_API_VERSION",
		"max-response-size": "AZD_REST_MAX_RESPONSE_SIZE",
		"scope":             "AZD_REST_SCOPE",
	}
	for flagName, want := range cases {
		assert.Equal(t, want, envVarName(flagName))
	}
}

// newTestFlags builds an isolated flag set mirroring a couple of the real
// persistent flags so applyEnvDefaults can be unit tested without the SDK.
func newTestFlags() (*pflag.FlagSet, *int, *string) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	retryVal := fs.Int("retry", 3, "")
	scopeVal := fs.String("scope", "", "")
	return fs, retryVal, scopeVal
}

func TestApplyEnvDefaults_AppliesWhenNotSet(t *testing.T) {
	fs, retryVal, _ := newTestFlags()
	env := map[string]string{"AZD_REST_RETRY": "7"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	err := applyEnvDefaults(fs, []string{"retry", "scope"}, lookup)

	require.NoError(t, err)
	assert.Equal(t, 7, *retryVal, "env value should override the default")
}

func TestApplyEnvDefaults_CommandLineWins(t *testing.T) {
	fs, retryVal, _ := newTestFlags()
	require.NoError(t, fs.Parse([]string{"--retry", "5"}))
	env := map[string]string{"AZD_REST_RETRY": "7"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	err := applyEnvDefaults(fs, []string{"retry"}, lookup)

	require.NoError(t, err)
	assert.Equal(t, 5, *retryVal, "command-line value should win over env")
}

func TestApplyEnvDefaults_DefaultWhenNoEnv(t *testing.T) {
	fs, retryVal, scopeVal := newTestFlags()
	lookup := func(string) (string, bool) { return "", false }

	err := applyEnvDefaults(fs, []string{"retry", "scope"}, lookup)

	require.NoError(t, err)
	assert.Equal(t, 3, *retryVal, "default should be kept when no env is set")
	assert.Equal(t, "", *scopeVal)
}

func TestApplyEnvDefaults_EmptyEnvIgnored(t *testing.T) {
	fs, _, scopeVal := newTestFlags()
	env := map[string]string{"AZD_REST_SCOPE": ""}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	err := applyEnvDefaults(fs, []string{"scope"}, lookup)

	require.NoError(t, err)
	assert.Equal(t, "", *scopeVal)
}

func TestApplyEnvDefaults_InvalidValueIsConfigError(t *testing.T) {
	fs, _, _ := newTestFlags()
	env := map[string]string{"AZD_REST_RETRY": "not-a-number"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	err := applyEnvDefaults(fs, []string{"retry"}, lookup)

	require.Error(t, err)
	var coder ExitCoder
	require.True(t, errors.As(err, &coder), "invalid env value should be an ExitCoder")
	assert.Equal(t, 2, coder.ExitCode())
	assert.Contains(t, err.Error(), "AZD_REST_RETRY")
}

// TestApplyEnvDefaults_UpdatesBoundGlobal verifies the env value flows through
// the real command's flag binding into the config snapshot the service reads.
func TestApplyEnvDefaults_UpdatesBoundGlobal(t *testing.T) {
	resetGlobalFlags()
	root := NewRootCmd()
	env := map[string]string{"AZD_REST_RETRY": "9"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	err := applyEnvDefaults(root.PersistentFlags(), []string{"retry"}, lookup)

	require.NoError(t, err)
	assert.Equal(t, 9, retry, "env default should update the bound global")
	assert.Equal(t, 9, snapshotConfig().Retry, "env default should reach the config snapshot")
}
