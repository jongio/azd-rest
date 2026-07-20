// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"fmt"

	"github.com/jongio/azd-rest/src/internal/service"
	"github.com/spf13/cobra"
)

// NewCacheCommand returns the cache subcommand group, which inspects and purges
// the on-disk response cache used by --cache-ttl.
func NewCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the on-disk response cache",
		Long: `Manage the on-disk cache used by --cache-ttl.

Responses are cached only when you pass --cache-ttl on a GET request. Cached
bodies can contain sensitive data, so entries are stored with owner-only
permissions and caching stays off unless you opt in per call.`,
	}
	cmd.AddCommand(newCacheClearCommand(), newCachePathCommand())
	return cmd
}

// newCacheClearCommand returns "cache clear", which removes every cached entry.
func newCacheClearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all cached responses",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := service.ClearCache()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cleared response cache at %s\n", dir)
			return nil
		},
	}
}

// newCachePathCommand returns "cache path", which prints the cache directory.
func newCachePathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the response cache directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := service.CacheDir()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), dir)
			return nil
		},
	}
}
