# azd-rest CLI Extension

Azure Developer CLI extension for executing REST APIs with automatic authentication and context integration.

## Building

```bash
go build -o rest ./src/cmd/rest
```

## Testing

```bash
go test -v ./...
```

## Running Locally

```bash
# Build
go build -o rest ./src/cmd/rest

# Run
./rest get https://api.github.com/repos/jongio/azd-rest
```

## Project Structure

```
cli/
├── src/
│   ├── cmd/rest/          # Main entry point
│   └── internal/
│       ├── cmd/           # Command implementations
│       ├── client/        # HTTP client logic
│       ├── context/       # Azure context integration
│       └── formatter/     # Response formatting
├── extension.yaml         # Extension metadata
├── go.mod                 # Go module
└── go.sum                 # Go dependencies
```

## For Development & Testing

### Install Locally

```bash
# Build
go build -o rest ./src/cmd/rest

# Copy to azd bin directory (Unix/Mac)
cp rest ~/.azd/bin/

# Copy to azd bin directory (Windows)
copy rest.exe %USERPROFILE%\.azd\bin\
```

### Enable azd Extensions

```bash
azd config set alpha.extension.enabled on
```

### Test the Extension

```bash
# Test basic GET
azd rest get https://api.github.com/repos/jongio/azd-rest

# Test with verbose output
azd rest get https://api.github.com/repos/jongio/azd-rest -v

# Test POST with data
azd rest post https://httpbin.org/post --data '{"test":"data"}'
```

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## License

MIT
