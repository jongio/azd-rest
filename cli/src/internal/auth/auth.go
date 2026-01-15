package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	tokenExpirySkew    = 2 * time.Minute
	defaultAuthTimeout = 30 * time.Second
)

// TokenProvider supplies OAuth bearer tokens for a given scope.
type TokenProvider interface {
	GetToken(ctx context.Context, scope string) (string, error)
}

type tokenCredential interface {
	GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error)
}

// AzureTokenProvider implements TokenProvider using azd-core's credential chain
// (DefaultAzureCredential-equivalent) with in-memory token reuse.
type AzureTokenProvider struct {
	credential tokenCredential
	cache      map[string]azcore.AccessToken
	mu         sync.RWMutex
	now        func() time.Time
	timeout    time.Duration
}

var (
	defaultProvider   TokenProvider
	providerOnce      sync.Once
	providerErr       error
	credentialFactory = func() (tokenCredential, error) {
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build Azure credential chain: %w", err)
		}
		return cred, nil
	}
	timeNow = time.Now
)

// NewAzureTokenProvider creates a provider backed by DefaultAzureCredential.
// The provider caches tokens per scope until close to expiration.
func NewAzureTokenProvider() (*AzureTokenProvider, error) {
	cred, err := credentialFactory()
	if err != nil {
		return nil, err
	}

	return &AzureTokenProvider{
		credential: cred,
		cache:      make(map[string]azcore.AccessToken),
		now:        timeNow,
		timeout:    defaultAuthTimeout,
	}, nil
}

// GetAzureToken acquires a bearer token for the supplied scope using the
// shared provider instance (cached credential and token reuse).
func GetAzureToken(scope string) (string, error) {
	provider, err := getDefaultProvider()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()

	return provider.GetToken(ctx, scope)
}

func getDefaultProvider() (TokenProvider, error) {
	providerOnce.Do(func() {
		defaultProvider, providerErr = NewAzureTokenProvider()
	})

	return defaultProvider, providerErr
}

// GetToken retrieves an access token for the specified scope with caching.
func (p *AzureTokenProvider) GetToken(ctx context.Context, scope string) (string, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "", fmt.Errorf("scope cannot be empty")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	if token, ok := p.getCached(scope); ok {
		return token, nil
	}

	accessToken, err := p.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		return "", classifyAuthError(scope, err)
	}

	p.setCached(scope, accessToken)
	return accessToken.Token, nil
}

func (p *AzureTokenProvider) getCached(scope string) (string, bool) {
	p.mu.RLock()
	token, ok := p.cache[scope]
	p.mu.RUnlock()

	if !ok || token.Token == "" || token.ExpiresOn.IsZero() {
		return "", false
	}

	if token.ExpiresOn.After(p.now().Add(tokenExpirySkew)) {
		return token.Token, true
	}

	return "", false
}

func (p *AzureTokenProvider) setCached(scope string, token azcore.AccessToken) {
	if token.Token == "" || token.ExpiresOn.IsZero() {
		return
	}

	p.mu.Lock()
	p.cache[scope] = token
	p.mu.Unlock()
}

func classifyAuthError(scope string, err error) error {
	lower := strings.ToLower(err.Error())

	switch {
	case strings.Contains(lower, "insufficient") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "permission"):
		return fmt.Errorf("authentication failed: insufficient permissions for scope %s: %w", scope, err)
	case strings.Contains(lower, "credential unavailable") ||
		strings.Contains(lower, "login") ||
		strings.Contains(lower, "no accounts") ||
		strings.Contains(lower, "authentication required") ||
		strings.Contains(lower, "configure"):
		return fmt.Errorf("authentication failed: not logged in or credential unavailable. Run 'az login' or configure managed identity/environment credentials: %w", err)
	default:
		return fmt.Errorf("authentication failed for scope %s: %w", scope, err)
	}
}

// MockTokenProvider is a mock implementation for testing
type MockTokenProvider struct {
	Token string
	Error error
}

// GetToken returns the mock token or error
func (m *MockTokenProvider) GetToken(ctx context.Context, scope string) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	return m.Token, nil
}
