package client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jongio/azd-rest/cli/src/internal/context"
	"github.com/jongio/azd-rest/cli/src/internal/formatter"
)

// RequestConfig holds configuration for an HTTP request
type RequestConfig struct {
	Method      string
	URL         string
	Headers     []string
	Data        string
	DataFile    string
	ContentType string
	Output      string
	Verbose     bool
	Insecure    bool
	UseAzdAuth  bool
}

// ExecuteRequest performs an HTTP request with the given configuration.
//
// Behaviors:
//   - Automatically formats JSON responses with pretty-printing
//   - Returns errors for HTTP status codes >= 400
//   - Writes response to stdout or file based on config.Output
//   - Prints verbose information to stderr if config.Verbose is true
//   - Reads request body from config.Data or config.DataFile
//   - Adds Azure authentication headers if config.UseAzdAuth is true
//   - Validates that both Data and DataFile are not set simultaneously
//
// Returns an error for:
//   - Network failures
//   - Invalid HTTP status codes (>= 400)
//   - File read/write errors
//   - Configuration errors
func ExecuteRequest(config RequestConfig) error {
	// Validate configuration
	if config.Data != "" && config.DataFile != "" {
		return fmt.Errorf("cannot specify both --data and --data-file")
	}

	// Create HTTP client with configurable timeout
	timeout := 30 * time.Second
	if timeoutEnv := os.Getenv("AZD_REST_TIMEOUT"); timeoutEnv != "" {
		if parsed, err := time.ParseDuration(timeoutEnv); err == nil {
			timeout = parsed
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	if config.Insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// Prepare request body
	var body io.Reader
	if config.Data != "" {
		body = strings.NewReader(config.Data)
	} else if config.DataFile != "" {
		data, err := os.ReadFile(config.DataFile)
		if err != nil {
			return fmt.Errorf("failed to read data file: %w", err)
		}
		body = bytes.NewReader(data)
	}

	// Create request
	req, err := http.NewRequest(config.Method, config.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type for requests with body
	if body != nil && config.ContentType != "" {
		req.Header.Set("Content-Type", config.ContentType)
	}

	// Add custom headers
	for _, header := range config.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		} else {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Malformed header ignored (missing colon): %q\n", header)
			}
		}
	}

	// Add azd authentication if enabled
	if config.UseAzdAuth {
		token, err := context.GetAzdAuthToken()
		if err != nil {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to get azd auth token: %v\n", err)
			}
		} else if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	// Add azd context headers
	if azdContext, err := context.GetAzdContext(); err == nil {
		if azdContext.SubscriptionID != "" {
			req.Header.Set("X-Azd-Subscription-Id", azdContext.SubscriptionID)
		}
		if azdContext.Environment != "" {
			req.Header.Set("X-Azd-Environment", azdContext.Environment)
		}
	}

	if config.Verbose {
		fmt.Fprintf(os.Stderr, "> %s %s\n", config.Method, config.URL)
		for key, values := range req.Header {
			for _, value := range values {
				// Mask authorization tokens
				if key == "Authorization" {
					value = "Bearer ***"
				}
				fmt.Fprintf(os.Stderr, "> %s: %s\n", key, value)
			}
		}
		fmt.Fprintln(os.Stderr, ">")
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if config.Verbose {
		fmt.Fprintf(os.Stderr, "< HTTP %s\n", resp.Status)
		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Fprintf(os.Stderr, "< %s: %s\n", key, value)
			}
		}
		fmt.Fprintln(os.Stderr, "<")
	}

	// Check for error status before formatting/displaying
	if resp.StatusCode >= 400 {
		// Still format the error response for readability
		output := formatter.FormatResponse(respBody, resp.Header.Get("Content-Type"))

		if config.Output != "" {
			if err := os.WriteFile(config.Output, []byte(output), 0600); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Error response written to %s\n", config.Output)
			}
		} else {
			// Print error response to stderr instead of stdout
			fmt.Fprintln(os.Stderr, output)
		}

		return fmt.Errorf("request failed with status: %s", resp.Status)
	}

	// Format and output successful response
	output := formatter.FormatResponse(respBody, resp.Header.Get("Content-Type"))

	if config.Output != "" {
		if err := os.WriteFile(config.Output, []byte(output), 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Response written to %s\n", config.Output)
		}
	} else {
		fmt.Println(output)
	}

	return nil
}
