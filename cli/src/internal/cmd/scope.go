package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"text/tabwriter"

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

type scopeMapping struct {
	Service string   `json:"service"`
	Scope   string   `json:"scope"`
	Hosts   []string `json:"hosts"`
	Note    string   `json:"note,omitempty"`
}

const (
	authModeBearer = "bearer"
	authModeNone   = "none"

	serviceResourceManager = "Azure Resource Manager"
	serviceMicrosoftGraph  = "Microsoft Graph"
	serviceKeyVault        = "Azure Key Vault"
	serviceStorage         = "Azure Storage"
	serviceCosmosDB        = "Azure Cosmos DB"
	serviceDataExplorer    = "Azure Data Explorer"

	scopeResourceManager = "https://management.azure.com/.default"

	hostResourceManager = "management.azure.com"
	hostGraph           = "graph.microsoft.com"
	hostKeyVault        = "*.vault.azure.net"
)

var knownScopeMappings = []scopeMapping{
	{Service: serviceResourceManager, Scope: scopeResourceManager, Hosts: []string{hostResourceManager}},
	{Service: serviceMicrosoftGraph, Scope: "https://graph.microsoft.com/.default", Hosts: []string{hostGraph}},
	{Service: "Azure Monitor Logs", Scope: "https://api.loganalytics.io/.default", Hosts: []string{"api.loganalytics.io"}},
	{Service: "Azure DevOps", Scope: "499b84ac-1321-427f-aa17-267ca6975798/.default", Hosts: []string{"dev.azure.com", "*.visualstudio.com"}},
	{Service: serviceKeyVault, Scope: "https://vault.azure.net/.default", Hosts: []string{hostKeyVault}},
	{Service: serviceStorage, Scope: "https://storage.azure.com/.default", Hosts: []string{"*.blob.core.windows.net", "*.dfs.core.windows.net", "*.queue.core.windows.net", "*.table.core.windows.net"}},
	{Service: "Azure Container Registry", Scope: "https://containerregistry.azure.net/.default", Hosts: []string{"*.azurecr.io"}},
	{Service: serviceCosmosDB, Scope: "https://cosmos.azure.com/.default", Hosts: []string{"*.documents.azure.com"}},
	{Service: "Azure App Configuration", Scope: "https://azconfig.io/.default", Hosts: []string{"*.azconfig.io"}},
	{Service: "Azure Batch", Scope: "https://batch.core.windows.net/.default", Hosts: []string{"*.batch.azure.com", "*.batch.core.windows.net"}},
	{Service: "Azure Database for MySQL/PostgreSQL/MariaDB", Scope: "https://ossrdbms-aad.database.windows.net/.default", Hosts: []string{"*.mysql.database.azure.com", "*.postgres.database.azure.com", "*.mariadb.database.azure.com"}},
	{Service: "Azure SQL Database", Scope: "https://database.windows.net/.default", Hosts: []string{"*.database.windows.net"}},
	{Service: "Azure Synapse Analytics", Scope: "https://dev.azuresynapse.net/.default", Hosts: []string{"*.dev.azuresynapse.net"}},
	{Service: "Azure Data Lake Store", Scope: "https://datalake.azure.net/.default", Hosts: []string{"*.azuredatalakestore.net"}},
	{Service: "Azure Media Services", Scope: "https://rest.media.azure.net/.default", Hosts: []string{"rest.media.azure.net"}},
	{Service: "Azure Service Bus", Scope: "https://servicebus.azure.net/.default", Hosts: []string{"*.servicebus.windows.net"}},
	{Service: "Azure Event Hubs", Scope: "https://eventhubs.azure.net/.default", Hosts: []string{"*.servicebus.windows.net"}},
	{Service: serviceDataExplorer, Scope: "https://<cluster>.kusto.windows.net/.default", Hosts: []string{"*.kusto.windows.net"}, Note: "Scope is based on the cluster host."},
}

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
	for _, mapping := range knownScopeMappings {
		if mapping.Scope == scope {
			return mapping.Service
		}
	}
	if strings.HasSuffix(scope, ".kusto.windows.net/.default") {
		return serviceDataExplorer
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

// NewScopesCommand returns the scopes subcommand, which lists built-in scope mappings.
func NewScopesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scopes",
		Short: "List built-in Azure OAuth scope mappings",
		Long: `List the built-in Azure OAuth scope mappings used by azd rest.

The command prints service names, example hosts, and OAuth scopes. It does not
make network calls. Use the scope command with a URL to preview one request.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeScopeMappings(cmd.OutOrStdout(), outputFormat)
		},
	}
}

func writeScopeMappings(w io.Writer, format string) error {
	if strings.EqualFold(format, "json") {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(knownScopeMappings)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SERVICE	SCOPE	HOSTS"); err != nil {
		return err
	}
	for _, mapping := range knownScopeMappings {
		hosts := strings.Join(mapping.Hosts, ", ")
		if mapping.Note != "" {
			hosts += " (" + mapping.Note + ")"
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", mapping.Service, mapping.Scope, hosts); err != nil {
			return err
		}
	}
	return tw.Flush()
}
