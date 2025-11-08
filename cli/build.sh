#!/bin/bash
set -e

# Build script for azd-rest extension
VERSION=${EXTENSION_VERSION:-"0.1.0"}
BINARY_NAME="azd-rest"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
esac

OUTPUT_DIR="./bin"
mkdir -p "$OUTPUT_DIR"

echo "Building $BINARY_NAME version $VERSION for $OS/$ARCH..."

# Build the binary
if [ "$OS" = "windows" ]; then
    GOOS="$OS" GOARCH="$ARCH" go build -o "$OUTPUT_DIR/${BINARY_NAME}.exe" \
        -ldflags "-X main.version=$VERSION" \
        ./src/cmd/rest
else
    GOOS="$OS" GOARCH="$ARCH" go build -o "$OUTPUT_DIR/$BINARY_NAME" \
        -ldflags "-X main.version=$VERSION" \
        ./src/cmd/rest
fi

echo "Build complete: $OUTPUT_DIR/$BINARY_NAME"
