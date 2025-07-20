package service

import (
	"context"
	"fmt"

	"github.com/kholisrag/do-firewall-allowlister/pkg/config"
	"github.com/kholisrag/do-firewall-allowlister/pkg/digitalocean"
	"github.com/kholisrag/do-firewall-allowlister/pkg/sources/cloudflare"
	"github.com/kholisrag/do-firewall-allowlister/pkg/sources/netdata"
	"go.uber.org/zap"
)

// Service orchestrates the firewall update process
type Service struct {
	config             *config.Config
	digitalOceanClient *digitalocean.Client
	cloudflareClient   *cloudflare.Client
	netdataClient      *netdata.Client
	logger             *zap.Logger
	dryRun             bool
}

// NewService creates a new service instance
func NewService(cfg *config.Config, logger *zap.Logger, dryRun bool) *Service {
	doClient := digitalocean.NewClient(cfg.DigitalOcean.APIKey, logger)
	cfClient := cloudflare.NewClient(cfg.Cloudflare.IPsURL, logger)
	andClient := netdata.NewClient(logger)

	return &Service{
		config:             cfg,
		digitalOceanClient: doClient,
		cloudflareClient:   cfClient,
		netdataClient:      andClient,
		logger:             logger.Named("service"),
		dryRun:             dryRun,
	}
}

// UpdateFirewallRules performs the complete firewall update process
func (s *Service) UpdateFirewallRules(ctx context.Context) error {
	s.logger.Info("Starting firewall rules update",
		zap.String("firewall_id", s.config.DigitalOcean.FirewallID),
		zap.Bool("dry_run", s.dryRun))

	// Fetch Cloudflare IPs
	cloudflareIPs, err := s.fetchCloudflareIPs(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch Cloudflare IPs: %w", err)
	}

	// Resolve Netdata domain IPs
	netdataIPs, err := s.resolveNetdataIPs(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve Netdata IPs: %w", err)
	}

	// Combine all IPs
	allIPs := make([]string, 0, len(cloudflareIPs)+len(netdataIPs))
	allIPs = append(allIPs, cloudflareIPs...)
	allIPs = append(allIPs, netdataIPs...)
	s.logger.Info("Collected all source IPs",
		zap.Int("cloudflare_ips", len(cloudflareIPs)),
		zap.Int("netdata_ips", len(netdataIPs)),
		zap.Int("total_ips", len(allIPs)))

	// Convert config rules to service rules
	var firewallRules []digitalocean.FirewallRule
	for _, rule := range s.config.DigitalOcean.InboundRules {
		firewallRules = append(firewallRules, digitalocean.FirewallRule{
			Port:     rule.Port,
			Protocol: rule.Protocol,
			Sources:  allIPs,
		})
	}

	if s.dryRun {
		s.logger.Info("DRY RUN: Would update firewall with the following rules")
		for _, rule := range firewallRules {
			s.logger.Info("DRY RUN: Firewall rule",
				zap.Int("port", rule.Port),
				zap.String("protocol", rule.Protocol),
				zap.Int("source_count", len(rule.Sources)))
		}
		s.logger.Info("DRY RUN: Total source IPs that would be allowed", zap.Int("count", len(allIPs)))
		return nil
	}

	// Update firewall rules
	err = s.digitalOceanClient.UpdateFirewallRules(
		ctx,
		s.config.DigitalOcean.FirewallID,
		firewallRules,
		allIPs,
	)
	if err != nil {
		return fmt.Errorf("failed to update firewall rules: %w", err)
	}

	s.logger.Info("Successfully completed firewall rules update",
		zap.String("firewall_id", s.config.DigitalOcean.FirewallID),
		zap.Int("total_rules", len(firewallRules)),
		zap.Int("total_source_ips", len(allIPs)))

	return nil
}

// fetchCloudflareIPs fetches Cloudflare IP ranges with retry
func (s *Service) fetchCloudflareIPs(ctx context.Context) ([]string, error) {
	s.logger.Debug("Fetching Cloudflare IPs")

	ips, err := s.cloudflareClient.FetchIPsWithRetry(ctx, 3)
	if err != nil {
		s.logger.Error("Failed to fetch Cloudflare IPs", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Successfully fetched Cloudflare IPs", zap.Int("count", len(ips)))
	return ips, nil
}

// resolveNetdataIPs resolves Netdata domain IPs with retry
func (s *Service) resolveNetdataIPs(ctx context.Context) ([]string, error) {
	if len(s.config.Netdata.Domains) == 0 {
		s.logger.Info("No Netdata domains configured, skipping resolution")
		return []string{}, nil
	}

	s.logger.Debug("Resolving Netdata domain IPs", zap.Strings("domains", s.config.Netdata.Domains))

	ips, err := s.netdataClient.ResolveDomainsWithRetry(ctx, s.config.Netdata.Domains, 3)
	if err != nil {
		s.logger.Error("Failed to resolve Netdata domain IPs", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Successfully resolved Netdata domain IPs", zap.Int("count", len(ips)))
	return ips, nil
}

// ValidateConfiguration validates the service configuration
func (s *Service) ValidateConfiguration(ctx context.Context) error {
	s.logger.Info("Validating configuration")

	// Test DigitalOcean API access
	firewall, err := s.digitalOceanClient.GetFirewall(ctx, s.config.DigitalOcean.FirewallID)
	if err != nil {
		return fmt.Errorf("failed to access DigitalOcean firewall: %w", err)
	}

	s.logger.Info("Successfully validated DigitalOcean access",
		zap.String("firewall_id", firewall.ID),
		zap.String("firewall_name", firewall.Name))

	// Test Cloudflare API access
	_, err = s.cloudflareClient.FetchIPs(ctx)
	if err != nil {
		return fmt.Errorf("failed to access Cloudflare API: %w", err)
	}

	s.logger.Info("Successfully validated Cloudflare API access")

	// Test Netdata domain resolution if domains are configured
	if len(s.config.Netdata.Domains) > 0 {
		_, err = s.netdataClient.ResolveDomains(ctx, s.config.Netdata.Domains)
		if err != nil {
			return fmt.Errorf("failed to resolve Netdata domains: %w", err)
		}

		s.logger.Info("Successfully validated Netdata domain resolution")
	}

	s.logger.Info("Configuration validation completed successfully")
	return nil
}

// GetStatus returns the current status of external services
func (s *Service) GetStatus(ctx context.Context) (*Status, error) {
	status := &Status{}

	// Check DigitalOcean API
	firewall, err := s.digitalOceanClient.GetFirewall(ctx, s.config.DigitalOcean.FirewallID)
	if err != nil {
		status.DigitalOcean.Status = "error"
		status.DigitalOcean.Error = err.Error()
	} else {
		status.DigitalOcean.Status = "ok"
		status.DigitalOcean.FirewallName = firewall.Name
		status.DigitalOcean.InboundRuleCount = len(firewall.InboundRules)
	}

	// Check Cloudflare API
	cloudflareIPs, err := s.cloudflareClient.FetchIPs(ctx)
	if err != nil {
		status.Cloudflare.Status = "error"
		status.Cloudflare.Error = err.Error()
	} else {
		status.Cloudflare.Status = "ok"
		status.Cloudflare.IPCount = len(cloudflareIPs)
	}

	// Check Netdata domains
	if len(s.config.Netdata.Domains) > 0 {
		netdataIPs, err := s.netdataClient.ResolveDomains(ctx, s.config.Netdata.Domains)
		if err != nil {
			status.Netdata.Status = "error"
			status.Netdata.Error = err.Error()
		} else {
			status.Netdata.Status = "ok"
			status.Netdata.IPCount = len(netdataIPs)
			status.Netdata.DomainCount = len(s.config.Netdata.Domains)
		}
	} else {
		status.Netdata.Status = "disabled"
	}

	return status, nil
}

// Status represents the current status of external services
type Status struct {
	DigitalOcean struct {
		Status           string `json:"status"`
		Error            string `json:"error,omitempty"`
		FirewallName     string `json:"firewall_name,omitempty"`
		InboundRuleCount int    `json:"inbound_rule_count,omitempty"`
	} `json:"digitalocean"`
	Cloudflare struct {
		Status  string `json:"status"`
		Error   string `json:"error,omitempty"`
		IPCount int    `json:"ip_count,omitempty"`
	} `json:"cloudflare"`
	Netdata struct {
		Status      string `json:"status"`
		Error       string `json:"error,omitempty"`
		IPCount     int    `json:"ip_count,omitempty"`
		DomainCount int    `json:"domain_count,omitempty"`
	} `json:"netdata"`
}
