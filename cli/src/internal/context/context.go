package context

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AzdContext holds Azure Developer CLI context information
type AzdContext struct {
	SubscriptionID string
	TenantID       string
	Environment    string
	Location       string
}

// GetAzdContext retrieves the current azd context
func GetAzdContext() (*AzdContext, error) {
	ctx := &AzdContext{}

	// Try to get environment name from azd
	if env, err := getAzdEnvironment(); err == nil {
		ctx.Environment = env
	}

	// Try to get subscription from environment or azd config
	if sub := os.Getenv("AZURE_SUBSCRIPTION_ID"); sub != "" {
		ctx.SubscriptionID = sub
	}

	if tenant := os.Getenv("AZURE_TENANT_ID"); tenant != "" {
		ctx.TenantID = tenant
	}

	if location := os.Getenv("AZURE_LOCATION"); location != "" {
		ctx.Location = location
	}

	return ctx, nil
}

// GetAzdAuthToken retrieves the authentication token from azd
func GetAzdAuthToken() (string, error) {
	// Try to get token from azd auth token command
	cmd := exec.Command("azd", "auth", "token", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		// If azd command fails, try to read from environment
		if token := os.Getenv("AZURE_ACCESS_TOKEN"); token != "" {
			return token, nil
		}
		return "", fmt.Errorf("failed to get azd auth token: %w", err)
	}

	var result struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		// Try to use output as raw token
		return strings.TrimSpace(string(output)), nil
	}

	return result.Token, nil
}

// getAzdEnvironment gets the current azd environment name
func getAzdEnvironment() (string, error) {
	// First, check for environment variable
	if env := os.Getenv("AZURE_ENV_NAME"); env != "" {
		return env, nil
	}

	// Try to read from .azure directory
	configPath := filepath.Join(".azure", "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var config struct {
			DefaultEnvironment string `json:"defaultEnvironment"`
		}
		if err := json.Unmarshal(data, &config); err == nil && config.DefaultEnvironment != "" {
			return config.DefaultEnvironment, nil
		}
	}

	// Try using azd env list
	cmd := exec.Command("azd", "env", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get azd environment: %w", err)
	}

	var envs []struct {
		Name      string `json:"name"`
		IsDefault bool   `json:"isDefault"`
	}

	if err := json.Unmarshal(output, &envs); err != nil {
		return "", err
	}

	for _, env := range envs {
		if env.IsDefault {
			return env.Name, nil
		}
	}

	if len(envs) > 0 {
		return envs[0].Name, nil
	}

	return "", fmt.Errorf("no azd environment found")
}

// GetEnvironmentVariables returns all azd-related environment variables
func GetEnvironmentVariables() map[string]string {
	vars := make(map[string]string)

	prefixes := []string{"AZURE_", "AZD_"}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				vars[key] = value
				break
			}
		}
	}

	return vars
}
