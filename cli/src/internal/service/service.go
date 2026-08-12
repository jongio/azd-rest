// Package service extracts core business logic from the cmd package into a
// testable service layer. It defines interface contracts for auth and HTTP
// client dependencies (#44) and centralizes request building/execution (#42).
package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jmespath-community/go-jmespath"
	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
)

var stdoutWriter io.Writer = os.Stdout

const (
	// TraceparentAutoValue tells BuildRequestOptions to generate a fresh W3C traceparent header.
	TraceparentAutoValue = "generate"

	// clientRequestIDHeader is the Azure correlation header set by --client-request-id.
	clientRequestIDHeader = "x-ms-client-request-id"
	traceparentHeader     = "traceparent"
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

// loadHeaderFile reads headers from a file, one "Key: Value" per line. Blank
// lines and lines beginning with "#" are ignored. It returns a clear error for
// a missing file or a malformed line.
func loadHeaderFile(path string) (map[string]string, error) {
	file, err := os.Open(path) // #nosec G304 -- User-specified file path via --header-file flag is intentional.
	if err != nil {
		return nil, fmt.Errorf("failed to open header file: %w", err)
	}
	defer func() { _ = file.Close() }()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header on line %d of %s: %q (expected Key: Value)", lineNum, path, line)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid header on line %d of %s: %q (empty header name)", lineNum, path, line)
		}
		result[key] = strings.TrimSpace(parts[1])
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read header file: %w", err)
	}
	return result, nil
}

func loadHeaderEnv(entries []string, lookupEnv func(string) (string, bool)) (map[string]string, error) {
	result := make(map[string]string)
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --header-env format: %s (expected Key=ENV_VAR)", entry)
		}
		key := strings.TrimSpace(parts[0])
		envName := strings.TrimSpace(parts[1])
		if !validHeaderName(key) {
			return nil, fmt.Errorf("invalid --header-env header name: %q", key)
		}
		if !validEnvName(envName) {
			return nil, fmt.Errorf("invalid --header-env environment variable name: %q", envName)
		}
		value, ok := lookupEnv(envName)
		if !ok || value == "" {
			return nil, fmt.Errorf("environment variable %s for --header-env %s is not set or empty", envName, key)
		}
		result[key] = value
	}
	return result, nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r <= 32 || r >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
			return false
		}
	}
	return true
}

func validEnvName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func applyQueryToResponse(resp *client.Response, expression string) error {
	if expression == "" {
		return nil
	}
	if !strings.Contains(strings.ToLower(resp.Headers.Get("Content-Type")), "json") && !client.IsJSON(resp.Body) {
		return fmt.Errorf("--query requires a JSON response")
	}

	var data any
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return fmt.Errorf("failed to parse JSON response for --query: %w", err)
	}

	result, err := jmespath.Search(expression, data)
	if err != nil {
		return fmt.Errorf("invalid --query expression: %w", err)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to encode --query result: %w", err)
	}

	resp.Body = body
	return nil
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

func resolveRequestURL(rawURL, baseURL string) (string, error) {
	if baseURL == "" {
		return rawURL, nil
	}

	request, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid request URL: %w", err)
	}
	if request.IsAbs() {
		return rawURL, nil
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid --base-url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("--base-url must include scheme and host")
	}
	if !strings.HasPrefix(rawURL, "/") && base.Path != "" && !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	return base.ResolveReference(request).String(), nil
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

// loadURLParamFile reads URL parameters from a file, one "key=value" per line.
// Blank lines and lines beginning with "#" are ignored.
func loadURLParamFile(path string) ([]string, error) {
	file, err := os.Open(path) // #nosec G304 -- User-specified file path via --url-param-file flag is intentional.
	if err != nil {
		return nil, fmt.Errorf("failed to open URL parameter file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var result []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid URL parameter on line %d of %s: %q (expected key=value)", lineNum, path, line)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid URL parameter on line %d of %s: %q (empty parameter name)", lineNum, path, line)
		}
		result = append(result, key+"="+strings.TrimSpace(parts[1]))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read URL parameter file: %w", err)
	}
	return result, nil
}

// BuildRequestOptions constructs RequestOptions from a Config and method/URL.
// The caller owns the returned Body (if it is an *os.File, it must be closed).
//
// File handle ownership (#82): When Config.DataFile is set, this function opens
// the file and assigns it to opts.Body. The caller is responsible for closing
// the file after the request completes. The returned cleanup function handles
// this - call it on error paths. On success paths the caller should defer it.
func (s *RequestService) BuildRequestOptions(cfg config.Config, method, url string) (client.RequestOptions, func(), error) {
	requestURL, err := resolveRequestURL(url, cfg.BaseURL)
	if err != nil {
		return client.RequestOptions{}, nil, err
	}

	requestURL, err = applyAPIVersion(requestURL, cfg.APIVersion)
	if err != nil {
		return client.RequestOptions{}, nil, err
	}

	if cfg.URLParamFile != "" {
		fileParams, err := loadURLParamFile(cfg.URLParamFile)
		if err != nil {
			return client.RequestOptions{}, nil, err
		}
		requestURL, err = applyURLParams(requestURL, fileParams)
		if err != nil {
			return client.RequestOptions{}, nil, err
		}
	}

	requestURL, err = applyURLParams(requestURL, cfg.URLParams)
	if err != nil {
		return client.RequestOptions{}, nil, err
	}

	// Host allowlist (#219): when set, the request host must match an allowed
	// pattern before any token is acquired or request is sent. This runs early
	// so a disallowed host never triggers authentication.
	if len(cfg.AllowedHosts) > 0 {
		host, allowed, parseErr := requestHostAllowed(requestURL, cfg.AllowedHosts)
		if parseErr != nil {
			return client.RequestOptions{}, nil, fmt.Errorf("failed to parse request URL: %w", parseErr)
		}
		if !allowed {
			return client.RequestOptions{}, nil, fmt.Errorf("host %q is not in the --allow-host allowlist", host)
		}
		if cfg.FollowRedirects {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --allow-host is set and redirects are enabled; redirect targets are bounded by --max-redirects but are not checked against the allowlist\n")
		}
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

	// Load headers from --header-file first so an inline -H header with the
	// same key wins on conflict (parsed below).
	if cfg.HeaderFile != "" {
		fileHeaders, err := loadHeaderFile(cfg.HeaderFile)
		if err != nil {
			return opts, nil, err
		}
		for key, value := range fileHeaders {
			opts.Headers[key] = value
		}
	}

	if len(cfg.HeaderEnv) > 0 {
		envHeaders, err := loadHeaderEnv(cfg.HeaderEnv, os.LookupEnv)
		if err != nil {
			return opts, nil, err
		}
		for key, value := range envHeaders {
			opts.Headers[key] = value
		}
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

	if cfg.Accept != "" {
		opts.Headers["Accept"] = cfg.Accept
	}
	if cfg.ContentType != "" {
		opts.Headers[contentTypeHeader] = cfg.ContentType
	}

	// --data-format (#236) selects how --data / --data-file is interpreted before
	// it is sent. The default is JSON (raw passthrough). YAML is parsed and
	// re-encoded as a JSON body.
	dataFormat := cfg.DataFormat
	if dataFormat == "" {
		dataFormat = dataFormatJSON
	}
	if dataFormat != dataFormatJSON && dataFormat != dataFormatYAML {
		return opts, nil, newDataFormatError(fmt.Errorf("--data-format must be %q or %q, got %q", dataFormatJSON, dataFormatYAML, dataFormat))
	}
	if dataFormat == dataFormatYAML && (len(cfg.JSONFields) > 0 || len(cfg.JSONFieldsRaw) > 0 || len(cfg.FormFields) > 0) {
		return opts, nil, newDataFormatError(fmt.Errorf("--data-format yaml cannot be combined with --form-field, --json-field, or --json-field-raw"))
	}

	// JSON body fields (#215): assemble a JSON body from repeatable --json-field
	// and --json-field-raw flags. This is mutually exclusive with other bodies.
	if len(cfg.JSONFields) > 0 || len(cfg.JSONFieldsRaw) > 0 {
		if cfg.Data != "" || cfg.DataFile != "" || len(cfg.FormFields) > 0 {
			return opts, nil, fmt.Errorf("--json-field/--json-field-raw cannot be combined with --data, --data-file, or --form-field")
		}
		jsonBody, err := buildJSONBody(cfg.JSONFields, cfg.JSONFieldsRaw)
		if err != nil {
			return opts, nil, err
		}
		opts.Body = bytes.NewReader(jsonBody)
		if !hasHeader(opts.Headers, contentTypeHeader) {
			opts.Headers[contentTypeHeader] = applicationJSON
		}
	}

	// The --client-request-id flag is authoritative and overrides a matching -H header.
	if cfg.ClientRequestID != "" {
		opts.Headers[clientRequestIDHeader] = cfg.ClientRequestID
	}

	// The --traceparent flag is authoritative and overrides a matching -H header.
	if cfg.Traceparent != "" {
		value, traceErr := prepareTraceparentHeader(cfg.Traceparent)
		if traceErr != nil {
			return opts, nil, traceErr
		}
		opts.Headers[traceparentHeader] = value
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
	switch {
	case dataFormat == dataFormatYAML:
		// #236: read the whole body, convert YAML to a JSON body, and default the
		// Content-Type to application/json. No file handle is kept open here.
		raw, readErr := readRequestBody(cfg)
		if readErr != nil {
			return opts, nil, readErr
		}
		if len(raw) > 0 {
			jsonBody, convErr := yamlToJSON(raw)
			if convErr != nil {
				return opts, nil, newDataFormatError(fmt.Errorf("failed to parse the request body as YAML: %w", convErr))
			}
			opts.Body = bytes.NewReader(jsonBody)
			if !hasHeader(opts.Headers, contentTypeHeader) {
				opts.Headers[contentTypeHeader] = applicationJSON
			}
		}
	case cfg.DataFile != "":
		if readsFromStdin(cfg.DataFile) {
			if cfg.Data != "" {
				return opts, nil, fmt.Errorf("--data-file - cannot be combined with --data")
			}
			raw, readErr := readStdinBody()
			if readErr != nil {
				return opts, nil, readErr
			}
			opts.Body = bytes.NewReader(raw)
			break
		}
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
	case cfg.Data != "":
		if cfg.Data == "@-" {
			raw, readErr := readStdinBody()
			if readErr != nil {
				return opts, nil, readErr
			}
			opts.Body = bytes.NewReader(raw)
		} else {
			opts.Body = strings.NewReader(cfg.Data)
		}
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
	opts.SkipAuth = client.ShouldSkipAuth(requestURL, opts.Headers, cfg.NoAuth)

	// Create token provider only when authentication is needed
	if !opts.SkipAuth && !cfg.DryRun {
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
	if cfg.RedactFile != "" {
		paths, err := loadRedactFile(cfg.RedactFile)
		if err != nil {
			return err
		}
		cfg.Redact = append(paths, cfg.Redact...)
	}

	// Warn prominently when TLS verification is disabled.
	if cfg.Insecure {
		writeDiagnostic(os.Stderr, cfg.Silent, "Warning: TLS certificate verification is disabled (--insecure). Do not use this flag in production.\n")
	}

	if cfg.ReadOnly {
		if err := validateReadOnlyMethod(method); err != nil {
			return err
		}
	}

	if cfg.Repeat < 1 {
		return fmt.Errorf("--repeat must be at least 1, got %d", cfg.Repeat)
	}
	if cfg.Limit < 0 {
		return fmt.Errorf("--limit must be at least 1 when set, got %d", cfg.Limit)
	}
	if cfg.RepeatDelay < 0 {
		return fmt.Errorf("--repeat-delay cannot be negative, got %s", cfg.RepeatDelay)
	}

	if err := validateColorMode(cfg.Color); err != nil {
		return err
	}

	// --raw-output (#234) only makes sense with --query. Reject the combination
	// up front (exit 2, no network call) so the flag never silently does nothing.
	if cfg.RawOutput && cfg.Query == "" {
		return newRawOutputUsageError("--raw-output requires --query")
	}
	if cfg.Count && cfg.NoBody {
		return newCountUsageError("--count and --no-body cannot be used together")
	}
	if cfg.Template != "" && cfg.Count {
		return newTemplateConfigError("--template and --count cannot be used together")
	}
	if cfg.Template != "" && cfg.NoBody {
		return newTemplateConfigError("--template and --no-body cannot be used together")
	}

	allowedStatuses, err := parseAllowedStatuses(cfg.AllowStatus)
	if err != nil {
		return err
	}

	if err := validateHeaderExpectations(cfg.ExpectedHeaders); err != nil {
		return err
	}

	// --max-latency (#280): parse the budget up front so an invalid value exits
	// with code 2 before any network call is made. A zero budget disables it.
	maxLatencyBudget, err := parseMaxLatency(cfg.MaxLatency)
	if err != nil {
		return err
	}

	// --template (#279): compile the template up front so invalid syntax or a
	// missing @file exits with code 2 before any network call is made. The
	// compiled result is discarded here and rebuilt when the body is rendered.
	if cfg.Template != "" {
		if _, err := parseTemplate(cfg.Template); err != nil {
			return err
		}
	}

	// Echo the correlation ID so it can be quoted in an Azure support request.
	if cfg.ClientRequestID != "" {
		fmt.Fprintf(os.Stderr, "%s: %s\n", clientRequestIDHeader, cfg.ClientRequestID)
	}

	// --cache-ttl (#283): parse first so an invalid duration fails fast with
	// exit code 2 before any request work happens.
	cacheTTL, err := parseCacheTTL(cfg.CacheTTL)
	if err != nil {
		return err
	}

	opts, cleanup, err := s.BuildRequestOptions(cfg, method, url)
	if err != nil {
		return err
	}
	defer cleanup()

	if cfg.DryRun {
		return writeDryRun(stdoutWriter, cfg, opts)
	}

	// --max-time bounds the whole operation (retries and pagination included).
	// A value of zero leaves the context untouched, preserving prior behavior.
	if cfg.MaxTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.MaxTime)
		defer cancel()
	}

	// Caching applies only to a single GET without a body. Request construction and
	// host-policy validation always run first. Authenticated lookups acquire the
	// exact token used by the eventual request and include its fingerprint in
	// the key so entries cannot cross credential boundaries.
	cacheEnabled := cacheTTL > 0 &&
		cfg.Repeat == 1 &&
		strings.EqualFold(opts.Method, http.MethodGet) &&
		opts.Body == nil
	var cacheCtx cacheContext
	cacheReady := false
	if cacheEnabled {
		cacheCtx, err = newCacheContext(ctx, &opts)
		if err != nil {
			return err
		}
		cacheReady = true
		if cfg.NoCache {
			if err := removeCache(cacheCtx.dir, cacheCtx.key); err != nil {
				return err
			}
		} else {
			cacheResponseLimit := cfg.MaxResponseSize
			if cacheResponseLimit <= 0 {
				cacheResponseLimit = config.Defaults().MaxResponseSize
			}
			if cached, hit := readCache(cacheCtx.dir, cacheCtx.key, cacheTTL, cacheResponseLimit); hit {
				if cfg.Verbose {
					writeDiagnostic(os.Stderr, cfg.Silent, "> served from cache (max age %s)\n", cacheTTL)
				}
				return s.handleResponse(cfg, opts.Method, opts.URL, cached, allowedStatuses, maxLatencyBudget)
			}
		}
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

	// --cache-ttl (#283): store only successful GET responses. A write failure
	// is a note, not a request failure, so the response is still served.
	if cacheReady && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if writeErr := writeCache(cacheCtx.dir, cacheCtx.key, resp); writeErr != nil {
			writeDiagnostic(os.Stderr, cfg.Silent, "Warning: failed to write response cache: %v\n", writeErr)
		}
	}

	return s.handleResponse(cfg, opts.Method, opts.URL, resp, allowedStatuses, maxLatencyBudget)
}

// handleResponse runs the shared post-response pipeline for both live and
// cached responses: apply --query, emit throttle and header diagnostics, write
// the body, expand --write-out, and honor --fail. method and finalURL feed
// --write-out so a cache hit reports the same request metadata as a live call.
func (s *RequestService) handleResponse(
	cfg config.Config,
	method string,
	finalURL string,
	resp *client.Response,
	allowedStatuses allowedStatusRanges,
	maxLatencyBudget time.Duration,
) error {
	// --diff (#266): compare the JSON response against a baseline file and print
	// a unified diff. This is terminal: it replaces normal body output and exits
	// non-zero on drift so snapshot checks in CI can detect changes.
	if cfg.Diff != "" {
		return diffAgainstBaseline(os.Stdout, resp.Body, cfg.Diff)
	}

	// Capture the original response body before --query rewrites it so --expect
	// asserts on the full response regardless of what --query prints (#269).
	originalBody := resp.Body

	if cfg.Query != "" {
		if err := applyQueryToResponse(resp, cfg.Query); err != nil {
			return err
		}
	}

	if cfg.ShowThrottle {
		writeThrottleInfo(os.Stderr, resp.Headers)
	}

	if err := writeResponseMetadata(cfg.MetadataFile, method, finalURL, resp); err != nil {
		return err
	}

	if cfg.ShowRequestIDs {
		writeRequestIDs(os.Stderr, resp.Headers)
	}

	if cfg.DumpHeaders != "" {
		if err := dumpResponseHeaders(cfg.DumpHeaders, resp); err != nil {
			return err
		}
	}

	if err := s.writeResponseOutput(cfg, resp); err != nil {
		return err
	}

	if cfg.WriteOut != "" {
		fmt.Fprint(os.Stderr, ExpandWriteOut(cfg.WriteOut, method, finalURL, resp))
	}

	if err := checkExpectedHeaders(resp, cfg.ExpectedHeaders); err != nil {
		return err
	}

	// --expect (#269): after the body has been written, assert JMESPath
	// expressions against the original response and return a non-zero exit when
	// one does not hold. Evaluated before --fail so an explicit body assertion
	// is reported ahead of the coarser status-code gate.
	if len(cfg.Expect) > 0 {
		if err := evaluateExpectations(originalBody, resp.Headers.Get("Content-Type"), cfg.Expect); err != nil {
			return err
		}
	}

	// --validate-schema (#267): after the body has been written, check the
	// response against a JSON Schema and return non-zero when it does not
	// conform so contract checks in CI can fail the build.
	if cfg.ValidateSchema != "" {
		if err := validateResponseSchema(os.Stderr, resp.Body, cfg.ValidateSchema); err != nil {
			return err
		}
	}

	// --fail (#233): after the body and metadata have been written, return a
	// non-zero exit for an error status so scripts and CI can detect failures.
	if cfg.Fail && resp.StatusCode >= 400 && !allowedStatuses.allows(resp.StatusCode) {
		return newHTTPFailError(resp.StatusCode, hostFromURL(finalURL))
	}

	// --max-latency (#280): the response has already been written, so this only
	// changes the outcome, not the output. A request slower than the budget is
	// reported as a failure, letting CI gate on performance without aborting the
	// request mid-flight.
	if maxLatencyBudget > 0 && resp.Duration > maxLatencyBudget {
		writeDiagnostic(os.Stderr, cfg.Silent, "> response took %s, over the --max-latency budget of %s\n", resp.Duration, maxLatencyBudget)
		return newMaxLatencyExceededError(maxLatencyBudget, resp.Duration, resp.StatusCode, hostFromURL(finalURL))
	}

	return nil
}

func prepareTraceparentHeader(value string) (string, error) {
	if value == TraceparentAutoValue {
		return generateTraceparent()
	}

	normalized := strings.ToLower(strings.TrimSpace(value))
	if err := validateTraceparent(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

func generateTraceparent() (string, error) {
	traceID, err := randomNonZeroHex(16)
	if err != nil {
		return "", err
	}
	parentID, err := randomNonZeroHex(8)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("00-%s-%s-01", traceID, parentID), nil
}

func randomNonZeroHex(length int) (string, error) {
	buf := make([]byte, length)
	for {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to generate traceparent: %w", err)
		}
		if !allZeroBytes(buf) {
			return hex.EncodeToString(buf), nil
		}
	}
}

func validateTraceparent(value string) error {
	parts := strings.Split(value, "-")
	if len(parts) != 4 {
		return fmt.Errorf("invalid traceparent %q: expected version, trace ID, parent ID, and flags", value)
	}
	if parts[0] != "00" {
		return fmt.Errorf("invalid traceparent %q: only version 00 is supported", value)
	}
	if !isLowerHex(parts[1], 32) || allZeroHex(parts[1]) {
		return fmt.Errorf("invalid traceparent %q: trace ID must be 32 non-zero lowercase hex characters", value)
	}
	if !isLowerHex(parts[2], 16) || allZeroHex(parts[2]) {
		return fmt.Errorf("invalid traceparent %q: parent ID must be 16 non-zero lowercase hex characters", value)
	}
	if !isLowerHex(parts[3], 2) {
		return fmt.Errorf("invalid traceparent %q: trace flags must be 2 lowercase hex characters", value)
	}
	return nil
}

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func allZeroHex(value string) bool {
	for _, ch := range value {
		if ch != '0' {
			return false
		}
	}
	return true
}

func allZeroBytes(value []byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}

// validateReadOnlyMethod rejects mutating request methods when read-only mode is enabled.
func validateReadOnlyMethod(method string) error {
	normalized := strings.ToUpper(method)
	if safeMethods[normalized] {
		return nil
	}
	return fmt.Errorf("--read-only blocks %s requests; allowed methods are %s", normalized, safeMethodList)
}

// writeResponseOutput renders the response body to stdout or --output-file,
// choosing the raw path for binary content and the formatter path otherwise.
func (s *RequestService) writeResponseOutput(cfg config.Config, resp *client.Response) error {
	formatter := client.NewFormatter(cfg.Verbose, cfg.OutputFormat)

	// --template (#279): render the response through a Go text/template. It runs
	// after --query (applied in Execute) and takes precedence over --format and
	// the other output modes, so it returns before any of them run.
	if cfg.Template != "" {
		rendered, err := renderTemplate(cfg.Template, resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(rendered, cfg.OutputFile)
	}

	// --count (#282): print the number of records and nothing else. It runs
	// after --query (applied in Execute) and takes precedence over the body
	// output, so it returns before any format dispatch. A non-JSON body is
	// reported as a clear error.
	if cfg.Count {
		n, err := countRecords(resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(fmt.Sprintf("%d\n", n), cfg.OutputFile)
	}

	if cfg.NoBody {
		if cfg.Include {
			return formatter.WriteOutput(buildResponseHeaderBlock(resp), cfg.OutputFile)
		}
		return nil
	}

	// --raw-output (#234): after --query, print a string result unquoted and an
	// array of strings one per line. Other shapes fall through to JSON so
	// nothing is silently mangled.
	if cfg.RawOutput {
		if text, ok := rawOutputText(resp.Body); ok {
			return formatter.WriteRawOutput([]byte(text), cfg.OutputFile)
		}
	}

	// --limit (#329): cap a JSON collection to the first N records before any
	// other transform or format dispatch runs, so every downstream flag sees the
	// capped set. A non-JSON or non-collection body is left unchanged with a note.
	if cfg.Limit > 0 {
		limited, changed, err := limitJSONBody(resp.Body, cfg.Limit)
		switch {
		case err != nil:
			writeDiagnostic(os.Stderr, cfg.Silent, "> --limit needs a JSON response; leaving output unchanged\\n")
		case changed:
			resp.Body = limited
		default:
			writeDiagnostic(os.Stderr, cfg.Silent, "> --limit found no top-level JSON array or value[] array; leaving output unchanged\\n")
		}
	}

	// --fields (#281): keep only the listed top-level fields. Unlike redaction
	// and flatten, this applies across every output format (json, table, csv,
	// yaml) and downstream pipes, so it runs before the format dispatch. Raw and
	// binary output cannot be parsed as JSON and are left unchanged with a note.
	if len(cfg.Fields) > 0 {
		isBinary := cfg.Binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type"))
		if isBinary || cfg.OutputFormat == formatRaw {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --fields needs parsed JSON; leaving raw or binary output unchanged\n")
		} else if projected, ok := projectFields(resp.Body, cfg.Fields); ok {
			resp.Body = projected
		} else {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --fields could not parse the response as JSON; leaving it unchanged\n")
		}
	}

	// Redaction (#216): mask matched JSON response fields before formatting.
	// Raw, XML, and binary output cannot be parsed as JSON, so it is left
	// unchanged with a note on stderr.
	if len(cfg.Redact) > 0 {
		isBinary := cfg.Binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type"))
		if isBinary || cfg.OutputFormat == formatRaw || cfg.OutputFormat == formatXML {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --redact needs parsed JSON; leaving raw, XML, or binary output unchanged\n")
		} else if redacted, err := redactJSONBody(resp.Body, cfg.Redact); err != nil {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --redact could not parse the response as JSON; leaving it unchanged\n")
		} else {
			resp.Body = redacted
		}
	}

	// Omission: remove matched JSON response fields entirely before formatting.
	// This is the structural complement to --redact: redaction masks the value in
	// place, omission drops the key (or array elements). Raw and binary output
	// cannot be parsed as JSON, so it is left unchanged with a note on stderr.
	if len(cfg.Omit) > 0 {
		isBinary := cfg.Binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type"))
		if isBinary || cfg.OutputFormat == formatRaw {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --omit needs parsed JSON; leaving raw or binary output unchanged\n")
		} else if omitted, err := omitJSONBody(resp.Body, cfg.Omit); err != nil {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --omit could not parse the response as JSON; leaving it unchanged\n")
		} else {
			resp.Body = omitted
		}
	}

	// Secret redaction (#265): mask the value of any key that looks sensitive
	// anywhere in the response. Like --redact it needs parsed JSON, so raw and
	// binary output are left unchanged with a note on stderr. It runs after
	// --redact so both can apply in one invocation.
	if cfg.RedactSecrets {
		isBinary := cfg.Binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type"))
		if isBinary || cfg.OutputFormat == formatRaw {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --redact-secrets needs parsed JSON; leaving raw or binary output unchanged\n")
		} else if redacted, err := redactSecretsJSONBody(resp.Body); err != nil {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --redact-secrets could not parse the response as JSON; leaving it unchanged\n")
		} else {
			resp.Body = redacted
		}
	}

	// Flatten (#237): collapse a JSON response into a single-level object keyed
	// by dotted paths. Like redaction it needs the JSON output path, so binary,
	// raw, and the structured formats (table, jsonl, yaml, csv, xml) are left
	// unchanged with a note on stderr.
	if cfg.Flatten {
		isBinary := cfg.Binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type"))
		onJSONPath := cfg.OutputFormat == string(client.FormatAuto) || cfg.OutputFormat == string(client.FormatJSON)
		switch {
		case isBinary || !onJSONPath:
			writeDiagnostic(os.Stderr, cfg.Silent, "> --flatten needs the JSON output path; leaving this response unchanged\n")
		default:
			if flattened, err := flattenJSONBody(resp.Body); err != nil {
				writeDiagnostic(os.Stderr, cfg.Silent, "> --flatten could not parse the response as JSON; leaving it unchanged\n")
			} else {
				resp.Body = flattened
			}
		}
	}

	// When --include is set, prepend the HTTP status line and response headers
	// to the output (curl -i style). Sensitive header values are redacted.
	var headerBlock string
	if cfg.Include {
		headerBlock = buildResponseHeaderBlock(resp)
	}

	if cfg.Binary || client.DetectContentType(resp.Body, resp.Headers.Get("Content-Type")) {
		if cfg.Compact {
			writeDiagnostic(os.Stderr, cfg.Silent, "> --compact needs JSON output; leaving binary output unchanged\n")
		}
		if cfg.Include {
			data := make([]byte, 0, len(headerBlock)+len(resp.Body))
			data = append(data, headerBlock...)
			data = append(data, resp.Body...)
			return formatter.WriteRawOutput(data, cfg.OutputFile)
		}
		return formatter.WriteRawOutput(resp.Body, cfg.OutputFile)
	}

	// azd-rest renders formats that azd-core's formatter does not support
	// (currently "table", "jsonl", "yaml", "csv", "tsv", "dotenv", and "xml"), then delegates everything else to azd-core.
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

	if cfg.OutputFormat == "yaml" {
		out, err := renderYAML(resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(out, cfg.OutputFile)
	}

	if cfg.OutputFormat == "csv" {
		out, err := renderCSV(resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(out, cfg.OutputFile)
	}

	if cfg.OutputFormat == "dotenv" {
		out, err := renderDotenv(resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(out, cfg.OutputFile)
	}

	if cfg.OutputFormat == "tsv" {
		out, err := renderTSV(resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(out, cfg.OutputFile)
	}

	if cfg.OutputFormat == formatXML {
		out, err := renderXML(resp.Body)
		if err != nil {
			return err
		}
		return formatter.WriteOutput(out, cfg.OutputFile)
	}

	// --compact (#235): minify JSON to a single line for the auto and json
	// formats and --query output. Raw, binary, table, jsonl, yaml, csv, and xml are
	// left untouched. A non-JSON body is left unchanged with a note on stderr.
	if cfg.Compact && cfg.OutputFormat != formatRaw {
		if compacted, ok := compactJSONBody(resp.Body); ok {
			return formatter.WriteOutput(headerBlock+compacted+"\n", cfg.OutputFile)
		}
		writeDiagnostic(os.Stderr, cfg.Silent, "> --compact needs a JSON response; leaving output unchanged\n")
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

// dumpResponseHeaders writes the response status line and headers to the path
// named by --dump-headers. A path of "-" writes to stderr so it does not mix
// with body output on stdout. Sensitive header values are redacted the same way
// the --include path redacts them.
func dumpResponseHeaders(path string, resp *client.Response) error {
	block := buildResponseHeaderBlock(resp)
	if path == "-" {
		_, err := fmt.Fprint(os.Stderr, block)
		return err
	}
	// #nosec G304 -- User-specified file path via --dump-headers flag is intentional.
	if err := os.WriteFile(path, []byte(block), 0o600); err != nil {
		return fmt.Errorf("failed to write response headers to %s: %w", path, err)
	}
	return nil
}
