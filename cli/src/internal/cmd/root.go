package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Flags for all commands
	headers     []string
	output      string
	verbose     bool
	insecure    bool
	useAzdAuth  bool
	data        string
	dataFile    string
	contentType string

	// appVersion is set by main package
	appVersion = "dev"
)

// SetVersion sets the application version
func SetVersion(v string) {
	appVersion = v
}

var rootCmd = &cobra.Command{
	Use:   "rest",
	Short: "Execute REST APIs with azd context and authentication",
	Long: `azd-rest allows you to execute REST API calls with automatic integration 
of Azure Developer CLI context and authentication tokens.

Examples:
  # Simple GET request
  azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01

  # POST with JSON data
  azd rest post https://api.example.com/resource --data '{"name":"test"}'

  # Custom headers
  azd rest get https://api.example.com/resource -H "Custom-Header: value"

  # Save response to file
  azd rest get https://api.example.com/resource --output response.json`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Persistent flags for all commands
	rootCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom headers (can be specified multiple times)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Output file path (default: stdout)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS certificate verification")
	rootCmd.PersistentFlags().BoolVar(&useAzdAuth, "use-azd-auth", true, "Use azd authentication token")

	// Add subcommands
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(postCmd)
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(patchCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("azd-rest version %s\n", appVersion)
	},
}
