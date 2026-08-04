package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metadataFlagNames rebuilds the flag set the metadata command is given in
// root.go: every persistent flag the extension registers itself, plus
// --allow-host, and none of the SDK's own.
//
// This mirrors the derivation in NewRootCmd rather than importing it, because
// the point of most tests here is to catch the two drifting apart.
func metadataFlagNames(t *testing.T) (*cobra.Command, []string) {
	t.Helper()

	root := NewRootCmd()

	// The SDK's persistent flags are the ones a bare extension root carries.
	// Anything beyond that was added by this extension.
	sdk := map[string]bool{}
	for _, name := range []string{"help", "output", "cwd", "debug", "no-prompt"} {
		sdk[name] = true
	}

	var names []string
	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if !sdk[flag.Name] {
			names = append(names, flag.Name)
		}
	})

	require.NotEmpty(t, names, "no extension flags found, so these guards would pass vacuously")
	return root, names
}

// TestEveryExtensionFlagPublishesItsEnvironmentVariable is the guard that made
// generating the list worth it. applyEnvDefaults gives every extension flag a
// working AZD_REST_<FLAG> variable the moment the flag is registered, and
// nothing else announces that. If the published list were hand written it
// would silently fall behind on the next flag someone adds.
func TestEveryExtensionFlagPublishesItsEnvironmentVariable(t *testing.T) {
	root, names := metadataFlagNames(t)

	published := map[string]bool{}
	for _, variable := range ExtensionConfiguration(root, names).EnvironmentVariables {
		published[variable.Name] = true
	}

	for _, name := range names {
		expected := configEnvVarName(name)
		assert.True(
			t,
			published[expected],
			"flag --%s honors %s but the metadata never mentions it", name, expected,
		)
	}
}

// TestPublishedEnvironmentVariablesAreReadable is the other direction. A
// published variable that applyEnvDefaults never consults is worse than an
// omission, because a user who sets it gets silence rather than an error.
func TestPublishedEnvironmentVariablesAreReadable(t *testing.T) {
	root, names := metadataFlagNames(t)

	readable := map[string]string{}
	for _, name := range names {
		readable[configEnvVarName(name)] = name
	}

	for _, variable := range ExtensionConfiguration(root, names).EnvironmentVariables {
		_, ok := readable[variable.Name]
		assert.True(t, ok, "metadata publishes %s but no flag reads it", variable.Name)
	}
}

// TestAllowHostKeepsItsOwnEnvironmentVariableName pins the one flag whose
// variable does not follow the AZD_REST_<FLAG> rule. Publishing
// AZD_REST_ALLOW_HOST would send a user to a name nothing reads, and the
// failure would be silent: hosts simply would not be restricted.
func TestAllowHostKeepsItsOwnEnvironmentVariableName(t *testing.T) {
	root, names := metadataFlagNames(t)

	var found bool
	for _, variable := range ExtensionConfiguration(root, names).EnvironmentVariables {
		if variable.Name == allowedHostsEnv {
			found = true
		}
		assert.NotEqual(t, "AZD_REST_ALLOW_HOST", variable.Name,
			"published the mechanical name for --allow-host; applyAllowedHostsEnv reads %s", allowedHostsEnv)
	}

	assert.True(t, found, "%s is not published even though --allow-host reads it", allowedHostsEnv)
}

// TestSdkFlagsAreNotPublished keeps the document honest about scope.
// applyEnvDefaults deliberately skips the SDK's persistent flags, so an
// AZD_REST_OUTPUT entry would advertise a variable that does nothing.
func TestSdkFlagsAreNotPublished(t *testing.T) {
	root, names := metadataFlagNames(t)

	for _, variable := range ExtensionConfiguration(root, names).EnvironmentVariables {
		for _, sdkFlag := range []string{"help", "output", "cwd", "no-prompt"} {
			assert.NotEqual(t, configEnvVarName(sdkFlag), variable.Name,
				"published %s, but environment defaults are never applied to the SDK flag --%s", variable.Name, sdkFlag)
		}
	}
}

// TestEveryPublishedVariableHasADescription keeps the output useful, and
// TestPublishedDefaultsMatchTheFlags keeps it true. A default that disagrees
// with the flag is a documentation bug that costs someone an afternoon.
func TestEveryPublishedVariableHasADescription(t *testing.T) {
	root, names := metadataFlagNames(t)

	for _, variable := range ExtensionConfiguration(root, names).EnvironmentVariables {
		assert.NotEmpty(t, strings.TrimSpace(variable.Description), "%s has no description", variable.Name)
	}
}

func TestPublishedDefaultsMatchTheFlags(t *testing.T) {
	root, names := metadataFlagNames(t)

	byEnvName := map[string]string{}
	for _, name := range names {
		byEnvName[configEnvVarName(name)] = name
	}

	for _, variable := range ExtensionConfiguration(root, names).EnvironmentVariables {
		flagName, ok := byEnvName[variable.Name]
		require.True(t, ok)

		flag := root.PersistentFlags().Lookup(flagName)
		require.NotNil(t, flag)

		if variable.Default == "" {
			continue
		}
		assert.Equal(t, flag.DefValue, variable.Default,
			"%s advertises default %q but --%s defaults to %q", variable.Name, variable.Default, flagName, flag.DefValue)
	}
}

// TestEmptyDefaultsAreOmitted stops the document from suggesting a value where
// the flag has none. pflag reports "[]" for an unset slice and "0s" for a zero
// duration, neither of which a user should be told to expect.
func TestEmptyDefaultsAreOmitted(t *testing.T) {
	root, names := metadataFlagNames(t)

	for _, variable := range ExtensionConfiguration(root, names).EnvironmentVariables {
		switch variable.Default {
		case "[]", "0s":
			t.Errorf("%s advertises the placeholder default %q", variable.Name, variable.Default)
		}
	}
}

// TestExtensionConfigurationOwnsNoConfigScopes pins the decision to publish no
// schemas. azd-rest persists nothing, so empty schemas would claim a surface
// that does not exist.
func TestExtensionConfigurationOwnsNoConfigScopes(t *testing.T) {
	root, names := metadataFlagNames(t)
	configuration := ExtensionConfiguration(root, names)

	assert.Nil(t, configuration.Global, "Global schema is set, but azd-rest persists no user configuration")
	assert.Nil(t, configuration.Project, "Project schema is set, but azd-rest defines no azure.yaml project keys")
	assert.Nil(t, configuration.Service, "Service schema is set, but azd-rest defines no azure.yaml service keys")
	assert.NotEmpty(t, configuration.EnvironmentVariables, "the configuration document says nothing at all")
}

// TestNoUserConfigWrites backs the claim above against the source tree rather
// than against memory.
func TestNoUserConfigWrites(t *testing.T) {
	err := filepath.Walk(filepath.Join("..", ".."), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(content), "UserConfig().Set") {
			t.Errorf("%s writes azd user configuration, so ExtensionConfiguration must publish a Global schema", path)
		}
		return nil
	})
	require.NoError(t, err)
}

// TestMetadataCommandEmitsConfiguration is the regression this command exists
// to prevent. azdext.NewMetadataCommand drops Configuration on the floor, so
// the emitted document is the only place that can be proven.
func TestMetadataCommandEmitsConfiguration(t *testing.T) {
	_, names := metadataFlagNames(t)

	command := NewMetadataCommand(NewRootCmd, names)

	var out bytes.Buffer
	command.SetOut(&out)
	command.SetArgs([]string{})
	require.NoError(t, command.Execute())

	var document struct {
		SchemaVersion string `json:"schemaVersion"`
		ID            string `json:"id"`
		Configuration *struct {
			Global               json.RawMessage `json:"global"`
			EnvironmentVariables []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"environmentVariables"`
		} `json:"configuration"`
	}

	require.NoError(t, json.Unmarshal(out.Bytes(), &document), "metadata output is not valid json:\n%s", out.String())

	assert.Equal(t, metadataSchemaVersion, document.SchemaVersion)
	assert.Equal(t, metadataExtensionID, document.ID)
	require.NotNil(t, document.Configuration,
		"configuration is absent, which is the exact regression this command exists to prevent")
	assert.Nil(t, document.Configuration.Global, "a schema was emitted despite no configuration being owned")
	assert.Len(t, document.Configuration.EnvironmentVariables, len(names))

	var sawAllowedHosts bool
	for _, variable := range document.Configuration.EnvironmentVariables {
		if variable.Name == allowedHostsEnv {
			sawAllowedHosts = true
		}
	}
	assert.True(t, sawAllowedHosts, "%s missing from the emitted document", allowedHostsEnv)
}

// TestMetadataCommandIsHidden keeps it out of help output. azd calls it; users
// should not see it.
func TestMetadataCommandIsHidden(t *testing.T) {
	command := NewMetadataCommand(NewRootCmd, nil)

	assert.True(t, command.Hidden, "metadata command is visible in help output")
	assert.Equal(t, "metadata", command.Use)
}

// TestExtensionIDMatchesManifest catches the drift that would stop azd
// associating this metadata with the installed extension.
func TestExtensionIDMatchesManifest(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "extension.yaml"))
	require.NoError(t, err, "extension.yaml is not where this test expects it")

	assert.Contains(t, string(content), metadataExtensionID,
		"extension.yaml does not contain the id used by the metadata command")
}

// failingWriter always fails, so the metadata command's write error branch gets
// exercised. Left uncovered it is unreachable in tests, which is how an
// unchecked write survived there in the first place.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestMetadataCommandReportsWriteFailure(t *testing.T) {
	command := NewMetadataCommand(NewRootCmd, nil)
	command.SetOut(failingWriter{})
	command.SetErr(failingWriter{})

	err := command.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to write metadata")
}
