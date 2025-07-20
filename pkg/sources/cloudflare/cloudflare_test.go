package cloudflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestNewClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	baseURL := "https://api.cloudflare.com/client/v4/ips"

	client := NewClient(baseURL, logger)

	if client == nil {
		t.Fatal("expected client to be created, got nil")
	}

	if client.baseURL != baseURL {
		t.Errorf("expected baseURL %s, got %s", baseURL, client.baseURL)
	}

	if client.httpClient == nil {
		t.Error("expected HTTP client to be initialized")
	}

	if client.logger == nil {
		t.Error("expected logger to be set")
	}
}

func TestFetchIPs(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		statusCode   int
		expectedIPs  []string
		expectError  bool
	}{
		{
			name:       "successful response",
			statusCode: http.StatusOK,
			responseBody: `{
				"success": true,
				"errors": [],
				"result": {
					"ipv4_cidrs": ["192.168.1.0/24", "10.0.0.0/8"],
					"ipv6_cidrs": ["2001:db8::/32"]
				}
			}`,
			expectedIPs: []string{"192.168.1.0/24", "10.0.0.0/8", "2001:db8::/32"},
		},
		{
			name:       "API error response",
			statusCode: http.StatusOK,
			responseBody: `{
				"success": false,
				"errors": ["API error occurred"],
				"result": null
			}`,
			expectError: true,
		},
		{
			name:         "HTTP error",
			statusCode:   http.StatusInternalServerError,
			responseBody: "Internal Server Error",
			expectError:  true,
		},
		{
			name:         "invalid JSON",
			statusCode:   http.StatusOK,
			responseBody: "invalid json",
			expectError:  true,
		},
		{
			name:       "empty result",
			statusCode: http.StatusOK,
			responseBody: `{
				"success": true,
				"errors": [],
				"result": {
					"ipv4_cidrs": [],
					"ipv6_cidrs": []
				}
			}`,
			expectedIPs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			logger := zaptest.NewLogger(t)
			client := NewClient(server.URL, logger)

			ctx := context.Background()
			ips, err := client.FetchIPs(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(ips) != len(tt.expectedIPs) {
				t.Errorf("expected %d IPs, got %d", len(tt.expectedIPs), len(ips))
				return
			}

			for i, expected := range tt.expectedIPs {
				if ips[i] != expected {
					t.Errorf("expected IP[%d] = %s, got %s", i, expected, ips[i])
				}
			}
		})
	}
}

func TestFetchIPsWithRetry(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Test successful retry
	t.Run("successful after retry", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"success": true,
				"errors": [],
				"result": {
					"ipv4_cidrs": ["192.168.1.0/24"],
					"ipv6_cidrs": []
				}
			}`))
		}))
		defer server.Close()

		client := NewClient(server.URL, logger)
		ctx := context.Background()

		ips, err := client.FetchIPsWithRetry(ctx, 3)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(ips) != 1 || ips[0] != "192.168.1.0/24" {
			t.Errorf("expected [192.168.1.0/24], got %v", ips)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})

	// Test failure after all retries
	t.Run("failure after all retries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL, logger)
		ctx := context.Background()

		_, err := client.FetchIPsWithRetry(ctx, 2)
		if err == nil {
			t.Error("expected error after all retries failed")
		}
	})
}
