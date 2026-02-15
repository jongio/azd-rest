---
title: CLI Reference
description: Complete command reference for azd rest extension
lastUpdated: 2026-01-09
tags: [cli, reference, documentation, commands]
---

# CLI Reference

Complete reference for the `azd rest` extension commands and flags.

## Overview

The `azd rest` extension allows you to execute REST API calls with automatic Azure authentication. The extension intelligently detects Azure service endpoints and applies the correct OAuth scopes.

## Installation

```bash
# Enable azd extensions
azd config set alpha.extension.enabled on

# Add the extension registry
azd extension source add -n azd-rest -t url -l https://raw.githubusercontent.com/jongio/azd-rest/main/registry.json

# Install the extension
azd extension install jongio.azd.rest

# Verify installation
azd rest version
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `get` | Execute a GET request |
| `post` | Execute a POST request |
| `put` | Execute a PUT request |
| `patch` | Execute a PATCH request |
| `delete` | Execute a DELETE request |
| `head` | Execute a HEAD request |
| `options` | Execute an OPTIONS request |
| `version` | Display the extension version |

---

## HTTP Method Commands

All HTTP method commands follow the same pattern and support the same global flags.

### `azd rest get <url>`

Execute a GET request to the specified URL.

**Usage:**
```bash
azd rest get <url> [flags]
```

**Examples:**
```bash
# Simple GET request
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01

# GET with custom headers
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --header "Accept: application/json"

# GET with verbose output
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --verbose

# GET without authentication
azd rest get https://api.github.com/repos/Azure/azure-dev --no-auth
```

### `azd rest post <url>`

Execute a POST request with a request body.

**Usage:**
```bash
azd rest post <url> [flags]
```

**Examples:**
```bash
# POST with JSON body
azd rest post https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"location":"eastus","kind":"StorageV2","sku":{"name":"Standard_LRS"}}'

# POST with body from file
azd rest post https://api.example.com/resource --data-file request.json

# POST with @ shorthand for file
azd rest post https://api.example.com/resource --data-file @request.json
```

### `azd rest put <url>`

Execute a PUT request (typically for full resource updates).

**Usage:**
```bash
azd rest put <url> [flags]
```

**Examples:**
```bash
azd rest put https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"location":"eastus","kind":"StorageV2"}'
```

### `azd rest patch <url>`

Execute a PATCH request (typically for partial resource updates).

**Usage:**
```bash
azd rest patch <url> [flags]
```

**Examples:**
```bash
azd rest patch https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"tags":{"environment":"production"}}'
```

### `azd rest delete <url>`

Execute a DELETE request.

**Usage:**
```bash
azd rest delete <url> [flags]
```

**Examples:**
```bash
azd rest delete https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01
```

### `azd rest head <url>`

Execute a HEAD request (retrieves headers without body).

**Usage:**
```bash
azd rest head <url> [flags]
```

**Examples:**
```bash
azd rest head https://management.azure.com/subscriptions?api-version=2020-01-01
```

### `azd rest options <url>`

Execute an OPTIONS request (retrieves allowed methods).

**Usage:**
```bash
azd rest options <url> [flags]
```

**Examples:**
```bash
azd rest options https://api.example.com/resource
```

---

## Global Flags

These flags are available for all HTTP method commands:

### Authentication

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--scope` | `-s` | string | (auto-detected) | OAuth scope for authentication. Auto-detected for Azure services if not provided. |
| `--no-auth` | | bool | false | Skip authentication (no bearer token). Useful for public APIs. |

### Request Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--header` | `-H` | string[] | [] | Custom headers (repeatable, format: `Key:Value`). Can be used multiple times. |
| `--data` | `-d` | string | "" | Request body (JSON string). |
| `--data-file` | | string | "" | Read request body from file. Also accepts `@{file}` shorthand. |
| `--timeout` | `-t` | duration | 30s | Request timeout. Examples: `30s`, `5m`, `1h`. |
| `--insecure` | `-k` | bool | false | Skip TLS certificate verification (not recommended for production). |

### Response Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | `-f` | string | auto | Output format: `auto` (pretty JSON), `json` (compact JSON), `raw` (raw response). |
| `--output-file` | | string | "" | Write response to file (raw for binary content). |
| `--binary` | | bool | false | Stream request/response as binary without transformation. |
| `--verbose` | `-v` | bool | false | Verbose output (show headers, timing, request details). |

### Advanced Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--paginate` | bool | false | Follow continuation tokens/next links when supported. |
| `--retry` | int | 3 | Retry attempts with exponential backoff for transient errors. |
| `--follow-redirects` | bool | true | Follow HTTP redirects. |
| `--max-redirects` | int | 10 | Maximum redirect hops. |

---

## `azd rest version`

Display the extension version information.

**Usage:**
```bash
azd rest version [flags]
```

**Examples:**
```bash
# Show version with details
azd rest version

# Show only version number
azd rest version --quiet

# JSON output
azd rest version --format json
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--quiet` | `-q` | bool | false | Display only the version number |
| `--format` | `-f` | string | auto | Output format: `auto` or `json` |

**Output Examples:**

**Default:**
```
azd rest
Version: 0.1.0
Build Date: 2026-01-09T10:30:45Z
Git Commit: abc123def
```

**Quiet:**
```
0.1.0
```

**JSON:**
```json
{
  "version": "0.1.0",
  "buildDate": "2026-01-09T10:30:45Z",
  "gitCommit": "abc123def"
}
```

---

## Scope Detection

`azd rest` automatically detects the appropriate OAuth scope for Azure services based on the URL hostname. This eliminates the need to manually specify scopes for most Azure API calls.

### Supported Azure Services

The following Azure services have automatic scope detection:

| Service | Hostname Pattern | Scope |
|---------|------------------|-------|
| Azure Management API | `management.azure.com` | `https://management.azure.com/.default` |
| Microsoft Graph | `graph.microsoft.com` | `https://graph.microsoft.com/.default` |
| Azure Key Vault | `*.vault.azure.net` | `https://vault.azure.net/.default` |
| Azure Storage | `*.blob.core.windows.net`<br>`*.queue.core.windows.net`<br>`*.table.core.windows.net`<br>`*.file.core.windows.net`<br>`*.dfs.core.windows.net` | `https://storage.azure.com/.default` |
| Azure Container Registry | `*.azurecr.io` | `https://containerregistry.azure.net/.default` |
| Azure Cosmos DB | `*.documents.azure.com` | `https://cosmos.azure.com/.default` |
| Azure App Configuration | `*.azconfig.io` | `https://azconfig.io/.default` |
| Azure Batch | `*.batch.azure.com` | `https://batch.core.windows.net/.default` |
| Azure Database (PostgreSQL/MySQL/MariaDB) | `*.postgres.database.azure.com`<br>`*.mysql.database.azure.com`<br>`*.mariadb.database.azure.com` | `https://ossrdbms-aad.database.windows.net/.default` |
| Azure SQL Database | `*.database.windows.net` | `https://database.windows.net/.default` |
| Azure Synapse | `*.dev.azuresynapse.net` | `https://dev.azuresynapse.net/.default` |
| Azure Data Lake | `*.azuredatalakestore.net` | `https://datalake.azure.net/.default` |
| Azure Media Services | `*.media.azure.net` | `https://rest.media.azure.net/.default` |
| Azure Log Analytics | `api.loganalytics.io` | `https://api.loganalytics.io/.default` |
| Azure DevOps | `dev.azure.com`<br>`*.visualstudio.com` | `499b84ac-1321-427f-aa17-267ca6975798/.default` |
| Azure Kusto | `*.kusto.windows.net` | `https://{hostname}/.default` |
| Azure Service Bus | `*.servicebus.windows.net` (queues) | `https://servicebus.azure.net/.default` |
| Azure Event Hubs | `*.servicebus.windows.net` (event hubs) | `https://eventhubs.azure.net/.default` |

### Custom Scopes

For non-Azure endpoints or when you need a specific scope, use the `--scope` flag:

```bash
# Custom scope for non-Azure endpoint
azd rest get https://api.myservice.com/data --scope https://myservice.com/.default

# Override auto-detected scope
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --scope https://management.azure.com/.default
```

### No Authentication

For public APIs that don't require authentication, use `--no-auth`:

```bash
azd rest get https://api.github.com/repos/Azure/azure-dev --no-auth
```

---

## Request Body

### JSON String

Use the `--data` flag to provide a JSON string directly:

```bash
azd rest post https://api.example.com/resource \
  --data '{"name":"example","value":123}'
```

### From File

Use the `--data-file` flag to read the request body from a file:

```bash
# Standard file path
azd rest post https://api.example.com/resource --data-file request.json

# @ shorthand (also supported)
azd rest post https://api.example.com/resource --data-file @request.json
```

**File Format:**
- The file is read as-is (raw bytes)
- For JSON, ensure the file contains valid JSON
- Binary files are supported when using `--binary` flag

---

## Response Formatting

### Auto Format (Default)

Automatically pretty-prints JSON responses:

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
```

**Output:**
```json
{
  "value": [
    {
      "id": "/subscriptions/...",
      "subscriptionId": "...",
      "displayName": "My Subscription"
    }
  ]
}
```

### Compact JSON

Use `--format json` for compact JSON (no pretty-printing):

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --format json
```

### Raw Output

Use `--format raw` for raw response (no JSON parsing):

```bash
azd rest get https://api.example.com/data --format raw
```

### Binary Content

Use `--binary` flag to handle binary content without transformation:

```bash
azd rest get https://example.com/image.png --binary --output-file image.png
```

### Save to File

Use `--output-file` to save the response to a file:

```bash
# Save JSON response
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --output-file subscriptions.json

# Save binary content
azd rest get https://example.com/image.png --binary --output-file image.png
```

---

## Verbose Output

Use `--verbose` to see detailed request and response information:

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --verbose
```

**Output includes:**
- Request method and URL
- Request headers (with token redacted)
- Response status code
- Response headers
- Request timing
- Response size

**Example:**
```
> GET https://management.azure.com/subscriptions?api-version=2020-01-01
> Authorization: Bearer ***REDACTED***
> Accept: application/json
> 
< 200 OK
< Content-Type: application/json
< Content-Length: 1234
< 
Request completed in 234ms
{
  "value": [...]
}
```

---

## Headers

### Custom Headers

Add custom headers using the `--header` flag (can be used multiple times):

```bash
azd rest get https://api.example.com/resource \
  --header "X-Custom-Header: value" \
  --header "Accept: application/json"
```

**Format:** `Key:Value` (colon separates key from value)

### Content-Type

When using `--data` or `--data-file`, `Content-Type: application/json` is automatically set. Override with `--header`:

```bash
azd rest post https://api.example.com/resource \
  --data '{"key":"value"}' \
  --header "Content-Type: application/xml"
```

---

## Redirects

By default, `azd rest` follows HTTP redirects (3xx status codes) up to 10 redirects.

**Control redirect behavior:**

```bash
# Disable redirects
azd rest get https://example.com --follow-redirects=false

# Limit redirect hops
azd rest get https://example.com --max-redirects=5
```

---

## Timeouts

Set request timeout using the `--timeout` flag:

```bash
# 30 seconds (default)
azd rest get https://api.example.com/resource --timeout 30s

# 5 minutes
azd rest get https://api.example.com/resource --timeout 5m

# 1 hour
azd rest get https://api.example.com/resource --timeout 1h
```

**Supported units:** `s` (seconds), `m` (minutes), `h` (hours)

---

## Retries

`azd rest` automatically retries failed requests with exponential backoff for transient errors (5xx, network errors).

**Configure retries:**

```bash
# Default: 3 retries
azd rest get https://api.example.com/resource

# Custom retry count
azd rest get https://api.example.com/resource --retry 5

# Disable retries
azd rest get https://api.example.com/resource --retry 0
```

---

## TLS Verification

By default, `azd rest` verifies TLS certificates. Disable verification (not recommended for production):

```bash
azd rest get https://api.example.com/resource --insecure
```

**Warning:** This makes requests vulnerable to man-in-the-middle attacks. Only use for testing or internal networks.

---

## Proxy Support

`azd rest` automatically detects and uses proxy settings from environment variables:

- `HTTP_PROXY` or `http_proxy` - HTTP proxy URL
- `HTTPS_PROXY` or `https_proxy` - HTTPS proxy URL
- `NO_PROXY` or `no_proxy` - Comma-separated list of hosts to bypass proxy

**Example:**
```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1

azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Request failed (HTTP error, network error, etc.) |
| 2 | Invalid arguments or configuration |

---

## Common Workflows

### First Time Setup

```bash
# 1. Enable extensions
azd config set alpha.extension.enabled on

# 2. Add registry
azd extension source add -n azd-rest -t url -l https://raw.githubusercontent.com/jongio/azd-rest/main/registry.json

# 3. Install extension
azd extension install jongio.azd.rest

# 4. Verify
azd rest version
```

### List Azure Subscriptions

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
```

### Create Azure Resource

```bash
azd rest post https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"location":"eastus","kind":"StorageV2","sku":{"name":"Standard_LRS"}}'
```

### Query Key Vault Secret

```bash
azd rest get https://myvault.vault.azure.net/secrets/mysecret?api-version=7.4
```

### Microsoft Graph API

```bash
azd rest get https://graph.microsoft.com/v1.0/me
```

### Custom API with Scope

```bash
azd rest get https://api.myservice.com/data \
  --scope https://myservice.com/.default
```

### Save Response to File

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --output-file subscriptions.json
```

---

## Troubleshooting

### Authentication Errors

**Error:** `failed to get token: credential unavailable`

**Solution:**
```bash
# Ensure you're logged in to Azure
az login

# Or use service principal
export AZURE_CLIENT_ID="..."
export AZURE_CLIENT_SECRET="..."
export AZURE_TENANT_ID="..."
```

### Scope Detection Issues

**Error:** `Warning: Azure host detected but no scope found`

**Solution:**
```bash
# Manually specify scope
azd rest get https://management.azure.com/... --scope https://management.azure.com/.default
```

### Network Errors

**Error:** `timeout` or `connection refused`

**Solution:**
```bash
# Increase timeout
azd rest get https://api.example.com/resource --timeout 5m

# Check proxy settings
echo $HTTP_PROXY
echo $HTTPS_PROXY
```

### Invalid JSON

**Error:** `failed to format response: invalid JSON`

**Solution:**
```bash
# Use raw format to see actual response
azd rest get https://api.example.com/resource --format raw
```

---

## Related Documentation

- [Security Review](./security-review.md) - Detailed security analysis and best practices
- [Threat Model](./threat-model.md) - Security threat analysis and mitigations
- [Main README](../../README.md) - Getting started guide and examples
- [Azure REST API Documentation](https://learn.microsoft.com/rest/api/azure/) - Azure REST API reference

---

## Contributing

For development and contribution guidelines, see [CONTRIBUTING.md](../../CONTRIBUTING.md).

---

## Support

- [Report Issues](https://github.com/jongio/azd-rest/issues)
- [View Source](https://github.com/jongio/azd-rest)
- [Release Notes](https://github.com/jongio/azd-rest/releases)
