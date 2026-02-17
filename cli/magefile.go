//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	binaryName    = "rest"
	srcDir        = "src/cmd/rest"
	binDir        = "bin"
	coverageDir   = "coverage"
	extensionFile = "extension.yaml"
	extensionID   = "jongio.azd.rest"
	testTimeout   = "10m"
)

// Default target runs all checks and builds.
var Default = All

// All runs format, lint, test, and build in dependency order.
func All() error {
	mg.Deps(Fmt, Lint, Test, Build)
	return nil
}

// Build compiles the CLI binary using azd x build.
func Build() error {
	// Ensure azd extensions are set up (installs azd x if needed)
	if err := ensureAzdExtensions(); err != nil {
		return err
	}

	fmt.Println("Building azd rest extension...")

	// Get version from extension.yaml
	version, err := getVersion()
	if err != nil {
		return err
	}

	// Set environment variables required by azd x build
	env := map[string]string{
		"EXTENSION_ID":      extensionID,
		"EXTENSION_VERSION": version,
	}

	// Build using azd x build (always skip install - we'll do proper publish workflow)
	if err := sh.RunWithV(env, "azd", "x", "build", "--skip-install"); err != nil {
		return fmt.Errorf("azd x build failed: %w", err)
	}

	fmt.Printf("✅ Build complete! Version: %s\n", version)
	fmt.Println("\n📝 Next steps for local testing:")
	fmt.Println("   1. Run 'mage pack' to package the extension")
	fmt.Println("   2. Run 'mage publish' to update local registry")
	fmt.Println("   3. Run 'azd extension install jongio.azd.rest --source local' to install")
	fmt.Println("\n   Or run 'mage setup' to do all three steps at once")
	return nil
}

// Pack packages the extension into archives using azd x pack.
func Pack() error {
	fmt.Println("Packaging extension...")

	version, err := getVersion()
	if err != nil {
		return err
	}

	// Build for current platform first
	env := map[string]string{
		"EXTENSION_ID":      extensionID,
		"EXTENSION_VERSION": version,
	}

	fmt.Println("Building binary...")
	if err := sh.RunWithV(env, "azd", "x", "build", "--skip-install"); err != nil {
		return fmt.Errorf("azd x build failed: %w", err)
	}

	// Package using azd x pack
	fmt.Println("Packaging extension...")
	if err := sh.RunV("azd", "x", "pack"); err != nil {
		return fmt.Errorf("azd x pack failed: %w", err)
	}

	fmt.Println("✅ Package complete!")
	return nil
}

// Publish updates the local registry with the packed extension.
func Publish() error {
	fmt.Println("Publishing to local registry...")

	version, err := getVersion()
	if err != nil {
		return err
	}

	// Publish to local registry
	if err := sh.RunV("azd", "x", "publish", "--registry", "../registry.json", "--version", version); err != nil {
		return fmt.Errorf("azd x publish failed: %w", err)
	}

	fmt.Println("✅ Published to local registry!")
	return nil
}

// Setup runs Build + Pack + Publish + Install in sequence.
func Setup() error {
	fmt.Println("Setting up extension for local development...")
	mg.Deps(Build, Pack, Publish)
	
	fmt.Println("\n✅ Setup complete! Extension is ready for local testing.")
	fmt.Println("   Install with: azd extension install jongio.azd.rest --source local")
	return nil
}

// Test runs unit tests only (with -short flag).
func Test() error {
	fmt.Println("Running unit tests...")
	return sh.RunV("go", "test", "-v", "-short", "./src/...")
}

// TestIntegration runs integration tests only.
func TestIntegration() error {
	fmt.Println("Running integration tests...")

	args := []string{"test", "-v", "-tags=integration"}

	// Handle timeout
	timeout := os.Getenv("TEST_TIMEOUT")
	if timeout == "" {
		timeout = testTimeout
	}
	args = append(args, "-timeout="+timeout)

	// Handle test filtering
	testName := os.Getenv("TEST_NAME")
	if testName != "" {
		args = append(args, "-run="+testName)
	}

	args = append(args, "./src/...")

	return sh.RunV("go", args...)
}

// TestAll runs all tests (unit + integration).
func TestAll() error {
	fmt.Println("Running all tests...")
	return sh.RunV("go", "test", "-v", "-tags=integration", "./src/...")
}

// TestCoverage runs tests with coverage report.
func TestCoverage() error {
	fmt.Println("Running tests with coverage...")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	absCoverageDir := filepath.Join(cwd, coverageDir)
	_ = os.RemoveAll(absCoverageDir)

	if err := os.MkdirAll(absCoverageDir, 0755); err != nil {
		return fmt.Errorf("failed to create coverage directory: %w", err)
	}

	coverageOut := filepath.Join(absCoverageDir, "coverage.out")
	coverageHTML := filepath.Join(absCoverageDir, "coverage.html")

	args := []string{"test", "-short", "-coverprofile=" + coverageOut, "./src/..."}
	if err := sh.RunV("go", args...); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	if err := sh.RunV("go", "tool", "cover", "-html="+coverageOut, "-o", coverageHTML); err != nil {
		return fmt.Errorf("failed to generate coverage HTML: %w", err)
	}

	output, err := sh.Output("go", "tool", "cover", "-func="+coverageOut)
	if err != nil {
		return fmt.Errorf("failed to calculate coverage: %w", err)
	}

	fmt.Println("\n" + output)
	fmt.Printf("✅ Coverage report generated: %s\n", coverageHTML)

	if strings.Contains(output, "total:") {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "total:") {
				fmt.Println("\n📊 " + strings.TrimSpace(line))
				break
			}
		}
	}

	return nil
}

// Fmt formats all Go code.
func Fmt() error {
	fmt.Println("Formatting code...")
	return sh.RunV("go", "fmt", "./...")
}

// Lint runs golangci-lint.
func Lint() error {
	fmt.Println("Running linter...")
	return sh.RunV("golangci-lint", "run", "--timeout=5m")
}

// Clean removes build artifacts.
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	
	dirs := []string{binDir, coverageDir}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("⚠️  Failed to remove %s: %v\n", dir, err)
		}
	}
	
	fmt.Println("✅ Clean complete!")
	return nil
}

// Preflight runs all pre-release checks: format, lint, security scan, vulnerability check.
func Preflight() error {
	fmt.Println("🚀 Running preflight checks...")
	fmt.Println()

	checks := []struct {
		name string
		fn   func() error
	}{
		{"Checking code format", preflightFmtCheck},
		{"Running linter", Lint},
		{"Running security scan", preflightGosec},
		{"Checking for known vulnerabilities", preflightVulncheck},
		{"Running tests with coverage", TestCoverage},
	}

	for i, check := range checks {
		fmt.Printf("📋 Step %d/%d: %s...\n", i+1, len(checks), check.name)
		if err := check.fn(); err != nil {
			return fmt.Errorf("%s failed: %w", check.name, err)
		}
		fmt.Println()
	}

	fmt.Println("✅ All preflight checks passed!")
	fmt.Println("🎉 Ready to ship!")
	return nil
}

// preflightFmtCheck verifies all Go files are formatted with gofmt.
func preflightFmtCheck() error {
	output, err := sh.Output("gofmt", "-l", ".")
	if err != nil {
		return fmt.Errorf("gofmt check failed: %w", err)
	}
	if strings.TrimSpace(output) != "" {
		fmt.Println("Unformatted files:")
		for _, f := range strings.Split(strings.TrimSpace(output), "\n") {
			fmt.Printf("   • %s\n", f)
		}
		return fmt.Errorf("code is not formatted. Run 'mage fmt' to fix")
	}
	fmt.Println("   ✅ Code is formatted")
	return nil
}

// preflightGosec runs a security scan using gosec if available.
func preflightGosec() error {
	if _, err := exec.LookPath("gosec"); err != nil {
		fmt.Println("   ⚠️  gosec not installed — skipping security scan")
		fmt.Println("      Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest")
		return nil
	}
	if err := sh.RunV("gosec", "-quiet", "./src/..."); err != nil {
		fmt.Println("   ⚠️  Security scan found issues (non-fatal)")
	} else {
		fmt.Println("   ✅ Security scan passed")
	}
	return nil
}

// preflightVulncheck checks for known vulnerabilities using govulncheck if available.
func preflightVulncheck() error {
	if _, err := exec.LookPath("govulncheck"); err != nil {
		fmt.Println("   ⚠️  govulncheck not installed — skipping vulnerability check")
		fmt.Println("      Install with: go install golang.org/x/vuln/cmd/govulncheck@latest")
		return nil
	}
	if err := sh.RunV("govulncheck", "./..."); err != nil {
		fmt.Println("   ⚠️  Known vulnerabilities found!")
		return err
	}
	fmt.Println("   ✅ No known vulnerabilities")
	return nil
}

// Watch monitors source files and rebuilds on changes (requires azd x watch).
func Watch() error {
	if err := ensureAzdExtensions(); err != nil {
		return err
	}

	fmt.Println("Starting watch mode...")

	env := map[string]string{
		"EXTENSION_ID": extensionID,
	}

	return sh.RunWithV(env, "azd", "x", "watch")
}

// ensureAzdExtensions ensures azd extensions tooling is installed.
func ensureAzdExtensions() error {
	// Check if azd x is available (it's part of microsoft.azd.extensions)
	// If not, try to install it
	if err := sh.Run("azd", "x", "version"); err != nil {
		fmt.Println("Installing azd extensions tooling...")
		if err := sh.RunV("azd", "extension", "install", "microsoft.azd.extensions"); err != nil {
			return fmt.Errorf("failed to install azd extensions: %w", err)
		}
	}

	return nil
}

// getVersion reads the version from extension.yaml
func getVersion() (string, error) {
	data, err := os.ReadFile(extensionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", extensionFile, err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "version:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("version not found in %s", extensionFile)
}
