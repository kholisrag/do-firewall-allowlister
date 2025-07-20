package commands

import (
	"context"
	"fmt"

	"github.com/kholisrag/do-firewall-allowlister/pkg/config"
	"github.com/kholisrag/do-firewall-allowlister/pkg/daemon"
	"github.com/kholisrag/do-firewall-allowlister/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewOneshotCommand creates and returns the oneshot command
func NewOneshotCommand() *cobra.Command {
	var oneshotDryRun bool

	oneshotCmd := &cobra.Command{
		Use:   "oneshot",
		Short: "Run the firewall allowlister once and exit",
		Long: `Run the firewall allowlister once to update DigitalOcean firewall rules and then exit.

This command will:
- Fetch Cloudflare IP ranges
- Resolve Netdata domain IPs
- Update DigitalOcean firewall rules
- Exit after completion

This is useful for manual execution, testing, or integration with external schedulers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOneshot(cmd, args, oneshotDryRun)
		},
	}

	// Add command-specific flags
	oneshotCmd.Flags().BoolVar(&oneshotDryRun, "dry-run", false,
		"Show what would be done without making actual changes")

	return oneshotCmd
}

func runOneshot(cmd *cobra.Command, args []string, dryRun bool) error {
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
	log.Info("Starting firewall allowlister one-shot execution",
		zap.String("config_file", configFile),
		zap.String("log_level", cfg.LogLevel),
		zap.Bool("dry_run", dryRun))

	// Create daemon (we use daemon for the business logic)
	d, err := daemon.NewDaemon(cfg, log, dryRun)
	if err != nil {
		log.Error("Failed to create daemon", zap.Error(err))
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Run once
	ctx := context.Background()
	if err := d.RunOnce(ctx); err != nil {
		log.Error("One-shot execution failed", zap.Error(err))
		return fmt.Errorf("one-shot execution failed: %w", err)
	}

	log.Info("One-shot execution completed successfully")
	return nil
}
