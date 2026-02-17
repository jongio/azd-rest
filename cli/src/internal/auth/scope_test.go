package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectScope(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedScope string
		expectError   bool
	}{
		// Management API
		{
			name:          "Management API - subscriptions",
			url:           "https://management.azure.com/subscriptions?api-version=2020-01-01",
			expectedScope: "https://management.azure.com/.default",
		},
		{
			name:          "Management API - resource groups",
			url:           "https://management.azure.com/subscriptions/sub-id/resourceGroups/rg-name",
			expectedScope: "https://management.azure.com/.default",
		},

		// Storage - Blob
		{
			name:          "Storage Blob - list containers",
			url:           "https://mystorageacct.blob.core.windows.net/?comp=list",
			expectedScope: "https://storage.azure.com/.default",
		},
		{
			name:          "Storage Blob - get blob",
			url:           "https://mystorageacct.blob.core.windows.net/container/blob",
			expectedScope: "https://storage.azure.com/.default",
		},

		// Storage - Queue
		{
			name:          "Storage Queue",
			url:           "https://mystorageacct.queue.core.windows.net/myqueue/messages",
			expectedScope: "https://storage.azure.com/.default",
		},

		// Storage - Table
		{
			name:          "Storage Table",
			url:           "https://mystorageacct.table.core.windows.net/mytable",
			expectedScope: "https://storage.azure.com/.default",
		},

		// Storage - File
		{
			name:          "Storage File",
			url:           "https://mystorageacct.file.core.windows.net/myshare/myfile",
			expectedScope: "https://storage.azure.com/.default",
		},

		// Storage - Data Lake
		{
			name:          "Storage Data Lake",
			url:           "https://mystorageacct.dfs.core.windows.net/filesystem/path",
			expectedScope: "https://storage.azure.com/.default",
		},

		// Key Vault
		{
			name:          "Key Vault - get secret",
			url:           "https://myvault.vault.azure.net/secrets/my-secret?api-version=7.4",
			expectedScope: "https://vault.azure.net/.default",
		},
		{
			name:          "Key Vault - list secrets",
			url:           "https://myvault.vault.azure.net/secrets?api-version=7.4",
			expectedScope: "https://vault.azure.net/.default",
		},

		// Microsoft Graph
		{
			name:          "Microsoft Graph - me",
			url:           "https://graph.microsoft.com/v1.0/me",
			expectedScope: "https://graph.microsoft.com/.default",
		},
		{
			name:          "Microsoft Graph - users",
			url:           "https://graph.microsoft.com/v1.0/users",
			expectedScope: "https://graph.microsoft.com/.default",
		},

		// Azure DevOps
		{
			name:          "Azure DevOps - dev.azure.com",
			url:           "https://dev.azure.com/myorg/_apis/projects",
			expectedScope: "499b84ac-1321-427f-aa17-267ca6975798/.default",
		},
		{
			name:          "Azure DevOps - visualstudio.com",
			url:           "https://myorg.visualstudio.com/_apis/projects",
			expectedScope: "499b84ac-1321-427f-aa17-267ca6975798/.default",
		},

		// Azure Data Explorer (Kusto)
		{
			name:          "Data Explorer - query",
			url:           "https://mycluster.eastus.kusto.windows.net/v1/rest/query",
			expectedScope: "https://mycluster.eastus.kusto.windows.net/.default",
		},

		// Container Registry
		{
			name:          "Container Registry - catalog",
			url:           "https://myregistry.azurecr.io/v2/_catalog",
			expectedScope: "https://containerregistry.azure.net/.default",
		},

		// Event Hubs
		{
			name:          "Event Hubs",
			url:           "https://mynamespace.servicebus.windows.net/eventhub",
			expectedScope: "https://eventhubs.azure.net/.default",
		},
		{
			name:          "Event Hubs with query and port",
			url:           "https://mynamespace.servicebus.windows.net:443/eventhub?api-version=2017-04",
			expectedScope: "https://eventhubs.azure.net/.default",
		},

		// Service Bus
		{
			name:          "Service Bus queue path",
			url:           "https://mynamespace.servicebus.windows.net/queues/myqueue",
			expectedScope: "https://servicebus.azure.net/.default",
		},
		{
			name:          "Service Bus queue singular",
			url:           "https://mynamespace.servicebus.windows.net/queue",
			expectedScope: "https://servicebus.azure.net/.default",
		},

		// Cosmos DB
		{
			name:          "Cosmos DB",
			url:           "https://myaccount.documents.azure.com/dbs",
			expectedScope: "https://cosmos.azure.com/.default",
		},

		// App Configuration
		{
			name:          "App Configuration",
			url:           "https://myconfig.azconfig.io/kv?api-version=1.0",
			expectedScope: "https://azconfig.io/.default",
		},

		// Azure Batch
		{
			name:          "Azure Batch",
			url:           "https://mybatch.eastus.batch.azure.com/pools?api-version=2021-06-01",
			expectedScope: "https://batch.core.windows.net/.default",
		},

		// PostgreSQL
		{
			name:          "PostgreSQL",
			url:           "https://myserver.postgres.database.azure.com",
			expectedScope: "https://ossrdbms-aad.database.windows.net/.default",
		},

		// MySQL
		{
			name:          "MySQL",
			url:           "https://myserver.mysql.database.azure.com",
			expectedScope: "https://ossrdbms-aad.database.windows.net/.default",
		},

		// MariaDB
		{
			name:          "MariaDB",
			url:           "https://myserver.mariadb.database.azure.com",
			expectedScope: "https://ossrdbms-aad.database.windows.net/.default",
		},

		// SQL Database
		{
			name:          "SQL Database",
			url:           "https://myserver.database.windows.net",
			expectedScope: "https://database.windows.net/.default",
		},

		// Synapse
		{
			name:          "Synapse",
			url:           "https://myworkspace.dev.azuresynapse.net",
			expectedScope: "https://dev.azuresynapse.net/.default",
		},

		// Data Lake
		{
			name:          "Data Lake",
			url:           "https://mydatalake.azuredatalakestore.net/webhdfs/v1",
			expectedScope: "https://datalake.azure.net/.default",
		},

		// Media Services
		{
			name:          "Media Services",
			url:           "https://myaccount.restv2.eastus.media.azure.net/api/",
			expectedScope: "https://rest.media.azure.net/.default",
		},

		// Log Analytics
		{
			name:          "Log Analytics",
			url:           "https://api.loganalytics.io/v1/workspaces/workspace-id/query",
			expectedScope: "https://api.loganalytics.io/.default",
		},

		// Non-Azure endpoints
		{
			name:          "Non-Azure - GitHub API",
			url:           "https://api.github.com/repos/Azure/azure-dev",
			expectedScope: "",
		},
		{
			name:          "Non-Azure - custom API",
			url:           "https://api.example.com/data",
			expectedScope: "",
		},

		// Edge cases
		{
			name:          "Case insensitive - uppercase domain",
			url:           "https://MANAGEMENT.AZURE.COM/subscriptions",
			expectedScope: "https://management.azure.com/.default",
		},
		{
			name:          "With custom port",
			url:           "https://management.azure.com:443/subscriptions",
			expectedScope: "https://management.azure.com/.default",
		},
		{
			name:          "HTTP scheme (still detects scope)",
			url:           "http://management.azure.com/subscriptions",
			expectedScope: "https://management.azure.com/.default",
		},

		// Error cases
		{
			name:          "Relative URL no host",
			url:           "/subscriptions",
			expectedScope: "",
		},
		{
			name:          "Invalid URL - malformed",
			url:           "ht!tp://invalid url",
			expectedScope: "",
			expectError:   true,
		},
		{
			name:        "Invalid URL - bad bracket",
			url:         "http://[::1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, err := DetectScope(tt.url)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedScope, scope)
		})
	}
}

func TestIsAzureHost(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Management API",
			url:      "https://management.azure.com/subscriptions",
			expected: true,
		},
		{
			name:     "Storage Blob",
			url:      "https://mystorageacct.blob.core.windows.net/container",
			expected: true,
		},
		{
			name:     "Key Vault",
			url:      "https://myvault.vault.azure.net/secrets",
			expected: true,
		},
		{
			name:     "Azure DevOps",
			url:      "https://dev.azure.com/myorg",
			expected: true,
		},
		{
			name:     "Container Registry",
			url:      "https://myregistry.azurecr.io/v2",
			expected: true,
		},
		{
			name:     "Non-Azure - GitHub",
			url:      "https://api.github.com/repos",
			expected: false,
		},
		{
			name:     "Non-Azure - custom domain",
			url:      "https://api.example.com/data",
			expected: false,
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAzureHost(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkDetectScope(b *testing.B) {
	urls := []string{
		"https://management.azure.com/subscriptions",
		"https://mystorageacct.blob.core.windows.net/container",
		"https://myvault.vault.azure.net/secrets/my-secret",
		"https://api.github.com/repos/Azure/azure-dev",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_, _ = DetectScope(url)
		}
	}
}
