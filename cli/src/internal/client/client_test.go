package client

import (
"context"
"fmt"
"io"
"net/http"
"net/http/httptest"
"strings"
"testing"
"time"

"github.com/jongio/azd-rest/src/internal/auth"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestClient_Execute_GET(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
assert.Equal(t, "GET", r.Method)
assert.Equal(t, "/test", r.URL.Path)
w.WriteHeader(http.StatusOK)
w.Write([]byte(`{"message":"success"}`))
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:  "GET",
URL:     server.URL + "/test",
SkipAuth: true,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.Equal(t, http.StatusOK, resp.StatusCode)
assert.Equal(t, `{"message":"success"}`, string(resp.Body))
assert.Greater(t, resp.Duration, time.Duration(0))
}

func TestClient_Execute_POST_WithBody(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
assert.Equal(t, "POST", r.Method)

body, err := io.ReadAll(r.Body)
require.NoError(t, err)
assert.Equal(t, `{"data":"test"}`, string(body))

w.WriteHeader(http.StatusCreated)
w.Write([]byte(`{"id":"123"}`))
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:   "POST",
URL:      server.URL + "/create",
Body:     strings.NewReader(`{"data":"test"}`),
SkipAuth: true,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.Equal(t, http.StatusCreated, resp.StatusCode)
assert.Equal(t, `{"id":"123"}`, string(resp.Body))
}

func TestClient_Execute_WithAuthentication(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
authHeader := r.Header.Get("Authorization")
assert.Equal(t, "Bearer test-token", authHeader)

w.WriteHeader(http.StatusOK)
w.Write([]byte(`{"authenticated":true}`))
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:   "GET",
URL:      server.URL + "/secure",
Scope:    "https://management.azure.com/.default",
SkipAuth: false,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Execute_WithCustomHeaders(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

w.WriteHeader(http.StatusOK)
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:   "GET",
URL:      server.URL + "/test",
Headers: map[string]string{
"Content-Type":    "application/json",
"X-Custom-Header": "custom-value",
},
SkipAuth: true,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Execute_UserAgent(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
userAgent := r.Header.Get("User-Agent")
assert.Contains(t, userAgent, "azd-rest")

w.WriteHeader(http.StatusOK)
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:   "GET",
URL:      server.URL + "/test",
SkipAuth: true,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Execute_AuthenticationError(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
}))
defer server.Close()

provider := &auth.MockTokenProvider{
Error: fmt.Errorf("authentication failed"),
}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:   "GET",
URL:      server.URL + "/test",
Scope:    "https://management.azure.com/.default",
SkipAuth: false,
}

_, err := client.Execute(context.Background(), opts)

require.Error(t, err)
assert.Contains(t, err.Error(), "failed to get authentication token")
}

func TestClient_Execute_ContextCancellation(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
time.Sleep(100 * time.Millisecond)
w.WriteHeader(http.StatusOK)
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

ctx, cancel := context.WithCancel(context.Background())
cancel() // Cancel immediately

opts := RequestOptions{
Method:   "GET",
URL:      server.URL + "/test",
SkipAuth: true,
}

_, err := client.Execute(ctx, opts)

require.Error(t, err)
assert.Contains(t, err.Error(), "context canceled")
}

func TestClient_Execute_Timeout(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
time.Sleep(200 * time.Millisecond)
w.WriteHeader(http.StatusOK)
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 100*time.Millisecond)

opts := RequestOptions{
Method:   "GET",
URL:      server.URL + "/slow",
SkipAuth: true,
}

_, err := client.Execute(context.Background(), opts)

require.Error(t, err)
}

func TestClient_Execute_InvalidURL(t *testing.T) {
provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:   "GET",
URL:      "ht!tp://invalid url",
SkipAuth: true,
}

_, err := client.Execute(context.Background(), opts)

require.Error(t, err)
}

func TestShouldSkipAuth(t *testing.T) {
tests := []struct {
name     string
url      string
headers  map[string]string
skipAuth bool
expected bool
}{
{
name:     "Explicit skip flag",
url:      "https://management.azure.com/subscriptions",
headers:  map[string]string{},
skipAuth: true,
expected: true,
},
{
name:     "Authorization header present",
url:      "https://management.azure.com/subscriptions",
headers:  map[string]string{"Authorization": "Bearer token"},
skipAuth: false,
expected: true,
},
{
name:     "Authorization header case insensitive",
url:      "https://management.azure.com/subscriptions",
headers:  map[string]string{"authorization": "Bearer token"},
skipAuth: false,
expected: true,
},
{
name:     "HTTP URL",
url:      "http://example.com/api",
headers:  map[string]string{},
skipAuth: false,
expected: true,
},
{
name:     "HTTPS URL without skip or auth header",
url:      "https://management.azure.com/subscriptions",
headers:  map[string]string{},
skipAuth: false,
expected: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := ShouldSkipAuth(tt.url, tt.headers, tt.skipAuth)
assert.Equal(t, tt.expected, result)
})
}
}

func TestDetectContentType(t *testing.T) {
tests := []struct {
name        string
body        []byte
contentType string
expected    bool
}{
{
name:        "JSON content",
body:        []byte(`{"key":"value"}`),
contentType: "application/json",
expected:    false,
},
{
name:        "Plain text",
body:        []byte("Hello, world!"),
contentType: "text/plain",
expected:    false,
},
{
name:        "Octet stream",
body:        []byte{0x00, 0x01, 0x02},
contentType: "application/octet-stream",
expected:    true,
},
{
name:        "PDF",
body:        []byte("%PDF-1.4"),
contentType: "application/pdf",
expected:    true,
},
{
name:        "Image",
body:        []byte{0xFF, 0xD8, 0xFF},
contentType: "image/jpeg",
expected:    true,
},
{
name:        "Binary content with null bytes",
body:        []byte{0x48, 0x65, 0x00, 0x6C, 0x6C, 0x6F},
contentType: "application/octet-stream",
expected:    true,
},
{
name:        "Empty body",
body:        []byte{},
contentType: "text/plain",
expected:    false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := DetectContentType(tt.body, tt.contentType)
assert.Equal(t, tt.expected, result)
})
}
}

func TestClient_Execute_AllHTTPMethods(t *testing.T) {
methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

for _, method := range methods {
t.Run(method, func(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
assert.Equal(t, method, r.Method)
w.WriteHeader(http.StatusOK)
if method != "HEAD" {
w.Write([]byte(`{"method":"` + method + `"}`))
}
}))
defer server.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:   method,
URL:      server.URL + "/test",
SkipAuth: true,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.Equal(t, http.StatusOK, resp.StatusCode)
})
}
}

func TestClient_Execute_Redirects(t *testing.T) {
t.Run("Follow redirects by default", func(t *testing.T) {
finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
w.Write([]byte(`{"final":true}`))
}))
defer finalServer.Close()

redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
http.Redirect(w, r, finalServer.URL, http.StatusFound)
}))
defer redirectServer.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:          "GET",
URL:             redirectServer.URL,
SkipAuth:        true,
FollowRedirects: true,
MaxRedirects:    10,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.Equal(t, http.StatusOK, resp.StatusCode)
assert.Contains(t, string(resp.Body), `"final":true`)
})

t.Run("Do not follow redirects when disabled", func(t *testing.T) {
finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
}))
defer finalServer.Close()

redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
http.Redirect(w, r, finalServer.URL, http.StatusFound)
}))
defer redirectServer.Close()

provider := &auth.MockTokenProvider{Token: "test-token"}
client := NewClient(provider, false, 30*time.Second)

opts := RequestOptions{
Method:          "GET",
URL:             redirectServer.URL,
SkipAuth:        true,
FollowRedirects: false,
}

resp, err := client.Execute(context.Background(), opts)

require.NoError(t, err)
assert.True(t, resp.StatusCode >= 300 && resp.StatusCode < 400)
})
}
