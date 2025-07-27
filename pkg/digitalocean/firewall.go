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

	// Log droplets that will be preserved
	if len(firewall.DropletIDs) > 0 {
		c.logger.Debug("Preserving droplet attachments during firewall update",
			zap.String("firewall_id", firewallID),
			zap.Ints("droplet_ids", firewall.DropletIDs))
	}

	// Update the firewall
	updateRequest := &godo.FirewallRequest{
		Name:          firewall.Name,
		InboundRules:  newInboundRules,
		OutboundRules: firewall.OutboundRules,
		Tags:          firewall.Tags,
		DropletIDs:    firewall.DropletIDs, // Preserve existing droplet attachments
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
		zap.Int("total_inbound_rules", len(newInboundRules)),
		zap.Int("preserved_droplets", len(firewall.DropletIDs)))

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

// AddSSHRule adds an SSH rule for a specific IP address to the firewall
// If replaceExisting is true, it removes all existing SSH rules for the port and replaces with the new IP
// If replaceExisting is false, it appends the IP to existing SSH rules for the port
func (c *Client) AddSSHRule(ctx context.Context, firewallID string, sourceIP string, port int, replaceExisting bool) error {
	c.logger.Info("Adding SSH rule to firewall",
		zap.String("firewall_id", firewallID),
		zap.String("source_ip", sourceIP),
		zap.Int("port", port),
		zap.Bool("replace_existing", replaceExisting))

	// Get current firewall configuration
	firewall, err := c.GetFirewall(ctx, firewallID)
	if err != nil {
		return fmt.Errorf("failed to get current firewall: %w", err)
	}

	// Validate and normalize the source IP
	validSources, err := c.validateAndNormalizeSources([]string{sourceIP})
	if err != nil {
		c.logger.Error("Failed to validate source IP", zap.Error(err))
		return fmt.Errorf("failed to validate source IP: %w", err)
	}

	var newInboundRules []godo.InboundRule
	var existingSSHRule *godo.InboundRule
	var existingSSHRuleIndex int = -1

	// Find existing SSH rule for this port
	for i, existingRule := range firewall.InboundRules {
		if existingRule.Protocol == "tcp" && existingRule.PortRange == fmt.Sprintf("%d", port) {
			existingSSHRule = &existingRule
			existingSSHRuleIndex = i
			break
		}
	}

	if existingSSHRule != nil {
		// Check if IP already exists in the rule
		ipAlreadyExists := false
		if existingSSHRule.Sources != nil {
			for _, addr := range existingSSHRule.Sources.Addresses {
				if addr == validSources[0] {
					ipAlreadyExists = true
					c.logger.Info("SSH rule already exists for this IP",
						zap.String("source_ip", sourceIP),
						zap.Int("port", port))
					break
				}
			}
		}

		if ipAlreadyExists && !replaceExisting {
			return nil // IP already exists and we're not replacing, nothing to do
		}

		// Copy all rules except the existing SSH rule
		for i, rule := range firewall.InboundRules {
			if i != existingSSHRuleIndex {
				newInboundRules = append(newInboundRules, rule)
			}
		}

		// Create updated SSH rule
		var updatedAddresses []string
		if replaceExisting {
			// Replace mode: only use the new IP
			updatedAddresses = validSources
			c.logger.Info("Replacing existing SSH rule with current IP",
				zap.String("source_ip", sourceIP),
				zap.Int("port", port))
		} else {
			// Append mode: merge with existing IPs
			if existingSSHRule.Sources != nil {
				updatedAddresses = append(updatedAddresses, existingSSHRule.Sources.Addresses...)
			}
			if !ipAlreadyExists {
				updatedAddresses = append(updatedAddresses, validSources...)
				c.logger.Info("Appending IP to existing SSH rule",
					zap.String("source_ip", sourceIP),
					zap.Int("port", port),
					zap.Int("total_ips", len(updatedAddresses)))
			}
		}

		// Create the updated SSH rule
		updatedSSHRule := godo.InboundRule{
			Protocol:  "tcp",
			PortRange: fmt.Sprintf("%d", port),
			Sources: &godo.Sources{
				Addresses: updatedAddresses,
			},
		}
		newInboundRules = append(newInboundRules, updatedSSHRule)
	} else {
		// No existing SSH rule for this port, create a new one
		newInboundRules = append(newInboundRules, firewall.InboundRules...)

		sshRule := godo.InboundRule{
			Protocol:  "tcp",
			PortRange: fmt.Sprintf("%d", port),
			Sources: &godo.Sources{
				Addresses: validSources,
			},
		}
		newInboundRules = append(newInboundRules, sshRule)

		c.logger.Info("Creating new SSH rule",
			zap.String("source_ip", sourceIP),
			zap.Int("port", port))
	}

	// Log droplets that will be preserved
	if len(firewall.DropletIDs) > 0 {
		c.logger.Debug("Preserving droplet attachments during SSH rule addition",
			zap.String("firewall_id", firewallID),
			zap.Ints("droplet_ids", firewall.DropletIDs))
	}

	// Update the firewall
	updateRequest := &godo.FirewallRequest{
		Name:          firewall.Name,
		InboundRules:  newInboundRules,
		OutboundRules: firewall.OutboundRules,
		Tags:          firewall.Tags,
		DropletIDs:    firewall.DropletIDs, // Preserve existing droplet attachments
	}

	_, _, err = c.client.Firewalls.Update(ctx, firewallID, updateRequest)
	if err != nil {
		c.logger.Error("Failed to update firewall with SSH rule",
			zap.String("firewall_id", firewallID),
			zap.Error(err))
		return fmt.Errorf("failed to update firewall %s with SSH rule: %w", firewallID, err)
	}

	c.logger.Info("Successfully added SSH rule to firewall",
		zap.String("firewall_id", firewallID),
		zap.String("source_ip", sourceIP),
		zap.Int("port", port),
		zap.Int("total_inbound_rules", len(newInboundRules)),
		zap.Int("preserved_droplets", len(firewall.DropletIDs)))

	return nil
}
