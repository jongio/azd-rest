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

Use `azd rest scope <url>` to preview the detected OAuth scope and auth mode for a URL without sending a request.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--scope` | `-s` | auto | OAuth scope (auto-detected for Azure services) |
| `--no-auth` | | false | Skip authentication for public APIs |
| `--client-request-id` | | "" | Set the x-ms-client-request-id header for Azure request correlation (pass without a value to generate a random ID) |
| `--header` | `-H` | [] | Custom headers (repeatable, format: Key:Value) |
| `--url-param` | | [] | Set or append a URL query parameter (repeatable, format: key=value) |
| `--data` | `-d` | "" | Request body (JSON string) |
| `--data-file` | | "" | Read request body from file (supports @file shorthand) |
| `--output-file` | | "" | Write response to file |
| `--format` | `-f` | auto | Output format: auto, json, raw, table, jsonl |
| `--verbose` | `-v` | false | Show request/response details |
| `--paginate` | | false | Follow continuation tokens/next links |
| `--retry` | | 3 | Retry attempts with exponential backoff |
| `--binary` | | false | Stream as binary without transformation |
| `--insecure` | `-k` | false | Skip TLS certificate verification |
| `--timeout` | `-t` | 30s | Request timeout for a single attempt (e.g., 30s, 5m, 1h) |
| `--max-time` | | 0 | Overall time budget across retries and pagination (0 disables the limit) |
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

MCP tools accept per-request controls:

| Argument | Default | Description |
|----------|---------|-------------|
| `timeoutSeconds` | 30 | Request timeout from 1 to 600 seconds |
| `retry` | 3 | Retry attempts from 1 to 10 |
| `maxResponseSizeBytes` | 10485760 | Maximum response size up to 52428800 bytes |
| `noAuth` | false | Skip Azure bearer token authentication |

Use `--read-only` to expose only the read tools (`rest_get`, `rest_head`). The
mutating tools (`rest_post`, `rest_put`, `rest_patch`, `rest_delete`) are omitted
from the tool surface entirely, so an assistant cannot make write calls:

```bash
azd rest mcp serve --read-only
```

## Resource Graph

Run an Azure Resource Graph query with `azd rest graph` using Kusto Query
Language (KQL). Authentication and the api-version are handled for you, and the
query runs against every subscription you can access unless you narrow it with
`--subscription` or `--management-group`.

```bash
azd rest graph "Resources | summarize count() by type"
```

| Flag | Description |
|------|-------------|
| `--subscription` | Subscription ID to scope the query (repeatable) |
| `--management-group` | Management group ID to scope the query (repeatable) |
| `--top` | Maximum number of rows to return |
| `--skip` | Number of rows to skip |
| `--skip-token` | Continuation token from a previous response |

## Identity

Check which Azure identity your requests use:

```bash
azd rest whoami
```

This acquires a token, decodes it locally, and prints the tenant, object ID,
app ID, audience, granted scopes, and expiry. The raw token is never printed.
Use `--scope` to inspect a token for a different service and `--format json`
for machine-readable output.

## Diagnostics

When a request fails with an auth error, run the doctor to find out whether the
problem is your credentials, your scope, or something else:

```bash
azd rest doctor
```

It checks scope detection, acquires a token for the management API, and decodes
the token's tenant and expiry. Add `--format json` for machine-readable output.
The command exits non-zero if any check fails, so you can gate scripts on it.

## Examples

```bash
# List Azure subscriptions
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01

# Count resources by type with Resource Graph
azd rest graph "Resources | summarize count() by type"

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

# Correlate a call for an Azure support request
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --client-request-id my-trace-001

# Preview the detected scope without sending a request
azd rest scope https://management.azure.com/subscriptions?api-version=2020-01-01

# Custom headers + save response
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --header "Accept: application/json" --output-file subscriptions.json

# Verbose output with timing details
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --verbose

# Table output for arrays and ARM value[] responses
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --format table

# Newline-delimited JSON (one object per line) for piping to jq -c
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --format jsonl

# Custom scope for non-Azure endpoint
azd rest get https://api.myservice.com/data --scope https://myservice.com/.default

# Paginate through results
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --paginate

# Show the signed-in Azure identity
azd rest whoami

# Cap the whole call, including retries and pagination
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --paginate --max-time 20s

# Diagnose authentication issues
azd rest doctor
```
