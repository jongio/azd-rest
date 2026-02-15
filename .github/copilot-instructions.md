1. Don't update AGENTS.md with project status. It should only contain info to help coding agents work with the project.
2. Don't ever execute git commands without permission.
3. When working in the CLI (Go) code, always run `mage preflight` from `cli/` before considering a task complete.
4. When working in the web (Astro) code, run `pnpm build` from `web/` to verify changes.

## Project Overview

azd-rest is an Azure Developer CLI (`azd`) extension for making authenticated REST API calls to Azure services. It automatically detects OAuth scopes, acquires tokens, and handles retries/pagination.

## Project Structure

```
cli/                  # Go CLI extension (Cobra + Mage)
  src/
    cmd/rest/         # CLI entry point
    internal/
      auth/           # Azure authentication & scope detection
      azdcore/        # azd integration layer
      client/         # HTTP client with retry, pagination, redirects
      cmd/            # Cobra command definitions (get, post, put, patch, delete, head, options)
      version/        # Version info
  magefile.go         # Mage build targets (build, test, lint, fmt, install, preflight)
  extension.yaml      # azd extension manifest
  go.mod / go.sum     # Go dependencies
web/                  # Documentation site (Astro + Tailwind CSS)
  src/                # Astro pages and components
  astro.config.mjs    # Astro configuration
  tailwind.config.ts  # Tailwind configuration
docs/specs/           # Design specifications
```

## Development Workflow

### CLI (Go)

```bash
cd cli
mage build           # Build the binary
mage test            # Run unit tests (fast, <5s)
mage testIntegration # Run integration tests (requires Azure auth)
mage testAll         # Run unit + integration tests
mage lint            # Run golangci-lint
mage fmt             # Format code (gofmt + goimports)
mage preflight       # Run all checks: fmt, lint, test, build
mage install         # Build and install as azd extension
```

### Web (Astro)

```bash
cd web
pnpm install          # Install dependencies
pnpm dev              # Start dev server
pnpm build            # Production build
```

### Root-Level

```bash
pnpm test             # Run all tests (CLI unit + integration + web e2e)
cspell "**/*.{go,md,yaml,yml}" --config cspell.json  # Spell check
```

## Coding Conventions

### Go

- Follow standard Go formatting (`gofmt`, `goimports`).
- All exported functions and types must have doc comments ending with a period.
- Keep functions focused and concise.
- Unit tests use `-short` flag; integration tests use `integration` build tag.
- Test files live alongside the code they test (`_test.go` suffix).
- Use table-driven tests where appropriate.

### Web (Astro/TypeScript)

- Use TypeScript for all non-Astro files.
- Follow Tailwind CSS utility-first patterns.
- Keep components small and focused.

## Key Dependencies

- **Go 1.25.5+** with Mage build tool
- **Cobra** for CLI command framework
- **Azure Identity SDK** (`azidentity`) for authentication
- **Node.js 20+** for cspell and web tooling
- **pnpm** as the package manager
