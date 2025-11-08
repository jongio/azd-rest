package cmd

import (
	"github.com/spf13/cobra"
)

// NewPatchCommand creates the PATCH command
func NewPatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch <url>",
		Short: "Execute a PATCH request",
		Long:  `Execute a PATCH request to the specified URL with automatic Azure authentication.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "PATCH", args[0])
		},
	}
	return cmd
}
