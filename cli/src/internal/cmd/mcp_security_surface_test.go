package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Task 3.16 asked to extend the MCP security policy with
// ValidatePathsWithinBase for file-reading tools.
//
// There are no file-reading tools. Every tool azd-rest registers takes url,
// scope and headers, the body-carrying ones add body, and all of them add the
// four request controls. Not one accepts a filesystem path, and the body
// argument is handed to strings.NewReader verbatim, so the CLI's --body @file
// affordance does not reach the model. ValidatePathsWithinBase would therefore
// guard nothing, which is the same answer tasks 3.5 and 3.11 reached for
// azd-app and azd-copilot.
//
// The URL half of the policy is a different story and is already adopted:
// DefaultMCPSecurityPolicy blocks metadata endpoints and private networks,
// requires HTTPS outside loopback, resolves hostnames and checks every
// resulting address, and fails closed when resolution fails. executeMCPRequest
// calls CheckURL before building a request.
//
// What was missing is enforcement that survives a new tool. WithSecurityPolicy
// hands the policy to the builder, but the builder does not apply CheckURL to
// tool arguments; it cannot, since it has no idea which argument is a URL. The
// check runs because every handler happens to funnel into executeMCPRequest.
// That is a convention, and the existing tests named the three handler shapes
// by hand, so a seventh tool calling the HTTP client directly would ship with
// no test failing. These tests drive every tool the server actually registers,
// so the guard covers tools that do not exist yet.

// blockedURLProbes are URLs the policy must refuse. Each names a distinct
// reason so a failure says which protection regressed.
var blockedURLProbes = []struct {
	name   string
	url    string
	reason string
}{
	{"imds", "http://169.254.169.254/metadata/identity/oauth2/token", "cloud metadata endpoint"},
	{"gce metadata", "http://metadata.google.internal/computeMetadata/v1/", "cloud metadata endpoint"},
	{"loopback", "http://127.0.0.1:8080/admin", "private network"},
	{"rfc1918", "https://10.0.0.5/internal", "private network"},
	{"link local", "https://169.254.1.1/", "link local range"},
	{"file scheme", "file:///etc/passwd", "non http scheme"},
	{"gopher scheme", "gopher://example.com/", "non http scheme"},
}

// toolCallArgs builds a CallToolRequest carrying the given arguments.
func toolCallArgs(raw map[string]any) mcp.CallToolRequest {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = raw
	return request
}

// TestEveryRegisteredToolRefusesBlockedURLs drives each tool the server
// registers rather than each handler someone remembered to name. A tool added
// later that reaches the network without going through executeMCPRequest fails
// here, which is the whole point: the policy is only as good as the guarantee
// that every path consults it.
func TestEveryRegisteredToolRefusesBlockedURLs(t *testing.T) {
	server := newMCPServer(false)
	tools := server.ListTools()
	require.NotEmpty(t, tools, "server registered no tools, so this guard would pass vacuously")

	for name, tool := range tools {
		for _, probe := range blockedURLProbes {
			t.Run(name+"/"+probe.name, func(t *testing.T) {
				result, err := tool.Handler(context.Background(), toolCallArgs(map[string]any{
					"url": probe.url,
				}))

				require.NoError(t, err, "handler returned a transport error rather than a tool result")
				require.NotNil(t, result)
				assert.True(
					t,
					result.IsError,
					"tool %q accepted %s (%s); it must consult the security policy before making a request",
					name, probe.url, probe.reason,
				)
			})
		}
	}
}

// TestEveryRegisteredToolRefusesBlockedHeaders covers the other model
// controlled input. A model must not be able to set Authorization and have the
// request sent with credentials it chose, and it must not be able to overwrite
// headers the policy considers sensitive.
func TestEveryRegisteredToolRefusesBlockedHeaders(t *testing.T) {
	server := newMCPServer(false)
	tools := server.ListTools()
	require.NotEmpty(t, tools)

	blocked := []string{"Authorization", "authorization", "Cookie", "X-Api-Key", "Proxy-Authorization"}

	for name, tool := range tools {
		for _, header := range blocked {
			t.Run(name+"/"+header, func(t *testing.T) {
				result, err := tool.Handler(context.Background(), toolCallArgs(map[string]any{
					"url":     "https://management.azure.com/subscriptions?api-version=2020-01-01",
					"headers": map[string]any{header: "attacker supplied"},
				}))

				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(
					t,
					result.IsError,
					"tool %q accepted the %s header from the model", name, header,
				)
			})
		}
	}
}

// TestNoToolAcceptsAFilesystemPath is the trigger that would make
// ValidatePathsWithinBase worth adopting. Today nothing here reads a file, so
// the path half of the policy has nothing to protect. The day a tool gains a
// path shaped argument, this fails and that decision gets revisited rather
// than quietly skipped.
func TestNoToolAcceptsAFilesystemPath(t *testing.T) {
	server := newMCPServer(false)
	tools := server.ListTools()
	require.NotEmpty(t, tools)

	pathish := []string{"path", "file", "filename", "filepath", "dir", "directory", "output", "outfile", "cert", "keyfile"}

	for name, tool := range tools {
		properties := tool.Tool.InputSchema.Properties
		for parameter := range properties {
			lower := strings.ToLower(parameter)
			for _, candidate := range pathish {
				if lower == candidate || strings.HasSuffix(lower, "path") || strings.HasSuffix(lower, "file") {
					t.Errorf(
						"tool %q declares parameter %q, which looks like a filesystem path; "+
							"if a tool now reads files, adopt MCPSecurityPolicy.ValidatePathsWithinBase and delete this guard",
						name, parameter,
					)
				}
			}
		}
	}
}

// TestBodyArgumentIsNotFileExpanded backs the claim above. The CLI accepts
// --body @file.json, and if that expansion were shared with the MCP path a
// model could read arbitrary files through an argument documented as a JSON
// string. It is not shared: mcpHandlerFactory passes body straight through.
func TestBodyArgumentIsNotFileExpanded(t *testing.T) {
	server := newMCPServer(false)
	tools := server.ListTools()

	var checked int
	for name, tool := range tools {
		if _, hasBody := tool.Tool.InputSchema.Properties["body"]; !hasBody {
			continue
		}
		checked++

		// A blocked URL short circuits before any network call, so the only
		// thing under test is whether the leading @ was treated as a path.
		result, err := tool.Handler(context.Background(), toolCallArgs(map[string]any{
			"url":  "http://169.254.169.254/",
			"body": "@/etc/passwd",
		}))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError, "tool %q should have refused the blocked URL", name)

		text := resultText(t, result)
		assert.NotContains(t, text, "root:", "tool %q expanded @/etc/passwd as a file path", name)
		assert.NotContains(t, strings.ToLower(text), "no such file", "tool %q tried to open the body as a file", name)
	}

	require.NotZero(t, checked, "no body carrying tool was exercised, so this guard proved nothing")
}

// TestSecurityPolicyKeepsDefaultHardening pins the policy construction itself.
// RedactHeaders is chained onto DefaultMCPSecurityPolicy, and a chain is easy
// to rewrite in a way that silently drops BlockPrivateNetworks or
// RequireHTTPS, since neither has a visible call site afterwards.
func TestSecurityPolicyKeepsDefaultHardening(t *testing.T) {
	policy := getMCPSecurityPolicy()

	for _, probe := range blockedURLProbes {
		t.Run(probe.name, func(t *testing.T) {
			assert.Error(t, policy.CheckURL(probe.url), "policy no longer blocks %s (%s)", probe.url, probe.reason)
		})
	}

	t.Run("plain http to a public host", func(t *testing.T) {
		assert.Error(
			t,
			policy.CheckURL("http://example.com/"),
			"policy no longer requires HTTPS, so bearer tokens could travel in clear text",
		)
	})

	t.Run("https to a public host is allowed", func(t *testing.T) {
		assert.NoError(
			t,
			policy.CheckURL("https://management.azure.com/subscriptions?api-version=2020-01-01"),
			"policy rejects ordinary Azure calls, which would make the extension useless",
		)
	})

	for _, header := range []string{"Authorization", "Cookie", "X-Api-Key", "Host", "Proxy-Authorization"} {
		t.Run("blocks header "+header, func(t *testing.T) {
			assert.True(t, policy.IsHeaderBlocked(header), "policy no longer treats %s as sensitive", header)
		})
	}
}

// TestReadOnlyServerRegistersNoMutatingTools pins the read only mode promise.
// The mode removes write tools from the surface instead of rejecting them at
// call time, so the assertion has to be about what is registered.
func TestReadOnlyServerRegistersNoMutatingTools(t *testing.T) {
	tools := newMCPServer(true).ListTools()
	require.NotEmpty(t, tools)

	for name := range tools {
		switch name {
		case "rest_post", "rest_put", "rest_patch", "rest_delete":
			t.Errorf("read only server registered mutating tool %q", name)
		}
	}

	assert.Contains(t, tools, "rest_get", "read only server dropped rest_get")
	assert.Contains(t, tools, "rest_head", "read only server dropped rest_head")
}
