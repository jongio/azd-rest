package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/azure/azure-dev/cli/azd/pkg/extensions"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// metadataSchemaVersion is the version of the extension metadata schema this
// command emits. It describes the metadata document, not azd-rest.
const metadataSchemaVersion = "1.0"

// metadataExtensionID must match the id in extension.yaml.
const metadataExtensionID = "jongio.azd.rest"

// environmentVariablesFor derives the published environment variable list from
// the flags that actually honor one.
//
// A hand written list would be wrong the day someone adds a flag. azd-rest
// does not name its environment variables anywhere: applyEnvDefaults walks the
// extension's own persistent flags and derives AZD_REST_<FLAG> from each,
// with --allow-host mapped to AZD_REST_ALLOWED_HOSTS. Every flag therefore
// gains a working environment variable the moment it is registered, and
// nothing announces it.
//
// So the metadata is generated from the same flag names and the same
// configEnvVarName function the config command uses. That makes the published
// document correct by construction rather than by diligence, and it means a
// new flag shows up in azd's view of this extension without anyone
// remembering to add it.
//
// flagNames is the set applyEnvDefaults operates on, which deliberately
// excludes the SDK's own persistent flags. Publishing AZD_REST_ variants of
// those would advertise variables that are never read.
func environmentVariablesFor(flags *pflag.FlagSet, flagNames []string) []extensions.EnvironmentVariable {
	sorted := append([]string(nil), flagNames...)
	sort.Strings(sorted)

	variables := make([]extensions.EnvironmentVariable, 0, len(sorted))
	for _, name := range sorted {
		flag := flags.Lookup(name)
		if flag == nil {
			continue
		}

		description := flag.Usage
		if description == "" {
			description = fmt.Sprintf("Default value for --%s.", name)
		} else {
			description = fmt.Sprintf("%s. Supplies the default for --%s; a value on the command line wins.", description, name)
		}

		variable := extensions.EnvironmentVariable{
			Name:        configEnvVarName(name),
			Description: description,
		}

		// pflag reports "[]" for an unset slice and "0s" for a zero duration.
		// Publishing those as defaults would suggest a value where the flag
		// simply has none.
		switch flag.DefValue {
		case "", "[]", "0s", "0":
		default:
			variable.Default = flag.DefValue
		}

		variables = append(variables, variable)
	}

	return variables
}

// ExtensionConfiguration describes azd-rest's configuration surface.
//
// Global, Project and Service are deliberately nil. azd-rest persists no user
// configuration: config.Config holds flag values for the lifetime of one
// process and is never written anywhere, the extension never calls the
// UserConfig service, and it defines no azure.yaml keys at project or service
// scope. Publishing empty schemas would claim a surface that does not exist.
// TestExtensionConfigurationOwnsNoConfigScopes and TestNoUserConfigWrites pin
// that, so the day this extension starts persisting configuration a schema
// becomes owed and the build says so.
func ExtensionConfiguration(root *cobra.Command, flagNames []string) *extensions.ConfigurationMetadata {
	return &extensions.ConfigurationMetadata{
		EnvironmentVariables: environmentVariablesFor(root.PersistentFlags(), flagNames),
	}
}

// ExtensionMetadata builds the full metadata document for azd-rest.
//
// azdext.NewMetadataCommand cannot be used directly: it marshals the result of
// GenerateExtensionMetadata immediately and exposes no hook for setting the
// Configuration field, so every extension that calls it publishes commands and
// nothing else.
func ExtensionMetadata(root *cobra.Command, flagNames []string) *extensions.ExtensionCommandMetadata {
	metadata := azdext.GenerateExtensionMetadata(metadataSchemaVersion, metadataExtensionID, root)
	metadata.Configuration = ExtensionConfiguration(root, flagNames)
	return metadata
}

// NewMetadataCommand creates the hidden metadata command azd uses to discover
// this extension's commands and configuration.
//
// rootCmdProvider returns a freshly built root command, and flagNames is the
// same slice the config command receives, so both views of the environment
// variable surface come from one place.
func NewMetadataCommand(rootCmdProvider func() *cobra.Command, flagNames []string) *cobra.Command {
	return &cobra.Command{
		Use:    "metadata",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			metadata := ExtensionMetadata(rootCmdProvider(), flagNames)

			jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(jsonBytes))
			return nil
		},
	}
}
