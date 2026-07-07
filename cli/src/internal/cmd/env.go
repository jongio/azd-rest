// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// envPrefix is prepended to the upper-cased flag name to form the environment
// variable that supplies a default for a persistent flag (#172).
const envPrefix = "AZD_REST_"

// ExitCoder is an error that carries a specific process exit code. main uses it
// to translate configuration errors into the documented exit code 2 instead of
// the generic exit code 1.
type ExitCoder interface {
	error
	ExitCode() int
}

// configError signals invalid configuration (for example a malformed value in
// an AZD_REST_* environment variable). It reports exit code 2, matching the
// "invalid arguments or configuration" exit code in the CLI reference.
type configError struct{ err error }

func (e *configError) Error() string { return e.err.Error() }

func (e *configError) Unwrap() error { return e.err }

// ExitCode returns 2 for invalid configuration.
func (e *configError) ExitCode() int { return 2 }

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
// the real process environment. An invalid value is returned as a configError so
// the process exits with code 2 and no request is made.
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
			return &configError{fmt.Errorf(
				"invalid value %q for %s (default for --%s): %w", value, envVarName(name), name, err,
			)}
		}
		flag.Changed = true
	}
	return nil
}

// allowedHostsEnv is the environment variable that supplies a comma separated
// default for the repeatable --allow-host flag (#219).
const allowedHostsEnv = "AZD_REST_ALLOWED_HOSTS"

// applyAllowedHostsEnv sets --allow-host from AZD_REST_ALLOWED_HOSTS when the
// flag was not provided on the command line. The value is a comma separated
// list of host patterns; blank entries are ignored. The lookup function is
// injectable so tests can supply values without touching the process
// environment.
func applyAllowedHostsEnv(flags *pflag.FlagSet, lookup func(string) (string, bool)) error {
	flag := flags.Lookup("allow-host")
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
			return &configError{fmt.Errorf(
				"invalid value %q for %s (default for --allow-host): %w", host, allowedHostsEnv, err,
			)}
		}
		applied = true
	}
	if applied {
		flag.Changed = true
	}
	return nil
}
