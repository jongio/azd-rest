package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// sourceDefault and sourceEnvironment name where an effective value came from.
const (
	sourceDefault     = "default"
	sourceEnvironment = "environment"
)

// redactedValue replaces the reported value of a credential-bearing flag whose
// value differs from its built-in default. It avoids angle brackets so it reads
// the same in text output and in JSON, where the encoder escapes HTML.
const redactedValue = "***"

// The credential-bearing persistent flag names that config masks. They mirror
// the flag names registered on the root command.
const (
	headerFlag       = "header"
	dataFlag         = "data"
	formFieldFlag    = "form-field"
	jsonFieldFlag    = "json-field"
	jsonFieldRawFlag = "json-field-raw"
)

// sensitiveFlags names the persistent flags whose values routinely carry
// credentials: an Authorization header, a request body, or individual body
// fields such as an OAuth client_secret. Their AZD_REST_* environment defaults
// are applied before this command runs, so reporting the effective value
// verbatim would print the secret to stdout and into CI logs. The flag, its
// environment variable, and its source are still reported; only the value is
// masked. Path-valued flags such as --header-file and --data-file are not
// masked because a path is not itself a credential and is useful when
// debugging.
var sensitiveFlags = map[string]bool{
	headerFlag:       true,
	dataFlag:         true,
	formFieldFlag:    true,
	jsonFieldFlag:    true,
	jsonFieldRawFlag: true,
}

// reportedValue returns the value to display for a flag. A sensitive flag whose
// value still equals its built-in default is reported as-is, so an unset flag
// stays readable; any other value is masked.
func reportedValue(name, value, defValue string) string {
	if sensitiveFlags[name] && value != defValue {
		return redactedValue
	}
	return value
}

// configEntry describes one persistent flag: its built-in default, the
// AZD_REST_* environment variable that can override it, the effective value,
// and whether that value came from the default or the environment.
type configEntry struct {
	Flag    string `json:"flag"`
	EnvVar  string `json:"envVar"`
	Default string `json:"default"`
	Value   string `json:"value"`
	Source  string `json:"source"`
}

// NewConfigCommand returns the config subcommand. flagNames lists the extension
// persistent flags to report; the command copies and sorts them so the output
// order is stable regardless of registration order.
func NewConfigCommand(flagNames []string) *cobra.Command {
	names := append([]string(nil), flagNames...)
	return &cobra.Command{
		Use:   "config",
		Short: "Show the effective configuration and environment variable mappings",
		Long: `Print every persistent flag with its built-in default, the AZD_REST_*
environment variable that overrides it, the effective value, and whether that
value came from the default or the environment.

Flags that carry credentials (--header, --data, --form-field, --json-field,
--json-field-raw) report their value as *** once it differs from the built-in
default, so running this command in CI does not print a secret.

config makes no network call. Use it to see why a request behaves the way it
does when AZD_REST_* variables are set in your shell or CI.`,
		Example: `  # Show the effective configuration
  azd rest config

  # Machine-readable output
  azd rest config --format json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfig(cmd.Root().PersistentFlags(), names, os.LookupEnv, outputFormat, cmd.OutOrStdout())
		},
	}
}

// runConfig collects the configuration entries and writes them as text or JSON.
func runConfig(flags *pflag.FlagSet, names []string, lookup func(string) (string, bool), format string, out io.Writer) error {
	entries := collectConfigEntries(flags, names, lookup)
	if format == formatJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}
	writeConfigText(out, entries)
	return nil
}

// collectConfigEntries builds a sorted configEntry for each named flag. The
// lookup function supplies environment values so tests can avoid the real
// process environment. A flag name that is not registered is skipped. A value
// is reported as sourced from the environment only when its AZD_REST_* variable
// is set to a non-empty value, matching how environment defaults are applied.
// Credential-bearing flags are masked by reportedValue so a secret supplied
// through an AZD_REST_* variable is never printed.
func collectConfigEntries(flags *pflag.FlagSet, names []string, lookup func(string) (string, bool)) []configEntry {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)

	entries := make([]configEntry, 0, len(sorted))
	for _, name := range sorted {
		flag := flags.Lookup(name)
		if flag == nil {
			continue
		}
		envVar := configEnvVarName(name)
		source := sourceDefault
		if value, ok := lookup(envVar); ok && value != "" {
			source = sourceEnvironment
		}
		entries = append(entries, configEntry{
			Flag:    name,
			EnvVar:  envVar,
			Default: flag.DefValue,
			Value:   reportedValue(name, flag.Value.String(), flag.DefValue),
			Source:  source,
		})
	}
	return entries
}

// configEnvVarName returns the environment variable that supplies a default for
// the given flag. The repeatable --allow-host flag reads a comma separated list
// from AZD_REST_ALLOWED_HOSTS; every other flag follows the AZD_REST_<FLAG>
// convention.
func configEnvVarName(flagName string) string {
	if flagName == allowHostFlag {
		return allowedHostsEnv
	}
	return envVarName(flagName)
}

// writeConfigText writes the entries as an aligned, tab-separated table.
func writeConfigText(out io.Writer, entries []configEntry) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FLAG\tVALUE\tSOURCE\tENV VAR\tDEFAULT")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.Flag, e.Value, e.Source, e.EnvVar, e.Default)
	}
	_ = w.Flush()
}
