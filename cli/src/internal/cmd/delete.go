package cmd

import (
	"github.com/spf13/cobra"
)

// NewDeleteCommand creates the DELETE command
func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <url>",
		Short: "Execute a DELETE request",
		Long:  `Execute a DELETE request to the specified URL with automatic Azure authentication.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "DELETE", args[0])
		},
	}
	return cmd
}
