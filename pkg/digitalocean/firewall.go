package digitalocean

import (
	"context"
	"fmt"
	"net"

	"github.com/digitalocean/godo"
	"go.uber.org/zap"
)

// FirewallRule represents a simplified firewall rule
type FirewallRule struct {
	Port     int
	Protocol string
	Sources  []string // IP addresses or CIDR blocks
}

// UpdateFirewallRules updates the firewall with new inbound rules for the specified IPs
func (c *Client) UpdateFirewallRules(
	ctx context.Context,
	firewallID string,
	rules []FirewallRule,
	sourceIPs []string,
) error {
	c.logger.Info("Updating firewall rules",
		zap.String("firewall_id", firewallID),
		zap.Int("rule_count", len(rules)),
		zap.Int("source_ip_count", len(sourceIPs)))

	// Get current firewall configuration
	firewall, err := c.GetFirewall(ctx, firewallID)
	if err != nil {
		return fmt.Errorf("failed to get current firewall: %w", err)
	}

	// Build new inbound rules
	var newInboundRules []godo.InboundRule

	// Keep existing rules that don't match our managed ports
	managedPorts := make(map[string]bool)
	for _, rule := range rules {
		managedPorts[fmt.Sprintf("%d", rule.Port)] = true
	}

	for _, existingRule := range firewall.InboundRules {
		// Keep rules for ports we don't manage
		if !managedPorts[existingRule.PortRange] {
			newInboundRules = append(newInboundRules, existingRule)
		}
	}

	// Add new rules for our managed ports
	for _, rule := range rules {
		// Validate and normalize source IPs
		validSources, err := c.validateAndNormalizeSources(sourceIPs)
		if err != nil {
			c.logger.Error("Failed to validate source IPs", zap.Error(err))
			return fmt.Errorf("failed to validate source IPs: %w", err)
		}

		inboundRule := godo.InboundRule{
			Protocol:  rule.Protocol,
			PortRange: fmt.Sprintf("%d", rule.Port),
			Sources: &godo.Sources{
				Addresses: validSources,
			},
		}

		newInboundRules = append(newInboundRules, inboundRule)

		c.logger.Debug("Added inbound rule",
			zap.Int("port", rule.Port),
			zap.String("protocol", rule.Protocol),
			zap.Strings("sources", validSources))
	}

	// Update the firewall
	updateRequest := &godo.FirewallRequest{
		Name:          firewall.Name,
		InboundRules:  newInboundRules,
		OutboundRules: firewall.OutboundRules,
		Tags:          firewall.Tags,
	}

	_, _, err = c.client.Firewalls.Update(ctx, firewallID, updateRequest)
	if err != nil {
		c.logger.Error("Failed to update firewall",
			zap.String("firewall_id", firewallID),
			zap.Error(err))
		return fmt.Errorf("failed to update firewall %s: %w", firewallID, err)
	}

	c.logger.Info("Successfully updated firewall rules",
		zap.String("firewall_id", firewallID),
		zap.Int("total_inbound_rules", len(newInboundRules)))

	return nil
}

// validateAndNormalizeSources validates IP addresses and CIDR blocks
func (c *Client) validateAndNormalizeSources(sources []string) ([]string, error) {
	var validSources []string

	for _, source := range sources {
		// Try to parse as IP address first
		if ip := net.ParseIP(source); ip != nil {
			// Convert to CIDR notation
			if ip.To4() != nil {
				validSources = append(validSources, source+"/32")
			} else {
				validSources = append(validSources, source+"/128")
			}
			continue
		}

		// Try to parse as CIDR block
		if _, _, err := net.ParseCIDR(source); err == nil {
			validSources = append(validSources, source)
			continue
		}

		c.logger.Warn("Invalid IP address or CIDR block", zap.String("source", source))
		return nil, fmt.Errorf("invalid IP address or CIDR block: %s", source)
	}

	return validSources, nil
}

// ListFirewalls lists all firewalls in the account
func (c *Client) ListFirewalls(ctx context.Context) ([]godo.Firewall, error) {
	c.logger.Debug("Listing firewalls")

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	var allFirewalls []godo.Firewall

	for {
		firewalls, resp, err := c.client.Firewalls.List(ctx, opt)
		if err != nil {
			c.logger.Error("Failed to list firewalls", zap.Error(err))
			return nil, fmt.Errorf("failed to list firewalls: %w", err)
		}

		allFirewalls = append(allFirewalls, firewalls...)

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, fmt.Errorf("failed to get current page: %w", err)
		}

		opt.Page = page + 1
	}

	c.logger.Debug("Successfully listed firewalls", zap.Int("count", len(allFirewalls)))
	return allFirewalls, nil
}
