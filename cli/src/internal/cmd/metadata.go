package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

// NewMetadataCommand creates a metadata command that generates extension metadata
// using the official azdext SDK metadata generator.
func NewMetadataCommand(rootCmdProvider func() *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "metadata",
		Short:  "Output extension metadata for IntelliSense",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := rootCmdProvider()
			metadata := azdext.GenerateExtensionMetadata("1.0", "jongio.azd.rest", root)
			data, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return err
		},
	}
}
