---
name: azd-rest
description: |
  Execute REST API calls with automatic Azure authentication and scope detection.
  USE FOR: Azure REST API calls, HTTP requests with Azure auth, management API queries,
  Key Vault access, Microsoft Graph calls, authenticated REST requests, Azure service APIs.
  DO NOT USE FOR: script execution (use azd-exec), service orchestration (use azd-app),
  AI-powered development (use azd-copilot), Azure deployments (use azd deploy).
---

# azd-rest Extension

azd-rest is an Azure Developer CLI extension that makes authenticated REST API calls
to Azure services with automatic scope detection, retry logic, and JSON formatting.
No manual token management — just point at a URL and go.

## When to Use

- Making REST API calls to Azure Management API, Microsoft Graph, Key Vault, etc.
- Querying Azure resources without needing az cli or SDK
- Testing Azure REST APIs with automatic OAuth token handling
- Making HTTP requests to any endpoint with or without Azure authentication

## Command Syntax

```
azd rest <method> <url> [flags]
```

Supported HTTP methods: `get`, `post`, `put`, `patch`, `delete`, `head`, `options`

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--scope` | `-s` | auto | OAuth scope (auto-detected for Azure services) |
| `--no-auth` | | false | Skip authentication for public APIs |
| `--header` | `-H` | [] | Custom headers (repeatable, format: Key:Value) |
| `--data` | `-d` | "" | Request body (JSON string) |
| `--data-file` | | "" | Read request body from file (supports @file shorthand) |
| `--output-file` | | "" | Write response to file |
| `--format` | `-f` | auto | Output format: auto, json, raw |
| `--verbose` | `-v` | false | Show request/response details |
| `--paginate` | | false | Follow continuation tokens/next links |
| `--retry` | | 3 | Retry attempts with exponential backoff |
| `--binary` | | false | Stream as binary without transformation |
| `--insecure` | `-k` | false | Skip TLS certificate verification |
| `--timeout` | `-t` | 30s | Request timeout (e.g., 30s, 5m, 1h) |
| `--follow-redirects` | | true | Follow HTTP redirects |
| `--max-redirects` | | 10 | Maximum redirect hops |

## Automatic Scope Detection

azd-rest detects the correct OAuth scope for 20+ Azure services based on the URL hostname:

| Service | Hostname Pattern | Scope |
|---------|------------------|-------|
| Management API | `management.azure.com` | `https://management.azure.com/.default` |
| Microsoft Graph | `graph.microsoft.com` | `https://graph.microsoft.com/.default` |
| Key Vault | `*.vault.azure.net` | `https://vault.azure.net/.default` |
| Storage | `*.blob.core.windows.net` | `https://storage.azure.com/.default` |
| Container Registry | `*.azurecr.io` | `https://containerregistry.azure.net/.default` |
| Cosmos DB | `*.documents.azure.com` | `https://cosmos.azure.com/.default` |
| App Configuration | `*.azconfig.io` | `https://azconfig.io/.default` |
| SQL Database | `*.database.windows.net` | `https://database.windows.net/.default` |
| Azure DevOps | `dev.azure.com` | `499b84ac-.../.default` |
| Kusto | `*.kusto.windows.net` | `https://{hostname}/.default` |
| Service Bus | `*.servicebus.windows.net` | `https://servicebus.azure.net/.default` |

For non-Azure endpoints, use `--scope` to provide a custom scope or `--no-auth` to skip auth.

## Authentication

Uses the same credential chain as azd and Azure CLI:
1. Azure CLI (`az login`)
2. Managed Identity
3. Service Principal (`AZURE_CLIENT_ID`/`SECRET`/`TENANT_ID`)
4. VS Code authentication

Tokens are automatically cached and reused.

## MCP Server

azd-rest includes an MCP server for AI assistant integration:

```bash
azd rest mcp serve
```

## Examples

```bash
# List Azure subscriptions
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01

# Get Key Vault secret
azd rest get https://myvault.vault.azure.net/secrets/mysecret?api-version=7.4

# Microsoft Graph - get current user
azd rest get https://graph.microsoft.com/v1.0/me

# POST with JSON body
azd rest post https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"location":"eastus","kind":"StorageV2","sku":{"name":"Standard_LRS"}}'

# POST with body from file
azd rest post https://api.example.com/resource --data-file request.json

# PATCH for partial update
azd rest patch https://management.azure.com/.../storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"tags":{"environment":"production"}}'

# DELETE a resource
azd rest delete https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}?api-version=2021-04-01

# Public API without auth
azd rest get https://api.github.com/repos/Azure/azure-dev --no-auth

# Custom headers + save response
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --header "Accept: application/json" --output-file subscriptions.json

# Verbose output with timing details
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --verbose

# Custom scope for non-Azure endpoint
azd rest get https://api.myservice.com/data --scope https://myservice.com/.default

# Paginate through results
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --paginate
```
