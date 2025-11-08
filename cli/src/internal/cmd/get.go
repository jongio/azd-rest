package cmd

import (
	"github.com/spf13/cobra"
)

// NewGetCommand creates the GET command
func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <url>",
		Short: "Execute a GET request",
		Long:  `Execute a GET request to the specified URL with automatic Azure authentication.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "GET", args[0])
		},
	}
	return cmd
}
