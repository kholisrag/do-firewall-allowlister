package digitalocean

import (
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestNewClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	apiKey := "test-api-key"

	client := NewClient(apiKey, logger)

	if client == nil {
		t.Fatal("expected client to be created, got nil")
	}

	if client.client == nil {
		t.Error("expected godo client to be initialized")
	}

	if client.logger == nil {
		t.Error("expected logger to be set")
	}
}

func TestTokenSource_Token(t *testing.T) {
	ts := &TokenSource{
		AccessToken: "test-token",
	}

	token, err := ts.Token()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if token.AccessToken != "test-token" {
		t.Errorf("expected token 'test-token', got %s", token.AccessToken)
	}
}

func TestValidateAndNormalizeSources(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient("test-key", logger)

	tests := []struct {
		name        string
		sources     []string
		expected    []string
		expectError bool
	}{
		{
			name:     "valid IPv4 addresses",
			sources:  []string{"192.168.1.1", "10.0.0.1"},
			expected: []string{"192.168.1.1/32", "10.0.0.1/32"},
		},
		{
			name:     "valid IPv6 addresses",
			sources:  []string{"2001:db8::1", "::1"},
			expected: []string{"2001:db8::1/128", "::1/128"},
		},
		{
			name:     "valid CIDR blocks",
			sources:  []string{"192.168.1.0/24", "10.0.0.0/8"},
			expected: []string{"192.168.1.0/24", "10.0.0.0/8"},
		},
		{
			name:     "mixed valid sources",
			sources:  []string{"192.168.1.1", "10.0.0.0/8", "2001:db8::1"},
			expected: []string{"192.168.1.1/32", "10.0.0.0/8", "2001:db8::1/128"},
		},
		{
			name:        "invalid IP address",
			sources:     []string{"invalid-ip"},
			expectError: true,
		},
		{
			name:        "invalid CIDR block",
			sources:     []string{"192.168.1.0/33"},
			expectError: true,
		},
		{
			name:     "empty sources",
			sources:  []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.validateAndNormalizeSources(tt.sources)

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

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("expected result[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// Note: Testing GetFirewall, UpdateFirewallRules, and ListFirewalls would require
// mocking the DigitalOcean API, which is complex. In a real-world scenario,
// you would use a mocking library like gomock or testify/mock to create
// mock implementations of the godo.Client interface.

// Example of how you might structure a test with mocking:
/*
func TestGetFirewall(t *testing.T) {
	// This would require setting up a mock godo.Client
	// and injecting it into our Client struct

	logger := zaptest.NewLogger(t)

	// Mock setup would go here
	mockClient := &mockGodoClient{}

	client := &Client{
		client: mockClient,
		logger: logger,
	}

	// Test implementation would go here
}
*/

func TestFirewallRule(t *testing.T) {
	rule := FirewallRule{
		Port:     80,
		Protocol: "tcp",
		Sources:  []string{"192.168.1.0/24"},
	}

	if rule.Port != 80 {
		t.Errorf("expected port 80, got %d", rule.Port)
	}

	if rule.Protocol != "tcp" {
		t.Errorf("expected protocol tcp, got %s", rule.Protocol)
	}

	if len(rule.Sources) != 1 || rule.Sources[0] != "192.168.1.0/24" {
		t.Errorf("expected sources [192.168.1.0/24], got %v", rule.Sources)
	}
}
