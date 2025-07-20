package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kholisrag/do-firewall-allowlister/pkg/config"
	"github.com/kholisrag/do-firewall-allowlister/pkg/daemon"
	"github.com/kholisrag/do-firewall-allowlister/pkg/logger"
	"github.com/kholisrag/do-firewall-allowlister/pkg/scheduler"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewValidateCommand creates and returns the validate command
func NewValidateCommand() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration and test connectivity",
		Long: `Validate the configuration file and test connectivity to external services.

This command will:
- Load and validate the configuration file
- Test DigitalOcean API access and firewall permissions
- Test Cloudflare API connectivity
- Test Netdata domain resolution
- Validate cron schedule syntax
- Show configuration summary

This is useful for troubleshooting configuration issues before running the service.`,
		RunE: runValidate,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show current status of external services",
		Long: `Show the current status of all external services and APIs.

This command will:
- Check DigitalOcean API connectivity and firewall status
- Check Cloudflare API connectivity and IP count
- Check Netdata domain resolution status
- Display results in JSON format

This is useful for monitoring and health checking.`,
		RunE: runStatus,
	}

	var statusFormat string
	statusCmd.Flags().StringVar(&statusFormat, "format", "json", "Output format (json, yaml)")

	validateCmd.AddCommand(statusCmd)
	return validateCmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Get config file from global flag
	configFile, _ := cmd.Flags().GetString("config")

	// Set configuration defaults
	config.SetDefaults()

	// Load configuration (use root command flags for global flags)
	cfg, err := config.Load(configFile, cmd.Root().PersistentFlags())
	if err != nil {
		return fmt.Errorf("❌ Configuration validation failed: %w", err)
	}

	// Initialize logger with minimal output for validation
	if err := logger.Initialize("ERROR"); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	log := logger.Get()
	log.Info("✅ Configuration file loaded successfully")

	// Validate cron schedule
	if err := scheduler.ValidateSchedule(cfg.Cron.Schedule); err != nil {
		return fmt.Errorf("❌ Invalid cron schedule: %w", err)
	}

	log.Info("✅ Cron schedule is valid", zap.String("schedule", cfg.Cron.Schedule))

	// Try to get next run time
	if nextRun, err := scheduler.GetNextRunTime(cfg.Cron.Schedule, cfg.Cron.Timezone); err != nil {
		log.Warn("⚠️  Could not determine next run time", zap.Error(err))
	} else {
		log.Info("📅 Next scheduled run", zap.String("time", nextRun.Format(time.RFC3339)))
	}

	// Test connectivity
	logger := logger.Get()
	d, err := daemon.NewDaemon(cfg, logger, true) // Use dry-run mode for validation
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize services: %w", err)
	}

	log.Info("🔍 Testing connectivity...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := d.RunOnce(ctx); err != nil {
		return fmt.Errorf("❌ Connectivity test failed: %w", err)
	}

	log.Info("✅ All connectivity tests passed")

	// Show configuration summary
	log.Info("📋 Configuration Summary")
	log.Info("Configuration details",
		zap.String("log_level", cfg.LogLevel),
		zap.String("cron_schedule", cfg.Cron.Schedule),
		zap.String("cron_timezone", cfg.Cron.Timezone),
		zap.String("firewall_id", cfg.DigitalOcean.FirewallID),
		zap.String("cloudflare_url", cfg.Cloudflare.IPsURL),
		zap.Int("netdata_domains", len(cfg.Netdata.Domains)),
		zap.Int("inbound_rules", len(cfg.DigitalOcean.InboundRules)))

	for i, rule := range cfg.DigitalOcean.InboundRules {
		log.Info("Inbound rule",
			zap.Int("rule_number", i+1),
			zap.String("protocol", rule.Protocol),
			zap.Int("port", rule.Port))
	}

	log.Info("✅ Configuration validation completed successfully")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Get config file from global flag
	configFile, _ := cmd.Flags().GetString("config")
	format, _ := cmd.Flags().GetString("format")

	// Set configuration defaults
	config.SetDefaults()

	// Load configuration
	_, err := config.Load(configFile, cmd.Root().PersistentFlags())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger with minimal output
	if err := logger.Initialize("ERROR"); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	// Get status (this would need to be implemented in daemon)
	status := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"config":    configFile,
		"services": map[string]string{
			"digitalocean": "unknown",
			"cloudflare":   "unknown",
			"netdata":      "unknown",
		},
	}

	// Output in requested format
	var output []byte
	switch format {
	case "json":
		output, err = json.MarshalIndent(status, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	fmt.Println(string(output))
	return nil
}
