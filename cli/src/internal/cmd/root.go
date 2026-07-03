// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/jongio/azd-rest/src/internal/service"
	"github.com/jongio/azd-rest/src/internal/skills"
	"github.com/jongio/azd-rest/src/internal/version"
	"github.com/spf13/cobra"
)

// Global flags - retained for cobra binding; snapshotted into config.Config
// before any business logic executes. The service layer receives only Config,
// never these globals directly (#43, #80).
var (
	scope           string
	noAuth          bool
	apiVersion      string
	urlParams       []string
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
	maxPages        int
	maxResponseSize int64
)

// httpMethodDef defines one HTTP method subcommand for the table-driven factory (#68).
type httpMethodDef struct {
	Method string // HTTP method (uppercase)
	Use    string // cobra Use field
	Short  string // cobra Short description
	Long   string // cobra Long description
}

// httpMethods is the authoritative table of HTTP method commands.
// Adding a new method requires only a new entry here (#68).
var httpMethods = []httpMethodDef{
	{"GET", "get <url>", "Execute a GET request", "Execute a GET request to the specified URL with automatic Azure authentication."},
	{"POST", "post <url>", "Execute a POST request", "Execute a POST request to the specified URL with automatic Azure authentication."},
	{"PUT", "put <url>", "Execute a PUT request", "Execute a PUT request to the specified URL with automatic Azure authentication."},
	{"PATCH", "patch <url>", "Execute a PATCH request", "Execute a PATCH request to the specified URL with automatic Azure authentication."},
	{"DELETE", "delete <url>", "Execute a DELETE request", "Execute a DELETE request to the specified URL with automatic Azure authentication."},
	{"HEAD", "head <url>", "Execute a HEAD request", "Execute a HEAD request to the specified URL with automatic Azure authentication."},
	{"OPTIONS", "options <url>", "Execute an OPTIONS request", "Execute an OPTIONS request to the specified URL with automatic Azure authentication."},
}

// newHTTPMethodCommand is the factory that produces a cobra.Command for any
// HTTP method from its definition (#68). All method commands share identical
// structure; only the method string and descriptions differ.
func newHTTPMethodCommand(def httpMethodDef) *cobra.Command {
	method := def.Method // capture for closure
	return &cobra.Command{
		Use:   def.Use,
		Short: def.Short,
		Long:  def.Long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, method, args[0])
		},
	}
}

// NewGetCommand returns the GET subcommand.
func NewGetCommand() *cobra.Command { return newHTTPMethodCommand(httpMethods[0]) }

// NewPostCommand returns the POST subcommand.
func NewPostCommand() *cobra.Command { return newHTTPMethodCommand(httpMethods[1]) }

// NewPutCommand returns the PUT subcommand.
func NewPutCommand() *cobra.Command { return newHTTPMethodCommand(httpMethods[2]) }

// NewPatchCommand returns the PATCH subcommand.
func NewPatchCommand() *cobra.Command { return newHTTPMethodCommand(httpMethods[3]) }

// NewDeleteCommand returns the DELETE subcommand.
func NewDeleteCommand() *cobra.Command { return newHTTPMethodCommand(httpMethods[4]) }

// NewHeadCommand returns the HEAD subcommand.
func NewHeadCommand() *cobra.Command { return newHTTPMethodCommand(httpMethods[5]) }

// NewOptionsCommand returns the OPTIONS subcommand.
func NewOptionsCommand() *cobra.Command { return newHTTPMethodCommand(httpMethods[6]) }

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

	// Use config.Defaults() as the single source of truth for flag default values.
	defaults := config.Defaults()

	// Extension-specific flags
	rootCmd.PersistentFlags().StringVarP(&scope, "scope", "s", "", "OAuth scope for authentication (auto-detected if not provided)")
	rootCmd.PersistentFlags().BoolVar(&noAuth, "no-auth", false, "Skip authentication (no bearer token)")
	rootCmd.PersistentFlags().StringVar(&apiVersion, "api-version", "", "Set or replace the api-version query parameter")
	rootCmd.PersistentFlags().StringArrayVar(&urlParams, "url-param", []string{}, "Set or append a URL query parameter (repeatable, format: key=value)")
	rootCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom headers (repeatable, format: Key:Value)")
	rootCmd.PersistentFlags().StringVarP(&data, "data", "d", "", "Request body (JSON string)")
	rootCmd.PersistentFlags().StringVar(&dataFile, "data-file", "", "Read request body from file (also accepts @{file} shorthand)")
	rootCmd.PersistentFlags().StringVar(&outputFile, "output-file", "", "Write response to file (raw for binary content)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", defaults.OutputFormat, "Output format: auto, json, raw, jsonl")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (show headers, timing)")
	rootCmd.PersistentFlags().BoolVar(&paginate, "paginate", false, "Follow continuation tokens/next links when supported")
	rootCmd.PersistentFlags().IntVar(&retry, "retry", defaults.Retry, "Retry attempts with exponential backoff for transient errors")
	rootCmd.PersistentFlags().BoolVar(&binary, "binary", false, "Stream request/response as binary without transformation")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS certificate verification (unsafe — do not use in production)")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", defaults.Timeout, "Request timeout")
	rootCmd.PersistentFlags().BoolVar(&followRedirects, "follow-redirects", defaults.FollowRedirects, "Follow HTTP redirects")
	rootCmd.PersistentFlags().IntVar(&maxRedirects, "max-redirects", defaults.MaxRedirects, "Maximum redirect hops")
	rootCmd.PersistentFlags().IntVar(&maxPages, "max-pages", defaults.MaxPages, "Maximum number of pages to fetch when paginating")
	rootCmd.PersistentFlags().Int64Var(&maxResponseSize, "max-response-size", defaults.MaxResponseSize, "Maximum response size in bytes")

	// Add HTTP method subcommands from the table (#68)
	for _, def := range httpMethods {
		rootCmd.AddCommand(newHTTPMethodCommand(def))
	}

	// Add non-HTTP-method subcommands
	rootCmd.AddCommand(
		azdext.NewVersionCommand("jongio.azd.rest", version.Version, &outputFormat),
		azdext.NewMetadataCommand("1.0", "jongio.azd.rest", NewRootCmd),
		azdext.NewListenCommand(nil),
		NewMCPCommand(),
	)

	return rootCmd
}

// snapshotConfig captures the current global flag values into an immutable
// Config struct (#80). This is the single point where globals are read;
// all downstream code receives Config via parameters (#43).
func snapshotConfig() config.Config {
	return config.Config{
		Scope:           scope,
		NoAuth:          noAuth,
		APIVersion:      apiVersion,
		URLParams:       urlParams,
		Headers:         headers,
		Data:            data,
		DataFile:        dataFile,
		OutputFile:      outputFile,
		OutputFormat:    outputFormat,
		Verbose:         verbose,
		Paginate:        paginate,
		Retry:           retry,
		Binary:          binary,
		Insecure:        insecure,
		Timeout:         timeout,
		FollowRedirects: followRedirects,
		MaxRedirects:    maxRedirects,
		MaxPages:        maxPages,
		MaxResponseSize: maxResponseSize,
	}
}

// defaultService is the production RequestService, lazily initialized.
// Tests can replace it via the requestService variable.
var requestService *service.RequestService

func getRequestService() *service.RequestService {
	if requestService != nil {
		return requestService
	}
	requestService = service.NewRequestService(
		service.DefaultTokenProviderFactory,
		service.DefaultHTTPClientFactory,
	)
	return requestService
}

// buildRequestOptions constructs RequestOptions from global flags and method-specific args.
// Delegates to the service layer (#42) after snapshotting config (#80).
func buildRequestOptions(method string, url string) (client.RequestOptions, error) {
	cfg := snapshotConfig()
	svc := getRequestService()
	opts, cleanup, err := svc.BuildRequestOptions(cfg, method, url)
	if err != nil {
		return opts, err
	}
	// For backward compatibility with tests that call buildRequestOptions directly,
	// we don't call cleanup here - the caller (executeRequest) handles it.
	_ = cleanup
	return opts, nil
}

// executeRequest executes an HTTP request and handles the response.
// It snapshots global flags into a Config (#80), then delegates to the
// service layer (#42) which receives dependencies via injection (#43).
func executeRequest(cmd *cobra.Command, method string, url string) error {
	cfg := snapshotConfig()
	svc := getRequestService()

	// Use command context for cancellation support (Ctrl+C)
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	return svc.Execute(ctx, cfg, method, url)
}

// Ensure imports are used.
var _ = auth.DetectScope
