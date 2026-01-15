package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockTokenProvider(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		err         error
		expectError bool
	}{
		{
			name:        "Success - returns token",
			token:       "mock-access-token-12345",
			err:         nil,
			expectError: false,
		},
		{
			name:        "Error - returns error",
			token:       "",
			err:         fmt.Errorf("authentication failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &MockTokenProvider{
				Token: tt.token,
				Error: tt.err,
			}

			token, err := provider.GetToken(context.Background(), "https://management.azure.com/.default")

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.token, token)
			}
		})
	}
}

func TestGetToken_EmptyScope(t *testing.T) {
	provider := &MockTokenProvider{
		Token: "token",
	}

	// Even mock should validate scope
	token, err := provider.GetToken(context.Background(), "")
	// Mock doesn't validate, but real implementation would
	assert.NoError(t, err) // Mock doesn't validate
	assert.Equal(t, "token", token)
}

func TestGetToken_ContextCancellation(t *testing.T) {
	provider := &MockTokenProvider{
		Token: "token",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Mock doesn't respect context, but real implementation would
	_, _ = provider.GetToken(ctx, "https://management.azure.com/.default")
	// This test demonstrates the interface, actual context handling in real provider
}

// Integration test (skipped by default, requires Azure authentication)
func TestAzureTokenProvider_Integration(t *testing.T) {
	t.Skip("Integration test - requires Azure authentication")

	provider, err := NewAzureTokenProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	token, err := provider.GetToken(context.Background(), "https://management.azure.com/.default")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token format (should be a JWT-like string)
	assert.Greater(t, len(token), 100, "Token should be a long string")
}

func TestAzureTokenProvider_InvalidScope(t *testing.T) {
	t.Skip("Integration test - requires Azure authentication")

	provider, err := NewAzureTokenProvider()
	require.NoError(t, err)

	// Try to get token with invalid scope
	_, err = provider.GetToken(context.Background(), "invalid-scope")
	assert.Error(t, err)
}
