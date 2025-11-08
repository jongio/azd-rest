#!/usr/bin/env pwsh
# Build script called by azd x build
# This handles pre-build steps for the azd-rest extension

$ErrorActionPreference = 'Stop'

# Get the directory of the script
$EXTENSION_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path

# Change to the script directory
Set-Location -Path $EXTENSION_DIR

# Helper function to kill extension processes
function Stop-ExtensionProcesses {
    $extensionId = "jongio.azd.rest"
    $extensionBinaryPrefix = $extensionId -replace '\.', '-'

    # Kill extension binaries by their distinctive name
    foreach ($arch in @("windows-amd64", "windows-arm64")) {
        $procName = "$extensionBinaryPrefix-$arch"
        Get-Process -Name $procName -ErrorAction SilentlyContinue | ForEach-Object {
            Write-Host "  Stopping process: $($_.Name) (PID: $($_.Id))" -ForegroundColor Gray
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
        }
    }
    
    # Also kill any processes running from the installed extension directory
    $installedExtensionDir = Join-Path $env:USERPROFILE ".azd\extensions\$extensionId"
    if (Test-Path $installedExtensionDir) {
        Get-Process | Where-Object { 
            $_.Path -and $_.Path.StartsWith($installedExtensionDir) 
        } | ForEach-Object {
            Write-Host "  Stopping process: $($_.Name) (PID: $($_.Id))" -ForegroundColor Gray
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
        }
    }
    
    Start-Sleep -Milliseconds 500
}

# Check if we need to rebuild the Go binary
$needsGoBuild = $false
$existingBinaries = Get-ChildItem -Path "bin" -Filter "*.exe" -ErrorAction SilentlyContinue | Where-Object { $_.Name -notlike "*.old" }

if (-not $existingBinaries) {
    $needsGoBuild = $true
    Write-Host "No existing binary found, will build" -ForegroundColor Yellow
} else {
    $newestBinary = $existingBinaries | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    $binaryTime = $newestBinary.LastWriteTime
    
    # Check Go source files
    $goFiles = Get-ChildItem -Path "src" -Recurse -Filter "*.go" -ErrorAction SilentlyContinue
    if ($goFiles) {
        $newestGoFile = $goFiles | Sort-Object LastWriteTime -Descending | Select-Object -First 1
        if ($newestGoFile.LastWriteTime -gt $binaryTime) {
            $needsGoBuild = $true
            Write-Host "Go source files changed, will rebuild" -ForegroundColor Yellow
        }
    }
}

if ($needsGoBuild) {
    Write-Host "Stopping extension processes before rebuild..." -ForegroundColor Yellow
    Stop-ExtensionProcesses
} else {
    Write-Host "  ✓ Binary up to date, skipping build" -ForegroundColor Green
    exit 0
}

Write-Host "Building REST Extension..." -ForegroundColor Cyan

# Create a safe version of EXTENSION_ID replacing dots with dashes
$EXTENSION_ID_SAFE = $env:EXTENSION_ID -replace '\.', '-'

# Define output directory
$OUTPUT_DIR = if ($env:OUTPUT_DIR) { $env:OUTPUT_DIR } else { Join-Path $EXTENSION_DIR "bin" }

# Create output directory if it doesn't exist
if (-not (Test-Path -Path $OUTPUT_DIR)) {
    New-Item -ItemType Directory -Path $OUTPUT_DIR | Out-Null
}

# Get Git commit hash and build date
try {
    $COMMIT = git rev-parse HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { $COMMIT = "unknown" }
} catch {
    $COMMIT = "unknown"
}
$BUILD_DATE = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")

# Read version from extension.yaml if EXTENSION_VERSION not set
if (-not $env:EXTENSION_VERSION) {
    if (Test-Path "extension.yaml") {
        $yamlContent = Get-Content "extension.yaml" -Raw
        if ($yamlContent -match 'version:\s*(\S+)') {
            $env:EXTENSION_VERSION = $matches[1]
        } else {
            $env:EXTENSION_VERSION = "0.0.0-dev"
        }
    } else {
        $env:EXTENSION_VERSION = "0.0.0-dev"
    }
}

Write-Host "Building version $env:EXTENSION_VERSION" -ForegroundColor Cyan

# List of OS and architecture combinations
if ($env:EXTENSION_PLATFORM) {
    $PLATFORMS = @($env:EXTENSION_PLATFORM)
}
else {
    $PLATFORMS = @(
        "windows/amd64",
        "windows/arm64",
        "darwin/amd64",
        "darwin/arm64",
        "linux/amd64",
        "linux/arm64"
    )
}

$VERSION_PKG = "github.com/jongio/azd-rest/src/internal/version"

# Loop through platforms and build
foreach ($PLATFORM in $PLATFORMS) {
    $OS, $ARCH = $PLATFORM -split '/'

    $OUTPUT_NAME = Join-Path $OUTPUT_DIR "$EXTENSION_ID_SAFE-$OS-$ARCH"

    if ($OS -eq "windows") {
        $OUTPUT_NAME += ".exe"
    }

    Write-Host "  Building for $OS/$ARCH..." -ForegroundColor Gray

    # Handle locked files on Windows by renaming instead of deleting
    if (Test-Path -Path $OUTPUT_NAME) {
        $backupName = "$OUTPUT_NAME.old"
        try {
            if (Test-Path -Path $backupName) {
                Remove-Item -Path $backupName -Force -ErrorAction SilentlyContinue
            }
            Move-Item -Path $OUTPUT_NAME -Destination $backupName -Force -ErrorAction Stop
        } catch {
            Remove-Item -Path $OUTPUT_NAME -Force -ErrorAction SilentlyContinue
        }
    }

    # Set environment variables for Go build
    $env:GOOS = $OS
    $env:GOARCH = $ARCH
    $env:CGO_ENABLED = "0"

    $ldflags = "-s -w -X '$VERSION_PKG.Version=$env:EXTENSION_VERSION' -X '$VERSION_PKG.BuildDate=$BUILD_DATE' -X '$VERSION_PKG.GitCommit=$COMMIT'"

    go build `
        "-ldflags=$ldflags" `
        -o $OUTPUT_NAME `
        ./src/cmd/rest

    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Build failed for $OS/$ARCH" -ForegroundColor Red
        exit 1
    }
}

# Kill extension processes again right before azd x build copies to ~/.azd/extensions/
# This prevents "file in use" errors during the install step
Stop-ExtensionProcesses

Write-Host "`n✓ Build completed successfully!" -ForegroundColor Green
Write-Host "  Binaries are located in the $OUTPUT_DIR directory." -ForegroundColor Gray
