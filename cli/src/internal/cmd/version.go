package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jongio/azd-rest/src/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	var quiet bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display the extension version",
		Long:  `Display the version information for the azd rest extension.`,
		Run: func(cmd *cobra.Command, args []string) {
			if outputFormat == "json" {
				// JSON output mode
				output := map[string]string{
					"version": version.Version,
				}
				if version.BuildDate != "unknown" {
					output["buildDate"] = version.BuildDate
				}
				if version.GitCommit != "unknown" {
					output["gitCommit"] = version.GitCommit
				}
				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(string(data))
			} else {
				// Human-readable output
				if quiet {
					fmt.Println(version.Version)
				} else {
					fmt.Printf("azd rest\n")
					fmt.Printf("Version: %s\n", version.Version)
					if version.BuildDate != "unknown" {
						fmt.Printf("Build Date: %s\n", version.BuildDate)
					}
					if version.GitCommit != "unknown" {
						fmt.Printf("Git Commit: %s\n", version.GitCommit)
					}
				}
			}
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Display only the version number")
	return cmd
}
