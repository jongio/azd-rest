# azd-rest

**Execute REST APIs with Azure Developer CLI context and authentication**

[![CI](https://github.com/jongio/azd-rest/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-rest/actions/workflows/ci.yml)
[![Release](https://github.com/jongio/azd-rest/actions/workflows/release.yml/badge.svg)](https://github.com/jongio/azd-rest/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An Azure Developer CLI (azd) extension that enables seamless REST API calls with automatic authentication and context integration.

---

## Features

- ✅ **Automatic Authentication** - Uses azd auth tokens automatically
- 🔐 **Azure Context Integration** - Injects subscription, tenant, and environment info
- 📊 **Multiple HTTP Methods** - GET, POST, PUT, PATCH, DELETE support
- 📝 **Smart Formatting** - Auto-formats JSON responses
- 🛡️ **Secure** - TLS verification with optional skip for development
- 📤 **Flexible Input** - JSON data inline or from files
- 📥 **Flexible Output** - Pretty-print to stdout or save to files
- 🔍 **Verbose Mode** - See full request/response details

---

## Installation

### Prerequisites

- [Azure Developer CLI (azd)](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd) installed
- Go 1.23 or later (for building from source)

### Option 1: Install from Release

Download the latest release for your platform from the [Releases page](https://github.com/jongio/azd-rest/releases).

#### Windows
```powershell
# Download and extract
Invoke-WebRequest -Uri "https://github.com/jongio/azd-rest/releases/latest/download/rest-windows-amd64.zip" -OutFile rest.zip
Expand-Archive rest.zip -DestinationPath $env:USERPROFILE\.azd\bin
```

#### macOS
```bash
# Download and extract
curl -L "https://github.com/jongio/azd-rest/releases/latest/download/rest-darwin-$(uname -m).tar.gz" | tar xz -C ~/.azd/bin
```

#### Linux
```bash
# Download and extract
curl -L "https://github.com/jongio/azd-rest/releases/latest/download/rest-linux-$(uname -m).tar.gz" | tar xz -C ~/.azd/bin
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/jongio/azd-rest.git
cd azd-rest/cli

# Build
go build -o rest ./src/cmd/rest

# Move to azd extensions directory
mv rest ~/.azd/bin/
```

### Enable azd Extensions

```bash
azd config set alpha.extension.enabled on
```

---

## Quick Start

### Basic GET Request

```bash
# Simple GET request
azd rest get https://api.github.com/repos/jongio/azd-rest

# GET Azure Management API (uses azd auth automatically)
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
```

### POST with Data

```bash
# POST with JSON data
azd rest post https://api.example.com/users \
  --data '{"name":"John Doe","email":"john@example.com"}'

# POST with data from file
azd rest post https://api.example.com/users \
  --data-file user.json
```

### Custom Headers

```bash
# Add custom headers
azd rest get https://api.example.com/resource \
  -H "X-Custom-Header: value" \
  -H "X-API-Key: your-api-key"
```

### Save Response to File

```bash
# Save response to file
azd rest get https://api.example.com/data --output response.json
```

### Verbose Mode

```bash
# See full request and response details
azd rest get https://api.example.com/resource -v
```

---

## Usage

### Commands

```
azd rest <method> <url> [flags]
```

#### Available Methods

- `get` - Execute GET request
- `post` - Execute POST request
- `put` - Execute PUT request
- `patch` - Execute PATCH request
- `delete` - Execute DELETE request
- `version` - Print version information

### Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--header` | `-H` | Custom headers (can be used multiple times) | - |
| `--output` | `-o` | Output file path | stdout |
| `--verbose` | `-v` | Verbose output | false |
| `--insecure` | `-k` | Skip TLS certificate verification | false |
| `--use-azd-auth` | - | Use azd authentication token | true |

### Request Body Flags (POST, PUT, PATCH)

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--data` | `-d` | Request body data (JSON string) | - |
| `--data-file` | - | Read request body from file | - |
| `--content-type` | `-t` | Content-Type header | application/json |

---

## Examples

### Azure Management API

```bash
# List subscriptions
azd rest get "https://management.azure.com/subscriptions?api-version=2020-01-01"

# List resource groups
azd rest get "https://management.azure.com/subscriptions/{subscriptionId}/resourcegroups?api-version=2021-04-01"

# Create a resource group
azd rest put "https://management.azure.com/subscriptions/{subscriptionId}/resourcegroups/mygroup?api-version=2021-04-01" \
  --data '{"location":"eastus"}'
```

### GitHub API

```bash
# Get repository info
azd rest get https://api.github.com/repos/jongio/azd-rest

# Create an issue (with GitHub token)
azd rest post https://api.github.com/repos/jongio/azd-rest/issues \
  -H "Authorization: token YOUR_GITHUB_TOKEN" \
  --use-azd-auth=false \
  --data '{"title":"Bug report","body":"Description of the bug"}'
```

### Custom REST APIs

```bash
# GET with custom headers
azd rest get https://api.example.com/v1/users \
  -H "X-API-Key: your-api-key" \
  -H "X-Client-Id: your-client-id"

# POST with form data
azd rest post https://api.example.com/v1/login \
  --content-type "application/x-www-form-urlencoded" \
  --data "username=user&password=pass"

# DELETE with verbose output
azd rest delete https://api.example.com/v1/users/123 -v
```

---

## Azure Context Integration

The extension automatically adds Azure context information from azd:

### Automatic Headers

When using azd authentication, the following headers are automatically added:

- `Authorization: Bearer <azd-token>` - Azure auth token from azd
- `X-Azd-Subscription-Id` - Current subscription ID
- `X-Azd-Environment` - Current azd environment name

### Environment Variables

The extension respects these environment variables:

- `AZURE_SUBSCRIPTION_ID` - Azure subscription ID
- `AZURE_TENANT_ID` - Azure tenant ID
- `AZURE_LOCATION` - Azure region
- `AZURE_ENV_NAME` - azd environment name
- `AZURE_ACCESS_TOKEN` - Override auth token

---

## Development

### Prerequisites

- Go 1.23 or later
- golangci-lint for linting
- cspell for spell checking

### Build

```bash
cd cli
go build -o rest ./src/cmd/rest
```

### Test

```bash
cd cli
go test -v ./...
```

### Lint

```bash
cd cli
golangci-lint run
```

### Spell Check

```bash
cspell "**/*.{go,md,yaml,yml}"
```

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

---

## Security

For security issues, please see [SECURITY.md](SECURITY.md).

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

## Links

- [Documentation](docs/)
- [Specification](SPEC.md)
- [Issue Tracker](https://github.com/jongio/azd-rest/issues)
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [azd Extensions](https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-extensions)