# Contributing to azd-rest

Thank you for your interest in contributing to azd-rest! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions. We aim to foster an inclusive and welcoming environment.

## Getting Started

### Prerequisites

- Go 1.23 or later
- Git
- golangci-lint (for linting)
- cspell (for spell checking)

### Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/azd-rest.git
   cd azd-rest
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/jongio/azd-rest.git
   ```

4. Install dependencies:
   ```bash
   cd cli
   go mod download
   ```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `test/` - Test additions/updates

### 2. Make Changes

- Write clean, readable code
- Follow Go conventions and best practices
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
cd cli

# Run tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

### 4. Lint Your Code

```bash
cd cli

# Run golangci-lint
golangci-lint run

# Run spell check
cspell "**/*.{go,md,yaml,yml}"
```

### 5. Commit Your Changes

Write clear commit messages following conventional commits:

```bash
git commit -m "feat: add support for custom timeout"
git commit -m "fix: resolve authentication token refresh issue"
git commit -m "docs: update installation instructions"
```

Commit message format:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test additions/changes
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## Pull Request Guidelines

### Before Submitting

- [ ] Tests pass locally
- [ ] Linting passes
- [ ] Spell check passes
- [ ] Documentation updated
- [ ] Commit messages are clear
- [ ] Branch is up to date with main

### PR Description

Include:
- Summary of changes
- Motivation and context
- Related issues (if any)
- Testing performed
- Screenshots (for UI changes)

### Review Process

1. Automated checks run (CI pipeline)
2. Code review by maintainers
3. Address feedback
4. Approval and merge

## Coding Standards

### Go Style Guide

Follow standard Go conventions:

- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use meaningful variable names
- Keep functions focused and small
- Add comments for exported functions

### File Organization

```
cli/src/internal/
├── cmd/          # Command-line interface
├── client/       # HTTP client logic
├── context/      # Azure context handling
└── formatter/    # Response formatting
```

### Testing

- Write table-driven tests where appropriate
- Test both success and failure cases
- Mock external dependencies
- Aim for >80% code coverage

Example test structure:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Error Handling

- Return errors, don't panic
- Wrap errors with context
- Use meaningful error messages
- Handle errors at appropriate levels

```go
if err != nil {
    return fmt.Errorf("failed to read file: %w", err)
}
```

## Documentation

### Code Documentation

- Document all exported functions
- Use GoDoc format
- Include examples where helpful

```go
// ExecuteRequest performs an HTTP request with the given configuration.
// It returns an error if the request fails or the response status is >= 400.
func ExecuteRequest(config RequestConfig) error {
    // ...
}
```

### README Updates

Update README.md when:
- Adding new features
- Changing usage
- Adding new flags or commands
- Updating installation steps

### SPEC Updates

Update SPEC.md for:
- Architecture changes
- New features
- API changes
- Security considerations

## Testing

### Unit Tests

Located next to source files: `*_test.go`

```bash
# Run specific test
go test -v -run TestExecuteRequest

# Run with coverage
go test -v -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

### Integration Tests

Located in `cli/tests/` directory.

### Test Best Practices

- Test public APIs, not implementation
- Use meaningful test names
- Clean up test resources
- Use test fixtures when appropriate
- Mock external dependencies

## Security

### Reporting Security Issues

Do NOT open public issues for security vulnerabilities.

Email security concerns to the maintainers privately.

### Security Best Practices

- Never commit secrets or tokens
- Validate all inputs
- Use secure defaults
- Keep dependencies updated
- Follow OWASP guidelines

## Release Process

Releases are handled by maintainers:

1. Update version in `extension.yaml`
2. Update CHANGELOG.md
3. Create version tag: `git tag v0.x.0`
4. Push tag: `git push origin v0.x.0`
5. GitHub Actions creates release

## Getting Help

- Open an issue for bugs or feature requests
- Start a discussion for questions
- Check existing issues and discussions first

## Recognition

Contributors are recognized in:
- GitHub contributors page
- Release notes
- CHANGELOG.md (for significant contributions)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
