---
title: Testing Guide
description: Comprehensive testing guide covering unit, integration, and e2e tests
lastUpdated: 2026-01-09
tags: [testing, quality, automation]
---

# Testing Guide

This project includes comprehensive testing across multiple layers: unit tests, integration tests, and end-to-end (e2e) tests.

## Quick Start

Run all tests from the root:

```bash
pnpm test
```

This will execute:
1. CLI unit tests (Go)
2. CLI integration tests (Go)

To also run web e2e tests, use `pnpm test:all`.

## Test Structure

### CLI Tests (Go)

Location: `cli/src/`

**Unit Tests**
- Fast tests that don't require external dependencies
- Run with: `pnpm test:cli:unit` or `cd cli && go test -v -short ./src/...`
- Use `-short` flag to skip integration tests

**Integration Tests**
- Tests that may require external services or longer execution time
- Run with: `pnpm test:cli:integration` or `cd cli && go test -v -tags=integration -timeout=10m ./src/...`
- Require `integration` build tag

### Web Tests (Playwright)

Location: `web/tests/`

**E2E Tests**
- Browser-based end-to-end tests
- Run with: `pnpm test:web` or `cd web && pnpm test`
- Multi-browser testing (Chrome, Firefox, Safari)
- Responsive design testing (mobile, tablet, desktop)
- Accessibility (a11y) compliance tests

## Test Commands

### From Root Directory

```bash
# Run all tests
pnpm test

# Run only CLI tests (unit + integration)
pnpm test:cli

# Run only CLI unit tests
pnpm test:cli:unit

# Run only CLI integration tests
pnpm test:cli:integration

# Run only web e2e tests
pnpm test:web
```

### From CLI Directory

```bash
cd cli

# Using Mage (recommended)
mage test              # Unit tests only
mage testIntegration   # Integration tests only
mage testAll           # All tests
mage testCoverage      # Tests with coverage report

# Using Go directly
go test -v -short ./src/...                           # Unit tests
go test -v -tags=integration -timeout=10m ./src/...   # Integration tests
go test -v -tags=integration ./src/...                # All tests
```

### From Web Directory

```bash
cd web

# Run all tests
pnpm test

# Run with browser UI
pnpm test:headed

# Debug mode
pnpm test:debug

# View test report
pnpm test:report

# Run specific test file
pnpm exec playwright test homepage.spec.ts

# Run specific browser
pnpm exec playwright test --project=chromium
```

## Prerequisites

### CLI Tests
- Go 1.26.0 or later
- Optional: `golangci-lint` for linting

### Web Tests
- Node.js 20 or later
- pnpm 9 or later
- Playwright browsers (install with `pnpm exec playwright install`)

## Test Coverage

### CLI Coverage

Generate coverage report:

```bash
cd cli
mage testCoverage
```

This creates `cli/coverage/coverage.html` with a detailed coverage report.

### Continuous Integration

All tests run automatically in CI pipelines. The `pnpm test` command is designed to be CI-friendly:
- Exits with non-zero code on test failure
- Provides verbose output for debugging
- Web tests automatically start/stop dev server

## Adding New Tests

### CLI Unit Test
1. Create `*_test.go` file in appropriate package
2. Use `-short` flag checks for integration tests
3. Follow existing test patterns

### CLI Integration Test
1. Create `*_integration_test.go` file
2. Add `//go:build integration` build tag at top
3. May require external services/longer timeout

### Web E2E Test
1. Create `*.spec.ts` in `web/tests/`
2. Follow Playwright patterns
3. Test across multiple browsers
4. Include accessibility checks

## Troubleshooting

**CLI tests fail with "package not found"**
- Run `go mod download` in `cli/` directory

**Web tests fail with "browser not found"**
- Run `pnpm exec playwright install` in `web/` directory

**Integration tests timeout**
- Set `TEST_TIMEOUT` environment variable: `TEST_TIMEOUT=20m pnpm test:cli:integration`

**Specific package integration test**
- Set `TEST_PACKAGE` environment variable: `TEST_PACKAGE=client pnpm test:cli:integration`

**Specific test by name**
- Set `TEST_NAME` environment variable: `TEST_NAME=TestClient pnpm test:cli:integration`

## Best Practices

1. **Keep unit tests fast** - Mock external dependencies
2. **Mark integration tests** - Use build tags and skip in unit test runs
3. **Write descriptive test names** - Test names should explain what's being tested
4. **Test edge cases** - Include error scenarios and boundary conditions
5. **Maintain test independence** - Tests should not depend on execution order
6. **Update tests with code** - Keep tests in sync with implementation changes
