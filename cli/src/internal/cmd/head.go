package cmd

import (
	"github.com/spf13/cobra"
)

// NewHeadCommand creates the HEAD command
func NewHeadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "head <url>",
		Short: "Execute a HEAD request",
		Long:  `Execute a HEAD request to the specified URL with automatic Azure authentication.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "HEAD", args[0])
		},
	}
	return cmd
}
