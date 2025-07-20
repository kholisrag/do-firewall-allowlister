package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jpillora/backoff"
	"go.uber.org/zap"
)

// Client handles Cloudflare IP fetching
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
	baseURL    string
}

// CloudflareIPsResponse represents the response from Cloudflare IPs API
type CloudflareIPsResponse struct {
	Success bool     `json:"success"`
	Errors  []string `json:"errors"`
	Result  struct {
		IPv4CIDRs []string `json:"ipv4_cidrs"`
		IPv6CIDRs []string `json:"ipv6_cidrs"`
	} `json:"result"`
}

// NewClient creates a new Cloudflare client
func NewClient(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  logger.Named("cloudflare"),
		baseURL: baseURL,
	}
}

// FetchIPs fetches Cloudflare IP ranges from their API
func (c *Client) FetchIPs(ctx context.Context) ([]string, error) {
	c.logger.Debug("Fetching Cloudflare IPs", zap.String("url", c.baseURL))

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL, nil)
	if err != nil {
		c.logger.Error("Failed to create request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "do-firewall-allowlister/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to fetch Cloudflare IPs", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch Cloudflare IPs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Unexpected status code from Cloudflare API",
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status))
		return nil, fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body", zap.Error(err))
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response CloudflareIPsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		c.logger.Error("Failed to parse JSON response", zap.Error(err))
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if !response.Success {
		c.logger.Error("Cloudflare API returned error", zap.Strings("errors", response.Errors))
		return nil, fmt.Errorf("cloudflare API returned errors: %v", response.Errors)
	}

	// Combine IPv4 and IPv6 CIDR blocks
	var allIPs []string
	allIPs = append(allIPs, response.Result.IPv4CIDRs...)
	allIPs = append(allIPs, response.Result.IPv6CIDRs...)

	c.logger.Info("Successfully fetched Cloudflare IPs",
		zap.Int("ipv4_count", len(response.Result.IPv4CIDRs)),
		zap.Int("ipv6_count", len(response.Result.IPv6CIDRs)),
		zap.Int("total_count", len(allIPs)))

	return allIPs, nil
}

// FetchIPsWithRetry fetches Cloudflare IPs with retry logic using exponential backoff with jitter
func (c *Client) FetchIPsWithRetry(ctx context.Context, maxRetries int) ([]string, error) {
	var lastErr error

	// Configure exponential backoff with jitter
	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: true,
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		c.logger.Debug("Attempting to fetch Cloudflare IPs",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries))

		ips, err := c.FetchIPs(ctx)
		if err == nil {
			return ips, nil
		}

		lastErr = err
		c.logger.Warn("Failed to fetch Cloudflare IPs, retrying",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))

		if attempt < maxRetries {
			// Use exponential backoff with jitter
			backoffDuration := b.Duration()
			c.logger.Debug("Waiting before retry", zap.Duration("backoff", backoffDuration))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffDuration):
				// Continue to next attempt
			}
		}
	}

	c.logger.Error("Failed to fetch Cloudflare IPs after all retries",
		zap.Int("max_retries", maxRetries),
		zap.Error(lastErr))

	return nil, fmt.Errorf("failed to fetch Cloudflare IPs after %d retries: %w", maxRetries, lastErr)
}
