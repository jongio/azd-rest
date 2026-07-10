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
# Add the extension registry
azd extension source add -n jongio -t url -l https://jongio.github.io/azd-extensions/registry.json

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
| `scope` | Preview the detected OAuth scope and auth mode for a URL |
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
| `--api-version` | | string | "" | Set or replace the `api-version` query parameter. |
| `--client-request-id` | | string | "" | Set the `x-ms-client-request-id` header for Azure request correlation. Pass the flag without a value to generate a random ID. |
| `--url-param` | | string[] | [] | Set or append a URL query parameter (repeatable, format: `key=value`). |

### Request Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--header` | `-H` | string[] | [] | Custom headers (repeatable, format: `Key:Value`). Can be used multiple times. |
| `--header-file` | | string | "" | Read headers from a file (one `Key: Value` per line; blank lines and `#` comments ignored). `-H` overrides on conflict. |
| `--data` | `-d` | string | "" | Request body (JSON string). |
| `--data-file` | | string | "" | Read request body from file. Also accepts `@{file}` shorthand. |
| `--json-field` | | string[] | [] | Add a string field to a JSON request body (repeatable, format: `key=value`). Dotted keys nest. |
| `--json-field-raw` | | string[] | [] | Add a raw JSON field to a JSON request body (repeatable, format: `key:=json`). Dotted keys nest. |
| `--timeout` | `-t` | duration | 30s | Request timeout for a single attempt. Examples: `30s`, `5m`, `1h`. |
| `--max-time` | | duration | 0 | Overall time budget across retries and pagination. `0` disables the limit. |
| `--insecure` | `-k` | bool | false | Skip TLS certificate verification (not recommended for production). |
| `--query` | `-q` | string | "" | JMESPath query to apply to JSON responses. |

### Response Configuration

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | `-f` | string | auto | Output format: `auto` (pretty JSON), `json` (compact JSON), `raw` (raw response), `table`, `jsonl` (one object per line), `yaml`, `csv`. |
| `--output-file` | | string | "" | Write response to file (raw for binary content). |
| `--redact` | | string[] | [] | Mask a JSON response field before output (repeatable, dotted path, `*` matches array elements). |
| `--binary` | | bool | false | Stream request/response as binary without transformation. |
| `--include` | `-i` | bool | false | Include the HTTP status line and response headers in the output (curl `-i` style). Sensitive header values are redacted. |
| `--verbose` | `-v` | bool | false | Verbose output (show headers, timing, request details). |
| `--silent` | | bool | false | Suppress non-error diagnostic messages on stderr (warnings and notices). Errors and response output are unaffected. |

### Advanced Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--paginate` | bool | false | Follow continuation tokens/next links when supported. |
| `--retry` | int | 3 | Retry attempts with exponential backoff for transient errors. |
| `--repeat` | int | 1 | Send the request N times and report latency statistics. |
| `--follow-redirects` | bool | true | Follow HTTP redirects. |
| `--max-redirects` | int | 10 | Maximum redirect hops. |
| `--allow-host` | stringArray | [] | Restrict requests to hosts matching a pattern (repeatable; leading `*.` matches subdomains). See [Restricting Request Hosts](#restricting-request-hosts). |
| `--repeat-delay` | duration | 0s | Wait between repeated requests when `--repeat` is greater than 1. |

### Environment Variable Defaults

Every global flag can take its default from an environment variable. This lets you set an option once for a shell session or CI job instead of repeating it on every call.

The variable name is the flag name upper-cased, with dashes replaced by underscores, and prefixed with `AZD_REST_`:

| Flag | Environment variable |
|------|----------------------|
| `--scope` | `AZD_REST_SCOPE` |
| `--api-version` | `AZD_REST_API_VERSION` |
| `--timeout` | `AZD_REST_TIMEOUT` |
| `--retry` | `AZD_REST_RETRY` |
| `--repeat` | `AZD_REST_REPEAT` |
| `--repeat-delay` | `AZD_REST_REPEAT_DELAY` |
| `--format` | `AZD_REST_FORMAT` |
| `--max-response-size` | `AZD_REST_MAX_RESPONSE_SIZE` |

Precedence is command line over environment over built-in default. A value passed on the command line always wins; an environment value is used only when the flag is not passed. An invalid value (for example `AZD_REST_RETRY=abc`) exits with code 2 and makes no request.

The repeatable `--allow-host` flag reads its default from `AZD_REST_ALLOWED_HOSTS`, a comma separated list of host patterns (for example `management.azure.com,*.vault.azure.net`). Blank entries are ignored. This is the one flag whose variable name is not the generic upper-cased mapping, because the value is a list rather than a single value.

```bash
export AZD_REST_RETRY=5
export AZD_REST_TIMEOUT=60s

# Both calls use retry=5, timeout=60s without repeating the flags
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
azd rest get https://management.azure.com/tenants?api-version=2020-01-01

# Command line still overrides the environment
azd rest get https://api.example.com/data --retry 1
```

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

## `azd rest scope <url>`

Preview how `azd rest` would authenticate a request to a URL without sending it. The command reports the resolved authentication mode, the OAuth scope, and the matched Azure service when known. It makes no network call, so it is safe to run against any URL.

Scope honors the same flags the request pipeline uses: `--scope` overrides the detected scope, `--no-auth` and a `-H "Authorization: ..."` header both report an unauthenticated request, and an `http://` URL reports that authentication is skipped.

**Usage:**
```bash
azd rest scope <url> [flags]
```

**Examples:**
```bash
# Preview the scope for a Management API URL
azd rest scope https://management.azure.com/subscriptions?api-version=2020-01-01

# See the effect of --no-auth
azd rest scope https://api.github.com/repos/Azure/azure-dev --no-auth

# Machine-readable output
azd rest scope https://graph.microsoft.com/v1.0/me --format json
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--scope` | `-s` | string | (auto) | Override the OAuth scope reported for the URL |
| `--no-auth` | | bool | false | Report the request as unauthenticated |
| `--header` | `-H` | string | | Headers used to evaluate auth skip (repeatable) |
| `--format` | `-f` | string | auto | Output format: `auto` or `json` |

**Output Examples:**

**Default:**
```
URL:      https://management.azure.com/subscriptions?api-version=2020-01-01
Auth:     bearer
Scope:    https://management.azure.com/.default
Service:  Azure Resource Manager
```

**JSON:**
```json
{
  "url": "https://graph.microsoft.com/v1.0/me",
  "authMode": "bearer",
  "scope": "https://graph.microsoft.com/.default",
  "service": "Microsoft Graph"
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

### API Version Helper

Use `--api-version` to add or replace the Azure `api-version` query parameter:

```bash
azd rest get https://management.azure.com/subscriptions --api-version 2020-01-01

azd rest get https://management.azure.com/subscriptions?api-version=2019-01-01 \
  --api-version 2020-01-01
```

### URL Query Parameters

Use `--url-param key=value` to set or append query parameters without hand-encoding them into the URL. The flag is repeatable. The first use of a key replaces any existing value on the URL, and repeating the same key appends another value:

```bash
# Add query parameters
azd rest get https://management.azure.com/subscriptions \
  --url-param api-version=2020-01-01 --url-param '$top=10'

# Replace an existing value
azd rest get "https://api.example.com/items?filter=all" --url-param filter=active

# Repeat a key to send multiple values
azd rest get https://api.example.com/items --url-param tag=a --url-param tag=b
```

### No Authentication

For public APIs that don't require authentication, use `--no-auth`:

```bash
azd rest get https://api.github.com/repos/Azure/azure-dev --no-auth
```

### Client Request ID

Azure support engineers often ask for the `x-ms-client-request-id` value to trace a call through the service logs. Use `--client-request-id` to set it, and the value is echoed to stderr so you can copy it into a support ticket:

```bash
# Provide your own correlation ID
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --client-request-id my-trace-001

# Pass the flag without a value to generate a random ID
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --client-request-id
```

The flag takes precedence over an `x-ms-client-request-id` value supplied with `-H`.

### Timeouts and Overall Budget

`--timeout` bounds a single request attempt. `--max-time` bounds the entire operation, including retries and pagination, so a slow endpoint cannot hang a script far past the point you expect:

```bash
# Cap the whole call at 20 seconds, even while paginating
azd rest get https://management.azure.com/subscriptions/{sub}/resources?api-version=2021-04-01 \
  --paginate --max-time 20s

# Per-attempt timeout and overall budget together
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --timeout 5s --max-time 30s
```

The two flags are independent. `--timeout` still applies to each attempt, while `--max-time` is the ceiling for the whole run. A value of `0` (the default) means no overall limit. Exceeding the budget cancels in-flight work and returns a timeout error with a non-zero exit code.

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

### JSON Body Fields

Use `--json-field` and `--json-field-raw` to build a JSON body from `key=value` pairs instead of writing raw JSON. `--json-field` sets a string value, and `--json-field-raw` parses the value as JSON so numbers, booleans, arrays, objects, and null keep their type. Dotted keys build nested objects, and repeated prefixes merge into the same parent object. `Content-Type: application/json` is set when you do not provide one:

```bash
azd rest post https://api.example.com/resource \
  --json-field name=example \
  --json-field-raw enabled:=true \
  --json-field-raw retries:=3 \
  --json-field sku.name=Standard_LRS
```

This sends `{"name":"example","enabled":true,"retries":3,"sku":{"name":"Standard_LRS"}}`. These flags cannot be combined with `--data`, `--data-file`, or `--form-field`.

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

### YAML Output

Use `--format yaml` to render a JSON response as YAML with two-space indentation and stable key order. Arrays and ARM `value` wrapper responses render as a YAML sequence of rows, and a single resource renders as a mapping:

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --format yaml
```

### CSV

Use `--format csv` to export arrays and ARM `value[]` responses as RFC 4180 CSV with a header row. Column order matches `--format table`, nested values are written as compact JSON, and scalar lists use a single `value` column:

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --format csv
```

### Query JSON Responses

Use `--query` to select data from JSON responses with JMESPath:

```bash
# Return subscription display names
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --query "value[].displayName"

# Return the first item
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --query "value[0]"
```

### Binary Content

Use `--binary` flag to handle binary content without transformation:

```bash
azd rest get https://example.com/image.png --binary --output-file image.png
```

### Redacting Response Fields

Use `--redact` to replace sensitive JSON values with a fixed placeholder before the response is printed or written to `--output-file`. The flag is repeatable and uses dotted paths, where `*` matches every element of an array:

```bash
# Mask the value of a Key Vault secret
azd rest get "https://myvault.vault.azure.net/secrets/db?api-version=7.4" --redact value

# Mask a field inside every item of an ARM list response
azd rest get "https://management.azure.com/subscriptions/.../providers/...?api-version=2023-01-01" \
  --redact value.*.properties.connectionString
```

Redaction runs for the `json`, `auto`, `table`, and `jsonl` formats. Raw and binary output is left unchanged, with a note on stderr, because it cannot be parsed as JSON. A path that matches nothing is a safe no-op.

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

## Include Response Headers

Use `--include` (or `-i`) to prepend the HTTP status line and response headers to the output, similar to `curl -i`. The header block is written to stdout ahead of the body, or to `--output-file` when that flag is set:

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --include
```

**Example:**
```
200 OK
Content-Length: 1234
Content-Type: application/json
x-ms-request-id: 6f1c...

{
  "value": [...]
}
```

Sensitive header values (for example `Authorization` and cookies) are redacted. Unlike `--verbose`, which writes request diagnostics and timing to stderr, `--include` writes only the status line and response headers alongside the body on stdout, which is convenient for scripts that need a header such as `Location`, `ETag`, or `x-ms-request-id`. `--include` works with the `auto`, `json`, and `raw` formats and with binary responses.

## Silent Mode

Use `--silent` to suppress non-error diagnostic messages that `azd rest` writes to stderr. This covers the insecure TLS warning, the "no scope found" warning, and the pagination notice. Errors, exit codes, and the response body on stdout are unaffected, so you never lose a genuine failure by silencing diagnostics.

```bash
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --silent
```

This is useful in CI logs and scripts where advisory output is noise. Unlike redirecting stderr, `--silent` keeps real error messages visible.

```bash
# Quiet output in a pipeline, errors still surface
azd rest get https://api.example.com/data --insecure --silent > data.json
```

## Restricting Request Hosts

Use `--allow-host` to restrict which hosts `azd rest` will call. When one or more patterns are set, the request host must match at least one pattern before any access token is acquired or any request is sent. A disallowed host fails fast with a non-zero exit code and never triggers authentication, which keeps a mistyped or unexpected host from receiving a bearer token.

```bash
# Only allow the ARM control plane and any Key Vault data-plane host
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --allow-host management.azure.com \
  --allow-host "*.vault.azure.net"
```

Matching rules:

- Matching is case insensitive and the port is ignored.
- A pattern that begins with `*.` matches any subdomain of the remaining suffix. For example `*.vault.azure.net` matches `kv.vault.azure.net` but not the bare `vault.azure.net`.
- Any other pattern must match the host exactly.

The flag is repeatable, and `AZD_REST_ALLOWED_HOSTS` supplies a comma separated default for shells and CI jobs:

```bash
export AZD_REST_ALLOWED_HOSTS="management.azure.com,*.vault.azure.net"
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
```

When `--allow-host` is combined with `--follow-redirects`, only the initial request host is checked. Redirect targets are still bounded by `--max-redirects` but are not matched against the allowlist, so a notice is printed to stderr unless `--silent` is set.

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

### Headers from a File

Keep a reusable header set in a file and load it with `--header-file`. Use one `Key: Value` per line. Blank lines and lines that start with `#` are ignored:

```bash
# headers.txt
# Shared headers for the widgets API
Accept: application/json
X-Api-Version: 2

azd rest get https://api.example.com/widgets --header-file headers.txt
```

Inline `--header` values take precedence, so you can load a base set from a file and override a single entry on the command line:

```bash
azd rest get https://api.example.com/widgets \
  --header-file headers.txt \
  --header "Accept: application/xml"
```

A missing file or a malformed line (one without a colon) returns a clear error and a non-zero exit code.

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

# Pace repeated requests
azd rest get https://api.example.com/resource --repeat 3 --repeat-delay 2s

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
# 1. Add registry
azd extension source add -n jongio -t url -l https://jongio.github.io/azd-extensions/registry.json

# 2. Install extension
azd extension install jongio.azd.rest

# 3. Verify
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
