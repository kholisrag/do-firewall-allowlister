package publicip

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client handles public IP detection
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
	serviceURL string
}

// NewClient creates a new public IP detection client
func NewClient(logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:     logger.Named("publicip"),
		serviceURL: "https://icanhazip.com/",
	}
}

// NewClientWithURL creates a new public IP detection client with custom service URL
func NewClientWithURL(serviceURL string, logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:     logger.Named("publicip"),
		serviceURL: serviceURL,
	}
}

// GetPublicIP detects the current public IP address
func (c *Client) GetPublicIP(ctx context.Context) (string, error) {
	c.logger.Debug("Detecting public IP address", zap.String("service_url", c.serviceURL))

	req, err := http.NewRequestWithContext(ctx, "GET", c.serviceURL, nil)
	if err != nil {
		c.logger.Error("Failed to create request", zap.Error(err))
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "do-firewall-allowlister/1.0")
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to fetch public IP", zap.Error(err))
		return "", fmt.Errorf("failed to fetch public IP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code",
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status))
		return "", fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body", zap.Error(err))
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Clean up the IP address (remove whitespace and newlines)
	ipStr := strings.TrimSpace(string(body))
	
	// Validate the IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		c.logger.Error("Invalid IP address received", zap.String("ip", ipStr))
		return "", fmt.Errorf("invalid IP address received: %s", ipStr)
	}

	c.logger.Info("Successfully detected public IP", zap.String("ip", ipStr))
	return ipStr, nil
}

// GetPublicIPWithRetry detects the current public IP address with retry logic
func (c *Client) GetPublicIPWithRetry(ctx context.Context, maxRetries int) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		c.logger.Debug("Attempting to detect public IP",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries))

		ip, err := c.GetPublicIP(ctx)
		if err == nil {
			return ip, nil
		}

		lastErr = err
		c.logger.Warn("Failed to detect public IP, retrying",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))

		if attempt < maxRetries {
			// Simple backoff - wait 2 seconds between retries
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(2 * time.Second):
				// Continue to next attempt
			}
		}
	}

	c.logger.Error("Failed to detect public IP after all retries",
		zap.Int("max_retries", maxRetries),
		zap.Error(lastErr))
	return "", fmt.Errorf("failed to detect public IP after %d retries: %w", maxRetries, lastErr)
}
