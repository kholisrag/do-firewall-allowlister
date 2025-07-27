package commands

import (
	"context"
	"fmt"

	"github.com/kholisrag/do-firewall-allowlister/pkg/config"
	"github.com/kholisrag/do-firewall-allowlister/pkg/digitalocean"
	"github.com/kholisrag/do-firewall-allowlister/pkg/logger"
	"github.com/kholisrag/do-firewall-allowlister/pkg/sources/publicip"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewAllowCurrentIPCommand creates and returns the allow-current-ip command
func NewAllowCurrentIPCommand() *cobra.Command {
	var (
		dryRun         bool
		port           int
		removeExisting bool
	)

	allowCurrentIPCmd := &cobra.Command{
		Use:   "allow-current-ip",
		Short: "Allow current public IP address for SSH access",
		Long: `Detect the current public IP address and add it to the DigitalOcean firewall for SSH access.

This command will:
- Detect your current public IP address using icanhazip.com
- Add it to existing SSH rules for the specified port (append mode by default)
- Preserve existing firewall rules and droplet attachments
- Default to port 22 (SSH) but can be customized with --port flag

Modes:
- Default (append): Adds current IP to existing SSH rules for the port
- --remove flag: Removes all existing SSH rules for the port and replaces with current IP only

This is useful for quickly allowing SSH access from your current location without
manually managing firewall rules in the DigitalOcean control panel.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAllowCurrentIP(cmd, args, dryRun, port, removeExisting)
		},
	}

	// Add command-specific flags
	allowCurrentIPCmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would be done without making actual changes")
	allowCurrentIPCmd.Flags().IntVar(&port, "port", 22,
		"Port number for SSH access (default: 22)")
	allowCurrentIPCmd.Flags().BoolVar(&removeExisting, "remove", false,
		"Remove existing SSH rules for this port and replace with current IP only")

	return allowCurrentIPCmd
}

func runAllowCurrentIP(cmd *cobra.Command, args []string, dryRun bool, port int, removeExisting bool) error {
	// Get config file from global flag
	configFile, _ := cmd.Flags().GetString("config")

	// Set configuration defaults
	config.SetDefaults()

	// Load configuration (use root command flags for global flags)
	cfg, err := config.Load(configFile, cmd.Root().PersistentFlags())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	log := logger.Get()
	log.Info("Starting allow-current-ip execution",
		zap.String("config_file", configFile),
		zap.String("log_level", cfg.LogLevel),
		zap.Bool("dry_run", dryRun),
		zap.Int("port", port),
		zap.Bool("remove_existing", removeExisting))

	// Validate port range
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port %d (must be 1-65535)", port)
	}

	// Create public IP client
	publicIPClient := publicip.NewClient(log)

	// Detect current public IP
	ctx := context.Background()
	currentIP, err := publicIPClient.GetPublicIPWithRetry(ctx, 3)
	if err != nil {
		log.Error("Failed to detect current public IP", zap.Error(err))
		return fmt.Errorf("failed to detect current public IP: %w", err)
	}

	log.Info("Detected current public IP", zap.String("ip", currentIP))

	if dryRun {
		if removeExisting {
			log.Info("DRY RUN: Would remove existing SSH rules and add current IP",
				zap.String("firewall_id", cfg.DigitalOcean.FirewallID),
				zap.String("source_ip", currentIP),
				zap.Int("port", port),
				zap.String("protocol", "tcp"))
		} else {
			log.Info("DRY RUN: Would append current IP to existing SSH rules",
				zap.String("firewall_id", cfg.DigitalOcean.FirewallID),
				zap.String("source_ip", currentIP),
				zap.Int("port", port),
				zap.String("protocol", "tcp"))
		}
		log.Info("DRY RUN: Execution completed successfully")
		return nil
	}

	// Create DigitalOcean client
	doClient := digitalocean.NewClient(cfg.DigitalOcean.APIKey, log)

	// Add SSH rule to firewall
	err = doClient.AddSSHRule(ctx, cfg.DigitalOcean.FirewallID, currentIP, port, removeExisting)
	if err != nil {
		log.Error("Failed to add SSH rule to firewall", zap.Error(err))
		return fmt.Errorf("failed to add SSH rule to firewall: %w", err)
	}

	log.Info("Successfully added current IP to firewall for SSH access",
		zap.String("firewall_id", cfg.DigitalOcean.FirewallID),
		zap.String("source_ip", currentIP),
		zap.Int("port", port))

	return nil
}
