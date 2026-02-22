package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-core/azdextutil"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// limiter uses the shared azdextutil token bucket.
// 10 burst tokens, refills at 1 token/second (≈60/min).
var limiter = azdextutil.NewRateLimiter(10, 1.0)

// cachedTokenProvider is reused across MCP requests to avoid
// creating a new Azure credential on every call.
var (
	cachedTokenProvider auth.TokenProvider
	tokenProviderMu    sync.Mutex
)

// blockedHeaders are headers that must not be set via custom headers.
var blockedHeaders = map[string]bool{
	"authorization":       true,
	"host":                true,
	"cookie":              true,
	"proxy-authorization": true,
}

// blockedHosts are cloud metadata endpoints that must never be contacted.
var blockedHosts = []string{
	"169.254.169.254",
	"fd00:ec2::254",
	"metadata.google.internal",
	"100.100.100.200",
}

// getOrCreateTokenProvider returns the cached token provider, retrying on failure.
func getOrCreateTokenProvider() (auth.TokenProvider, error) {
	tokenProviderMu.Lock()
	defer tokenProviderMu.Unlock()
	if cachedTokenProvider != nil {
		return cachedTokenProvider, nil
	}
	tp, err := auth.NewAzureTokenProvider()
	if err != nil {
		return nil, err
	}
	cachedTokenProvider = tp
	return tp, nil
}

// isBlockedURL returns true if the URL targets a cloud metadata endpoint.
func isBlockedURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	for _, b := range blockedHosts {
		if host == b {
			return true
		}
	}
	return false
}

// validateScopeURLMatch ensures the scope domain matches the request URL domain.
func validateScopeURLMatch(scope, rawURL string) error {
	scopeURL, err := url.Parse(scope)
	if err != nil {
		return fmt.Errorf("invalid scope URL: %w", err)
	}
	reqURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid request URL: %w", err)
	}
	scopeHost := strings.ToLower(scopeURL.Hostname())
	reqHost := strings.ToLower(reqURL.Hostname())
	if scopeHost == "" || reqHost == "" {
		return fmt.Errorf("scope and URL must have valid hosts")
	}
	if reqHost != scopeHost && !strings.HasSuffix(reqHost, "."+scopeHost) {
		return fmt.Errorf("scope host %q does not match request URL host %q", scopeHost, reqHost)
	}
	return nil
}

// mcpResponse is the JSON structure returned by MCP tool handlers.
type mcpResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
}

// executeMCPRequest performs an authenticated HTTP request for MCP tools.
func executeMCPRequest(ctx context.Context, method, reqURL, body, scopeOverride string, customHeaders map[string]string) (*mcpResponse, error) {
	if isBlockedURL(reqURL) {
		return nil, fmt.Errorf("requests to cloud metadata endpoints are blocked")
	}

	if !limiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded (60 requests/minute)")
	}

	opts := client.RequestOptions{
		Method:          method,
		URL:             reqURL,
		Headers:         make(map[string]string),
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		Retry:           3,
		MaxResponseSize: 10 * 1024 * 1024,
	}

	for k, v := range customHeaders {
		opts.Headers[k] = v
	}

	if body != "" {
		opts.Body = strings.NewReader(body)
	}

	// Determine scope
	detectedScope := scopeOverride
	if detectedScope == "" {
		s, err := auth.DetectScope(reqURL)
		if err != nil {
			return nil, fmt.Errorf("failed to detect scope: %w", err)
		}
		detectedScope = s
	}

	if scopeOverride != "" {
		if err := validateScopeURLMatch(scopeOverride, reqURL); err != nil {
			return nil, fmt.Errorf("scope/URL mismatch: %w", err)
		}
	}

	opts.Scope = detectedScope

	opts.SkipAuth = client.ShouldSkipAuth(reqURL, opts.Headers, false)

	tp, err := getOrCreateTokenProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create token provider: %w", err)
	}
	opts.TokenProvider = tp

	httpClient := client.NewClient(opts.TokenProvider, false, opts.Timeout)

	resp, err := httpClient.Execute(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	respHeaders := make(map[string]string)
	for key, values := range resp.Headers {
		if len(values) > 0 {
			respHeaders[key] = values[0]
		}
	}

	return &mcpResponse{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       string(resp.Body),
	}, nil
}

// parseHeaders extracts custom headers from MCP tool arguments.
func parseHeaders(request mcp.CallToolRequest) (map[string]string, error) {
	headers := make(map[string]string)
	args := request.GetArguments()
	if h, ok := args["headers"]; ok {
		if hMap, ok := h.(map[string]any); ok {
			for k, v := range hMap {
				if blockedHeaders[strings.ToLower(k)] {
					return nil, fmt.Errorf("header %q is not allowed", k)
				}
				if s, ok := v.(string); ok {
					headers[k] = s
				}
			}
		}
	}
	return headers, nil
}

func formatResponse(resp *mcpResponse) string {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"statusCode":%d,"error":"failed to marshal response"}`, resp.StatusCode)
	}
	return string(data)
}

// Tool handler for methods with a body (POST, PUT, PATCH)
func handleBodyMethod(method string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		url, err := request.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError("missing required argument: url"), nil
		}

		body := request.GetString("body", "")
		scopeOverride := request.GetString("scope", "")
		headers, err := parseHeaders(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := executeMCPRequest(ctx, method, url, body, scopeOverride, headers)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(formatResponse(resp)), nil
	}
}

// Tool handler for methods without a body (GET, DELETE)
func handleNoBodyMethod(method string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		url, err := request.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError("missing required argument: url"), nil
		}

		scopeOverride := request.GetString("scope", "")
		headers, err := parseHeaders(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := executeMCPRequest(ctx, method, url, "", scopeOverride, headers)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(formatResponse(resp)), nil
	}
}

// handleHead handles HEAD requests (returns status + headers only).
func handleHead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := request.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError("missing required argument: url"), nil
	}

	scopeOverride := request.GetString("scope", "")
	headers, err := parseHeaders(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resp, err := executeMCPRequest(ctx, "HEAD", url, "", scopeOverride, headers)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// HEAD responses omit body
	resp.Body = ""
	return mcp.NewToolResultText(formatResponse(resp)), nil
}

const mcpInstructions = `You are an Azure REST API assistant powered by the azd-rest extension.
You can execute authenticated HTTP requests against Azure and other REST APIs.
OAuth scopes are automatically detected from the URL for known Azure services
(management.azure.com, graph.microsoft.com, etc.). Use the scope parameter
to override when needed. All requests include Azure bearer token authentication
by default.`

func newMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"azd-rest",
		"0.1.0",
		server.WithInstructions(mcpInstructions),
		server.WithToolCapabilities(true),
	)

	// URL + scope + headers options (for GET, DELETE)
	urlScopeHeaderOpts := func(desc string, annotations ...mcp.ToolOption) []mcp.ToolOption {
		opts := []mcp.ToolOption{
			mcp.WithDescription(desc),
			mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
			mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
			mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
		}
		return append(opts, annotations...)
	}

	// URL + body + scope + headers options (for POST, PUT, PATCH)
	urlBodyScopeHeaderOpts := func(desc string, annotations ...mcp.ToolOption) []mcp.ToolOption {
		opts := []mcp.ToolOption{
			mcp.WithDescription(desc),
			mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
			mcp.WithString("body", mcp.Description("Request body (JSON string)")),
			mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
			mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
		}
		return append(opts, annotations...)
	}

	// GET - readonly
	s.AddTool(
		mcp.NewTool("rest_get", urlScopeHeaderOpts(
			"Execute an authenticated GET request against an Azure or REST API endpoint",
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		)...),
		handleNoBodyMethod("GET"),
	)

	// POST
	s.AddTool(
		mcp.NewTool("rest_post", urlBodyScopeHeaderOpts(
			"Execute an authenticated POST request against an Azure or REST API endpoint",
			mcp.WithDestructiveHintAnnotation(true),
		)...),
		handleBodyMethod("POST"),
	)

	// PUT
	s.AddTool(
		mcp.NewTool("rest_put", urlBodyScopeHeaderOpts(
			"Execute an authenticated PUT request against an Azure or REST API endpoint",
			mcp.WithIdempotentHintAnnotation(true),
		)...),
		handleBodyMethod("PUT"),
	)

	// PATCH
	s.AddTool(
		mcp.NewTool("rest_patch", urlBodyScopeHeaderOpts(
			"Execute an authenticated PATCH request against an Azure or REST API endpoint",
			mcp.WithDestructiveHintAnnotation(true),
		)...),
		handleBodyMethod("PATCH"),
	)

	// DELETE - destructive
	s.AddTool(
		mcp.NewTool("rest_delete", urlScopeHeaderOpts(
			"Execute an authenticated DELETE request against an Azure or REST API endpoint",
			mcp.WithDestructiveHintAnnotation(true),
		)...),
		handleNoBodyMethod("DELETE"),
	)

	// HEAD - readonly
	s.AddTool(
		mcp.NewTool("rest_head",
			mcp.WithDescription("Execute an authenticated HEAD request to retrieve response headers without body"),
			mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
			mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleHead,
	)

	return s
}

func NewMCPCommand() *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:    "mcp",
		Short:  "MCP server commands",
		Hidden: true,
	}

	serveCmd := &cobra.Command{
		Use:    "serve",
		Short:  "Start MCP stdio server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := newMCPServer()
			return server.ServeStdio(s)
		},
	}

	mcpCmd.AddCommand(serveCmd)
	return mcpCmd
}
