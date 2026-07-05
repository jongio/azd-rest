package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/spf13/cobra"
)

// scopeResult describes how azd rest would authenticate a request to a URL.
// It is the JSON payload emitted by the scope command with --format json.
type scopeResult struct {
	URL      string `json:"url"`
	AuthMode string `json:"authMode"`
	Reason   string `json:"reason,omitempty"`
	Scope    string `json:"scope,omitempty"`
	Service  string `json:"service,omitempty"`
	Note     string `json:"note,omitempty"`
}

const (
	authModeBearer = "bearer"
	authModeNone   = "none"

	serviceResourceManager = "Azure Resource Manager"
	scopeResourceManager   = "https://management.azure.com/.default"
)

// NewScopeCommand returns the scope subcommand, which previews the OAuth scope
// and authentication mode azd rest would use for a URL without sending a request.
func NewScopeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scope <url>",
		Short: "Preview the detected OAuth scope and auth mode for a URL",
		Long: `Preview how azd rest would authenticate a request to a URL without sending it.

Scope reports the detected OAuth scope, the resolved authentication mode, and the
matched Azure service when known. It respects --scope, --no-auth, and -H headers so
you can verify authentication before running a real request. No network call is made.

Examples:
  # Inspect the scope for a Management API URL
  azd rest scope https://management.azure.com/subscriptions?api-version=2020-01-01

  # See the effect of --no-auth
  azd rest scope https://api.github.com/repos/Azure/azure-dev --no-auth

  # Machine-readable output
  azd rest scope https://graph.microsoft.com/v1.0/me --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := resolveScope(args[0], scope, noAuth, headers)
			if err != nil {
				return err
			}
			return writeScopeResult(cmd.OutOrStdout(), res, outputFormat)
		},
	}
}

// resolveScope computes the authentication preview for a URL using the same rules
// the request pipeline applies: --no-auth, a supplied Authorization header, and the
// non-HTTPS skip all disable bearer auth, otherwise the scope is taken from the
// --scope override or auto-detected from the host.
func resolveScope(rawURL, scopeOverride string, noAuth bool, headerArgs []string) (scopeResult, error) {
	if strings.TrimSpace(rawURL) == "" {
		return scopeResult{}, fmt.Errorf("url is required")
	}
	if _, err := url.Parse(rawURL); err != nil {
		return scopeResult{}, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	headers, err := parseHeaderArgs(headerArgs)
	if err != nil {
		return scopeResult{}, err
	}

	res := scopeResult{URL: rawURL}

	if client.ShouldSkipAuth(rawURL, headers, noAuth) {
		res.AuthMode = authModeNone
		res.Reason = skipReason(rawURL, headers, noAuth)
		return res, nil
	}

	res.AuthMode = authModeBearer

	if scopeOverride != "" {
		res.Scope = scopeOverride
		res.Service = friendlyService(scopeOverride)
		return res, nil
	}

	detected, err := auth.DetectScope(rawURL)
	if err != nil {
		return scopeResult{}, fmt.Errorf("failed to detect scope: %w", err)
	}
	res.Scope = detected
	if detected == "" {
		if auth.IsAzureHost(rawURL) {
			res.Note = "Azure host detected but no scope matched. Pass --scope to set one or --no-auth to skip authentication."
		} else {
			res.Note = "No scope detected for this host. Pass --scope to set one or --no-auth to skip authentication."
		}
		return res, nil
	}
	res.Service = friendlyService(detected)
	return res, nil
}

// parseHeaderArgs converts repeatable Key:Value header flags into a map, matching
// the parsing the request builder uses so the preview honors an Authorization header.
func parseHeaderArgs(headerArgs []string) (map[string]string, error) {
	headers := make(map[string]string)
	for _, h := range headerArgs {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format: %s (expected Key:Value)", h)
		}
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return headers, nil
}

// skipReason explains why authentication is skipped for a URL.
func skipReason(rawURL string, headers map[string]string, noAuth bool) string {
	if noAuth {
		return "authentication skipped (--no-auth)"
	}
	for key := range headers {
		if strings.EqualFold(key, "authorization") {
			return "authentication skipped (Authorization header supplied)"
		}
	}
	if strings.HasPrefix(strings.ToLower(rawURL), "http://") {
		return "authentication skipped (non-HTTPS URL)"
	}
	return "authentication skipped"
}

// friendlyService maps a detected OAuth scope to a human-readable Azure service
// name. It returns an empty string when the scope does not match a known service.
func friendlyService(scope string) string {
	names := map[string]string{
		scopeResourceManager:                                 serviceResourceManager,
		"https://graph.microsoft.com/.default":               "Microsoft Graph",
		"https://api.loganalytics.io/.default":               "Azure Monitor Logs",
		"499b84ac-1321-427f-aa17-267ca6975798/.default":      "Azure DevOps",
		"https://vault.azure.net/.default":                   "Azure Key Vault",
		"https://storage.azure.com/.default":                 "Azure Storage",
		"https://containerregistry.azure.net/.default":       "Azure Container Registry",
		"https://cosmos.azure.com/.default":                  "Azure Cosmos DB",
		"https://azconfig.io/.default":                       "Azure App Configuration",
		"https://batch.core.windows.net/.default":            "Azure Batch",
		"https://ossrdbms-aad.database.windows.net/.default": "Azure Database for MySQL/PostgreSQL/MariaDB",
		"https://database.windows.net/.default":              "Azure SQL Database",
		"https://dev.azuresynapse.net/.default":              "Azure Synapse Analytics",
		"https://datalake.azure.net/.default":                "Azure Data Lake Store",
		"https://rest.media.azure.net/.default":              "Azure Media Services",
		"https://servicebus.azure.net/.default":              "Azure Service Bus",
		"https://eventhubs.azure.net/.default":               "Azure Event Hubs",
	}
	if name, ok := names[scope]; ok {
		return name
	}
	if strings.HasSuffix(scope, ".kusto.windows.net/.default") {
		return "Azure Data Explorer"
	}
	return ""
}

// writeScopeResult renders a scopeResult as aligned text or, when format is json,
// as indented JSON.
func writeScopeResult(w io.Writer, res scopeResult, format string) error {
	if strings.EqualFold(format, "json") {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	fmt.Fprintf(w, "URL:      %s\n", res.URL)
	fmt.Fprintf(w, "Auth:     %s\n", res.AuthMode)
	if res.Reason != "" {
		fmt.Fprintf(w, "Reason:   %s\n", res.Reason)
	}
	if res.Scope != "" {
		fmt.Fprintf(w, "Scope:    %s\n", res.Scope)
	}
	if res.Service != "" {
		fmt.Fprintf(w, "Service:  %s\n", res.Service)
	}
	if res.Note != "" {
		fmt.Fprintf(w, "Note:     %s\n", res.Note)
	}
	return nil
}
