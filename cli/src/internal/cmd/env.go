// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"fmt"
	"strings"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/pflag"
)

// envPrefix is prepended to the upper-cased flag name to form the environment
// variable that supplies a default for a persistent flag (#172).
const envPrefix = "AZD_REST_"

// envVarName maps a flag name to its environment variable name by upper-casing
// it, replacing dashes with underscores, and adding the AZD_REST_ prefix. For
// example "api-version" becomes "AZD_REST_API_VERSION".
func envVarName(flagName string) string {
	return envPrefix + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}

// applyEnvDefaults applies AZD_REST_<FLAG> environment variables to the named
// persistent flags that were not set on the command line. Precedence is command
// line over environment over built-in default: a flag already set on the command
// line is left untouched, and only a non-empty environment value is applied.
// The lookup function is injectable so tests can supply values without touching
// the real process environment. An invalid value is returned as a structured
// [azdext.LocalError] so the azd host can classify it as a validation failure
// and show the suggestion, and so no request is made.
func applyEnvDefaults(flags *pflag.FlagSet, names []string, lookup func(string) (string, bool)) error {
	for _, name := range names {
		flag := flags.Lookup(name)
		if flag == nil || flag.Changed {
			continue
		}
		value, ok := lookup(envVarName(name))
		if !ok || value == "" {
			continue
		}
		if err := flag.Value.Set(value); err != nil {
			envVar := envVarName(name)
			return &azdext.LocalError{
				Message: fmt.Sprintf(
					"invalid value %q for %s (default for --%s): %v", value, envVar, name, err,
				),
				Code:     ErrCodeInvalidEnvDefault,
				Category: azdext.LocalErrorCategoryValidation,
				Suggestion: fmt.Sprintf(
					"Unset %s, or set it to a value that --%s accepts. A flag passed on the command line always wins over the environment variable.",
					envVar, name,
				),
			}
		}
		flag.Changed = true
	}
	return nil
}

// allowHostFlag is the name of the repeatable host-allowlist flag (#219).
const allowHostFlag = "allow-host"

// allowedHostsEnv is the environment variable that supplies a comma separated
// default for the repeatable --allow-host flag (#219).
const allowedHostsEnv = "AZD_REST_ALLOWED_HOSTS"

// applyAllowedHostsEnv sets --allow-host from AZD_REST_ALLOWED_HOSTS when the
// flag was not provided on the command line. The value is a comma separated
// list of host patterns; blank entries are ignored. The lookup function is
// injectable so tests can supply values without touching the process
// environment.
func applyAllowedHostsEnv(flags *pflag.FlagSet, lookup func(string) (string, bool)) error {
	flag := flags.Lookup(allowHostFlag)
	if flag == nil || flag.Changed {
		return nil
	}
	value, ok := lookup(allowedHostsEnv)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}
	applied := false
	for _, part := range strings.Split(value, ",") {
		host := strings.TrimSpace(part)
		if host == "" {
			continue
		}
		if err := flag.Value.Set(host); err != nil {
			return &azdext.LocalError{
				Message: fmt.Sprintf(
					"invalid value %q for %s (default for --%s): %v", host, allowedHostsEnv, allowHostFlag, err,
				),
				Code:     ErrCodeInvalidEnvDefault,
				Category: azdext.LocalErrorCategoryValidation,
				Suggestion: fmt.Sprintf(
					"%s takes a comma separated list of host patterns. Fix or unset the offending entry %q.",
					allowedHostsEnv, host,
				),
			}
		}
		applied = true
	}
	if applied {
		flag.Changed = true
	}
	return nil
}
