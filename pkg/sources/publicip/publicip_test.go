package publicip

import (
	"context"
	"net"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient(logger)

	if client == nil {
		t.Fatal("expected client to be created, got nil")
	}

	if client.serviceURL != "https://icanhazip.com/" {
		t.Errorf("expected default service URL to be https://icanhazip.com/, got %s", client.serviceURL)
	}

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("expected timeout to be 10 seconds, got %v", client.httpClient.Timeout)
	}
}

func TestNewClientWithURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	customURL := "https://example.com/ip"
	client := NewClientWithURL(customURL, logger)

	if client == nil {
		t.Fatal("expected client to be created, got nil")
	}

	if client.serviceURL != customURL {
		t.Errorf("expected service URL to be %s, got %s", customURL, client.serviceURL)
	}
}

func TestGetPublicIP_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t)
	client := NewClient(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ip, err := client.GetPublicIP(ctx)
	if err != nil {
		t.Fatalf("failed to get public IP: %v", err)
	}

	// Validate that we got a valid IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		t.Errorf("expected valid IP address, got %s", ip)
	}

	// Should not be a loopback or private IP
	if parsedIP.IsLoopback() {
		t.Errorf("expected public IP, got loopback address: %s", ip)
	}

	if parsedIP.IsPrivate() {
		t.Errorf("expected public IP, got private address: %s", ip)
	}

	t.Logf("Successfully detected public IP: %s", ip)
}

func TestGetPublicIPWithRetry_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t)
	client := NewClient(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ip, err := client.GetPublicIPWithRetry(ctx, 3)
	if err != nil {
		t.Fatalf("failed to get public IP with retry: %v", err)
	}

	// Validate that we got a valid IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		t.Errorf("expected valid IP address, got %s", ip)
	}

	t.Logf("Successfully detected public IP with retry: %s", ip)
}
