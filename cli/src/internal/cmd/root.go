// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/skills"
	"github.com/jongio/azd-rest/src/internal/version"
	"github.com/spf13/cobra"
)

// Global flags
var (
	scope           string
	noAuth          bool
	headers         []string
	data            string
	dataFile        string
	outputFile      string
	outputFormat    string
	verbose         bool
	paginate        bool
	retry           int
	binary          bool
	insecure        bool
	timeout         time.Duration
	followRedirects bool
	maxRedirects    int
)

// NewRootCmd creates the root command for azd rest
func NewRootCmd() *cobra.Command {
	rootCmd, _ := azdext.NewExtensionRootCommand(azdext.ExtensionCommandOptions{
		Name:    "rest",
		Version: version.Version,
		Short:   "Execute REST API calls with Azure authentication",
		Long: `azd rest is an Azure Developer CLI extension that enables you to execute REST API calls
with automatic Azure authentication. The extension intelligently detects Azure service
endpoints and applies the correct OAuth scopes.

Examples:
  # Simple GET (auto-detects Management API scope)
  azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01

  # POST with JSON body
  azd rest post https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 --data '{"location":"eastus"}'

  # Custom scope for non-Azure endpoint
  azd rest get https://api.myservice.com/data --scope https://myservice.com/.default

  # Non-Azure endpoint without auth
  azd rest get https://api.github.com/repos/Azure/azure-dev --no-auth`,
	})

	// Chain extension-specific PersistentPreRunE after SDK's built-in one
	sdkPreRunE := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if sdkPreRunE != nil {
			if err := sdkPreRunE(cmd, args); err != nil {
				return err
			}
		}
		// Install Copilot skill
		if err := skills.InstallSkill(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to install copilot skill: %v\n", err)
		}
		return nil
	}

	// Extension-specific flags
	rootCmd.PersistentFlags().StringVarP(&scope, "scope", "s", "", "OAuth scope for authentication (auto-detected if not provided)")
	rootCmd.PersistentFlags().BoolVar(&noAuth, "no-auth", false, "Skip authentication (no bearer token)")
	rootCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom headers (repeatable, format: Key:Value)")
	rootCmd.PersistentFlags().StringVarP(&data, "data", "d", "", "Request body (JSON string)")
	rootCmd.PersistentFlags().StringVar(&dataFile, "data-file", "", "Read request body from file (also accepts @{file} shorthand)")
	rootCmd.PersistentFlags().StringVar(&outputFile, "output-file", "", "Write response to file (raw for binary content)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "auto", "Output format: auto, json, raw")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (show headers, timing)")
	rootCmd.PersistentFlags().BoolVar(&paginate, "paginate", false, "Follow continuation tokens/next links when supported")
	rootCmd.PersistentFlags().IntVar(&retry, "retry", 3, "Retry attempts with exponential backoff for transient errors")
	rootCmd.PersistentFlags().BoolVar(&binary, "binary", false, "Stream request/response as binary without transformation")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS certificate verification")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 30*time.Second, "Request timeout")
	rootCmd.PersistentFlags().BoolVar(&followRedirects, "follow-redirects", true, "Follow HTTP redirects")
	rootCmd.PersistentFlags().IntVar(&maxRedirects, "max-redirects", 10, "Maximum redirect hops")

	// Add subcommands
	rootCmd.AddCommand(
		NewGetCommand(),
		NewPostCommand(),
		NewPutCommand(),
		NewPatchCommand(),
		NewDeleteCommand(),
		NewHeadCommand(),
		NewOptionsCommand(),
		azdext.NewVersionCommand("jongio.azd.rest", version.Version, &outputFormat),
		azdext.NewMetadataCommand("1.0", "jongio.azd.rest", NewRootCmd),
		azdext.NewListenCommand(nil),
		NewMCPCommand(),
	)

	return rootCmd
}

// buildRequestOptions constructs RequestOptions from global flags and method-specific args
func buildRequestOptions(method string, url string) (client.RequestOptions, error) {
	opts := client.RequestOptions{
		Method:          method,
		URL:             url,
		Headers:         make(map[string]string),
		Scope:           scope,
		SkipAuth:        noAuth,
		Verbose:         verbose,
		Timeout:         timeout,
		Insecure:        insecure,
		FollowRedirects: followRedirects,
		MaxRedirects:    maxRedirects,
		OutputFile:      outputFile,
		Format:          outputFormat,
		Binary:          binary,
		Retry:           retry,
		MaxResponseSize: 100 * 1024 * 1024, // 100MB default
		Paginate:        paginate,
	}

	// Parse headers
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			return opts, fmt.Errorf("invalid header format: %s (expected Key:Value)", header)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		opts.Headers[key] = value
	}

	// Handle request body
	var bodyFile *os.File
	if dataFile != "" {
		// Support @{file} shorthand
		filePath := dataFile
		if strings.HasPrefix(dataFile, "@") {
			filePath = strings.TrimPrefix(dataFile, "@")
		}
		file, err := os.Open(filePath) // #nosec G304 -- User-specified file path via --data-file flag is intentional.
		if err != nil {
			return opts, fmt.Errorf("failed to open data file: %w", err)
		}
		bodyFile = file
		opts.Body = file
	} else if data != "" {
		opts.Body = strings.NewReader(data)
	}

	// closeBodyFile is a helper to close the opened file on error paths.
	closeBodyFile := func() {
		if bodyFile != nil {
			_ = bodyFile.Close()
		}
	}

	// Detect scope if not provided
	if opts.Scope == "" && !opts.SkipAuth {
		detectedScope, err := auth.DetectScope(url)
		if err != nil {
			closeBodyFile()
			return opts, fmt.Errorf("failed to detect scope: %w", err)
		}
		opts.Scope = detectedScope

		// Warn if Azure host but no scope detected
		if opts.Scope == "" && auth.IsAzureHost(url) {
			fmt.Fprintf(os.Stderr, "Warning: Azure host detected but no scope found. Use --scope to provide a scope or --no-auth to skip authentication.\n")
		}
	}

	// Check if auth should be skipped
	opts.SkipAuth = client.ShouldSkipAuth(url, opts.Headers, noAuth)

	// Create token provider only when authentication is needed
	if !opts.SkipAuth {
		tokenProvider, err := auth.NewAzureTokenProvider()
		if err != nil {
			closeBodyFile()
			return opts, fmt.Errorf("failed to create token provider: %w", err)
		}
		opts.TokenProvider = tokenProvider
	}

	return opts, nil
}

// executeRequest executes an HTTP request and handles the response
func executeRequest(cmd *cobra.Command, method string, url string) error {
	opts, err := buildRequestOptions(method, url)
	if err != nil {
		return err
	}

	// Track if we opened a file so we can close it after the request
	var fileToClose *os.File
	if file, ok := opts.Body.(*os.File); ok {
		fileToClose = file
	}

	// Ensure file is closed even on error
	defer func() {
		if fileToClose != nil {
			_ = fileToClose.Close()
		}
	}()

	// Create HTTP client
	httpClient := client.NewClient(opts.TokenProvider, insecure, timeout)

	// Use command context for cancellation support (Ctrl+C)
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := httpClient.Execute(ctx, opts)
	if err != nil {
		return err
	}

	// Handle binary output
	if binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type")) {
		formatter := client.NewFormatter(verbose, outputFormat)
		return formatter.WriteRawOutput(resp.Body, outputFile)
	}

	// Format and output response
	formatter := client.NewFormatter(verbose, outputFormat)
	formatted, err := formatter.Format(resp)
	if err != nil {
		return fmt.Errorf("failed to format response: %w", err)
	}

	return formatter.WriteOutput(formatted, outputFile)
}
