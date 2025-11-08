package cmd

import (
	"github.com/spf13/cobra"
)

// NewPutCommand creates the PUT command
func NewPutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put <url>",
		Short: "Execute a PUT request",
		Long:  `Execute a PUT request to the specified URL with automatic Azure authentication.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "PUT", args[0])
		},
	}
	return cmd
}
