package config

import (
	"os"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		envVars     map[string]string
		flags       map[string]string
		expectError bool
		validate    func(*Config) error
	}{
		{
			name:       "valid config file",
			configFile: "testdata/valid_config.yaml",
			validate: func(cfg *Config) error {
				if cfg.LogLevel != "DEBUG" {
					t.Errorf("expected LogLevel DEBUG, got %s", cfg.LogLevel)
				}
				if cfg.Cron.Schedule != "0 0 * * *" {
					t.Errorf("expected cron schedule '0 0 * * *', got %s", cfg.Cron.Schedule)
				}
				if cfg.DigitalOcean.APIKey != "test-api-key" {
					t.Errorf("expected API key 'test-api-key', got %s", cfg.DigitalOcean.APIKey)
				}
				return nil
			},
		},
		{
			name:        "missing config file",
			configFile:  "nonexistent.yaml",
			expectError: true,
		},
		{
			name:       "environment variable override",
			configFile: "", // No config file, only env vars
			envVars: map[string]string{
				"FIREWALL_ALLOWLISTER_LOG_LEVEL":                "ERROR",
				"FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY":     "env-api-key",
				"FIREWALL_ALLOWLISTER_DIGITALOCEAN_FIREWALL_ID": "env-firewall-id",
				"FIREWALL_ALLOWLISTER_CLOUDFLARE_IPS_URL":       "https://api.cloudflare.com/client/v4/ips",
				"FIREWALL_ALLOWLISTER_CRON_SCHEDULE":            "0 1 * * *",
			},
			validate: func(cfg *Config) error {
				if cfg.LogLevel != "ERROR" {
					t.Errorf("expected LogLevel ERROR from env, got %s", cfg.LogLevel)
				}
				if cfg.DigitalOcean.APIKey != "env-api-key" {
					t.Errorf("expected API key from env, got %s", cfg.DigitalOcean.APIKey)
				}
				if cfg.DigitalOcean.FirewallID != "env-firewall-id" {
					t.Errorf("expected firewall ID from env, got %s", cfg.DigitalOcean.FirewallID)
				}
				return nil
			},
		},
		{
			name:       "flag override",
			configFile: "", // No config file, only flags
			flags: map[string]string{
				"log-level":                "WARN",
				"digitalocean.api-key":     "flag-api-key",
				"digitalocean.firewall-id": "flag-firewall-id",
				"cloudflare.ips-url":       "https://api.cloudflare.com/client/v4/ips",
				"cron.schedule":            "0 2 * * *",
			},
			validate: func(cfg *Config) error {
				if cfg.LogLevel != "WARN" {
					t.Errorf("expected LogLevel WARN from flag, got %s", cfg.LogLevel)
				}
				if cfg.DigitalOcean.APIKey != "flag-api-key" {
					t.Errorf("expected API key from flag, got %s", cfg.DigitalOcean.APIKey)
				}
				if cfg.DigitalOcean.FirewallID != "flag-firewall-id" {
					t.Errorf("expected firewall ID from flag, got %s", cfg.DigitalOcean.FirewallID)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset koanf instance
			k = koanf.New(".")
			SetDefaults()

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Create flags
			var flags *pflag.FlagSet
			if len(tt.flags) > 0 {
				flags = pflag.NewFlagSet("test", pflag.ContinueOnError)
				for key := range tt.flags {
					flags.String(key, "", "test flag")
				}
				for key, value := range tt.flags {
					flags.Set(key, value)
				}
			}

			cfg, err := Load(tt.configFile, flags)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				if err := tt.validate(cfg); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				LogLevel: "INFO",
				Cron: CronConfig{
					Schedule: "0 0 * * *",
					Timezone: "UTC",
				},
				DigitalOcean: DigitalOceanConfig{
					APIKey:     "test-key",
					FirewallID: "test-firewall",
					InboundRules: []InboundRule{
						{Port: 80, Protocol: "tcp"},
						{Port: 443, Protocol: "tcp"},
					},
				},
				Cloudflare: CloudflareConfig{
					IPsURL: "https://api.cloudflare.com/client/v4/ips",
				},
			},
		},
		{
			name: "missing API key",
			config: &Config{
				LogLevel: "INFO",
				Cron: CronConfig{
					Schedule: "0 0 * * *",
				},
				DigitalOcean: DigitalOceanConfig{
					FirewallID: "test-firewall",
				},
				Cloudflare: CloudflareConfig{
					IPsURL: "https://api.cloudflare.com/client/v4/ips",
				},
			},
			expectError: true,
			errorMsg:    "digitalocean.api-key is required",
		},
		{
			name: "invalid log level",
			config: &Config{
				LogLevel: "INVALID",
				Cron: CronConfig{
					Schedule: "0 0 * * *",
				},
				DigitalOcean: DigitalOceanConfig{
					APIKey:     "test-key",
					FirewallID: "test-firewall",
				},
				Cloudflare: CloudflareConfig{
					IPsURL: "https://api.cloudflare.com/client/v4/ips",
				},
			},
			expectError: true,
			errorMsg:    "invalid log level",
		},
		{
			name: "invalid port",
			config: &Config{
				LogLevel: "INFO",
				Cron: CronConfig{
					Schedule: "0 0 * * *",
				},
				DigitalOcean: DigitalOceanConfig{
					APIKey:     "test-key",
					FirewallID: "test-firewall",
					InboundRules: []InboundRule{
						{Port: 0, Protocol: "tcp"},
					},
				},
				Cloudflare: CloudflareConfig{
					IPsURL: "https://api.cloudflare.com/client/v4/ips",
				},
			},
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "invalid protocol",
			config: &Config{
				LogLevel: "INFO",
				Cron: CronConfig{
					Schedule: "0 0 * * *",
				},
				DigitalOcean: DigitalOceanConfig{
					APIKey:     "test-key",
					FirewallID: "test-firewall",
					InboundRules: []InboundRule{
						{Port: 80, Protocol: "invalid"},
					},
				},
				Cloudflare: CloudflareConfig{
					IPsURL: "https://api.cloudflare.com/client/v4/ips",
				},
			},
			expectError: true,
			errorMsg:    "invalid protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	// Reset koanf instance
	k = koanf.New(".")

	SetDefaults()

	if k.String("log-level") != "INFO" {
		t.Errorf("expected default log-level INFO, got %s", k.String("log-level"))
	}

	if k.String("cron.schedule") != "0 0 * * *" {
		t.Errorf("expected default cron schedule '0 0 * * *', got %s", k.String("cron.schedule"))
	}

	if k.String("cron.timezone") != "UTC" {
		t.Errorf("expected default timezone UTC, got %s", k.String("cron.timezone"))
	}

	if k.String("cloudflare.ips-url") != "https://api.cloudflare.com/client/v4/ips" {
		t.Errorf("expected default Cloudflare URL, got %s", k.String("cloudflare.ips-url"))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
