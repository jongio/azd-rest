package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// cachedTokenProvider is reused across MCP requests to avoid
// creating a new Azure credential on every call.
var (
	cachedTokenProvider auth.TokenProvider
	tokenProviderMu     sync.Mutex
)

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

// securityPolicy is the shared security policy for MCP tools.
// Initialized via securityPolicyOnce for thread-safe lazy init.
var (
	securityPolicy     *azdext.MCPSecurityPolicy
	securityPolicyOnce sync.Once
)

func getMCPSecurityPolicy() *azdext.MCPSecurityPolicy {
	securityPolicyOnce.Do(func() {
		securityPolicy = azdext.DefaultMCPSecurityPolicy().
			RedactHeaders("Host", "Proxy-Authorization")
	})
	return securityPolicy
}

// resetSecurityPolicyForTest resets the security policy singleton so tests
// can inject a custom policy (e.g. to allow httptest loopback addresses).
// This must only be called from tests.
func resetSecurityPolicyForTest() {
	securityPolicyOnce = sync.Once{}
	securityPolicy = nil
}

// setSecurityPolicyForTest replaces the security policy singleton with a
// custom policy for testing. It resets the sync.Once and immediately marks
// it as consumed so getMCPSecurityPolicy() returns the injected policy.
func setSecurityPolicyForTest(p *azdext.MCPSecurityPolicy) {
	securityPolicyOnce = sync.Once{}
	securityPolicyOnce.Do(func() {
		securityPolicy = p
	})
}

// executeMCPRequest performs an authenticated HTTP request for MCP tools.
func executeMCPRequest(ctx context.Context, method, reqURL, body, scopeOverride string, customHeaders map[string]string) (*mcpResponse, error) {
	policy := getMCPSecurityPolicy()
	if err := policy.CheckURL(reqURL); err != nil {
		return nil, fmt.Errorf("requests to cloud metadata endpoints are blocked: %w", err)
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
func parseHeaders(args azdext.ToolArgs) (map[string]string, error) {
	headers := make(map[string]string)
	policy := getMCPSecurityPolicy()
	if args.Has("headers") {
		raw := args.Raw()
		if h, ok := raw["headers"]; ok {
			if hMap, ok := h.(map[string]any); ok {
				for k, v := range hMap {
					if policy.IsHeaderBlocked(k) {
						return nil, fmt.Errorf("header %q is not allowed", k)
					}
					if s, ok := v.(string); ok {
						headers[k] = s
					}
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
func handleBodyMethod(method string) azdext.MCPToolHandler {
	return func(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
		url, err := args.RequireString("url")
		if err != nil {
			return azdext.MCPErrorResult("missing required argument: url"), nil
		}

		body := args.OptionalString("body", "")
		scopeOverride := args.OptionalString("scope", "")
		headers, err := parseHeaders(args)
		if err != nil {
			return azdext.MCPErrorResult("%s", err.Error()), nil
		}

		resp, err := executeMCPRequest(ctx, method, url, body, scopeOverride, headers)
		if err != nil {
			return azdext.MCPErrorResult("%s", err.Error()), nil
		}

		return azdext.MCPTextResult("%s", formatResponse(resp)), nil
	}
}

// Tool handler for methods without a body (GET, DELETE)
func handleNoBodyMethod(method string) azdext.MCPToolHandler {
	return func(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
		url, err := args.RequireString("url")
		if err != nil {
			return azdext.MCPErrorResult("missing required argument: url"), nil
		}

		scopeOverride := args.OptionalString("scope", "")
		headers, err := parseHeaders(args)
		if err != nil {
			return azdext.MCPErrorResult("%s", err.Error()), nil
		}

		resp, err := executeMCPRequest(ctx, method, url, "", scopeOverride, headers)
		if err != nil {
			return azdext.MCPErrorResult("%s", err.Error()), nil
		}

		return azdext.MCPTextResult("%s", formatResponse(resp)), nil
	}
}

// handleHead handles HEAD requests (returns status + headers only).
func handleHead(ctx context.Context, args azdext.ToolArgs) (*mcp.CallToolResult, error) {
	url, err := args.RequireString("url")
	if err != nil {
		return azdext.MCPErrorResult("missing required argument: url"), nil
	}

	scopeOverride := args.OptionalString("scope", "")
	headers, err := parseHeaders(args)
	if err != nil {
		return azdext.MCPErrorResult("%s", err.Error()), nil
	}

	resp, err := executeMCPRequest(ctx, "HEAD", url, "", scopeOverride, headers)
	if err != nil {
		return azdext.MCPErrorResult("%s", err.Error()), nil
	}

	// HEAD responses omit body
	resp.Body = ""
	return azdext.MCPTextResult("%s", formatResponse(resp)), nil
}

const mcpInstructions = `You are an Azure REST API assistant powered by the azd-rest extension.
You can execute authenticated HTTP requests against Azure and other REST APIs.
OAuth scopes are automatically detected from the URL for known Azure services
(management.azure.com, graph.microsoft.com, etc.). Use the scope parameter
to override when needed. All requests include Azure bearer token authentication
by default.`

func newMCPServer() *server.MCPServer {
	policy := getMCPSecurityPolicy()
	builder := azdext.NewMCPServerBuilder("azd-rest", version.Version).
		WithRateLimit(10, 1.0).
		WithInstructions(mcpInstructions).
		WithSecurityPolicy(policy)

	// GET - readonly
	builder.AddTool("rest_get", handleNoBodyMethod("GET"),
		azdext.MCPToolOptions{
			Description: "Execute an authenticated GET request against an Azure or REST API endpoint",
			ReadOnly:    true,
		},
		mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
		mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
		mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
	)

	// POST
	builder.AddTool("rest_post", handleBodyMethod("POST"),
		azdext.MCPToolOptions{
			Description: "Execute an authenticated POST request against an Azure or REST API endpoint",
			Destructive: true,
		},
		mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
		mcp.WithString("body", mcp.Description("Request body (JSON string)")),
		mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
		mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
	)

	// PUT
	builder.AddTool("rest_put", handleBodyMethod("PUT"),
		azdext.MCPToolOptions{
			Description: "Execute an authenticated PUT request against an Azure or REST API endpoint",
			Idempotent:  true,
		},
		mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
		mcp.WithString("body", mcp.Description("Request body (JSON string)")),
		mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
		mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
	)

	// PATCH
	builder.AddTool("rest_patch", handleBodyMethod("PATCH"),
		azdext.MCPToolOptions{
			Description: "Execute an authenticated PATCH request against an Azure or REST API endpoint",
			Destructive: true,
		},
		mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
		mcp.WithString("body", mcp.Description("Request body (JSON string)")),
		mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
		mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
	)

	// DELETE - destructive
	builder.AddTool("rest_delete", handleNoBodyMethod("DELETE"),
		azdext.MCPToolOptions{
			Description: "Execute an authenticated DELETE request against an Azure or REST API endpoint",
			Destructive: true,
		},
		mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
		mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
		mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
	)

	// HEAD - readonly
	builder.AddTool("rest_head", handleHead,
		azdext.MCPToolOptions{
			Description: "Execute an authenticated HEAD request to retrieve response headers without body",
			ReadOnly:    true,
		},
		mcp.WithString("url", mcp.Required(), mcp.Description("The request URL")),
		mcp.WithString("scope", mcp.Description("OAuth scope override (auto-detected if omitted)")),
		mcp.WithObject("headers", mcp.Description("Custom HTTP headers as key-value pairs")),
	)

	return builder.Build()
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
