package cmd

import (
	"github.com/spf13/cobra"
)

// NewPostCommand creates the POST command
func NewPostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post <url>",
		Short: "Execute a POST request",
		Long:  `Execute a POST request to the specified URL with automatic Azure authentication.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "POST", args[0])
		},
	}
	return cmd
}
