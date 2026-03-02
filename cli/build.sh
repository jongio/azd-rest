#!/bin/bash
# Build script called by azd x build
# This handles pre-build steps for the azd-rest extension

set -e

# Get the directory of the script (cli folder)
EXTENSION_DIR="$(cd "$(dirname "$0")" && pwd)"

# Change to the script directory
cd "$EXTENSION_DIR" || exit

# Helper function to kill extension processes
# This prevents "file in use" errors when azd x watch copies to ~/.azd/extensions/
stop_extension_processes() {
    EXTENSION_ID_FOR_KILL="jongio.azd.rest"
    EXTENSION_BINARY_PREFIX="${EXTENSION_ID_FOR_KILL//./-}"

    # Kill processes by name silently (ignore errors if not running)
    pkill -f "$EXTENSION_BINARY_PREFIX" 2>/dev/null || true

    # Also kill any processes running from the installed extension directory
    INSTALLED_EXT_DIR="$HOME/.azd/extensions/$EXTENSION_ID_FOR_KILL"
    if [ -d "$INSTALLED_EXT_DIR" ]; then
        pkill -f "$INSTALLED_EXT_DIR" 2>/dev/null || true
    fi

    # Give processes time to fully terminate and release file handles
    sleep 0.5
}

# Check if we need to rebuild the Go binary
NEEDS_GO_BUILD=false

if [ -d "bin" ]; then
    # Find newest binary (excluding .old files)
    NEWEST_BINARY_TIME=$(find bin -type f ! -name "*.old" -exec stat -c %Y {} \; 2>/dev/null | sort -n | tail -1 || \
                         find bin -type f ! -name "*.old" -exec stat -f %m {} \; 2>/dev/null | sort -n | tail -1)

    if [ -z "$NEWEST_BINARY_TIME" ]; then
        NEEDS_GO_BUILD=true
        echo "No existing binary found, will build"
    else
        # Check Go source files
        if [ -d "src" ]; then
            NEWEST_GO_TIME=$(find src -name "*.go" -type f -exec stat -c %Y {} \; 2>/dev/null | sort -n | tail -1 || \
                             find src -name "*.go" -type f -exec stat -f %m {} \; 2>/dev/null | sort -n | tail -1)
            if [ -n "$NEWEST_GO_TIME" ] && [ "$NEWEST_GO_TIME" -gt "$NEWEST_BINARY_TIME" ]; then
                NEEDS_GO_BUILD=true
                echo "Go source files changed, will rebuild"
            fi
        fi
    fi
else
    NEEDS_GO_BUILD=true
    echo "No bin directory found, will build"
fi

# Only kill extension processes if we're actually going to rebuild the binary
if [ "$NEEDS_GO_BUILD" = true ]; then
    echo "Stopping extension processes before rebuild..."
    stop_extension_processes
else
    # Nothing to rebuild - exit early to prevent azd x watch from trying to install
    echo "  ✓ Binary up to date, skipping build"
    exit 0
fi

echo "Building REST Extension..."

# Create a safe version of EXTENSION_ID replacing dots with dashes
EXTENSION_ID_SAFE="${EXTENSION_ID//./-}"

# Define output directory
OUTPUT_DIR="${OUTPUT_DIR:-$EXTENSION_DIR/bin}"

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Get Git commit hash and build date
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Read version from extension.yaml if EXTENSION_VERSION not set
if [ -z "$EXTENSION_VERSION" ]; then
    if [ -f "extension.yaml" ]; then
        EXTENSION_VERSION=$(grep -E '^version:' extension.yaml | awk '{print $2}' | tr -d '[:space:]')
        if [ -z "$EXTENSION_VERSION" ]; then
            EXTENSION_VERSION="0.0.0-dev"
        fi
    else
        EXTENSION_VERSION="0.0.0-dev"
    fi
fi

echo "Building version $EXTENSION_VERSION"

# List of OS and architecture combinations
if [ -n "$EXTENSION_PLATFORM" ]; then
    PLATFORMS=("$EXTENSION_PLATFORM")
else
    PLATFORMS=(
        "windows/amd64"
        "windows/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "linux/amd64"
        "linux/arm64"
    )
fi

VERSION_PKG="github.com/jongio/azd-rest/src/internal/version"

# Loop through platforms and build
for PLATFORM in "${PLATFORMS[@]}"; do
    OS=$(echo "$PLATFORM" | cut -d'/' -f1)
    ARCH=$(echo "$PLATFORM" | cut -d'/' -f2)

    OUTPUT_NAME="$OUTPUT_DIR/$EXTENSION_ID_SAFE-$OS-$ARCH"

    if [ "$OS" = "windows" ]; then
        OUTPUT_NAME+='.exe'
    fi

    echo "  Building for $OS/$ARCH..."

    # Delete the output file if it already exists
    [ -f "$OUTPUT_NAME" ] && rm -f "$OUTPUT_NAME"

    LDFLAGS="-s -w -X '$VERSION_PKG.Version=$EXTENSION_VERSION' -X '$VERSION_PKG.BuildDate=$BUILD_DATE' -X '$VERSION_PKG.GitCommit=$COMMIT'"

    # Set environment variables for Go build
    CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build \
        -ldflags="$LDFLAGS" \
        -o "$OUTPUT_NAME" \
        ./src/cmd/rest

    if [ $? -ne 0 ]; then
        echo "ERROR: Build failed for $OS/$ARCH"
        exit 1
    fi
done

# Kill extension processes again right before azd x build copies to ~/.azd/extensions/
# This prevents "file in use" errors during the install step
stop_extension_processes

echo ""
echo "✓ Build completed successfully!"
echo "  Binaries are located in the $OUTPUT_DIR directory."
