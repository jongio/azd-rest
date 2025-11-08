package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteRequest_GET(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	config := RequestConfig{
		Method:     "GET",
		URL:        server.URL,
		UseAzdAuth: false,
	}

	err := ExecuteRequest(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteRequest_POST(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"123","status":"created"}`))
	}))
	defer server.Close()

	config := RequestConfig{
		Method:      "POST",
		URL:         server.URL,
		Data:        `{"name":"test"}`,
		ContentType: "application/json",
		UseAzdAuth:  false,
	}

	err := ExecuteRequest(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteRequest_CustomHeaders(t *testing.T) {
	customHeader := "X-Custom-Header"
	customValue := "custom-value"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(customHeader) != customValue {
			t.Errorf("Expected header %s: %s, got %s", customHeader, customValue, r.Header.Get(customHeader))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	config := RequestConfig{
		Method:     "GET",
		URL:        server.URL,
		Headers:    []string{customHeader + ": " + customValue},
		UseAzdAuth: false,
	}

	err := ExecuteRequest(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteRequest_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	config := RequestConfig{
		Method:     "GET",
		URL:        server.URL,
		UseAzdAuth: false,
	}

	err := ExecuteRequest(config)
	if err == nil {
		t.Error("Expected error for 404 status, got nil")
	}
}
