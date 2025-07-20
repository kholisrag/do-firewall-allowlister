package netdata

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jpillora/backoff"
	"go.uber.org/zap"
)

// Client handles Netdata domain IP resolution
type Client struct {
	resolver *net.Resolver
	logger   *zap.Logger
}

// NewClient creates a new Netdata client
func NewClient(logger *zap.Logger) *Client {
	return &Client{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Second * 10,
				}
				return d.DialContext(ctx, network, address)
			},
		},
		logger: logger.Named("netdata"),
	}
}

// ResolveDomains resolves IP addresses for the given domains
func (c *Client) ResolveDomains(ctx context.Context, domains []string) ([]string, error) {
	c.logger.Info("Resolving Netdata domains", zap.Strings("domains", domains))

	var allIPs []string
	var resolveErrors []error

	for _, domain := range domains {
		c.logger.Debug("Resolving domain", zap.String("domain", domain))

		ips, err := c.resolveDomain(ctx, domain)
		if err != nil {
			c.logger.Error("Failed to resolve domain",
				zap.String("domain", domain),
				zap.Error(err))
			resolveErrors = append(resolveErrors, fmt.Errorf("failed to resolve %s: %w", domain, err))
			continue
		}

		c.logger.Debug("Successfully resolved domain",
			zap.String("domain", domain),
			zap.Strings("ips", ips),
			zap.Int("count", len(ips)))

		allIPs = append(allIPs, ips...)
	}

	if len(resolveErrors) > 0 && len(allIPs) == 0 {
		// All domains failed to resolve
		c.logger.Error("Failed to resolve any domains", zap.Int("error_count", len(resolveErrors)))
		return nil, fmt.Errorf("failed to resolve any domains: %v", resolveErrors)
	}

	if len(resolveErrors) > 0 {
		// Some domains failed, but we have some IPs
		c.logger.Warn("Some domains failed to resolve",
			zap.Int("error_count", len(resolveErrors)),
			zap.Int("successful_ips", len(allIPs)))
	}

	// Remove duplicates
	uniqueIPs := removeDuplicates(allIPs)

	c.logger.Info("Successfully resolved Netdata domains",
		zap.Int("total_domains", len(domains)),
		zap.Int("resolved_ips", len(uniqueIPs)),
		zap.Int("failed_domains", len(resolveErrors)))

	return uniqueIPs, nil
}

// resolveDomain resolves both IPv4 and IPv6 addresses for a domain
func (c *Client) resolveDomain(ctx context.Context, domain string) ([]string, error) {
	var allIPs []string

	// Resolve IPv4 addresses
	ipv4Addrs, err := c.resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		c.logger.Debug("Failed to resolve IPv4 for domain",
			zap.String("domain", domain),
			zap.Error(err))
	} else {
		for _, addr := range ipv4Addrs {
			if addr.IP.To4() != nil {
				allIPs = append(allIPs, addr.IP.String())
			}
		}
	}

	// Also try to get IPv6 addresses
	ipv6Addrs, err := c.resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		c.logger.Debug("Failed to resolve IPv6 for domain",
			zap.String("domain", domain),
			zap.Error(err))
	} else {
		for _, addr := range ipv6Addrs {
			if addr.IP.To4() == nil && addr.IP.To16() != nil {
				allIPs = append(allIPs, addr.IP.String())
			}
		}
	}

	if len(allIPs) == 0 {
		return nil, fmt.Errorf("no IP addresses found for domain %s", domain)
	}

	return allIPs, nil
}

// ResolveDomainsWithRetry resolves domains with retry logic using exponential backoff with jitter
func (c *Client) ResolveDomainsWithRetry(ctx context.Context, domains []string, maxRetries int) ([]string, error) {
	var lastErr error

	// Configure exponential backoff with jitter
	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: true,
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		c.logger.Debug("Attempting to resolve Netdata domains",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries))

		ips, err := c.ResolveDomains(ctx, domains)
		if err == nil {
			return ips, nil
		}

		lastErr = err
		c.logger.Warn("Failed to resolve Netdata domains, retrying",
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

	c.logger.Error("Failed to resolve Netdata domains after all retries",
		zap.Int("max_retries", maxRetries),
		zap.Error(lastErr))

	return nil, fmt.Errorf("failed to resolve Netdata domains after %d retries: %w", maxRetries, lastErr)
}

// removeDuplicates removes duplicate IP addresses from the slice
func removeDuplicates(ips []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, ip := range ips {
		if !seen[ip] {
			seen[ip] = true
			unique = append(unique, ip)
		}
	}

	return unique
}
