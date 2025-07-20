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

// NewDaemonCommand creates and returns the daemon command
func NewDaemonCommand() *cobra.Command {
	var daemonDryRun bool

	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the firewall allowlister as a daemon with scheduled updates",
		Long: `Run the firewall allowlister as a daemon that continuously monitors and updates
DigitalOcean firewall rules based on the configured schedule.

The daemon will:
- Fetch Cloudflare IP ranges
- Resolve Netdata domain IPs
- Update DigitalOcean firewall rules
- Run on the configured cron schedule
- Handle graceful shutdown on SIGINT/SIGTERM`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon(cmd, args, daemonDryRun)
		},
	}

	// Add command-specific flags
	daemonCmd.Flags().BoolVar(&daemonDryRun, "dry-run", false, "Show what would be done without making actual changes")

	return daemonCmd
}

func runDaemon(cmd *cobra.Command, args []string, dryRun bool) error {
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
	log.Info("Starting firewall allowlister daemon",
		zap.String("schedule", cfg.Cron.Schedule),
		zap.String("timezone", cfg.Cron.Timezone),
		zap.Bool("dry_run", dryRun),
	)

	// Create daemon instance
	d, err := daemon.NewDaemon(cfg, log, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Run daemon
	ctx := context.Background()
	if err := d.Start(ctx); err != nil {
		return fmt.Errorf("daemon failed: %w", err)
	}

	return nil
}
