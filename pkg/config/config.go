package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// Config represents the application configuration
type Config struct {
	LogLevel     string             `koanf:"log-level" yaml:"log-level"`
	Cron         CronConfig         `koanf:"cron" yaml:"cron"`
	DigitalOcean DigitalOceanConfig `koanf:"digitalocean" yaml:"digitalocean"`
	Netdata      NetdataConfig      `koanf:"netdata" yaml:"netdata"`
	Cloudflare   CloudflareConfig   `koanf:"cloudflare" yaml:"cloudflare"`
}

// CronConfig represents cron scheduling configuration
type CronConfig struct {
	Schedule string `koanf:"schedule" yaml:"schedule"`
	Timezone string `koanf:"timezone" yaml:"timezone"`
}

// DigitalOceanConfig represents DigitalOcean API configuration
type DigitalOceanConfig struct {
	APIKey       string        `koanf:"api-key" yaml:"api-key"`
	FirewallID   string        `koanf:"firewall-id" yaml:"firewall-id"`
	InboundRules []InboundRule `koanf:"inbound-rules" yaml:"inbound-rules"`
}

// InboundRule represents a firewall inbound rule
type InboundRule struct {
	Port     int    `koanf:"port" yaml:"port"`
	Protocol string `koanf:"protocol" yaml:"protocol"`
}

// NetdataConfig represents Netdata domains configuration
type NetdataConfig struct {
	Domains []string `koanf:"domains" yaml:"domains"`
}

// CloudflareConfig represents Cloudflare API configuration
type CloudflareConfig struct {
	IPsURL string `koanf:"ips-url" yaml:"ips-url"`
}

var k = koanf.New(".")

// Load loads configuration from YAML file, environment variables, and command line flags
// Priority: CLI flags (highest) > Environment variables > YAML file (lowest)
func Load(configFile string, flags *pflag.FlagSet) (*Config, error) {
	// Create a new koanf instance for this load operation
	loader := koanf.New(".")

	// Load defaults first (lowest priority)
	_ = loader.Set("log-level", "INFO")
	_ = loader.Set("cron.schedule", "0 0 * * *") // Standard 5-field format: minute hour day month weekday
	_ = loader.Set("cron.timezone", "UTC")
	_ = loader.Set("cloudflare.ips-url", "https://api.cloudflare.com/client/v4/ips")

	// Load from YAML file (low priority)
	if configFile != "" {
		if err := loader.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
	}

	// Load from environment variables (medium priority)
	// Environment variables should be prefixed with FIREWALL_ALLOWLISTER_
	// and use underscores instead of dashes (e.g., FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY -> digitalocean-api-key)
	if err := loader.Load(env.Provider("FIREWALL_ALLOWLISTER_", ".", func(s string) string {
		// Remove prefix and convert to lowercase
		key := strings.ToLower(strings.TrimPrefix(s, "FIREWALL_ALLOWLISTER_"))

		// Handle specific mappings for nested structures
		switch key {
		case "digitalocean_api_key":
			return "digitalocean.api-key"
		case "digitalocean_firewall_id":
			return "digitalocean.firewall-id"
		case "cloudflare_ips_url":
			return "cloudflare.ips-url"
		case "cron_schedule":
			return "cron.schedule"
		case "cron_timezone":
			return "cron.timezone"
		case "log_level":
			return "log-level"
		default:
			// For other cases, replace first underscore with dot for section.key pattern
			parts := strings.SplitN(key, "_", 2)
			if len(parts) == 2 {
				return parts[0] + "." + strings.ReplaceAll(parts[1], "_", "-")
			}
			return key
		}
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Load from command line flags (highest priority)
	if flags != nil {
		// Manually load flags with proper key mapping
		flags.VisitAll(func(f *pflag.Flag) {
			if f.Changed {
				key := f.Name
				// Handle specific flag mappings
				switch key {
				case "log-level":
					key = "log-level"
				case "digitalocean.api-key":
					key = "digitalocean.api-key"
				case "digitalocean.firewall-id":
					key = "digitalocean.firewall-id"
				case "cloudflare.ips-url":
					key = "cloudflare.ips-url"
				case "cron.schedule":
					key = "cron.schedule"
				case "cron.timezone":
					key = "cron.timezone"
				default:
					// Keep hyphens as-is for other flags
					key = f.Name
				}

				_ = loader.Set(key, f.Value.String())
			}
		})
	}

	// Unmarshal into Config struct
	var config Config
	if err := loader.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validate performs basic validation on the configuration
func validate(config *Config) error {
	if config.DigitalOcean.APIKey == "" {
		return fmt.Errorf("digitalocean.api-key is required")
	}

	if config.DigitalOcean.FirewallID == "" {
		return fmt.Errorf("digitalocean.firewall-id is required")
	}

	if config.Cloudflare.IPsURL == "" {
		return fmt.Errorf("cloudflare.ips-url is required")
	}

	if config.Cron.Schedule == "" {
		return fmt.Errorf("cron.schedule is required")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
		"FATAL": true,
	}
	if !validLogLevels[strings.ToUpper(config.LogLevel)] {
		return fmt.Errorf("invalid log level: %s (must be DEBUG, INFO, WARN, ERROR, or FATAL)", config.LogLevel)
	}

	// Validate inbound rules
	for i, rule := range config.DigitalOcean.InboundRules {
		if rule.Port <= 0 || rule.Port > 65535 {
			return fmt.Errorf("invalid port %d in inbound rule %d (must be 1-65535)", rule.Port, i)
		}
		if rule.Protocol != "tcp" && rule.Protocol != "udp" && rule.Protocol != "icmp" {
			return fmt.Errorf("invalid protocol %s in inbound rule %d (must be tcp, udp, or icmp)", rule.Protocol, i)
		}
	}

	return nil
}

// SetDefaults sets default values for configuration
func SetDefaults() {
	_ = k.Set("log-level", "INFO")
	_ = k.Set("cron.schedule", "0 0 * * *") // Standard 5-field format: minute hour day month weekday
	_ = k.Set("cron.timezone", "UTC")
	_ = k.Set("cloudflare.ips-url", "https://api.cloudflare.com/client/v4/ips")
}

// GetKoanf returns the koanf instance for advanced usage
func GetKoanf() *koanf.Koanf {
	return k
}
