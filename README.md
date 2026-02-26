<div align="center">

# azd rest

### **Authenticated Azure REST Calls**

Make REST API calls with automatic Azure authentication and scope detection — no manual token management.

[![CI](https://github.com/jongio/azd-rest/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-rest/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<br />

[**🌐 Visit the Website →**](https://jongio.github.io/azd-rest/)

*Full documentation, CLI reference, and security architecture*

[**📦 Part of azd Extensions →**](https://jongio.github.io/azd-extensions/)

*Browse all Azure Developer CLI extensions by Jon Gallant*

<br />

---

</div>

## ⚡ One-Command REST Calls

Stop managing tokens. Run `azd rest` and authentication happens automatically.

```bash
# Add the extension registry
azd extension source add -n jongio -t url -l https://jongio.github.io/azd-extensions/registry.json

# Install the extension
azd extension install jongio.azd.rest

# Make your first request
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01
```

That's it. The extension detects the correct OAuth scope, acquires tokens, handles retries, and formats JSON responses.

---

## ✨ Features

<table>
<tr>
<td width="50%">

### 🔐 Automatic Authentication
Uses your Azure CLI credentials with automatic OAuth scope detection for 20+ Azure services — Management API, Graph, Key Vault, Storage, Cosmos DB, and more.

### 🛡️ Security Hardened
SSRF protection with DNS resolution validation, blocked CIDR ranges, rate limiting, header sanitization, and response size limits. [See security architecture →](https://jongio.github.io/azd-rest/security/)

### 🤖 MCP Server
Built-in Model Context Protocol server for AI agent integration. Copilot and other AI tools can make authenticated Azure REST calls through `azd rest`.

</td>
<td width="50%">

### 🔄 All HTTP Methods
GET, POST, PUT, PATCH, DELETE, HEAD, and OPTIONS with JSON body support from inline data or files.

### 📊 Verbose Diagnostics
Request/response details, traceparent injection for distributed tracing, and redacted sensitive headers in logs.

### ✅ Battle-Tested
Comprehensive CI with CodeQL security scanning, spell checking, multi-platform testing (Linux/Windows/macOS), and 80%+ test coverage.

</td>
</tr>
</table>

---

## 📖 Usage Examples

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

For the complete command and flag reference, see the [CLI Reference](https://jongio.github.io/azd-rest/reference/cli/) on the website.

## ⚙️ Development

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
```

For detailed testing information, see [TESTING.md](TESTING.md).

## 🔐 Security

`azd rest` uses your Azure credentials to authenticate API requests. Only make requests to trusted endpoints, use HTTPS (default), and never use `--insecure` in production.

See the [Security Architecture](https://jongio.github.io/azd-rest/security/) page for the full threat model, SSRF protections, and hardening details.

## 📚 Documentation

- [**Website**](https://jongio.github.io/azd-rest/) — Full documentation and guided tour
- [CLI Reference](https://jongio.github.io/azd-rest/reference/cli/) — Complete command and flag reference
- [Security Architecture](https://jongio.github.io/azd-rest/security/) — Threat model and security hardening
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contribution guidelines

## 🔗 azd Extensions

azd rest is part of a suite of Azure Developer CLI extensions by [Jon Gallant](https://github.com/jongio).

| Extension | Description | Website |
|-----------|-------------|---------|
| **[azd app](https://github.com/jongio/azd-app)** | Run Azure apps locally with auto-dependencies, dashboard, and AI debugging | [jongio.github.io/azd-app](https://jongio.github.io/azd-app/) |
| **[azd copilot](https://github.com/jongio/azd-copilot)** | AI-powered Azure development with 16 agents and 28 skills | [jongio.github.io/azd-copilot](https://jongio.github.io/azd-copilot/) |
| **[azd exec](https://github.com/jongio/azd-exec)** | Execute scripts with azd environment context and Key Vault integration | [jongio.github.io/azd-exec](https://jongio.github.io/azd-exec/) |
| **[azd rest](https://github.com/jongio/azd-rest)** | Authenticated REST API calls with automatic scope detection | [jongio.github.io/azd-rest](https://jongio.github.io/azd-rest/) |

🌐 **Extension Hub**: [jongio.github.io/azd-extensions](https://jongio.github.io/azd-extensions/) — Browse all extensions, quick install, and registry info.

## License

MIT — see [LICENSE](LICENSE) for details.
