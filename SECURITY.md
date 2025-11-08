# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to the project maintainers.

Please include the following information:

- Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Response Timeline

- We will acknowledge receipt of your vulnerability report within 48 hours
- We will provide a more detailed response within 7 days indicating next steps
- We will keep you informed of the progress towards a fix and announcement
- We may ask for additional information or guidance

## Disclosure Policy

- Security issues are fixed in private and disclosed publicly only after a fix is available
- We will credit researchers who responsibly disclose security issues (unless they prefer to remain anonymous)
- We follow coordinated disclosure practices

## Security Best Practices

When using azd-rest:

1. **Keep Updated** - Always use the latest version
2. **Secure Auth Tokens** - Never commit auth tokens to version control
3. **Use TLS** - Keep `--insecure` flag disabled in production
4. **Review Verbose Output** - Be cautious when sharing verbose output (may contain sensitive data)
5. **Environment Variables** - Protect environment variables containing Azure credentials
6. **File Permissions** - Ensure proper permissions on output files containing sensitive data

## Security Features

azd-rest includes the following security features:

- **TLS Verification** - Enabled by default
- **Token Masking** - Auth tokens masked in verbose output
- **No Credential Storage** - Does not store credentials
- **Minimal Dependencies** - Reduces attack surface
- **Security Scanning** - gosec in CI pipeline
- **Dependency Checks** - Regular vulnerability scans

## Known Limitations

1. Auth tokens may be visible in process lists when passed via environment variables
2. Verbose output may expose sensitive HTTP headers
3. Output files inherit default file permissions (use umask appropriately)

## Security Updates

Security updates are released as soon as possible after a vulnerability is confirmed.

Check the [Releases](https://github.com/jongio/azd-rest/releases) page for security updates.
