// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	// graphURL is the Azure Resource Graph query endpoint.
	graphURL = "https://management.azure.com/providers/Microsoft.ResourceGraph/resources"
	// graphAPIVersion is the default stable Resource Graph API version.
	graphAPIVersion = "2022-10-01"
)

// graphOptions maps to the Resource Graph request "options" object. Fields use
// the "$"-prefixed names the API expects and are omitted when unset.
type graphOptions struct {
	Top       int    `json:"$top,omitempty"`
	Skip      int    `json:"$skip,omitempty"`
	SkipToken string `json:"$skipToken,omitempty"`
}

// graphRequest is the JSON body sent to the Resource Graph endpoint.
type graphRequest struct {
	Query            string        `json:"query"`
	Subscriptions    []string      `json:"subscriptions,omitempty"`
	ManagementGroups []string      `json:"managementGroups,omitempty"`
	Options          *graphOptions `json:"options,omitempty"`
}

// buildGraphRequestBody builds the JSON request body for a Resource Graph query.
// Subscriptions and management groups are omitted when empty, which tells the
// service to query every subscription the caller can access. The options object
// is included only when at least one paging field is set.
func buildGraphRequestBody(query string, subscriptions, managementGroups []string, top, skip int, skipToken string) (string, error) {
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	req := graphRequest{
		Query:            query,
		Subscriptions:    subscriptions,
		ManagementGroups: managementGroups,
	}

	if top > 0 || skip > 0 || skipToken != "" {
		req.Options = &graphOptions{Top: top, Skip: skip, SkipToken: skipToken}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to build query body: %w", err)
	}
	return string(body), nil
}

// NewGraphCommand returns the graph subcommand, which runs an Azure Resource
// Graph (KQL) query against the management endpoint.
func NewGraphCommand() *cobra.Command {
	var (
		subscriptions    []string
		managementGroups []string
		top              int
		skip             int
		skipToken        string
	)

	cmd := &cobra.Command{
		Use:   "graph <kql-query>",
		Short: "Run an Azure Resource Graph query",
		Long: `Run an Azure Resource Graph query using Kusto Query Language (KQL).

The query runs against every subscription you can access unless you narrow it
with --subscription or --management-group. Authentication and the api-version
are handled for you.`,
		Example: `  # Count resources by type
  azd rest graph "Resources | summarize count() by type"

  # Scope to specific subscriptions and return the first 5 rows
  azd rest graph "Resources | project name, type" --subscription <sub-id> --top 5

  # Continue a paged result set
  azd rest graph "Resources | project name" --skip-token <token>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGraph(cmd, args[0], subscriptions, managementGroups, top, skip, skipToken)
		},
	}

	cmd.Flags().StringArrayVar(&subscriptions, "subscription", nil, "Subscription ID to scope the query (repeatable; defaults to all accessible subscriptions)")
	cmd.Flags().StringArrayVar(&managementGroups, "management-group", nil, "Management group ID to scope the query (repeatable)")
	cmd.Flags().IntVar(&top, "top", 0, "Maximum number of rows to return (maps to options.$top)")
	cmd.Flags().IntVar(&skip, "skip", 0, "Number of rows to skip (maps to options.$skip)")
	cmd.Flags().StringVar(&skipToken, "skip-token", "", "Continuation token from a previous response (maps to options.$skipToken)")

	return cmd
}

// runGraph builds the Resource Graph request body and delegates to the request
// service, reusing the same auth, retry, and formatting path as other commands.
func runGraph(cmd *cobra.Command, query string, subscriptions, managementGroups []string, top, skip int, skipToken string) error {
	body, err := buildGraphRequestBody(query, subscriptions, managementGroups, top, skip, skipToken)
	if err != nil {
		return err
	}

	cfg := snapshotConfig()
	cfg.Data = body
	cfg.DataFile = ""
	if cfg.APIVersion == "" {
		cfg.APIVersion = graphAPIVersion
	}
	// Prepend so an explicit --header Content-Type still wins.
	cfg.Headers = append([]string{"Content-Type: application/json"}, cfg.Headers...)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	return getRequestService().Execute(ctx, cfg, "POST", graphURL)
}
