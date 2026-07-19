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
			Value:   flag.Value.String(),
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
