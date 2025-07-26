package commands

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates and returns the root cobra command
func NewRootCommand(buildInfo BuildInfo) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "do-firewall-allowlister",
		Short: "DigitalOcean Firewall Allowlister manages firewall rules for Cloudflare and Netdata IPs",
		Long: `A service that automatically updates DigitalOcean firewall rules to allow traffic from:
- Cloudflare IP ranges
- Netdata domain IPs

The service can run as a daemon with scheduled updates or as a one-shot command.`,
	}

	// Add global persistent flags that are common across all commands
	rootCmd.PersistentFlags().StringP("config", "c", "config.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().String("log-level", "", "Log level (DEBUG, INFO, WARN, ERROR, FATAL)")
	rootCmd.PersistentFlags().String("digitalocean.api-key", "", "DigitalOcean API key")
	rootCmd.PersistentFlags().String("digitalocean.firewall-id", "", "DigitalOcean firewall ID")
	rootCmd.PersistentFlags().String("cron.schedule", "", "Cron schedule expression")
	rootCmd.PersistentFlags().String("cron.timezone", "", "Timezone for cron schedule")
	rootCmd.PersistentFlags().String("cloudflare.ips-url", "", "Cloudflare IPs API URL")

	// Add subcommands
	rootCmd.AddCommand(NewDaemonCommand())
	rootCmd.AddCommand(NewOneshotCommand())
	rootCmd.AddCommand(NewAllowCurrentIPCommand())
	rootCmd.AddCommand(NewValidateCommand())
	rootCmd.AddCommand(NewVersionCommand(buildInfo))

	return rootCmd
}
