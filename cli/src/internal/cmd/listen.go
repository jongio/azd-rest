package cmd

import (
	"context"
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

			host := azdext.NewExtensionHost(azdClient).
				WithProjectEventHandler("postprovision", func(ctx context.Context, args *azdext.ProjectEventArgs) error {
					fmt.Printf("Post-provision completed for project: %s\n", args.Project.Name)
					return nil
				}).
				WithProjectEventHandler("postdeploy", func(ctx context.Context, args *azdext.ProjectEventArgs) error {
					fmt.Printf("Deployment completed for project: %s\n", args.Project.Name)
					return nil
				}).
				WithServiceEventHandler("postdeploy", func(ctx context.Context, args *azdext.ServiceEventArgs) error {
					fmt.Printf("Service %s deployed successfully\n", args.Service.Name)
					for _, artifact := range args.ServiceContext.Deploy {
						if artifact.Kind == azdext.ArtifactKind_ARTIFACT_KIND_ENDPOINT {
							fmt.Printf("  Endpoint: %s\n", artifact.Location)
						}
					}
					return nil
				}, nil)

			if err := host.Run(ctx); err != nil {
				return fmt.Errorf("failed to run extension: %w", err)
			}

			return nil
		},
	}
}
