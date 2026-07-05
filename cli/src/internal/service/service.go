// Package service extracts core business logic from the cmd package into a
// testable service layer. It defines interface contracts for auth and HTTP
// client dependencies (#44) and centralizes request building/execution (#42).
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
)

// TokenProviderFactory creates a TokenProvider. Abstracting this allows tests
// to inject mocks without touching real Azure credentials.
type TokenProviderFactory func() (client.TokenProvider, error)

// HTTPClientFactory creates an HTTP client given a token provider and config.
type HTTPClientFactory func(tp client.TokenProvider, insecure bool, timeout time.Duration) *client.Client

// RequestService encapsulates the business logic for building and executing
// HTTP requests. It receives its dependencies via constructor injection (#43).
type RequestService struct {
	tokenProviderFactory TokenProviderFactory
	httpClientFactory    HTTPClientFactory
}

// NewRequestService constructs a RequestService with injected dependencies.
func NewRequestService(tpf TokenProviderFactory, hcf HTTPClientFactory) *RequestService {
	return &RequestService{
		tokenProviderFactory: tpf,
		httpClientFactory:    hcf,
	}
}

// DefaultTokenProviderFactory is the production factory using Azure credentials.
func DefaultTokenProviderFactory() (client.TokenProvider, error) {
	return auth.NewAzureTokenProvider()
}

// DefaultHTTPClientFactory is the production factory using the real HTTP client.
func DefaultHTTPClientFactory(tp client.TokenProvider, insecure bool, timeout time.Duration) *client.Client {
	return client.NewClient(tp, insecure, timeout)
}

// writeDiagnostic writes a non-error advisory message (warning or notice) to w
// unless silent mode is enabled. It is only for informational diagnostics;
// errors and response output must never be routed through it, so silencing
// diagnostics can never hide a genuine failure (#171).
func writeDiagnostic(w io.Writer, silent bool, format string, args ...any) {
	if silent {
		return
	}
	fmt.Fprintf(w, format, args...)
}

func applyAPIVersion(rawURL, apiVersion string) (string, error) {
	if apiVersion == "" {
		return rawURL, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL for --api-version: %w", err)
	}
	query := parsed.Query()
	query.Set("api-version", apiVersion)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// applyURLParams sets or appends query parameters from repeatable key=value flags.
// The first occurrence of a key replaces any existing value on the URL; further
// occurrences of the same key append, so multi-valued parameters are possible.
func applyURLParams(rawURL string, params []string) (string, error) {
	if len(params) == 0 {
		return rawURL, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL for --url-param: %w", err)
	}
	query := parsed.Query()
	seen := make(map[string]bool)
	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return "", fmt.Errorf("invalid --url-param format: %s (expected key=value)", param)
		}
		key, value := parts[0], parts[1]
		if seen[key] {
			query.Add(key, value)
		} else {
			query.Set(key, value)
			seen[key] = true
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// BuildRequestOptions constructs RequestOptions from a Config and method/URL.
// The caller owns the returned Body (if it is an *os.File, it must be closed).
//
// File handle ownership (#82): When Config.DataFile is set, this function opens
// the file and assigns it to opts.Body. The caller is responsible for closing
// the file after the request completes. The returned cleanup function handles
// this - call it on error paths. On success paths the caller should defer it.
func (s *RequestService) BuildRequestOptions(cfg config.Config, method, url string) (client.RequestOptions, func(), error) {
	requestURL, err := applyAPIVersion(url, cfg.APIVersion)
	if err != nil {
		return client.RequestOptions{}, nil, err
	}

	requestURL, err = applyURLParams(requestURL, cfg.URLParams)
	if err != nil {
		return client.RequestOptions{}, nil, err
	}

	opts := client.RequestOptions{
		Method:          method,
		URL:             requestURL,
		Headers:         make(map[string]string),
		Scope:           cfg.Scope,
		SkipAuth:        cfg.NoAuth,
		Verbose:         cfg.Verbose,
		Timeout:         cfg.Timeout,
		Insecure:        cfg.Insecure,
		FollowRedirects: cfg.FollowRedirects,
		MaxRedirects:    cfg.MaxRedirects,
		OutputFile:      cfg.OutputFile,
		Format:          cfg.OutputFormat,
		Binary:          cfg.Binary,
		Retry:           cfg.Retry,
		MaxResponseSize: cfg.MaxResponseSize,
		Paginate:        cfg.Paginate,
	}

	// Parse headers
	for _, header := range cfg.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			return opts, nil, fmt.Errorf("invalid header format: %s (expected Key:Value)", header)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		opts.Headers[key] = value
	}

	// Form fields (#202): build an application/x-www-form-urlencoded body from
	// repeatable --form-field flags. This is mutually exclusive with a raw body.
	if len(cfg.FormFields) > 0 {
		if cfg.Data != "" || cfg.DataFile != "" {
			return opts, nil, fmt.Errorf("--form-field cannot be combined with --data or --data-file")
		}
		encoded, err := encodeFormFields(cfg.FormFields)
		if err != nil {
			return opts, nil, err
		}
		opts.Body = strings.NewReader(encoded)
		if !hasHeader(opts.Headers, contentTypeHeader) {
			opts.Headers[contentTypeHeader] = formURLEncoded
		}
	}

	// File handle ownership (#82): bodyFile tracks the opened file so we can
	// provide a cleanup function to the caller. The caller MUST call cleanup
	// after the request completes (or on error).
	var bodyFile *os.File
	if cfg.DataFile != "" {
		filePath := cfg.DataFile
		if strings.HasPrefix(cfg.DataFile, "@") {
			filePath = strings.TrimPrefix(cfg.DataFile, "@")
		}
		file, err := os.Open(filePath) // #nosec G304 -- User-specified file path via --data-file flag is intentional.
		if err != nil {
			return opts, nil, fmt.Errorf("failed to open data file: %w", err)
		}
		bodyFile = file
		opts.Body = file
	} else if cfg.Data != "" {
		opts.Body = strings.NewReader(cfg.Data)
	}

	// cleanup closes the file handle if one was opened. The caller owns this.
	cleanup := func() {
		if bodyFile != nil {
			_ = bodyFile.Close()
		}
	}

	// Detect scope if not provided
	if opts.Scope == "" && !opts.SkipAuth {
		detectedScope, err := auth.DetectScope(requestURL)
		if err != nil {
			cleanup()
			return opts, nil, fmt.Errorf("failed to detect scope: %w", err)
		}
		opts.Scope = detectedScope

		if opts.Scope == "" && auth.IsAzureHost(requestURL) {
			writeDiagnostic(os.Stderr, cfg.Silent, "Warning: Azure host detected but no scope found. Use --scope to provide a scope or --no-auth to skip authentication.\n")
		}
	}

	// Check if auth should be skipped
	opts.SkipAuth = client.ShouldSkipAuth(url, opts.Headers, cfg.NoAuth)

	// Create token provider only when authentication is needed
	if !opts.SkipAuth {
		tokenProvider, err := s.tokenProviderFactory()
		if err != nil {
			cleanup()
			return opts, nil, fmt.Errorf("failed to create token provider: %w", err)
		}
		opts.TokenProvider = tokenProvider
	}

	return opts, cleanup, nil
}

// Execute performs the full request lifecycle: build options, execute, format output.
func (s *RequestService) Execute(ctx context.Context, cfg config.Config, method, url string) error {
	// Warn prominently when TLS verification is disabled.
	if cfg.Insecure {
		writeDiagnostic(os.Stderr, cfg.Silent, "Warning: TLS certificate verification is disabled (--insecure). Do not use this flag in production.\n")
	}

	if cfg.Repeat < 1 {
		return fmt.Errorf("--repeat must be at least 1, got %d", cfg.Repeat)
	}

	if err := validateColorMode(cfg.Color); err != nil {
		return err
	}

	opts, cleanup, err := s.BuildRequestOptions(cfg, method, url)
	if err != nil {
		return err
	}
	defer cleanup()

	// --max-time bounds the whole operation (retries and pagination included).
	// A value of zero leaves the context untouched, preserving prior behavior.
	if cfg.MaxTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.MaxTime)
		defer cancel()
	}

	httpClient := s.httpClientFactory(opts.TokenProvider, cfg.Insecure, cfg.Timeout)

	if cfg.Paginate && cfg.Verbose {
		writeDiagnostic(os.Stderr, cfg.Silent, "> Pagination enabled (max %d pages)\n", cfg.MaxPages)
	}

	if cfg.Repeat > 1 {
		return s.executeRepeat(ctx, cfg, httpClient, opts)
	}

	resp, err := httpClient.Execute(ctx, opts)
	if err != nil {
		// Distinguish the overall budget from a per-attempt timeout: when the
		// max-time context is the one that fired, ctx.Err() is non-nil here.
		if cfg.MaxTime > 0 && ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("overall time budget of %s exceeded (--max-time): %w", cfg.MaxTime, err)
		}
		return err
	}

	if cfg.ShowThrottle {
		writeThrottleInfo(os.Stderr, resp.Headers)
	}

	if err := s.writeResponseOutput(cfg, resp); err != nil {
		return err
	}

	if cfg.WriteOut != "" {
		fmt.Fprint(os.Stderr, ExpandWriteOut(cfg.WriteOut, opts.Method, opts.URL, resp))
	}

	return nil
}

// writeResponseOutput renders the response body to stdout or --output-file,
// choosing the raw path for binary content and the formatter path otherwise.
func (s *RequestService) writeResponseOutput(cfg config.Config, resp *client.Response) error {
	formatter := client.NewFormatter(cfg.Verbose, cfg.OutputFormat)

	// When --include is set, prepend the HTTP status line and response headers
	// to the output (curl -i style). Sensitive header values are redacted.
	var headerBlock string
	if cfg.Include {
		headerBlock = buildResponseHeaderBlock(resp)
	}

	if cfg.Binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type")) {
		if cfg.Include {
			data := make([]byte, 0, len(headerBlock)+len(resp.Body))
			data = append(data, headerBlock...)
			data = append(data, resp.Body...)
			return formatter.WriteRawOutput(data, cfg.OutputFile)
		}
		return formatter.WriteRawOutput(resp.Body, cfg.OutputFile)
	}

	// azd-rest renders formats that azd-core's formatter does not support
	// (currently "table" and "jsonl"), then delegates everything else to azd-core.
	if cfg.OutputFormat == "table" {
		out, err := renderTableWithColumns(resp.Body, cfg.TableColumns)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(out, cfg.OutputFile)
	}

	if cfg.OutputFormat == "jsonl" {
		out, err := renderJSONL(resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(out, cfg.OutputFile)
	}

	formatted, err := formatter.Format(resp)
	if err != nil {
		return fmt.Errorf("failed to format response: %w", err)
	}

	if shouldColorize(cfg, resp) {
		fmt.Print(headerBlock + colorizeJSON(formatted))
		return nil
	}

	return formatter.WriteOutput(headerBlock+formatted, cfg.OutputFile)
}

// buildResponseHeaderBlock renders the HTTP status line and response headers as
// a curl -i style block terminated by a blank line. Header keys are sorted for
// deterministic output and sensitive values are redacted.
func buildResponseHeaderBlock(resp *client.Response) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", resp.Status)

	keys := make([]string, 0, len(resp.Headers))
	for key := range resp.Headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, value := range resp.Headers[key] {
			fmt.Fprintf(&b, "%s: %s\n", key, client.RedactSensitiveHeader(key, value))
		}
	}
	b.WriteString("\n")
	return b.String()
}

// RedactSensitiveHeader re-exports from client for MCP use.
var RedactSensitiveHeader = client.RedactSensitiveHeader

// NewFormatter re-exports from client.
var NewFormatter = client.NewFormatter
