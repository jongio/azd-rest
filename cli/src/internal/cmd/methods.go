package cmd

import (
	"github.com/jongio/azd-rest/cli/src/internal/client"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <url>",
	Short: "Execute GET request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRequest("GET", args[0])
	},
}

var postCmd = &cobra.Command{
	Use:   "post <url>",
	Short: "Execute POST request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRequest("POST", args[0])
	},
}

var putCmd = &cobra.Command{
	Use:   "put <url>",
	Short: "Execute PUT request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRequest("PUT", args[0])
	},
}

var patchCmd = &cobra.Command{
	Use:   "patch <url>",
	Short: "Execute PATCH request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRequest("PATCH", args[0])
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <url>",
	Short: "Execute DELETE request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRequest("DELETE", args[0])
	},
}

func init() {
	// Add data flags to commands that support request bodies
	for _, cmd := range []*cobra.Command{postCmd, putCmd, patchCmd} {
		cmd.Flags().StringVarP(&data, "data", "d", "", "Request body data")
		cmd.Flags().StringVar(&dataFile, "data-file", "", "Read request body from file")
		cmd.Flags().StringVarP(&contentType, "content-type", "t", "application/json", "Content-Type header")
	}
}

func executeRequest(method, url string) error {
	config := client.RequestConfig{
		Method:      method,
		URL:         url,
		Headers:     headers,
		Data:        data,
		DataFile:    dataFile,
		ContentType: contentType,
		Output:      output,
		Verbose:     verbose,
		Insecure:    insecure,
		UseAzdAuth:  useAzdAuth,
	}

	return client.ExecuteRequest(config)
}
