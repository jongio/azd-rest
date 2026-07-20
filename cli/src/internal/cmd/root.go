// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/google/uuid"
	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/jongio/azd-rest/src/internal/service"
	"github.com/jongio/azd-rest/src/internal/skills"
	"github.com/jongio/azd-rest/src/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Global flags - retained for cobra binding; snapshotted into config.Config
// before any business logic executes. The service layer receives only Config,
// never these globals directly (#43, #80).
var (
	scope           string
	noAuth          bool
	apiVersion      string
	clientRequestID string
	urlParams       []string
	headers         []string
	headerFile      string
	data            string
	dataFile        string
	dataFormat      string
	query           string
	formFields      []string
	jsonFields      []string
	jsonFieldsRaw   []string
	outputFile      string
	outputFormat    string
	verbose         bool
	paginate        bool
	flatten         bool
	retry           int
	binary          bool
	insecure        bool
	silent          bool
	timeout         time.Duration
	maxTime         time.Duration
	followRedirects bool
	maxRedirects    int
	maxPages        int
	maxResponseSize int64
	showThrottle    bool
	repeat          int
	colorMode       string
	writeOut        string
	include         bool
	allowHosts      []string
	redactPaths     []string
	redactFile      string
	tableColumns    []string
	dumpHeaders     string
	fail            bool
	rawOutput       bool
	compact         bool
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

	// Capture the SDK-provided persistent flags so environment-variable defaults
	// are applied only to the extension's own flags (#172).
	sdkFlagNames := map[string]bool{}
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		sdkFlagNames[f.Name] = true
	})

	// extensionFlagNames is populated after the extension flags are registered
	// below; the PersistentPreRunE closure reads it at execution time.
	var extensionFlagNames []string

	// Chain extension-specific PersistentPreRunE after SDK's built-in one
	sdkPreRunE := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if sdkPreRunE != nil {
			if err := sdkPreRunE(cmd, args); err != nil {
				return err
			}
		}
		// Apply AZD_REST_<FLAG> environment defaults before any request runs, so
		// an invalid value fails fast with exit code 2 (#172).
		if err := applyEnvDefaults(cmd.Flags(), extensionFlagNames, os.LookupEnv); err != nil {
			return err
		}
		// AZD_REST_ALLOWED_HOSTS is a comma separated default for --allow-host (#219).
		if err := applyAllowedHostsEnv(cmd.Flags(), os.LookupEnv); err != nil {
			return err
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
	rootCmd.PersistentFlags().StringVar(&clientRequestID, "client-request-id", "", "Set the x-ms-client-request-id header for Azure request correlation. Pass the flag without a value to generate a random ID.")
	// Passing --client-request-id without a value generates a fresh ID for this invocation.
	rootCmd.PersistentFlags().Lookup("client-request-id").NoOptDefVal = uuid.NewString()
	rootCmd.PersistentFlags().StringArrayVar(&urlParams, "url-param", []string{}, "Set or append a URL query parameter (repeatable, format: key=value)")
	rootCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom headers (repeatable, format: Key:Value)")
	rootCmd.PersistentFlags().StringVar(&headerFile, "header-file", "", "Read headers from a file (one Key: Value per line; blank lines and # comments ignored). -H overrides on conflict.")
	rootCmd.PersistentFlags().StringVarP(&data, "data", "d", "", "Request body (JSON string)")
	rootCmd.PersistentFlags().StringVar(&dataFile, "data-file", "", "Read request body from file (also accepts @{file} shorthand)")
	rootCmd.PersistentFlags().StringVar(&dataFormat, "data-format", "json", "Interpret --data / --data-file as this format before sending: json or yaml. YAML is converted to a JSON body.")
	rootCmd.PersistentFlags().StringVarP(&query, "query", "q", "", "JMESPath query to apply to JSON responses")
	rootCmd.PersistentFlags().StringArrayVar(&formFields, "form-field", []string{}, "Add an application/x-www-form-urlencoded field (repeatable, format: key=value)")
	rootCmd.PersistentFlags().StringArrayVar(&jsonFields, "json-field", []string{}, "Add a string field to a JSON request body (repeatable, format: key=value; dotted keys nest)")
	rootCmd.PersistentFlags().StringArrayVar(&jsonFieldsRaw, "json-field-raw", []string{}, "Add a raw JSON field to a JSON request body (repeatable, format: key:=json; dotted keys nest)")
	rootCmd.PersistentFlags().StringVar(&outputFile, "output-file", "", "Write response to file (raw for binary content)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", defaults.OutputFormat, "Output format: auto, json, raw, table, jsonl, yaml, csv")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (show headers, timing)")
	rootCmd.PersistentFlags().BoolVar(&paginate, "paginate", false, "Follow continuation tokens/next links when supported")
	rootCmd.PersistentFlags().BoolVar(&flatten, "flatten", false, "Flatten a JSON response into a single-level object keyed by dotted paths (e.g. properties.state, value[0].name)")
	rootCmd.PersistentFlags().IntVar(&retry, "retry", defaults.Retry, "Retry attempts with exponential backoff for transient errors")
	rootCmd.PersistentFlags().BoolVar(&binary, "binary", false, "Stream request/response as binary without transformation")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS certificate verification (unsafe — do not use in production)")
	rootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "Suppress non-error diagnostic messages on stderr (warnings and notices)")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", defaults.Timeout, "Request timeout")
	rootCmd.PersistentFlags().DurationVar(&maxTime, "max-time", defaults.MaxTime, "Overall time budget across retries and pagination (0 disables the limit)")
	rootCmd.PersistentFlags().BoolVar(&followRedirects, "follow-redirects", defaults.FollowRedirects, "Follow HTTP redirects")
	rootCmd.PersistentFlags().IntVar(&maxRedirects, "max-redirects", defaults.MaxRedirects, "Maximum redirect hops")
	rootCmd.PersistentFlags().IntVar(&maxPages, "max-pages", defaults.MaxPages, "Maximum number of pages to fetch when paginating")
	rootCmd.PersistentFlags().Int64Var(&maxResponseSize, "max-response-size", defaults.MaxResponseSize, "Maximum response size in bytes")
	rootCmd.PersistentFlags().BoolVar(&showThrottle, "show-throttle", false, "Print Azure rate-limit and quota headers to stderr, with a low-quota warning")
	rootCmd.PersistentFlags().IntVar(&repeat, "repeat", defaults.Repeat, "Send the request N times and report latency statistics")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", defaults.Color, "Colorize JSON output: auto, always, never")
	rootCmd.PersistentFlags().StringVarP(&writeOut, "write-out", "w", "", "Print curl-style response metadata to stderr after the request (e.g. \"%{http_code} %{time_total}\")")
	rootCmd.PersistentFlags().BoolVarP(&include, "include", "i", false, "Include the HTTP status line and response headers in the output")
	rootCmd.PersistentFlags().StringArrayVar(&allowHosts, "allow-host", []string{}, "Restrict requests to hosts matching a pattern (repeatable; leading *. matches subdomains). Env: AZD_REST_ALLOWED_HOSTS (comma separated)")
	rootCmd.PersistentFlags().StringArrayVar(&redactPaths, "redact", []string{}, "Mask a JSON response field before output (repeatable, dotted path, * matches array elements)")
	rootCmd.PersistentFlags().StringVar(&redactFile, "redact-file", "", "Read JSON response redaction paths from a file (one dotted path per line; blank lines and # comments ignored)")
	rootCmd.PersistentFlags().StringSliceVar(&tableColumns, "table-columns", nil, "Comma-separated columns to show, in order, for --format table (ignored for other formats)")
	rootCmd.PersistentFlags().StringVar(&dumpHeaders, "dump-headers", "", "Write response status line and headers to a file (use - for stderr)")
	rootCmd.PersistentFlags().BoolVar(&fail, "fail", false, "Exit with code 22 when the response status is 400 or higher (the response body is still printed)")
	rootCmd.PersistentFlags().BoolVarP(&rawOutput, "raw-output", "r", false, "With --query, print a string result unquoted and an array of strings one per line (like jq -r)")
	rootCmd.PersistentFlags().BoolVarP(&compact, "compact", "c", false, "Minify JSON output to a single line (applies to auto and json formats and --query results)")

	// Record the extension's own persistent flag names (those not added by the
	// SDK) so environment-variable defaults apply only to them (#172).
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !sdkFlagNames[f.Name] && f.Name != "allow-host" {
			extensionFlagNames = append(extensionFlagNames, f.Name)
		}
	})

	// Add HTTP method subcommands from the table (#68)
	for _, def := range httpMethods {
		rootCmd.AddCommand(newHTTPMethodCommand(def))
	}

	// Add non-HTTP-method subcommands
	rootCmd.AddCommand(
		NewScopeCommand(),
		azdext.NewVersionCommand("jongio.azd.rest", version.Version, &outputFormat),
		azdext.NewMetadataCommand("1.0", "jongio.azd.rest", NewRootCmd),
		azdext.NewListenCommand(nil),
		NewMCPCommand(),
		NewDoctorCommand(),
		NewGraphCommand(),
		NewWhoamiCommand(),
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
		ClientRequestID: clientRequestID,
		URLParams:       urlParams,
		Headers:         headers,
		HeaderFile:      headerFile,
		Data:            data,
		DataFile:        dataFile,
		DataFormat:      dataFormat,
		Query:           query,
		FormFields:      formFields,
		JSONFields:      jsonFields,
		JSONFieldsRaw:   jsonFieldsRaw,
		OutputFile:      outputFile,
		OutputFormat:    outputFormat,
		Verbose:         verbose,
		Flatten:         flatten,
		Paginate:        paginate,
		Retry:           retry,
		Binary:          binary,
		Insecure:        insecure,
		Silent:          silent,
		Timeout:         timeout,
		MaxTime:         maxTime,
		FollowRedirects: followRedirects,
		MaxRedirects:    maxRedirects,
		MaxPages:        maxPages,
		MaxResponseSize: maxResponseSize,
		ShowThrottle:    showThrottle,
		Repeat:          repeat,
		Color:           colorMode,
		WriteOut:        writeOut,
		Include:         include,
		AllowedHosts:    allowHosts,
		Redact:          redactPaths,
		RedactFile:      redactFile,
		TableColumns:    tableColumns,
		DumpHeaders:     dumpHeaders,
		Fail:            fail,
		RawOutput:       rawOutput,
		Compact:         compact,
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
