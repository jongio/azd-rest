# azd-rest Extension Specification

Version: 0.1.0  
Last Updated: 2025-11-08

---

## Overview

azd-rest is an Azure Developer CLI (azd) extension that enables developers to execute REST API calls with automatic integration of Azure authentication tokens and context information.

### Goals

1. Simplify REST API testing and development with Azure services
2. Automatically integrate Azure authentication from azd
3. Provide Azure context (subscription, tenant, environment) to API calls
4. Support all common HTTP methods (GET, POST, PUT, PATCH, DELETE)
5. Offer flexible request and response handling
6. Maintain security best practices

### Non-Goals

1. Replace full-featured API testing tools (e.g., Postman, httpie)
2. Provide API mocking or simulation
3. Support GraphQL or other non-REST protocols
4. Implement request recording or replay

---

## Architecture

### Components

```
azd-rest/
├── cli/                        # CLI extension
│   ├── src/
│   │   ├── cmd/rest/          # Main entry point
│   │   └── internal/
│   │       ├── cmd/           # Command implementations
│   │       ├── client/        # HTTP client logic
│   │       ├── context/       # Azure context integration
│   │       └── formatter/     # Response formatting
│   ├── extension.yaml         # Extension metadata
│   └── go.mod                 # Go module definition
└── .github/workflows/         # CI/CD pipelines
```

### Design Principles

1. **Minimal Dependencies** - Use standard library where possible
2. **Security First** - TLS verification enabled by default
3. **User-Friendly** - Simple CLI with sensible defaults
4. **Extensible** - Easy to add new features and formatters
5. **Well-Tested** - Comprehensive unit and integration tests

---

## Features

### HTTP Methods

Support for standard HTTP methods:

- **GET** - Retrieve resources
- **POST** - Create new resources
- **PUT** - Update/replace resources
- **PATCH** - Partially update resources
- **DELETE** - Remove resources

### Authentication

#### Azure Authentication

- Automatically retrieves auth token from azd via `azd auth token`
- Falls back to `AZURE_ACCESS_TOKEN` environment variable
- Can be disabled with `--use-azd-auth=false`
- Adds `Authorization: Bearer <token>` header

#### Custom Authentication

Users can provide custom auth headers:
- API keys via `-H "X-API-Key: value"`
- Bearer tokens via `-H "Authorization: Bearer token"`
- Basic auth via `-H "Authorization: Basic base64"`

### Context Integration

Automatically injects Azure context as HTTP headers:

| Header | Source | Description |
|--------|--------|-------------|
| `X-Azd-Subscription-Id` | azd context | Current subscription ID |
| `X-Azd-Environment` | azd context | Current environment name |
| `X-Azd-Tenant-Id` | azd context | Current tenant ID |

Context is retrieved from:
1. azd CLI commands (`azd env list`, `azd auth token`)
2. Environment variables (`AZURE_*`, `AZD_*`)
3. azd configuration files (`.azure/config.json`)

### Request Body Handling

#### Inline Data

```bash
azd rest post https://api.example.com/resource \
  --data '{"key":"value"}'
```

#### File Data

```bash
azd rest post https://api.example.com/resource \
  --data-file request.json
```

#### Content Types

- `application/json` (default)
- `application/x-www-form-urlencoded`
- `text/plain`
- Custom via `--content-type` flag

### Response Formatting

#### JSON Pretty-Printing

Automatically detects and formats JSON responses:

```json
{
  "id": "123",
  "name": "example",
  "status": "active"
}
```

#### Content-Type Detection

- Checks `Content-Type` response header
- Auto-detects JSON by parsing response
- Falls back to raw output for non-JSON

#### Output Options

- **stdout** (default) - Print to console
- **file** - Save to file with `--output file.json`

### Headers Management

#### Custom Headers

```bash
azd rest get https://api.example.com \
  -H "X-Custom: value" \
  -H "X-Another: value2"
```

#### Default Headers

Automatically added:
- `Content-Type: application/json` (for POST/PUT/PATCH with data)
- `Authorization: Bearer <token>` (when using azd auth)
- `X-Azd-*` headers (from Azure context)

### Security Features

#### TLS Verification

- Enabled by default
- Validates server certificates
- Can be disabled with `--insecure` for development

#### Token Masking

- Auth tokens masked in verbose output
- Displayed as `Bearer ***`
- Prevents accidental token exposure

#### Security Scanning

- gosec static analysis in CI
- Dependency vulnerability checks
- Code quality checks with golangci-lint

### Verbose Mode

Enable with `--verbose` or `-v`:

```
> GET https://api.example.com/resource
> Authorization: Bearer ***
> X-Azd-Subscription-Id: xxx-xxx-xxx
>
< HTTP 200 OK
< Content-Type: application/json
< Content-Length: 123
<
{
  "result": "success"
}
```

---

## Command-Line Interface

### Command Structure

```
azd rest <method> <url> [flags]
```

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--header, -H` | string[] | [] | Custom HTTP headers |
| `--output, -o` | string | "" | Output file path |
| `--verbose, -v` | bool | false | Verbose output |
| `--insecure, -k` | bool | false | Skip TLS verification |
| `--use-azd-auth` | bool | true | Use azd authentication |

### Request Body Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--data, -d` | string | "" | Request body data |
| `--data-file` | string | "" | Request body from file |
| `--content-type, -t` | string | "application/json" | Content-Type header |

### Commands

#### get

Execute HTTP GET request.

```bash
azd rest get <url> [flags]
```

#### post

Execute HTTP POST request.

```bash
azd rest post <url> [--data JSON | --data-file FILE] [flags]
```

#### put

Execute HTTP PUT request.

```bash
azd rest put <url> [--data JSON | --data-file FILE] [flags]
```

#### patch

Execute HTTP PATCH request.

```bash
azd rest patch <url> [--data JSON | --data-file FILE] [flags]
```

#### delete

Execute HTTP DELETE request.

```bash
azd rest delete <url> [flags]
```

#### version

Print version information.

```bash
azd rest version
```

---

## Error Handling

### HTTP Errors

- Status codes >= 400 return non-zero exit code
- Error message includes status code and text
- Response body still displayed (may contain error details)

### Network Errors

- Connection failures return descriptive error
- Timeout after 30 seconds
- DNS resolution failures reported

### Authentication Errors

- Missing auth token shows warning (if verbose)
- Continues with request (may fail at server)
- Suggests running `azd auth login`

### File Errors

- Missing data file returns error before request
- Output file write failures reported
- File permission issues shown

---

## Testing

### Unit Tests

- HTTP client logic
- Request/response handling
- Header manipulation
- Response formatting
- Context retrieval

### Integration Tests

- End-to-end command execution
- Authentication flow
- Error handling
- File I/O operations

### Test Coverage

- Target: 80% code coverage
- Required for all new features
- Measured in CI pipeline

---

## CI/CD Pipeline

### Continuous Integration

Runs on every pull request:

1. **Lint** - golangci-lint with strict rules
2. **Test** - Unit tests on Linux, macOS, Windows
3. **Spell Check** - cspell on all text files
4. **Security Scan** - gosec for security issues
5. **Build** - Multi-platform binary builds

### Continuous Deployment

Triggered on version tags:

1. Build binaries for all platforms
2. Create release archives (zip, tar.gz)
3. Generate checksums
4. Create GitHub release
5. Upload artifacts

### Platforms

- Windows (amd64)
- Linux (amd64, arm64)
- macOS (amd64, arm64)

---

## Extension Registration

### Metadata (extension.yaml)

```yaml
id: jongio.azd.rest
namespace: rest
displayName: REST API Extension
description: Execute REST APIs with azd context and authentication
usage: azd rest <method> <url> [options]
version: 0.1.0
language: go
capabilities:
  - custom-commands
examples:
  - name: get
    description: Execute GET request
    usage: azd rest get https://api.example.com/resource
tags:
  - rest
  - api
  - http
```

### Installation Methods

1. **Manual** - Download binary, place in `~/.azd/bin`
2. **From Release** - Download from GitHub releases
3. **Build from Source** - Clone and build locally

---

## Future Enhancements

### Planned Features

1. **Request Templates** - Save and reuse common requests
2. **Response Caching** - Cache responses for repeated calls
3. **Batch Requests** - Execute multiple requests from file
4. **Environment Variables** - Substitute variables in URLs and data
5. **Certificate Management** - Custom CA certificates
6. **Proxy Support** - HTTP/HTTPS proxy configuration
7. **Request History** - Store and replay previous requests
8. **JSON Path Filtering** - Extract specific fields from responses

### Under Consideration

1. **GraphQL Support** - Execute GraphQL queries
2. **WebSocket Support** - WebSocket connections
3. **OAuth Flows** - OAuth 2.0 authentication flows
4. **API Documentation** - Generate docs from API calls
5. **Performance Testing** - Load testing capabilities

---

## Compatibility

### azd Versions

- Minimum: azd 1.0.0
- Recommended: Latest stable release
- Extensions alpha feature must be enabled

### Go Versions

- Minimum: Go 1.23
- Tested: Go 1.23, 1.24

### Operating Systems

- Windows 10/11
- macOS 12+
- Linux (Ubuntu 20.04+, Debian 11+, RHEL 8+)

---

## Performance

### Benchmarks

- Request overhead: < 50ms
- JSON formatting: < 10ms for typical responses
- Context retrieval: < 100ms (cached after first call)

### Resource Usage

- Memory: < 20MB typical
- CPU: Minimal (I/O bound)
- Disk: < 10MB binary size

---

## Security Considerations

### Best Practices

1. **Never log auth tokens** - Always mask in output
2. **Verify TLS by default** - Only skip for development
3. **Minimal permissions** - No unnecessary file access
4. **Secure defaults** - Safe configuration out-of-the-box
5. **Regular updates** - Dependency security patches

### Known Limitations

1. Auth tokens passed via environment are visible in process list
2. Verbose output may expose sensitive headers
3. File permissions on output files use default umask

---

## License

MIT License - See LICENSE file for details.

---

## References

- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [azd Extensions Overview](https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-extensions)
- [Extension Framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Go HTTP Client](https://pkg.go.dev/net/http)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
