---
title: Contributing to azd-rest
description: Guidelines for contributing to the azd-rest project
lastUpdated: 2026-01-15
tags: [contributing, development, guidelines]
---

# Contributing to azd-rest

We love your input! We want to make contributing to azd-rest as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Development Setup

### Prerequisites

- Go 1.25.5 or later
- golangci-lint
- Node.js 20+ (for cspell)
- Git

### Setting Up Your Development Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/azd-rest.git
cd azd-rest/cli

# Install dependencies
go mod download

# Install mage (build tool)
go install github.com/magefile/mage@latest

# Enable azd extensions
azd config set alpha.extension.enabled on

# Build and install (will mirror azd-exec flow)
mage install

# Verify installation
azd rest version

# Run tests
mage test

# Run all quality checks
mage preflight
```

## Code Style

### Go Code

- Follow the standard Go formatting guidelines (use `gofmt` and `goimports`)
- Write meaningful comments for exported functions and types
- All comments should end with a period
- Keep functions focused and concise
- Write tests for new functionality

### Running Linters

```bash
cd cli

# Format code
mage fmt

# Run all linters
mage lint

# Run spell checker (from root)
cd ..
cspell "**/*.{go,md,yaml,yml}" --config cspell.json
```

## Testing

### Unit Tests

```bash
cd cli

# Run unit tests (fast, <5s)
mage test

# Run with verbose output
go test -v -short ./src/...
```

### Integration Tests

```bash
# Run all integration tests
mage testIntegration

# Run specific package
TEST_PACKAGE=auth mage testIntegration

# Run specific test
TEST_NAME=TestAuthIntegration mage testIntegration

# Run all tests (unit + integration)
mage testAll
```

### Running Tests with Coverage

```bash
# Generate coverage report
mage testCoverage

# View coverage in browser
# Windows
start coverage/coverage.html
# macOS
open coverage/coverage.html
# Linux
xdg-open coverage/coverage.html
```

## Pull Request Process

1. Update the README.md with details of changes to the interface, if applicable.
2. Update the CHANGELOG.md with notes on your changes.
3. The PR will be merged once you have the sign-off of at least one maintainer.

## Pull Request Guidelines

- **Keep it small**: Small, focused PRs are easier to review and merge.
- **Write good commit messages**: Follow [Conventional Commits](https://www.conventionalcommits.org/) (feat:, fix:, docs:, etc.)
- **Add tests**: Ensure your changes are covered by tests (>=80% coverage).
- **Update documentation**: If you change functionality, update the docs.
- **Follow the code style**: Make sure your code passes all linters.
- **Run preflight**: Execute `mage preflight` before submitting.

### Quality Gates (Must Pass)

Before submitting a PR:

```bash
# Run all quality checks
mage preflight
```

This ensures:
- ✅ Code formatting (`go fmt`)
- ✅ Linting (`go vet`, `golangci-lint`)
- ✅ Unit tests (100% pass)
- ✅ Integration tests (100% pass)
- ✅ Coverage report (>=80%)

## Commit Message Guidelines

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line

## Any Contributions You Make Will Be Under the MIT License

In short, when you submit code changes, your submissions are understood to be under the same [MIT License](LICENSE) that covers the project. Feel free to contact the maintainers if that's a concern.

## Report Bugs Using GitHub Issues

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/jongio/azd-rest/issues/new).

### Write Bug Reports with Detail, Background, and Sample Code

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## Feature Requests

We welcome feature requests! Please create an issue describing:

- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered
- Additional context

## Code of Conduct

### Our Pledge

In the interest of fostering an open and welcoming environment, we as contributors and maintainers pledge to make participation in our project and our community a harassment-free experience for everyone.

### Our Standards

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

## License

By contributing, you agree that your contributions will be licensed under its MIT License.

## Questions?

Feel free to open an issue with your question or contact the maintainers directly.

## Thank You!

Your contributions to open source, large or small, make projects like this possible. Thank you for taking the time to contribute.
