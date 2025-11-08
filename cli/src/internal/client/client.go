package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
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

// ExecuteRequest performs an HTTP request with the given configuration
func ExecuteRequest(config RequestConfig) error {
	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
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

	// Format and output response
	output := formatter.FormatResponse(respBody, resp.Header.Get("Content-Type"))

	if config.Output != "" {
		if err := os.WriteFile(config.Output, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Response written to %s\n", config.Output)
		}
	} else {
		fmt.Println(output)
	}

	// Return error if status is not successful
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status: %s", resp.Status)
	}

	return nil
}

// Response represents an HTTP response for JSON output
type Response struct {
	Status     string              `json:"status"`
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       interface{}         `json:"body"`
}

// FormatJSONResponse formats the response as JSON
func FormatJSONResponse(resp *http.Response, body []byte) (string, error) {
	var bodyData interface{}
	if err := json.Unmarshal(body, &bodyData); err != nil {
		bodyData = string(body)
	}

	response := Response{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       bodyData,
	}

	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output), nil
}
