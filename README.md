# azd rest

**REST API calls with automatic Azure authentication and scope detection.**

`azd rest` is an [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/) extension that makes authenticated REST API calls to Azure services — no manual token management, no scope configuration.

[![CI](https://github.com/jongio/azd-rest/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-rest/actions/workflows/ci.yml)
[![CodeQL](https://github.com/jongio/azd-rest/actions/workflows/codeql.yml/badge.svg)](https://github.com/jongio/azd-rest/actions/workflows/codeql.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Quick Start

```bash
# 1. Install the extension
azd extension install jongio.azd.rest

# 2. Make your first request
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
```

The extension automatically detects the correct OAuth scope, acquires tokens, handles retries, and formats JSON responses.

## Command Reference

| Command | Description |
|---------|-------------|
| `azd rest get <url>` | HTTP GET request |
| `azd rest post <url>` | HTTP POST request |
| `azd rest put <url>` | HTTP PUT request |
| `azd rest patch <url>` | HTTP PATCH request |
| `azd rest delete <url>` | HTTP DELETE request |
| `azd rest head <url>` | HTTP HEAD request |
| `azd rest options <url>` | HTTP OPTIONS request |
| `azd rest version` | Show extension version |

### Common Flags

| Flag | Description |
|------|-------------|
| `--data <json>` | Inline JSON request body |
| `--data-file <path>` | Read request body from file |
| `--header <key: value>` | Add custom header (repeatable) |
| `--scope <url>` | Override OAuth scope |
| `--no-auth` | Skip authentication (public APIs) |
| `--verbose` | Show request/response details |
| `--output-file <path>` | Save response to file |
| `--insecure` | Skip TLS verification (testing only) |

For the complete reference, see [CLI Reference](cli/docs/cli-reference.md).

## Usage Examples

```bash
# POST with JSON body
azd rest post https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \
  --data '{"location":"eastus","kind":"StorageV2","sku":{"name":"Standard_LRS"}}'

# POST with body from file
azd rest post https://management.azure.com/.../storageAccounts/{name}?api-version=2021-04-01 \
  --data-file storage-account.json

# Key Vault secret
azd rest get https://myvault.vault.azure.net/secrets/mysecret?api-version=7.4

# Microsoft Graph
azd rest get https://graph.microsoft.com/v1.0/me

# Public API (no auth)
azd rest get https://api.github.com/repos/Azure/azure-dev --no-auth

# Custom headers + save response
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \
  --header "Accept: application/json" --output-file subscriptions.json
```

### Supported Azure Services

Automatic scope detection for 20+ services including Management API, Graph, Key Vault, Storage, Container Registry, Cosmos DB, App Configuration, SQL Database, DevOps, Kusto, Service Bus, and more. See [CLI Reference](cli/docs/cli-reference.md) for the complete list.

## Architecture

| Layer | Tech | Location |
|-------|------|----------|
| **CLI** | Go + Cobra | `cli/` |
| **Web** | Astro + Tailwind | `web/` |
| **Shared** | [azd-core](https://github.com/jongio/azd-core) library | Go module dependency |

### Authentication

Uses the same credential chain as `azd` and Azure CLI: Azure CLI (`az login`), Managed Identity, Service Principal (`AZURE_CLIENT_ID`/`SECRET`/`TENANT_ID`), and VS Code authentication. Tokens are automatically cached and reused.

## Development

### Prerequisites

- [Go 1.25+](https://golang.org/dl/)
- [Node.js 20+](https://nodejs.org/) and [pnpm](https://pnpm.io/)
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd)

### Build & Test

```bash
# Build
cd cli && mage build

# Test
cd cli && mage test

# Lint
cd cli && mage lint

# All (fmt → lint → test → build → install)
cd cli && mage

# Web dev
cd web && pnpm dev

# Full test suite (unit + integration + e2e)
pnpm test
```

For detailed testing information, see [TESTING.md](TESTING.md).

### Documentation

- [CLI Reference](cli/docs/cli-reference.md) — Complete command and flag reference
- [Security Review](cli/docs/security-review.md) — Security analysis and best practices
- [Threat Model](cli/docs/threat-model.md) — Threat analysis

## Security

`azd rest` uses your Azure credentials to authenticate API requests. Only make requests to trusted endpoints, use HTTPS (default), and never use `--insecure` in production. See [Security Documentation](cli/docs/security-review.md) for details.

## CI/CD

- **CI**: Lint, spell check, tests (Linux/Windows/macOS), security scanning, coverage
- **CodeQL**: Security analysis on push to main and weekly
- **Release**: Automated multi-platform binary releases
- **Website**: Automated website deployment

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Related Projects

- [Azure Developer CLI](https://github.com/Azure/azure-dev) — Core azd tool
- [azd-exec](https://github.com/jongio/azd-exec) — Execute scripts with azd context
- [azd-app](https://github.com/jongio/azd-app) — Run Azure apps locally

## License

MIT — see [LICENSE](LICENSE) for details.
