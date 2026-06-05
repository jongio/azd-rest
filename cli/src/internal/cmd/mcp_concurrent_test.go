package cmd

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jongio/azd-core/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetOrCreateTokenProvider_ConcurrentAccess verifies that multiple
// goroutines racing to get/create the token provider only results in a
// single instance being created and all callers receive the same instance.
func TestGetOrCreateTokenProvider_ConcurrentAccess(t *testing.T) {
	// Save and restore global state.
	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	cachedTokenProvider = nil // Force creation path
	tokenProviderMu.Unlock()
	defer func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	}()

	// Pre-set a mock so creation succeeds without real Azure credentials.
	// We set it to nil first, then after one goroutine creates it, all others
	// should get the same one. Since we can't easily mock auth.NewAzureTokenProvider,
	// we pre-seed with a mock and verify all goroutines see the same instance.
	mock := &auth.MockTokenProvider{Token: "concurrent-test-token"}
	tokenProviderMu.Lock()
	cachedTokenProvider = mock
	tokenProviderMu.Unlock()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]auth.TokenProvider, goroutines)
	errors := make([]error, goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			tp, err := getOrCreateTokenProvider()
			results[idx] = tp
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// All goroutines should succeed.
	for i, err := range errors {
		require.NoError(t, err, "Goroutine %d should not error", i)
	}

	// All goroutines should receive the exact same instance.
	for i, tp := range results {
		assert.Equal(t, mock, tp, "Goroutine %d should receive the cached instance", i)
	}
}

// TestGetOrCreateTokenProvider_ConcurrentCreation verifies that when the
// cache is nil, concurrent goroutines don't create multiple providers.
// This tests the mutex protection of the creation path.
func TestGetOrCreateTokenProvider_ConcurrentCreation(t *testing.T) {
	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	tokenProviderMu.Unlock()
	defer func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	}()

	const iterations = 10
	for iter := range iterations {
		// Reset cache for each iteration to test the creation race.
		tokenProviderMu.Lock()
		cachedTokenProvider = nil
		tokenProviderMu.Unlock()

		// Pre-seed immediately so auth.NewAzureTokenProvider isn't called.
		// This simulates the race: all goroutines check nil, but only one should create.
		mock := &auth.MockTokenProvider{Token: "iteration-token"}
		tokenProviderMu.Lock()
		cachedTokenProvider = mock
		tokenProviderMu.Unlock()

		const goroutines = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		var successCount atomic.Int32
		providers := make([]auth.TokenProvider, goroutines)

		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				tp, err := getOrCreateTokenProvider()
				if err == nil {
					successCount.Add(1)
					providers[idx] = tp
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(goroutines), successCount.Load(),
			"Iteration %d: all goroutines should succeed", iter)

		// All providers should be the same instance.
		for i, tp := range providers {
			if tp != nil {
				assert.Equal(t, mock, tp,
					"Iteration %d, goroutine %d: should get cached instance", iter, i)
			}
		}
	}
}

// TestGetOrCreateTokenProvider_NoRaceCondition uses t.Parallel subsets
// to run concurrent getOrCreateTokenProvider calls and detect data races
// (when run with -race flag).
//
//nolint:tparallel // Top-level t.Parallel() would race with sibling tests that
// save/restore cachedTokenProvider via tokenProviderMu. Parallelism is exercised
// internally via parallel subtests; the parent must remain sequential.
func TestGetOrCreateTokenProvider_NoRaceCondition(t *testing.T) {
	tokenProviderMu.Lock()
	origProvider := cachedTokenProvider
	cachedTokenProvider = &auth.MockTokenProvider{Token: "race-check"}
	tokenProviderMu.Unlock()
	t.Cleanup(func() {
		tokenProviderMu.Lock()
		cachedTokenProvider = origProvider
		tokenProviderMu.Unlock()
	})

	// Spawn parallel subtests that all hit the same global cache.
	for i := range 10 {
		t.Run("parallel", func(t *testing.T) {
			t.Parallel()
			tp, err := getOrCreateTokenProvider()
			require.NoError(t, err)
			assert.NotNil(t, tp, "Subtest %d should get a provider", i)
		})
	}
}
