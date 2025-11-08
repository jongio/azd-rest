---
title: azd-rest Tasks
created: 2026-01-15
updated: 2026-01-15
status: active
type: feature
category: documentation
author: azd-rest team
percentComplete: 20
related: []
tags: [rest, planning, tasks]
---

# azd-rest: REST API Extension with Azure Authentication - Tasks

<!-- NEXT: 4 -->

## TODO

### 4. HTTP Client Module

**Priority**: P0  
**Size**: L  
**Dependencies**: Task 3  
**Description**: Implement HTTP client for REST API execution

**Acceptance Criteria**:
- [ ] `cli/src/internal/client/client.go` with request execution
- [ ] Support all HTTP methods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- [ ] Add `Authorization: Bearer {token}` header when token provided
- [ ] Support custom headers (repeatable `-H` flag)
- [ ] Support request body from string (`--data`) or file (`--data-file`)
- [ ] Timeout handling (default 30s, configurable via `--timeout`)
- [ ] Redirect handling (follow by default, max 10 hops)
- [ ] TLS certificate validation (enabled by default, skip with `--insecure`)
- [ ] Proxy support via environment variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
- [ ] Unit tests `cli/src/internal/client/client_test.go` with 80%+ coverage
- [ ] Mock HTTP server for testing error scenarios
- [ ] Test timeout, redirects, TLS validation, headers, body handling

**Notes**: Use Go's standard `net/http` package. Keep it simple.

---

### 5. Response Formatting Module

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 4  
**Description**: Implement response formatting and output options

**Acceptance Criteria**:
- [ ] `cli/src/internal/client/formatter.go` with formatting functions
- [ ] Auto-detect JSON responses and pretty-print
- [ ] Raw output mode (`--format raw`)
- [ ] Write to stdout by default, file with `--output` flag
- [ ] Verbose mode (`--verbose`) shows: request headers, response headers, status code, timing
- [ ] Token redaction in verbose output (show first/last 4 chars only)
- [ ] Error response formatting (display error message + status code)
- [ ] Unit tests `cli/src/internal/client/formatter_test.go`
- [ ] Test JSON pretty-print, raw output, file writing, verbose mode

**Notes**: Match Azure CLI output quality.

---

### 6. CLI Command Structure (Cobra)

**Priority**: P0  
**Size**: L  
**Dependencies**: Tasks 3, 4, 5  
**Description**: Implement CLI commands and flags using Cobra

**Acceptance Criteria**:
- [ ] `cli/src/internal/cmd/root.go` with root command and global flags
- [ ] `cli/src/internal/cmd/get.go` for GET requests
- [ ] `cli/src/internal/cmd/post.go` for POST requests
- [ ] `cli/src/internal/cmd/put.go` for PUT requests
- [ ] `cli/src/internal/cmd/patch.go` for PATCH requests
- [ ] `cli/src/internal/cmd/delete.go` for DELETE requests
- [ ] `cli/src/internal/cmd/head.go` for HEAD requests
- [ ] `cli/src/internal/cmd/options.go` for OPTIONS requests
- [ ] `cli/src/internal/cmd/version.go` for version command
- [ ] All flags from spec: `--scope`, `--no-auth`, `--header`, `--data`, `--data-file`, `--output`, `--format`, `--verbose`, `--insecure`, `--timeout`, `--follow-redirects`, `--max-redirects`
- [ ] Flag validation (e.g., timeout must be positive)
- [ ] Help text for all commands and flags
- [ ] Usage examples in help
- [ ] Unit tests for command parsing and flag validation

**Notes**: Follow azd-exec patterns for Cobra setup.

---

### 7. Main Entry Point

**Priority**: P0  
**Size**: S  
**Dependencies**: Task 6  
**Description**: Implement main.go entry point

**Acceptance Criteria**:
- [ ] `cli/src/cmd/rest/main.go` with version injection via ldflags
- [ ] Proper error handling and exit codes
- [ ] Version set from build process
- [ ] Simple, clean entry point (delegate to cmd package)

**Notes**: Copy pattern from azd-exec.

---

### 8. Build System (Mage)

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 7  
**Description**: Implement Mage build automation

**Acceptance Criteria**:
- [ ] `cli/magefile.go` with all targets from azd-exec:
  - `Build`: Build binary using `azd x build`
  - `Pack`: Package using `azd x pack`
  - `Publish`: Publish to local registry
  - `Setup`: Build + Pack + Publish + Install
  - `Test`: Run unit tests only
  - `TestIntegration`: Run integration tests
  - `TestAll`: Run all tests
  - `TestCoverage`: Generate coverage report
  - `Lint`: Run golangci-lint
  - `Fmt`: Format code
  - `Clean`: Remove build artifacts
  - `Preflight`: Run all checks
  - `Watch`: Watch mode (if needed)
- [ ] `cli/build.sh` shell script for Linux/macOS
- [ ] `cli/build.ps1` PowerShell script for Windows (if needed)
- [ ] Version extraction from `extension.yaml`
- [ ] Integration with azd extensions (`azd x build`, `azd x pack`)

**Notes**: Copy from azd-exec and adapt for rest extension.

---

### 9. Unit Tests

**Priority**: P0  
**Size**: L  
**Dependencies**: Tasks 2-6  
**Description**: Comprehensive unit test coverage (80%+ target)

**Acceptance Criteria**:
- [ ] `cli/src/internal/auth/scope_test.go`: 100% coverage of scope detection
- [ ] `cli/src/internal/auth/auth_test.go`: Auth logic with mocked azd-core
- [ ] `cli/src/internal/client/client_test.go`: HTTP client with mock server
- [ ] `cli/src/internal/client/formatter_test.go`: All formatting scenarios
- [ ] `cli/src/internal/cmd/*_test.go`: Command parsing and validation
- [ ] Table-driven tests for scope detection (all services)
- [ ] Error scenario tests (network failures, auth failures, timeouts)
- [ ] Edge case tests (empty responses, large bodies, special characters)
- [ ] Run with `mage test` or `go test -short ./src/...`
- [ ] Generate coverage report with `mage testCoverage`
- [ ] Achieve 80%+ overall coverage

**Notes**: Use `-short` flag to skip integration tests.

---

### 10. Integration Tests

**Priority**: P0  
**Size**: L  
**Dependencies**: Task 9  
**Description**: Integration tests with real Azure services

**Acceptance Criteria**:
- [ ] `cli/tests/integration/auth_test.go`: Real Azure authentication
- [ ] `cli/tests/integration/management_test.go`: ARM API calls (list subscriptions)
- [ ] `cli/tests/integration/storage_test.go`: Storage API calls (require storage account)
- [ ] `cli/tests/integration/keyvault_test.go`: Key Vault API calls (require vault)
- [ ] All tests tagged with `//go:build integration`
- [ ] Tests skip if not authenticated (`az login` required)
- [ ] Run with `mage testIntegration` or `go test -tags=integration ./tests/...`
- [ ] CI runs integration tests on main branch only (not PRs)
- [ ] Tests clean up resources after execution

**Notes**: May require Azure subscription. Use test fixtures.

---

### 11. E2E Tests

**Priority**: P1  
**Size**: M  
**Dependencies**: Task 10  
**Description**: End-to-end command execution tests

**Acceptance Criteria**:
- [ ] `cli/tests/e2e/e2e_test.go`: Full command execution
- [ ] Test all HTTP methods (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
- [ ] Test custom scopes, headers, output formats
- [ ] Test `--verbose`, `--output`, `--data`, `--data-file` flags
- [ ] Test error scenarios (invalid URL, auth failures, network errors)
- [ ] Test non-Azure endpoints with `--no-auth`
- [ ] Run with `mage testAll`

**Notes**: These test the full CLI flow, not individual functions.

---

### 12. CI Workflow (GitHub Actions)

**Priority**: P0  
**Size**: M  
**Dependencies**: Tasks 8, 9  
**Description**: Implement CI pipeline

**Acceptance Criteria**:
- [ ] `.github/workflows/ci.yml` based on azd-exec pattern
- [ ] Jobs: preflight, test (Linux/Windows/macOS), lint, build, integration
- [ ] Preflight: format check, spell check, lint, security scan
- [ ] Test: unit tests with race detection, upload coverage to Codecov
- [ ] Lint: golangci-lint, go vet
- [ ] Build: multi-platform binaries (Windows/Linux/macOS, x64/ARM64)
- [ ] Integration: integration tests on Linux only
- [ ] Runs on PR to main, workflow_dispatch
- [ ] Uses Go 1.26.0
- [ ] Artifacts uploaded for binaries
- [ ] Coverage report in PR summary

**Notes**: Copy from azd-exec and adapt.

---

### 13. CodeQL Security Scanning

**Priority**: P0  
**Size**: S  
**Dependencies**: None  
**Description**: Implement CodeQL security analysis

**Acceptance Criteria**:
- [ ] `.github/workflows/codeql.yml` based on azd-exec
- [ ] Runs on push to main and weekly schedule
- [ ] Scans Go code for security vulnerabilities
- [ ] Reports findings to GitHub Security tab
- [ ] No critical vulnerabilities found

**Notes**: Standard CodeQL setup for Go projects.

---

### 14. PR Build Workflow

**Priority**: P1  
**Size**: S  
**Dependencies**: Task 12  
**Description**: Fast PR validation workflow

**Acceptance Criteria**:
- [ ] `.github/workflows/pr-build.yml`
- [ ] Runs subset of CI checks: lint, unit tests (Linux only), build
- [ ] Faster than full CI for quick feedback
- [ ] Runs on PR events (opened, synchronize)

**Notes**: Optimize for speed while maintaining quality.

---

### 15. Release Workflow

**Priority**: P0  
**Size**: L  
**Dependencies**: Task 12  
**Description**: Automated release workflow

**Acceptance Criteria**:
- [ ] `.github/workflows/release.yml` based on azd-exec
- [ ] Manual trigger with `patch`/`minor`/`major` version bump input
- [ ] Calculate next version from `extension.yaml`
- [ ] Update `extension.yaml` and `CHANGELOG.md`
- [ ] Run full CI (preflight, test, lint, build)
- [ ] Build multi-platform binaries (Windows/Linux/macOS, x64/ARM64)
- [ ] Package with `azd x pack`
- [ ] Create GitHub release with binaries and changelog
- [ ] Update `registry.json`
- [ ] Commit and push version bump
- [ ] Uses `GITHUB_TOKEN` for authentication

**Notes**: This is the most complex workflow. Test thoroughly.

---

### 16. Documentation: CLI Reference

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 6  
**Description**: Complete command and flag reference

**Acceptance Criteria**:
- [ ] `cli/docs/cli-reference.md` with all commands documented
- [ ] Each command: description, usage, flags, examples
- [ ] All flags documented with types, defaults, descriptions
- [ ] Real-world examples for each command
- [ ] Cross-references to other docs (azure-scopes.md, security-review.md)

**Notes**: Similar to azd-exec's cli-reference.md.

---

### 17. Documentation: Azure Scopes Reference

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 2  
**Description**: Comprehensive Azure scope mapping guide

**Acceptance Criteria**:
- [ ] `cli/docs/azure-scopes.md` with all service scopes
- [ ] Table with: Service, URL Pattern, OAuth Scope, Example
- [ ] Explanation of scope detection algorithm
- [ ] How to override with `--scope` flag
- [ ] Special cases (Kusto, DevOps, Service Bus vs Event Hubs)
- [ ] Links to official Azure documentation

**Notes**: This is a key reference document for users.

---

### 18. Documentation: Security Review

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 4  
**Description**: Security analysis and best practices

**Acceptance Criteria**:
- [ ] `cli/docs/security-review.md` based on azd-exec pattern
- [ ] Token handling security
- [ ] TLS/HTTPS best practices
- [ ] Input validation
- [ ] Audit trail and logging
- [ ] Warning about `--insecure` flag
- [ ] Recommendations for secure usage
- [ ] Threat model (similar to azd-exec)

**Notes**: Security is critical for an auth tool.

---

### 19. Documentation: Usage Examples

**Priority**: P1  
**Size**: M  
**Dependencies**: Task 6  
**Description**: Real-world usage examples

**Acceptance Criteria**:
- [ ] `cli/docs/examples.md` with practical scenarios
- [ ] Examples for all major Azure services (ARM, Storage, Key Vault, Graph)
- [ ] Examples with custom scopes, headers, data
- [ ] Error handling examples
- [ ] Non-Azure endpoint examples
- [ ] Copy-paste ready commands

**Notes**: Help users get started quickly.

---

### 20. Documentation: README.md

**Priority**: P0  
**Size**: L  
**Dependencies**: Tasks 16, 17, 18  
**Description**: Comprehensive README with quick start

**Acceptance Criteria**:
- [ ] Project description and badges (CI, CodeQL, License, Coverage)
- [ ] Features overview with visual appeal (similar to azd-exec README)
- [ ] Quick start guide
- [ ] Installation instructions (enable extensions, add source, install)
- [ ] Common usage patterns with examples
- [ ] Security notice section
- [ ] Link to detailed documentation
- [ ] Contributing guidelines
- [ ] License information
- [ ] Release notes link

**Notes**: This is the first thing users see. Make it great.

---

### 21. Documentation: TESTING.md

**Priority**: P1  
**Size**: S  
**Dependencies**: Tasks 8, 9, 10, 11  
**Description**: Testing guide

**Acceptance Criteria**:
- [ ] Test structure and organization
- [ ] How to run unit tests, integration tests, e2e tests
- [ ] Coverage requirements (80%+)
- [ ] How to add new tests
- [ ] CI/CD testing process
- [ ] Troubleshooting test failures

**Notes**: Copy structure from azd-exec.

---

### 22. Spell Check Configuration

**Priority**: P2  
**Size**: S  
**Dependencies**: None  
**Description**: Configure cspell for documentation

**Acceptance Criteria**:
- [ ] `cspell.json` at root with custom dictionary
- [ ] Include Azure service names (Kusto, Synapse, etc.)
- [ ] Include technical terms (OAuth, bearer, scope, etc.)
- [ ] Run in CI preflight checks
- [ ] All docs pass spell check

**Notes**: Maintain documentation quality.

---

### 23. Linting Configuration

**Priority**: P1  
**Size**: S  
**Dependencies**: None  
**Description**: Configure golangci-lint

**Acceptance Criteria**:
- [ ] `cli/.golangci.yml` with strict settings
- [ ] Enable: gofmt, goimports, govet, staticcheck, gosec, errcheck
- [ ] Disable: none (strict mode)
- [ ] Run in CI lint job
- [ ] All code passes linting

**Notes**: Use azd-exec config as baseline.

---

### 24. Local Development Setup Script

**Priority**: P2  
**Size**: S  
**Dependencies**: Task 8  
**Description**: Script for setting up local development

**Acceptance Criteria**:
- [ ] `test-release-local.ps1` for Windows (similar to azd-exec)
- [ ] Automate: build, pack, publish, install to local azd
- [ ] Verify installation with `azd rest version`
- [ ] Clean up previous installations

**Notes**: Make local testing easy.

---

### 25. Registry Configuration

**Priority**: P0  
**Size**: S  
**Dependencies**: Task 1  
**Description**: Configure extension registry

**Acceptance Criteria**:
- [ ] `registry.json` at root with extension metadata
- [ ] JSON schema: `{"extensions": [{"id": "jongio.azd.rest", ...}]}`
- [ ] Include version, download URLs (to be updated by release workflow)
- [ ] Validate JSON structure

**Notes**: Required for azd extension installation.

---

### 26. Test Package.json Orchestration

**Priority**: P1  
**Size**: S  
**Dependencies**: Tasks 8, 9, 10, 11  
**Description**: Root package.json for test orchestration

**Acceptance Criteria**:
- [ ] `package.json` at root with scripts:
  - `test`: Run all tests (unit + integration)
  - `test:cli`: Run CLI tests
  - `test:cli:unit`: Run unit tests only
  - `test:cli:integration`: Run integration tests only
- [ ] Works with `pnpm test` from root
- [ ] Matches azd-exec pattern

**Notes**: Consistent test execution across projects.

---

### 27. Final Integration Test with Real Azure

**Priority**: P0  
**Size**: M  
**Dependencies**: All previous tasks  
**Description**: End-to-end validation with real Azure services

**Acceptance Criteria**:
- [ ] Install extension locally: `azd extension install jongio.azd.rest --source local`
- [ ] Test Management API: `azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01`
- [ ] Test Storage API with real storage account
- [ ] Test Key Vault API with real vault
- [ ] Test custom scope with non-Azure endpoint
- [ ] Test all HTTP methods (GET, POST, PUT, DELETE)
- [ ] Test verbose mode, output to file, custom headers
- [ ] Verify token auto-detection for all Azure services
- [ ] All tests pass without errors

**Notes**: This is the final validation before first release.

---

### 28. Release v0.1.0

**Priority**: P0  
**Size**: S  
**Dependencies**: Task 27  
**Description**: First production release

**Acceptance Criteria**:
- [ ] Trigger release workflow with `patch` bump
- [ ] Version 0.1.0 created in GitHub releases
- [ ] Binaries for all platforms uploaded
- [ ] CHANGELOG.md updated
- [ ] registry.json updated with v0.1.0 download URLs
- [ ] Extension installable via: `azd extension install jongio.azd.rest`
- [ ] README.md installation instructions verified

**Notes**: Celebrate! 🎉

---

### 29. Azure CLI Parity Features

**Priority**: P0  
**Size**: M  
**Dependencies**: Tasks 2, 3, 4, 6  
**Description**: Implement parity behaviors with Azure CLI `az rest` (body @file, uri parameters, auth skip rules, output-file, binary mode)

**Acceptance Criteria**:
- [ ] Support `@{file}` shorthand for request bodies with UTF-8 default and binary-safe reads
- [ ] Implement `--uri-parameters` substitution and ARM placeholder replacement (subscriptionId, resourceGroup)
- [ ] Add `--output-file` with binary-safe streaming (no formatting) when binary detected or `--binary` set
- [ ] Add `--binary` passthrough mode to stream request/response bodies without transformation
- [ ] Auth skip logic matches parity rules: if `--no-auth` or no scope and no override, skip token and warn on Azure-looking hosts
- [ ] Unit tests cover each parity behavior; e2e covers `@file`, `--uri-parameters`, `--output-file`, `--binary`, and auth-skip warning

---

### 30. Scope Detection Edge Cases

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 2  
**Description**: Expand scope detection for Service Bus vs Event Hubs paths, sovereign clouds, custom ports, and fallback warnings

**Acceptance Criteria**:
- [ ] Path-based selection for `.servicebus.windows.net`: Event Hubs default, Service Bus when path includes `/queue` or `/queues`
- [ ] Recognize sovereign suffixes (`.azure.cn`, `.usgovcloudapi.net`, `.microsoft.scloud`) with same scope mapping
- [ ] Ignore port when matching hostnames; add tests with non-default ports
- [ ] Warn when host matches Azure patterns but scope is empty and no override provided
- [ ] Table-driven unit tests cover all edge cases and negative cases

---

### 31. Reliability and Pagination Controls

**Priority**: P1  
**Size**: M  
**Dependencies**: Tasks 4, 5, 6  
**Description**: Add retries/backoff, pagination, timeouts, payload limits, and streaming safeguards

**Acceptance Criteria**:
- [ ] Default retries=3 with exponential backoff on 429/5xx/timeouts; configurable via `--retry`
- [ ] Default timeout 30s with upper bound (e.g., 5m); flag validation for positive duration
- [ ] `--paginate` follows `nextLink`/`@odata.nextLink`/`Continuation-Token`; stops on limit or error
- [ ] Payload size guard: switch to streaming beyond threshold (configurable) to avoid memory blowup
- [ ] Streaming rules: when `--binary` or non-text content-type, stream to stdout or `--output-file` without formatting
- [ ] Unit/e2e tests cover retries, pagination happy/stop paths, timeout enforcement, and streaming of large/binary content

---

### 32. Security and Telemetry Guardrails

**Priority**: P0  
**Size**: M  
**Dependencies**: Tasks 4, 5  
**Description**: Implement redaction, `--insecure` safeguards, telemetry policy, and binary handling protections

**Acceptance Criteria**:
- [ ] Verbose redaction shows only first/last 4 token chars; no body/header content logged by default
- [ ] `--insecure` requires confirmation (or `--yes`) and emits a clear warning
- [ ] Telemetry is opt-in; collects only method/host/status/duration; never tokens/headers/bodies
- [ ] Binary handling avoids buffering large payloads; streams directly when flagged or detected
- [ ] Threat model documented in security-review.md reflecting these controls
- [ ] Tests cover redaction, `--insecure` prompt, and telemetry opt-in/off behavior

---

### 33. Docs and Tests Alignment for Parity

**Priority**: P0  
**Size**: S  
**Dependencies**: Tasks 29, 30, 31, 32  
**Description**: Update docs and tests to reflect new parity/reliability/security behaviors

**Acceptance Criteria**:
- [ ] CLI reference updated with `--uri-parameters`, `@{file}`, `--output-file`, `--binary`, `--paginate`, `--retry`
- [ ] Examples include parity scenarios and binary/download cases
- [ ] Security-review.md and azure-scopes.md updated with new behaviors and scope edge cases
- [ ] Tests added/updated to cover the new flags and documented behaviors

---

### 34. Tooling and Compatibility Enforcement

**Priority**: P1  
**Size**: S  
**Dependencies**: Tasks 12, 15  
**Description**: Enforce Go/toolchain and azd-core compatibility; ensure rollback/install safety in release flow

**Acceptance Criteria**:
- [ ] CI workflows pin Go 1.26.0 and validate azd-core >= v0.3.0
- [ ] Compatibility matrix documented (OS/arch) and validated in CI job matrix
- [ ] Release workflow retains prior version in registry.json for rollback; README/TESTING describe rollback steps
- [ ] Checks fail fast if tooling versions drift from pinned values

---

### 35. Documentation Hygiene & Frontmatter Compliance

**Priority**: P0  
**Size**: S  
**Dependencies**: None  
**Description**: Clean up azd-rest docs to DevX standards (frontmatter, naming, placement, stale dates)

**Acceptance Criteria**:
- [ ] All docs under `docs/` only; no stray docs elsewhere
- [ ] Required frontmatter fields present and valid where applicable (title, created, updated, status, type, tags optional)
- [ ] Files follow lowercase-dash naming (exceptions: README, LICENSE)
- [ ] Stale `updated` dates (>90 days) refreshed where content is touched
- [ ] `devx-frontmatter-validate` passes cleanly

---

### 36. Integrate Design Points and Remove Research Doc

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 35 (or run in parallel)  
**Description**: Fold all useful design points from the retired Azure CLI research into the spec, remove research references, and delete the research doc.

**Acceptance Criteria**:
- [ ] Spec incorporates necessary design behaviors (auth skip logic, token/URI parameter handling, request/response I/O, telemetry, pagination/retry choices, binary/output handling) without citing the research doc or `az rest`
- [ ] References to the retired research doc removed from spec, summary, and related docs
- [ ] Research doc deleted
- [ ] Summary updated to reflect current state without pointing to the removed research doc
- [ ] Frontmatter remains valid; docs stay within `docs/` and naming rules upheld

---

### 38. Remove summary.md Doc

**Priority**: P2  
**Size**: XS  
**Dependencies**: None  
**Description**: Remove the summary.md doc now that the spec contains the integrated design points.

**Acceptance Criteria**:
- [ ] summary.md removed from docs/specs/azd-rest/
- [ ] No links in spec/tasks/other docs point to summary.md
- [ ] Frontmatter validation remains clean after removal

---

## IN PROGRESS

_No active tasks._

### 3. Authentication Module with azd-core

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 2  
**Description**: Implement Azure token acquisition using azd-core library

**Acceptance Criteria**:
- [ ] `cli/src/internal/auth/auth.go` with `GetAzureToken(scope string) (string, error)`
- [ ] Integration with azd-core's authentication (DefaultAzureCredential pattern)
- [ ] Token caching via azd-core
- [ ] Error handling for auth failures (not logged in, insufficient permissions)
- [ ] Unit tests with mocked azd-core calls
- [ ] Integration tests with real Azure authentication (tagged `integration`)
- [ ] Supports all credential types: Azure CLI, Managed Identity, Environment Variables

**Notes**: Leverage azd-core security module. Don't reinvent authentication.

**Status (In Progress)**:
- Kickoff: handoff to Developer to wire `GetAzureToken` via azd-core credentials + caching; preserve azd-core v0.3.0 pin and Go 1.26.0 guard.

---

---

---

## DONE

### 1. Project Foundation & Scaffolding

**Priority**: P0  
**Size**: M  
**Dependencies**: None  
**Description**: Set up initial project structure following azd-exec patterns

**Acceptance Criteria**:
- [x] Directory structure created (`cli/`, `docs/`, `.github/workflows/`, etc.)
- [x] `cli/go.mod` with dependencies (azd-core v0.3.0+, cobra, Azure SDK)
- [x] `cli/extension.yaml` with metadata (id: jongio.azd.rest, namespace: rest)
- [x] Root `package.json` for test orchestration (similar to azd-exec)
- [x] `.gitignore`, `LICENSE` (MIT), `cspell.json`
- [x] Basic `README.md` with project description and installation instructions
- [x] `CHANGELOG.md` initialized with v0.1.0
- [x] `CONTRIBUTING.md` copied from azd-exec
- [x] `registry.json` initialized

**Progress (2026-01-15)**:
- [x] Added scaffolding files aligned to azd-exec baseline (gitignore, license, cspell, package, README, changelog, contributing, registry) and advanced NEXT to Task 2.
- [x] Updated extension metadata with id/namespace and preserved go.mod dependencies on azd-core v0.3.0, Azure SDK, and Cobra.

---

### 2. Azure Scope Detection Module

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 1  
**Description**: Implement scope detection algorithm for all Azure services

**Acceptance Criteria**:
- [x] `cli/src/internal/auth/scope.go` with `DetectScope(url string) (string, error)`
- [x] Support for all services in spec: Management, Storage, Key Vault, Graph, DevOps, Kusto, ACR, Event Hubs, Service Bus, Cosmos, App Config, Batch, OSSRDBMS, SQL, Synapse, Data Lake, Media, Log Analytics
- [x] Exact match patterns (management.azure.com, graph.microsoft.com, etc.)
- [x] Suffix match patterns (*.vault.azure.net, *.blob.core.windows.net, etc.)
- [x] Special cases (Kusto with cluster-specific scope, DevOps GUID)
- [x] Unit tests `cli/src/internal/auth/scope_test.go` with 100% coverage
- [x] Test all URL patterns from spec table
- [x] Test edge cases (http vs https, port numbers, query strings)
- [x] Returns empty string for non-Azure URLs

**Progress (2026-01-15)**:
- Implemented scope detection covering management, storage variants, Key Vault, Graph, DevOps GUID, Kusto cluster-aware scopes, ACR, Event Hubs vs Service Bus queue paths, Cosmos, App Config, Batch, OSSRDBMS, SQL, Synapse, Data Lake, Media, and Log Analytics in [cli/src/internal/auth/scope.go](cli/src/internal/auth/scope.go#L1-L108).
- Added exhaustive table-driven tests (edge cases include HTTP scheme, ports, query strings, relative URLs, malformed URLs) with 100% coverage in [cli/src/internal/auth/scope_test.go](cli/src/internal/auth/scope_test.go#L1-L354); validated with `GOWORK=off GOFLAGS=-mod=mod go test ./src/internal/auth/...`.

---

### 37. azd-core Alignment for azd-rest

### 37. azd-core Alignment for azd-rest

**Priority**: P0  
**Size**: M  
**Dependencies**: Task 36  
**Description**: Specify and coordinate any azd-core updates required to support azd-rest features.

**Acceptance Criteria**:
- [x] Document required azd-core surface in spec (token helper with expires-on, subscription/resource group accessors, cloud endpoint map, user-agent/telemetry hook, redaction utilities, version pin)
- [x] Open tracking item/PR in azd-core (or cross-repo task) for these changes
- [x] Update dependency/pinning in azd-rest to minimum azd-core version once available
- [x] Tests/CI in azd-rest enforce the pinned azd-core version

**Progress (2026-01-15)**:
- [x] Tracking issue content ready to file in `jongio/azd-core` (title “Expose extension helper surface for azd-rest (token w/ expiry, context accessors, endpoints, UA/telemetry hook, redaction)”) with body documented in [docs/specs/azd-rest/spec.md](docs/specs/azd-rest/spec.md#L138-L156); needs manual filing due to repo access.
- [x] go.mod remains pinned to `github.com/jongio/azd-core v0.3.0` with local replace; guard test location is `cli/src/internal/azdcore/version_test.go`.
- [x] Version guard workflow now uses Go 1.26.0 with `GOWORK=off` and azd-core v0.3.0 checkout so the pin stays enforced in CI ([.github/workflows/version-guard.yml](.github/workflows/version-guard.yml)).
- [x] Guard test passes locally with `GOWORK=off GOFLAGS=-mod=mod go test ./src/internal/azdcore/...`.
