package cmd

import (
	"fmt"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

func NewListenCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "listen",
		Short:  "Start extension listener (internal use only)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := azdext.WithAccessToken(cmd.Context())

			azdClient, err := azdext.NewAzdClient()
			if err != nil {
				return fmt.Errorf("failed to create azd client: %w", err)
			}
			defer azdClient.Close()

			host := azdext.NewExtensionHost(azdClient)

			if err := host.Run(ctx); err != nil {
				return fmt.Errorf("failed to run extension: %w", err)
			}

			return nil
		},
	}
}
