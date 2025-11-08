package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-core/azdextutil"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// limiter uses the shared azdextutil token bucket.
// 10 burst tokens, refills at 1 token/second (≈60/min).
// TODO: migrate to azdext.MCPServerBuilder.WithRateLimit() when MCP server is refactored.
var limiter = azdextutil.NewRateLimiter(10, 1.0) //nolint:staticcheck // deprecated but functional; migration tracked

// cachedTokenProvider is reused across MCP requests to avoid
// creating a new Azure credential on every call.
var (
	cachedTokenProvider auth.TokenProvider
	tokenProviderMu     sync.Mutex
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

// blockedCIDRs are IP ranges that must never be contacted via MCP.
// Includes loopback, link-local, and RFC 1918 private ranges.
var blockedCIDRs []*net.IPNet

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8",      // "this" network (reaches loopback on Linux/macOS)
		"127.0.0.0/8",    // IPv4 loopback
		"::/128",         // IPv6 unspecified (reaches loopback)
		"::1/128",        // IPv6 loopback
		"169.254.0.0/16", // IPv4 link-local (cloud metadata)
		"fe80::/10",      // IPv6 link-local
		"10.0.0.0/8",     // RFC 1918 private
		"172.16.0.0/12",  // RFC 1918 private
		"192.168.0.0/16", // RFC 1918 private
	} {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid blocked CIDR %q: %v", cidr, err))
		}
		blockedCIDRs = append(blockedCIDRs, ipNet)
	}
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

// isBlockedIP checks whether an IP falls within any blocked CIDR range.
func isBlockedIP(ip net.IP) bool {
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// isBlockedURL returns true if the URL targets a cloud metadata endpoint,
// loopback address, or private network. Resolves hostnames via DNS to
// prevent bypass via alternate IP representations.
//
// NOTE: This check has a TOCTOU limitation — DNS is resolved here but the
// HTTP transport performs a separate resolution at connect time. A DNS
// rebinding attack could theoretically bypass this check by switching the
// DNS response between the two resolutions. A proper fix requires a custom
// net.Dialer with a Control function that validates IPs at connect time,
// which would need changes to the shared httpclient package in azd-core.
func isBlockedURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())

	// Check against hostname blocklist (for non-IP hosts like metadata.google.internal).
	for _, b := range blockedHosts {
		if host == b {
			return true
		}
	}

	// If the host parses as an IP, check it directly.
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			ip = v4
		}
		return isBlockedIP(ip)
	}

	// Resolve hostname to IPs and check each resolved address.
	// This prevents bypass via hex/octal/decimal IP representations
	// and DNS names that resolve to blocked addresses.
	addrs, err := net.LookupHost(host)
	if err != nil {
		// If DNS resolution fails, block the request to be safe.
		return true
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			if isBlockedIP(ip) {
				return true
			}
		}
	}
	return false
}

// validateScopeURLMatch ensures the scope domain matches the request URL domain.
// It allows the request URL to be a subdomain of the scope host (e.g., scope
// management.azure.com allows sub.management.azure.com). Cross-domain Azure
// scope mappings (e.g., storage.azure.com scope for *.blob.core.windows.net)
// are handled by the auto-detection path in auth.DetectScope and do not go
// through this validation — this function only checks explicit scope overrides.
//
// Security note: subdomain matching has a theoretical risk where an attacker
// controlling a subdomain (e.g., evil.example.com) could receive tokens scoped
// to the parent domain (example.com). This is unexploitable for Azure-controlled
// domains since subdomains are not registrable by third parties. The single-label
// check below prevents bare TLD matching (e.g., scope "com" is rejected).
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
	// Reject single-label scope hosts (e.g., bare TLDs like "com") to
	// prevent overly broad subdomain matching. IPv6 addresses are exempt
	// since they use colons rather than dots as separators.
	if !strings.Contains(scopeHost, ".") && !strings.Contains(scopeHost, ":") {
		return fmt.Errorf("scope host %q must have at least two labels (e.g., example.com)", scopeHost)
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
		return nil, fmt.Errorf("rate limit exceeded (10 burst, 1 request/second sustained)")
	}

	opts := client.RequestOptions{
		Method:          method,
		URL:             reqURL,
		Headers:         make(map[string]string),
		Timeout:         30 * time.Second,
		FollowRedirects: false,
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

	if !opts.SkipAuth {
		tp, err := getOrCreateTokenProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to create token provider: %w", err)
		}
		opts.TokenProvider = tp
	}

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
		version.Version,
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
		mcp.NewTool("rest_head", urlScopeHeaderOpts(
			"Execute an authenticated HEAD request to retrieve response headers without body",
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		)...),
		handleHead,
	)

	return s
}

// NewMCPCommand creates the MCP server command group.
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
