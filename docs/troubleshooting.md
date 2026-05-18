# Troubleshooting

Common error scenarios and solutions for `azd rest`.

## Authentication Errors

### "no credential available" or "authentication failed"

**Cause**: No Azure CLI login session or expired token.

**Solution**:
```bash
# Log in to Azure CLI
az login

# Verify your session
az account show
```

### "token acquisition failed for scope ..."

**Cause**: The detected OAuth scope doesn't match your permissions, or you lack access to the target resource.

**Solution**:
```bash
# Check which account is active
az account show

# Switch subscription if needed
az account set --subscription <subscription-id>

# Force a specific scope
azd rest get <url> --scope https://management.azure.com/.default
```

## Network Errors

### "connection refused" or "dial tcp: lookup ... no such host"

**Cause**: DNS resolution failure, network connectivity issue, or incorrect URL.

**Solution**:
1. Verify the URL is correct and the service exists.
2. Check your network connection and proxy settings.
3. For private endpoints, ensure you're on the correct network (VPN/private link).

### "SSRF protection: blocked request to private IP range"

**Cause**: The URL resolves to a private/internal IP address. `azd rest` blocks these by default to prevent SSRF attacks.

**Solution**:
- Verify the URL is correct - this usually indicates a misconfigured endpoint.
- If you intentionally need to reach a private endpoint, use `--allow-private` (not recommended for untrusted URLs).

## Request Errors

### "HTTP 401 Unauthorized"

**Cause**: Valid token but insufficient permissions for the target resource.

**Solution**:
```bash
# Verify your identity
az account show

# Check role assignments on the resource
az role assignment list --scope <resource-id>

# Try with explicit scope
azd rest get <url> --scope <scope-url>
```

### "HTTP 403 Forbidden"

**Cause**: Your identity lacks the required RBAC role or policy blocks the operation.

**Solution**:
1. Verify you have the correct role assignment on the target resource.
2. Check Azure Policy for deny rules.
3. For management plane operations, ensure you have at least Reader role.

### "HTTP 404 Not Found"

**Cause**: The resource doesn't exist, incorrect API version, or wrong URL path.

**Solution**:
1. Verify the resource exists: `az resource show --ids <resource-id>`
2. Check the API version in the URL is valid for the resource type.
3. Verify the URL path structure matches the Azure REST API reference.

### "HTTP 429 Too Many Requests"

**Cause**: Azure throttling due to too many requests.

**Solution**:
- `azd rest` automatically retries with exponential backoff.
- If persistent, wait a few minutes and reduce request frequency.
- Check `Retry-After` header in verbose output: `azd rest get <url> --verbose`

## Installation Errors

### "extension not found" or "failed to install"

**Cause**: Extension registry not configured or network issue.

**Solution**:
```bash
# Add the extension registry
azd extension source add -n jongio -t url -l https://jongio.github.io/azd-extensions/registry.json

# Install the extension
azd extension install jongio.azd.rest

# Verify installation
azd rest version
```

### "azd: command not found"

**Cause**: Azure Developer CLI not installed or not in PATH.

**Solution**:
```bash
# Install azd
curl -fsSL https://aka.ms/install-azd.sh | bash

# Or on Windows (PowerShell)
powershell -ex AllSigned -c "Invoke-RestMethod 'https://aka.ms/install-azd.ps1' | Invoke-Expression"
```

## Build Errors (Development)

### "go: module requires Go >= 1.26.1"

**Cause**: Your Go version is too old.

**Solution**:
```bash
# Check current version
go version

# Install Go 1.26.1+ from https://go.dev/dl/
```

### "mage: command not found"

**Cause**: Mage build tool not installed.

**Solution**:
```bash
go install github.com/magefile/mage@latest
```

## MCP Server Errors

### "failed to start MCP server" or connection timeout

**Cause**: Port conflict, missing configuration, or extension not properly installed.

**Solution**:
1. Verify the extension is installed: `azd rest version`
2. Check if another process is using the MCP port.
3. Restart the MCP client (e.g., VS Code, Copilot).

### MCP client can't discover azd rest tools

**Cause**: MCP configuration not pointing to the correct extension path.

**Solution**:
```bash
# Verify mcp serve works directly
azd rest mcp serve

# Check your MCP client configuration references the correct azd binary path
```

## Getting More Help

- **Verbose output**: Add `--verbose` to any command for detailed request/response info.
- **GitHub Issues**: [Report a bug](https://github.com/jongio/azd-rest/issues/new)
- **Discussions**: [Ask a question](https://github.com/jongio/azd-rest/discussions)
