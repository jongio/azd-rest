---
title: azd-rest Specification
created: 2026-01-15
updated: 2026-01-15
status: active
type: feature
category: documentation
author: azd-rest team
percentComplete: 0
related: []
tags: [rest, azure, cli, specification]
---

# azd-rest: REST API Extension with Azure Authentication

**Version**: 1.0  
**Status**: Planning  
**Last Updated**: 2026-01-15

## Summary

`azd-rest` is an Azure Developer CLI extension that enables developers to execute REST API calls with automatic Azure authentication. The extension intelligently detects Azure service endpoints and applies the correct OAuth scopes, making it effortless to interact with Azure Management API, Storage, Key Vault, and any other REST endpoint.

## Motivation

Developers frequently need to interact with Azure REST APIs for testing, debugging, and automation. Current challenges include:

- **Authentication complexity**: Managing bearer tokens manually is error-prone
- **Scope management**: Different Azure services require different OAuth scopes (e.g., `https://management.azure.com/.default` for ARM, `https://storage.azure.com/.default` for Storage)
- **Context switching**: Copying tokens from `az account get-access-token` or maintaining separate authentication scripts
- **azd integration**: No native way to leverage azd's authentication context for REST calls

`azd-rest` solves these problems by providing a simple interface: `azd rest [url]` that handles authentication automatically.

## Objectives

- ✅ **Simple interface**: `azd rest [url]` for all REST calls
- ✅ **Automatic Azure authentication**: Leverage azd's authentication context (shared with azd login)
- ✅ **Intelligent scope detection**: Automatically determine correct OAuth scope based on endpoint URL
- ✅ **Comprehensive Azure service support**: Management API, Storage, Key Vault, Graph, and all major Azure services
- ✅ **Custom scope support**: Allow manual scope override for edge cases or non-Azure endpoints
- ✅ **Full HTTP method support**: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- ✅ **Developer-friendly**: Request/response bodies, custom headers, JSON formatting, verbose output
- ✅ **Non-Azure endpoints**: Support any REST endpoint with optional bearer token
- ✅ **Production-ready**: Comprehensive testing (unit + integration), CI/CD, security scanning, 80%+ coverage
- ✅ **Consistent with azd-exec**: Follow established patterns, release workflows, and documentation standards

## Non-Goals

- **Not a replacement for `curl`**: For complex HTTP scenarios (multipart uploads, websockets), use specialized tools
- **Not a full API client**: We provide REST execution, not service-specific SDKs or abstractions
- **Not session management**: Each request is independent; no cookie/session handling
- **Not GraphQL**: REST endpoints only (though GraphQL over HTTP POST would work)
- **Not a browser**: No JavaScript execution, HTML rendering, or DOM interaction

## Technical Architecture

### Component Structure

Based on `azd-exec` patterns:

```
azd-rest/
├── cli/
│   ├── src/
│   │   ├── cmd/
│   │   │   └── rest/
│   │   │       └── main.go              # Entry point
│   │   └── internal/
│   │       ├── cmd/
│   │       │   ├── root.go              # Root command + flags
│   │       │   ├── get.go               # GET command
│   │       │   ├── post.go              # POST command
│   │       │   ├── put.go               # PUT command
│   │       │   ├── patch.go             # PATCH command
│   │       │   ├── delete.go            # DELETE command
│   │       │   ├── head.go              # HEAD command
│   │       │   ├── options.go           # OPTIONS command
│   │       │   └── version.go           # Version command
│   │       ├── auth/
│   │       │   ├── auth.go              # Azure authentication via azd-core
│   │       │   ├── scope.go             # Scope detection logic
│   │       │   └── auth_test.go
│   │       ├── client/
│   │       │   ├── client.go            # HTTP client implementation
│   │       │   ├── client_test.go
│   │       │   └── formatter.go         # Response formatting
│   │       └── context/
│   │           └── context.go           # azd environment context
│   ├── tests/
│   │   ├── integration/
│   │   │   ├── auth_test.go             # Integration tests for auth
│   │   │   ├── management_test.go       # ARM API tests
│   │   │   ├── storage_test.go          # Storage API tests
│   │   │   └── keyvault_test.go         # Key Vault API tests
│   │   └── e2e/
│   │       └── e2e_test.go              # End-to-end scenarios
│   ├── extension.yaml                    # Extension metadata
│   ├── go.mod
│   ├── magefile.go                       # Build automation
│   └── build.sh
├── docs/
│   ├── cli-reference.md                  # Command documentation
│   ├── azure-scopes.md                   # Azure scope reference
│   ├── security-review.md                # Security analysis
│   └── examples.md                       # Usage examples
├── .github/
│   └── workflows/
│       ├── ci.yml                        # CI pipeline
│       ├── release.yml                   # Release automation
│       ├── codeql.yml                    # Security scanning
│       └── pr-build.yml                  # PR validation
├── README.md
├── CHANGELOG.md
├── TESTING.md
├── registry.json                         # Extension registry
├── package.json                          # Test orchestration
└── LICENSE
```

### Dependencies

- **azd-core** (target v0.3.0 once helper surface lands; temporary `replace ../azd-core` during development): Authentication, Azure context, shared utilities
- **cobra**: CLI framework (same as azd-exec)
- **Azure SDK for Go**: `azcore`, `azidentity` for authentication
- **Standard library**: `net/http` for REST calls

### Required azd-core Enhancements (v0.3.0 target)

- **Token acquisition helper**: Provide a stable `GetToken(ctx, scope) (azcore.AccessToken, error)` surface that returns the bearer value plus `ExpiresOn` for retry/pagination reuse, backed by azd's credential chain and cache.
- **Context accessors**: Public helpers to retrieve active subscription ID and default resource group (from azd login/context) for ARM placeholder substitution; must no-op gracefully when unset.
- **Cloud endpoint map**: Expose a map of public and sovereign endpoints (management/resource manager host + audience) so extensions do not duplicate constants when inferring scopes.
- **User-Agent/telemetry hook**: Policy/builder to append extension name/version and command metadata to the User-Agent plus telemetry envelope while guaranteeing metadata-only collection (no headers/bodies/tokens).
- **Header redaction utilities**: Shared helpers to redact sensitive headers (Authorization first/last 4 chars) and body snippets for verbose logging so extensions do not reimplement.
- **Version contract and guardrails**: azd-rest pins to `github.com/jongio/azd-core v0.3.0` (with local `replace` to `../../azd-core` until the release is cut) and enforces the pin via `cli/src/internal/azdcore/version_test.go`; CI should fail when go.mod drifts.
- **Tracking item**: Create an azd-core issue titled “Expose extension helper surface for azd-rest (token w/ expiry, context accessors, endpoints, UA/telemetry hook, redaction)” containing the above acceptance criteria and the planned version pin to v0.3.0.

### azd-core tracking issue (draft)

File this in the azd-core repository (github.com/jongio/azd-core) with the exact title **“Expose extension helper surface for azd-rest (token w/ expiry, context accessors, endpoints, UA/telemetry hook, redaction)”** and body:

```
## Summary
Expose helper surface needed by the azd-rest extension so it can rely on azd-core for tokens, context, endpoints, telemetry, and redaction instead of duplicating logic. Target release: azd-core v0.3.0 (azd-rest pins to this version with a guardrail test).

## Acceptance Criteria
- [ ] Token helper: public function that returns `azcore.AccessToken` with `ExpiresOn` for a given scope using azd credential chain (cache respected, retries for transient failures).
- [ ] Context accessors: helpers to fetch active subscription ID and default resource group from azd context; safe no-op/default behavior when unset.
- [ ] Cloud endpoint map: exported map for public and sovereign clouds that includes resource manager host + audience values (used for scope and placeholder substitution) so extensions do not hardcode constants.
- [ ] User-Agent/telemetry hook: policy/builder to append extension name/version and command metadata to UA + telemetry envelope; metadata-only collection (no tokens/headers/bodies), opt-in friendly.
- [ ] Redaction utilities: helpers to redact sensitive headers (Authorization first/last 4 chars) and body snippets for verbose logging; reusable across extensions.
- [ ] Version pin: deliver in v0.3.0; azd-rest will stay pinned to `github.com/jongio/azd-core v0.3.0` and CI enforces the pin.

## Notes
- Please link the azd-rest spec section “Required azd-core Enhancements (v0.3.0 target)” for context.
- azd-rest currently uses a local replace to the azd-core repo during development.
```

### Azure Scope Detection

**Core Logic**: Analyze URL pattern to determine required OAuth scope.

#### Scope Mapping

| Service | URL Pattern | OAuth Scope | Example |
|---------|------------|-------------|---------|
| **Management API** | `management.azure.com` | `https://management.azure.com/.default` | `https://management.azure.com/subscriptions?api-version=2020-01-01` |
| **Storage (Data Plane)** | `*.blob.core.windows.net`<br>`*.queue.core.windows.net`<br>`*.table.core.windows.net`<br>`*.file.core.windows.net`<br>`*.dfs.core.windows.net` | `https://storage.azure.com/.default` | `https://mystorageacct.blob.core.windows.net/container/blob` |
| **Key Vault** | `*.vault.azure.net` | `https://vault.azure.net/.default` | `https://myvault.vault.azure.net/secrets/my-secret?api-version=7.4` |
| **Microsoft Graph** | `graph.microsoft.com` | `https://graph.microsoft.com/.default` | `https://graph.microsoft.com/v1.0/me` |
| **Azure DevOps** | `dev.azure.com`<br>`*.visualstudio.com` | `499b84ac-1321-427f-aa17-267ca6975798/.default` | `https://dev.azure.com/{org}/_apis/projects` |
| **Azure Data Explorer** | `*.kusto.windows.net` | `https://{cluster}.{region}.kusto.windows.net/.default` | `https://mycluster.eastus.kusto.windows.net/v1/rest/query` |
| **Container Registry** | `*.azurecr.io` | `https://containerregistry.azure.net/.default` | `https://myregistry.azurecr.io/v2/_catalog` |
| **Event Hubs** | `*.servicebus.windows.net` | `https://eventhubs.azure.net/.default` | `https://mynamespace.servicebus.windows.net/eventhub` |
| **Service Bus** | `*.servicebus.windows.net` | `https://servicebus.azure.net/.default` | `https://mynamespace.servicebus.windows.net/queue` |
| **Cosmos DB** | `*.documents.azure.com` | `https://cosmos.azure.com/.default` | `https://myaccount.documents.azure.com/dbs` |
| **App Configuration** | `*.azconfig.io` | `https://azconfig.io/.default` | `https://myconfig.azconfig.io/kv?api-version=1.0` |
| **Azure Batch** | `*.batch.azure.com` | `https://batch.core.windows.net/.default` | `https://mybatch.eastus.batch.azure.com/pools?api-version=2021-06-01` |
| **OSSRDBMS** | `*.postgres.database.azure.com`<br>`*.mysql.database.azure.com`<br>`*.mariadb.database.azure.com` | `https://ossrdbms-aad.database.windows.net/.default` | `https://myserver.postgres.database.azure.com` |
| **SQL Database** | `*.database.windows.net` | `https://database.windows.net/.default` | `https://myserver.database.windows.net` |
| **Synapse** | `*.dev.azuresynapse.net` | `https://dev.azuresynapse.net/.default` | `https://myworkspace.dev.azuresynapse.net` |
| **Data Lake** | `*.azuredatalakestore.net` | `https://datalake.azure.net/.default` | `https://mydatalake.azuredatalakestore.net/webhdfs/v1` |
| **Media Services** | `*.media.azure.net` | `https://rest.media.azure.net/.default` | `https://myaccount.restv2.eastus.media.azure.net/api/` |
| **Log Analytics** | `api.loganalytics.io` | `https://api.loganalytics.io/.default` | `https://api.loganalytics.io/v1/workspaces/{id}/query` |

#### Scope Detection Algorithm

```go
func DetectScope(url string) (string, error) {
    parsedURL, err := url.Parse(url)
    if err != nil {
        return "", err
    }
    
    host := strings.ToLower(parsedURL.Hostname())
    
    // Exact matches
    exactMatches := map[string]string{
        "management.azure.com":     "https://management.azure.com/.default",
        "graph.microsoft.com":      "https://graph.microsoft.com/.default",
        "api.loganalytics.io":      "https://api.loganalytics.io/.default",
    }
    if scope, ok := exactMatches[host]; ok {
        return scope, nil
    }
    
    // Suffix matches (e.g., *.vault.azure.net)
    suffixMatches := map[string]string{
        ".vault.azure.net":            "https://vault.azure.net/.default",
        ".blob.core.windows.net":      "https://storage.azure.com/.default",
        ".queue.core.windows.net":     "https://storage.azure.com/.default",
        ".table.core.windows.net":     "https://storage.azure.com/.default",
        ".file.core.windows.net":      "https://storage.azure.com/.default",
        ".dfs.core.windows.net":       "https://storage.azure.com/.default",
        ".azurecr.io":                 "https://containerregistry.azure.net/.default",
        ".servicebus.windows.net":     "https://eventhubs.azure.net/.default",  // Default to Event Hubs; override based on path
        ".documents.azure.com":        "https://cosmos.azure.com/.default",
        ".azconfig.io":                "https://azconfig.io/.default",
        ".batch.azure.com":            "https://batch.core.windows.net/.default",
        ".postgres.database.azure.com":"https://ossrdbms-aad.database.windows.net/.default",
        ".mysql.database.azure.com":   "https://ossrdbms-aad.database.windows.net/.default",
        ".mariadb.database.azure.com": "https://ossrdbms-aad.database.windows.net/.default",
        ".database.windows.net":       "https://database.windows.net/.default",
        ".dev.azuresynapse.net":       "https://dev.azuresynapse.net/.default",
        ".azuredatalakestore.net":     "https://datalake.azure.net/.default",
        ".media.azure.net":            "https://rest.media.azure.net/.default",
    }
    
    for suffix, scope := range suffixMatches {
        if strings.HasSuffix(host, suffix) {
            return scope, nil
        }
    }
    
    // Azure DevOps patterns
    if host == "dev.azure.com" || strings.HasSuffix(host, ".visualstudio.com") {
        return "499b84ac-1321-427f-aa17-267ca6975798/.default", nil
    }
    
    // Kusto/Data Explorer (special case: scope includes cluster)
    if strings.HasSuffix(host, ".kusto.windows.net") {
        return fmt.Sprintf("https://%s/.default", host), nil
    }
    
    // Sovereign clouds: allow suffix-based detection on hosts like *.azure.cn, *.usgovcloudapi.net, *.microsoft.scloud
    // Custom ports: ignore port component when matching hosts

    // No Azure scope detected - return empty (user can provide --scope)
    return "", nil
}
```

**Service Bus vs Event Hubs resolution**: default to Event Hubs scope; if path contains `/queue` or `/queues`, use Service Bus scope. Log chosen scope in verbose mode.

**Fallback behavior**: When no scope is detected and `--scope` is not provided, skip auth and warn; if host matches Azure public/sovereign domains but scope is empty, suggest `--scope` override.

**Sovereign clouds**: Treat known sovereign suffixes (`.azure.cn`, `.usgovcloudapi.net`, `.microsoft.scloud`) identically for pattern matching; scope strings remain the same unless service requires regional host.

**Custom ports**: Scope detection ignores port; matching is done on hostname.

### Authentication Flow

1. **Parse URL**: Extract hostname from target URL; require absolute URL with scheme
2. **Detect scope**: Use scope detection algorithm
3. **Decide on auth**: Skip token acquisition when any of the following are true: `--no-auth` flag present, `Authorization` header already provided (case-insensitive), URL scheme is HTTP, or no scope is detected and no override is set (emit warning when host appears Azure).
4. **Get token**: Call `azd-core` auth with detected/custom scope when auth is enabled
5. **Execute request**: Add `Authorization: Bearer {token}` header when token is present
6. **Return response**: Format and display response

```go
// Simplified flow
func ExecuteRequest(method, url string, options RequestOptions) error {
    // 1. Detect scope
    scope := options.CustomScope
    if scope == "" {
        detected, err := DetectScope(url)
        if err != nil {
            return err
        }
        scope = detected
    }
    
    // 2. Decide whether to acquire a token
    var token string
    if options.SkipAuth || options.HasAuthorizationHeader || options.IsHTTP {
        scope = ""
    }

    // 3. Get token (if scope detected or --use-auth=true)
    if scope != "" && options.UseAuth {
        token, err = GetAzureToken(scope)
        if err != nil {
            return err
        }
    }
    
    // 3. Build request
    req, err := http.NewRequest(method, url, options.Body)
    if err != nil {
        return err
    }
    
    // 4. Add authorization header
    if token != "" {
        req.Header.Set("Authorization", "Bearer " + token)
    }
    
    // 5. Add custom headers
    for k, v := range options.Headers {
        req.Header.Set(k, v)
    }
    
    // 6. Execute request
    resp, err := httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 7. Format and display response
    return FormatResponse(resp, options.Output)
}
```

### Request Construction and Input Handling

- **Token substitution**: Replace `{subscriptionId}` and `{resourceGroup}` placeholders from the active azd context before sending requests; fail fast when placeholders are present but values are missing.
- **URI parameters**: `--uri-parameters` accepts key=value pairs or JSON (including `@{file}`) and merges into the query string; preserves existing query parameters.
- **File shorthands**: `@{file}` supported for bodies, headers, and URI parameters; default read is UTF-8, with binary-safe path for `--binary` or when file sniffing detects non-text content.
- **Content-Type detection**: If body parses as JSON and no `Content-Type` is provided, set `application/json`; user-provided header always wins; honor explicit `--binary` to avoid any content-type inference.
- **Telemetry headers**: Every request adds `User-Agent: azd-rest/{version} (azd extension)`, `x-ms-client-request-id: {UUID}`, and `x-ms-command-name: azd rest` for correlation; telemetry remains metadata-only.
- **HTTPS-first**: Enforce HTTPS by default; HTTP schemes are allowed but disable auth automatically and log a warning.

### Response Handling

- **Binary detection**: Inspect `Content-Type` and response bytes to detect non-text; when binary or `--binary` is set, stream directly to `--output-file` or stdout with a safety warning.
- **Output selection**: `--output-file` writes raw bytes without formatting. Otherwise, auto-pretty-print JSON and default to raw text for non-JSON.
- **Pagination**: When `--paginate` is set, follow `nextLink`/`@odata.nextLink`/`Continuation-Token` values, reuse auth headers, and merge `value` arrays while preserving ordering.

## Command-Line Interface

### Request Handling and Safety Defaults

- `@{file}` ingestion for bodies, headers, and URI parameters alongside `--data-file`, preserving UTF-8 defaults while allowing binary payloads.
- `--uri-parameters` (key=value or JSON) with token substitution for ARM placeholders such as `{subscriptionId}` and `{resourceGroup}`; substitution uses the active azd context.
- `--output-file` for direct writes; when content is binary or `--binary` is set, bypass formatting and stream to disk/stdout with a warning before stdout writes.
- Binary passthrough mode to stream request/response bodies without transformation for octet-stream scenarios.
- Authentication skip rules: skip token acquisition when `--no-auth` is set, when an `Authorization` header is already provided (case-insensitive), when the URL uses HTTP, or when no scope is detected and no override is provided; warn if the host matches Azure patterns but auth is skipped.
- Telemetry headers applied on every request: `User-Agent: azd-rest/{version}`, `x-ms-client-request-id` (UUID), and `x-ms-command-name: azd rest`; telemetry remains metadata-only and opt-in per policy.
- No auto-prefixing of relative URLs; full URLs are required to avoid ambiguity.

### Command Structure

```bash
azd rest <url> [flags]

# OR using method as subcommand (preferred)
azd rest get <url> [flags]
azd rest post <url> [flags]
azd rest put <url> [flags]
azd rest patch <url> [flags]
azd rest delete <url> [flags]
azd rest head <url> [flags]
azd rest options <url> [flags]
```

### Global Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--scope` | `-s` | string | (auto-detect) | OAuth scope for authentication |
| `--no-auth` | | bool | false | Skip authentication (no bearer token) |
| `--header` | `-H` | string[] | [] | Custom headers (repeatable) |
| `--data` | `-d` | string | | Request body (JSON string) |
| `--data-file` | | string | | Read request body from file (also accept `@{file}` shorthand) |
| `--uri-parameters` | | string[] | | Key=value pairs substituted into templated URLs (e.g., subscriptionId, resourceGroup) |
| `--output-file` | | string | | Write response to file (raw for binary content) |
| `--output` | `-o` | string | (stdout) | Save response to file |
| `--format` | `-f` | string | `auto` | Output format: `auto`, `json`, `raw` |
| `--verbose` | `-v` | bool | false | Verbose output (show headers, timing) |
| `--paginate` | | bool | false | Follow continuation tokens/next links when supported |
| `--retry` | | int | 3 | Retry attempts with exponential backoff for transient errors |
| `--binary` | | bool | false | Stream request/response as binary without transformation |
| `--insecure` | `-k` | bool | false | Skip TLS certificate verification |
| `--timeout` | `-t` | duration | 30s | Request timeout |
| `--follow-redirects` | | bool | true | Follow HTTP redirects |
| `--max-redirects` | | int | 10 | Maximum redirect hops |

### Usage Examples

```bash
# Simple GET (auto-detects Management API scope)
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01

# Storage API (auto-detects storage scope)
azd rest get https://mystorageacct.blob.core.windows.net/container?restype=container&comp=list

# Key Vault (auto-detects vault scope)
azd rest get https://myvault.vault.azure.net/secrets/my-secret?api-version=7.4

# POST with JSON body
azd rest post https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"location":"eastus","sku":{"name":"Standard_LRS"},"kind":"StorageV2"}'

# POST with file
azd rest post https://api.example.com/resource \
  --data-file ./payload.json \
  --header "Content-Type: application/json"

# Custom scope for non-Azure endpoint
azd rest get https://api.myservice.com/data \
  --scope https://myservice.com/.default

# Non-Azure endpoint without auth
azd rest get https://api.github.com/repos/Azure/azure-dev \
  --no-auth \
  --header "Accept: application/vnd.github.v3+json"

# Verbose output with timing
azd rest get https://management.azure.com/subscriptions \
  --verbose

# Save response to file
azd rest get https://api.example.com/data \
  --output response.json

# Custom headers
azd rest get https://api.example.com/resource \
  --header "X-Custom-Header: value" \
  --header "Accept: application/json"

# DELETE with verbose output
azd rest delete https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --verbose
```

## Security Considerations

### Threat Model and Guardrails

- Assets: tokens, request/response bodies, output files, logs, telemetry records. Threats: token leakage, MITM when `--insecure`, PII in telemetry, disk leaks for binary dumps.
- Enforce token redaction in verbose output (show first/last 4 chars only) and never log bodies unless explicitly requested; gated by `--verbose` with warning.
- `--insecure` requires confirmation (or `--yes`) and prints a clear warning before execution.
- Binary handling: never pretty-print; stream to disk/stdout; avoid buffering large binaries in memory.
- Telemetry: opt-in; collect command metadata only (method, host, status, duration). No bodies/headers/tokens. Document the policy.

### Token Handling

- ✅ **No token exposure**: Tokens never printed to stdout/stderr (only in `--verbose` debug mode with explicit warning)
- ✅ **Secure storage**: Leverage azd-core's token cache (encrypted on disk)
- ✅ **Short-lived tokens**: Azure tokens expire in 1 hour, automatically refreshed
- ✅ **Scope isolation**: Each request gets minimum required scope

### TLS/HTTPS

- ✅ **HTTPS by default**: All Azure endpoints use HTTPS
- ✅ **Certificate validation**: Enabled by default, `--insecure` flag with warning for testing
- ⚠️ **Warning on --insecure**: Display security warning when certificate validation disabled

### Input Validation

- ✅ **URL validation**: Parse and validate URL before request
- ✅ **Header injection prevention**: Sanitize custom headers
- ✅ **Body size limits**: Prevent memory exhaustion from large payloads

### Audit Trail

- ✅ **Request logging**: Optional verbose mode shows full request details
- ✅ **No sensitive data in logs**: Tokens/secrets redacted in verbose output

## Reliability and Performance

- Default timeout 30s; configurable per request; enforce an upper bound (e.g., 5m) to avoid hangs.
- Retries: default 3 with exponential backoff on 429/5xx and network timeouts; disabled in `--binary` mode unless explicitly set.
- Pagination: `--paginate` follows `nextLink`/`@odata.nextLink` or `Continuation-Token` where present; stop on limits or user `--max-pages` (future enhancement).
- Payload limits: configurable max in-memory size (e.g., 32MB) before switching to streaming file writes/reads.
- Streaming: when `--binary` or content-type is non-text, stream to stdout or `--output-file` without transformations.

## Testing Strategy

### Unit Tests (80%+ coverage target)

- **auth/scope_test.go**: Test all scope detection patterns
- **client/client_test.go**: HTTP client behavior, error handling
- **client/formatter_test.go**: Response formatting (JSON, raw)
- **cmd/*_test.go**: Command-line parsing, flag handling

### Integration Tests (require Azure resources)

- **auth_test.go**: Real Azure authentication flow
- **management_test.go**: ARM API calls (list subscriptions)
- **storage_test.go**: Storage API calls (list containers)
- **keyvault_test.go**: Key Vault API calls (list secrets)

### E2E Tests

- **e2e_test.go**: Full command execution scenarios
- Test all HTTP methods (GET, POST, PUT, PATCH, DELETE)
- Test custom scopes, headers, output formats
- Test error scenarios (invalid URL, auth failures)

### CI/CD Pipeline

Based on `azd-exec` workflows:

- **CI** (`ci.yml`): Lint, unit tests, integration tests (Linux/Windows/macOS), security scanning
- **CodeQL** (`codeql.yml`): Security analysis on push to main
- **Release** (`release.yml`): Automated releases with multi-platform binaries
- **PR Build** (`pr-build.yml`): PR validation with all checks

### Current Test Status

**Status**: Not yet implemented (project in planning phase)

| Category | Target Coverage | Current Coverage | Status |
|----------|----------------|------------------|--------|
| Unit Tests | 80%+ | 0% | ❌ Not started |
| Integration Tests | Key scenarios | 0% | ❌ Not started |
| E2E Tests | Critical paths | 0% | ❌ Not started |
| Security Tests | All auth flows | 0% | ❌ Not started |

**Next Steps**:
1. Set up test infrastructure (test files, mocks, fixtures)
2. Implement scope detection unit tests
3. Implement auth flow unit tests
4. Implement HTTP client unit tests
5. Add integration tests for Azure services (requires test resources)
6. Add E2E tests for command execution
7. Configure CI pipeline for automated testing

**Blockers**: None (awaiting implementation start)

## Documentation Requirements

### CLI Documentation

- **cli-reference.md**: Complete command and flag reference
- **azure-scopes.md**: Comprehensive Azure scope mapping guide
- **examples.md**: Real-world usage examples for all major Azure services
- **security-review.md**: Security analysis and best practices

### README.md

- Feature overview with badges (CI, CodeQL, License, Coverage)
- Quick start guide
- Installation instructions
- Common usage patterns
- Security notice
- Contributing guidelines
- Link to detailed docs

### TESTING.md

- Test structure and organization
- How to run tests (unit, integration, e2e)
- Coverage requirements
- Adding new tests

### CHANGELOG.md

- Release history
- Breaking changes
- New features
- Bug fixes

## Release Process

Following `azd-exec` automated release pattern:

- Tooling baseline: Go 1.26.0, azd-core >= v0.3.0; workflows enforce these versions.
- Compatibility: maintain a matrix for supported OS/arch (Windows/Linux/macOS; x64/ARM64) and validate in CI.
- Rollback safety: releases remain installable via previous version in registry.json; document rollback steps in README/TESTING.

1. **Version bump**: Manual workflow trigger with `patch`/`minor`/`major` selection
2. **Automated steps**:
   - Calculate next version
   - Update `extension.yaml` and `CHANGELOG.md`
   - Run full CI (preflight, tests, lint, build)
   - Build multi-platform binaries (Windows/Linux/macOS, x64/ARM64)
   - Package extension (`azd x pack`)
   - Create GitHub release with binaries
   - Update registry.json
   - Commit and push changes

3. **Manual publish** (optional for testing):
   ```bash
   cd cli
   export EXTENSION_ID="jongio.azd.rest"
   export EXTENSION_VERSION="0.1.0"
   azd x build --all
   azd x pack
   azd x release --repo "jongio/azd-rest" --version "0.1.0" --draft
   azd x publish --registry ../registry.json --version "0.1.0"
   ```

## Implementation Plan

See [tasks.md](./tasks.md) for detailed task breakdown.

## Open Questions

### Q1: Should we support request/response interceptors or middleware?

**Decision**: No (out of scope for v1.0). Keep simple. Users can pipe to `jq` or other tools for transformation.

### Q2: Should we cache tokens across requests in a single session?

**Decision**: Yes. `azd-core` already provides token caching. We'll use that.

### Q3: Should we support built-in JMESPath filtering?

**Decision**: No. Users can pipe to `jq` or other tooling for filtering. Keep focused on REST execution.

### Q4: Should we support batch requests?

**Decision**: No (v1.0). Each `azd rest` call is one request. Users can script multiple calls.

### Q5: Should we detect API version automatically?

**Decision**: No. Users must provide `api-version` query parameter where required (Azure convention).

### Q6: Should we validate request bodies against OpenAPI specs?

**Decision**: No. Too complex for v1.0. Focus on executing requests, not validating schemas.

### Q7: How to handle Service Bus vs Event Hubs (both use .servicebus.windows.net)?

**Decision**: Default to Event Hubs scope. Users can override with `--scope https://servicebus.azure.net/.default` if needed.

### Q8: Should we support proxy configuration?

**Decision**: Yes, respect `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` environment variables (standard Go net/http behavior).

### Q9: Should we support certificate pinning?

**Decision**: No. Out of scope for v1.0.

### Q10: Should we support HTTP/2 or HTTP/3?

**Decision**: Use Go's default HTTP client (supports HTTP/2 automatically, HTTP/1.1 fallback). HTTP/3 is out of scope.
