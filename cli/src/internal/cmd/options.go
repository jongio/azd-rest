package cmd

import (
	"github.com/spf13/cobra"
)

// NewOptionsCommand creates the OPTIONS command
func NewOptionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "options <url>",
		Short: "Execute an OPTIONS request",
		Long:  `Execute an OPTIONS request to the specified URL with automatic Azure authentication.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "OPTIONS", args[0])
		},
	}
	return cmd
}
