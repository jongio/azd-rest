package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type extensionMetadata struct {
	SchemaVersion string            `json:"schemaVersion"`
	ID            string            `json:"id"`
	Commands      []commandMetadata `json:"commands"`
}

type commandMetadata struct {
	Name        []string          `json:"name"`
	Short       string            `json:"short"`
	Subcommands []commandMetadata `json:"subcommands,omitempty"`
	Flags       []flagMetadata    `json:"flags,omitempty"`
	Args        []argMetadata     `json:"args,omitempty"`
	Examples    []exampleMetadata `json:"examples,omitempty"`
}

type flagMetadata struct {
	Name         string `json:"name"`
	Shorthand    string `json:"shorthand,omitempty"`
	Type         string `json:"type"`
	Default      string `json:"default,omitempty"`
	Description  string `json:"description"`
	Repeatable   bool   `json:"repeatable,omitempty"`
	Required     bool   `json:"required,omitempty"`
}

type argMetadata struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type exampleMetadata struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

func NewMetadataCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "metadata",
		Short:  "Generate extension metadata",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			meta := buildMetadata()
			data, err := json.MarshalIndent(meta, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}

func buildMetadata() extensionMetadata {
	httpMethods := []struct {
		name  string
		short string
	}{
		{"get", "Execute an HTTP GET request"},
		{"post", "Execute an HTTP POST request"},
		{"put", "Execute an HTTP PUT request"},
		{"patch", "Execute an HTTP PATCH request"},
		{"delete", "Execute an HTTP DELETE request"},
		{"head", "Execute an HTTP HEAD request"},
		{"options", "Execute an HTTP OPTIONS request"},
	}

	urlArg := []argMetadata{
		{Name: "url", Required: true, Description: "The URL to send the request to"},
	}

	var subcommands []commandMetadata
	for _, m := range httpMethods {
		subcommands = append(subcommands, commandMetadata{
			Name:  []string{"rest", m.name},
			Short: m.short,
			Args:  urlArg,
		})
	}

	subcommands = append(subcommands, commandMetadata{
		Name:  []string{"rest", "version"},
		Short: "Display extension version",
	})

	flags := []flagMetadata{
		{Name: "scope", Shorthand: "s", Type: "string", Description: "Override OAuth scope for authentication"},
		{Name: "no-auth", Type: "bool", Description: "Skip authentication"},
		{Name: "header", Shorthand: "H", Type: "string", Description: "Custom headers (format: Key:Value)", Repeatable: true},
		{Name: "data", Shorthand: "d", Type: "string", Description: "Request body as JSON string"},
		{Name: "data-file", Type: "string", Description: "Read request body from file"},
		{Name: "output-file", Type: "string", Description: "Save response to file"},
		{Name: "format", Shorthand: "f", Type: "string", Default: "auto", Description: "Output format (auto, json, raw)"},
		{Name: "verbose", Shorthand: "v", Type: "bool", Description: "Show headers, timing, and diagnostics"},
		{Name: "paginate", Type: "bool", Description: "Follow pagination/continuation tokens"},
		{Name: "retry", Type: "int", Default: "3", Description: "Number of retry attempts"},
		{Name: "binary", Type: "bool", Description: "Stream response as binary"},
		{Name: "insecure", Shorthand: "k", Type: "bool", Description: "Skip TLS certificate verification"},
		{Name: "timeout", Shorthand: "t", Type: "duration", Default: "30s", Description: "Request timeout"},
		{Name: "follow-redirects", Type: "bool", Default: "true", Description: "Follow HTTP redirects"},
		{Name: "max-redirects", Type: "int", Default: "10", Description: "Maximum number of redirects"},
	}

	examples := []exampleMetadata{
		{Name: "GET request", Command: "azd rest get https://management.azure.com/subscriptions?api-version=2022-12-01"},
		{Name: "POST with body", Command: "azd rest post https://example.com/api -d '{\"key\": \"value\"}'"},
		{Name: "Custom headers", Command: "azd rest get https://api.example.com -H \"Accept:application/xml\""},
		{Name: "Save to file", Command: "azd rest get https://example.com/data --output-file response.json"},
	}

	return extensionMetadata{
		SchemaVersion: "1.0",
		ID:            "jongio.azd.rest",
		Commands: []commandMetadata{
			{
				Name:        []string{"rest"},
				Short:       "Execute REST API calls with Azure authentication and scope detection",
				Subcommands: subcommands,
				Flags:       flags,
				Examples:    examples,
			},
		},
	}
}
