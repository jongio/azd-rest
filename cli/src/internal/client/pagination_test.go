package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagination_NextLinkInBody(t *testing.T) {
	pageCount := 0
	var serverURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		switch pageCount {
		case 1:
			response := map[string]interface{}{
				"value": []interface{}{
					map[string]interface{}{"id": "1", "name": "item1"},
					map[string]interface{}{"id": "2", "name": "item2"},
				},
				"nextLink": serverURL + "?page=2",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case 2:
			response := map[string]interface{}{
				"value": []interface{}{
					map[string]interface{}{"id": "3", "name": "item3"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	serverURL = server.URL

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL,
		SkipAuth: true,
		Paginate: true,
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)

	var data map[string]interface{}
	err = json.Unmarshal(resp.Body, &data)
	require.NoError(t, err)

	valueArray, ok := data["value"].([]interface{})
	require.True(t, ok, "Response should have 'value' array")

	// Verify pagination collected all items from both pages.
	assert.Equal(t, 3, len(valueArray), "Pagination should aggregate items from all pages (2 from page 1, 1 from page 2)")

	// Verify the server was hit exactly twice (once per page).
	assert.Equal(t, 2, pageCount, "Server should have received exactly 2 requests (one per page)")

	// Verify nextLink is stripped from the final aggregated response.
	_, hasNextLink := data["nextLink"]
	assert.False(t, hasNextLink, "nextLink should be removed from the aggregated response")

	// Verify item ordering is preserved across pages.
	firstItem, ok := valueArray[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "1", firstItem["id"])

	lastItem, ok := valueArray[2].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "3", lastItem["id"])
}

func TestPagination_LinkHeader(t *testing.T) {
	pageCount := 0
	var serverURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		if pageCount == 1 {
			w.Header().Set("Link", `<`+serverURL+`?page=2>; rel="next"`)
			response := map[string]interface{}{
				"value": []interface{}{
					map[string]interface{}{"id": "1"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			response := map[string]interface{}{
				"value": []interface{}{
					map[string]interface{}{"id": "2"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	serverURL = server.URL

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL,
		SkipAuth: true,
		Paginate: true,
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)

	var data map[string]interface{}
	err = json.Unmarshal(resp.Body, &data)
	require.NoError(t, err)

	valueArray, ok := data["value"].([]interface{})
	require.True(t, ok)

	// Verify pagination followed the Link header and aggregated both pages.
	assert.Equal(t, 2, len(valueArray), "Pagination should aggregate items from both pages via Link header")

	// Verify server was hit twice.
	assert.Equal(t, 2, pageCount, "Server should have received exactly 2 requests for Link header pagination")

	// Verify item ordering.
	item1, ok := valueArray[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "1", item1["id"])

	item2, ok := valueArray[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2", item2["id"])
}

func TestPagination_NoPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{"id": "1"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := &auth.MockTokenProvider{Token: "test-token"}
	client := NewClient(provider, false, 30*time.Second)

	opts := RequestOptions{
		Method:   "GET",
		URL:      server.URL,
		SkipAuth: true,
		Paginate: false,
	}

	resp, err := client.Execute(context.Background(), opts)
	require.NoError(t, err)

	var data map[string]interface{}
	err = json.Unmarshal(resp.Body, &data)
	require.NoError(t, err)

	valueArray, ok := data["value"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, len(valueArray), "Should have only first page items")
}
