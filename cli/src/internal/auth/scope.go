package auth

import (
	"fmt"
	"net/url"
	"strings"
)

// DetectScope analyzes a URL and returns the appropriate Azure OAuth scope.
// Returns empty string when the hostname does not match a known Azure service.
func DetectScope(urlString string) (string, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	host := strings.ToLower(parsedURL.Hostname())
	if host == "" {
		return "", nil
	}

	path := parsedURL.EscapedPath()

	exactMatches := map[string]string{
		"management.azure.com": "https://management.azure.com/.default",
		"graph.microsoft.com":  "https://graph.microsoft.com/.default",
		"api.loganalytics.io":  "https://api.loganalytics.io/.default",
		"dev.azure.com":        "499b84ac-1321-427f-aa17-267ca6975798/.default",
	}

	if scope, ok := exactMatches[host]; ok {
		return scope, nil
	}

	if strings.HasSuffix(host, ".visualstudio.com") {
		return "499b84ac-1321-427f-aa17-267ca6975798/.default", nil
	}

	if strings.HasSuffix(host, ".kusto.windows.net") {
		return fmt.Sprintf("https://%s/.default", host), nil
	}

	if strings.HasSuffix(host, ".servicebus.windows.net") {
		if strings.Contains(path, "/queue") || strings.Contains(path, "/queues") {
			return "https://servicebus.azure.net/.default", nil
		}
		return "https://eventhubs.azure.net/.default", nil
	}

	suffixMatches := map[string]string{
		".vault.azure.net":             "https://vault.azure.net/.default",
		".blob.core.windows.net":       "https://storage.azure.com/.default",
		".queue.core.windows.net":      "https://storage.azure.com/.default",
		".table.core.windows.net":      "https://storage.azure.com/.default",
		".file.core.windows.net":       "https://storage.azure.com/.default",
		".dfs.core.windows.net":        "https://storage.azure.com/.default",
		".azurecr.io":                  "https://containerregistry.azure.net/.default",
		".documents.azure.com":         "https://cosmos.azure.com/.default",
		".azconfig.io":                 "https://azconfig.io/.default",
		".batch.azure.com":             "https://batch.core.windows.net/.default",
		".postgres.database.azure.com": "https://ossrdbms-aad.database.windows.net/.default",
		".mysql.database.azure.com":    "https://ossrdbms-aad.database.windows.net/.default",
		".mariadb.database.azure.com":  "https://ossrdbms-aad.database.windows.net/.default",
		".database.windows.net":        "https://database.windows.net/.default",
		".dev.azuresynapse.net":        "https://dev.azuresynapse.net/.default",
		".azuredatalakestore.net":      "https://datalake.azure.net/.default",
		".media.azure.net":             "https://rest.media.azure.net/.default",
	}

	for suffix, scope := range suffixMatches {
		if strings.HasSuffix(host, suffix) {
			return scope, nil
		}
	}

	return "", nil
}

// IsAzureHost checks if a hostname appears to be an Azure service
func IsAzureHost(urlString string) bool {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsedURL.Hostname())

	azurePatterns := []string{
		".azure.com",
		".azure.net",
		".windows.net",
		".azurecr.io",
		".azconfig.io",
		"management.azure.com",
		"graph.microsoft.com",
		"dev.azure.com",
		".visualstudio.com",
		".azuredatalakestore.net",
	}

	for _, pattern := range azurePatterns {
		if strings.Contains(host, pattern) || host == strings.TrimPrefix(pattern, ".") {
			return true
		}
	}

	return false
}
