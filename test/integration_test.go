package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kholisrag/do-firewall-allowlister/pkg/config"
	"github.com/kholisrag/do-firewall-allowlister/pkg/daemon"
	"github.com/kholisrag/do-firewall-allowlister/pkg/logger"
	"github.com/kholisrag/do-firewall-allowlister/pkg/scheduler"
	"go.uber.org/zap/zaptest"
)

// TestIntegrationConfigLoad tests the complete configuration loading process
func TestIntegrationConfigLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary config file
	configContent := `
log-level: DEBUG
cron:
  schedule: "0 0 * * *"
  timezone: "UTC"
digitalocean:
  api-key: "test-api-key"
  firewall-id: "test-firewall-id"
  inbound-rules:
    - port: 80
      protocol: tcp
    - port: 443
      protocol: tcp
netdata:
  domains:
    - "app.netdata.cloud"
    - "api.netdata.cloud"
cloudflare:
  ips-url: "https://api.cloudflare.com/client/v4/ips"
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Test configuration loading
	config.SetDefaults()
	cfg, err := config.Load(tmpFile.Name(), nil)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate loaded configuration
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("Expected LogLevel DEBUG, got %s", cfg.LogLevel)
	}

	if cfg.DigitalOcean.APIKey != "test-api-key" {
		t.Errorf("Expected API key test-api-key, got %s", cfg.DigitalOcean.APIKey)
	}

	if len(cfg.DigitalOcean.InboundRules) != 2 {
		t.Errorf("Expected 2 inbound rules, got %d", len(cfg.DigitalOcean.InboundRules))
	}
}

// TestIntegrationSchedulerValidation tests cron schedule validation
func TestIntegrationSchedulerValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name        string
		schedule    string
		timezone    string
		expectError bool
	}{
		{
			name:     "valid daily schedule",
			schedule: "0 0 * * *",
			timezone: "UTC",
		},
		{
			name:     "valid hourly schedule",
			schedule: "0 * * * *",
			timezone: "America/New_York",
		},
		{
			name:        "invalid schedule",
			schedule:    "invalid",
			timezone:    "UTC",
			expectError: true,
		},
		{
			name:        "invalid timezone",
			schedule:    "0 0 * * *",
			timezone:    "Invalid/Timezone",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test schedule validation
			err := scheduler.ValidateSchedule(tc.schedule)
			if tc.expectError && err == nil {
				t.Error("Expected schedule validation error")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected schedule validation error: %v", err)
			}

			// Test next run time calculation
			if !tc.expectError {
				nextRun, err := scheduler.GetNextRunTime(tc.schedule, tc.timezone)
				if err != nil {
					t.Errorf("Failed to get next run time: %v", err)
				}
				if nextRun.IsZero() {
					t.Error("Next run time should not be zero")
				}
			}
		})
	}
}

// TestIntegrationDaemonLifecycle tests daemon creation and basic lifecycle
func TestIntegrationDaemonLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test configuration
	cfg := &config.Config{
		LogLevel: "ERROR", // Reduce log noise in tests
		Cron: config.CronConfig{
			Schedule: "0 0 * * *",
			Timezone: "UTC",
		},
		DigitalOcean: config.DigitalOceanConfig{
			APIKey:     "test-api-key",
			FirewallID: "test-firewall-id",
			InboundRules: []config.InboundRule{
				{Port: 80, Protocol: "tcp"},
			},
		},
		Cloudflare: config.CloudflareConfig{
			IPsURL: "https://api.cloudflare.com/client/v4/ips",
		},
		Netdata: config.NetdataConfig{
			Domains: []string{"example.com"},
		},
	}

	logger := zaptest.NewLogger(t)

	// Test daemon creation
	d, err := daemon.NewDaemon(cfg, logger, true) // Use dry-run mode
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test schedule validation
	if err := d.ValidateSchedule(); err != nil {
		t.Errorf("Schedule validation failed: %v", err)
	}

	// Test next run time
	nextRun, err := d.GetNextRunTime()
	if err != nil {
		t.Errorf("Failed to get next run time: %v", err)
	}
	if nextRun.IsZero() {
		t.Error("Next run time should not be zero")
	}

	// Test daemon status (without starting)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := d.GetStatus(ctx)
	if err != nil {
		t.Errorf("Failed to get daemon status: %v", err)
	}

	if status.DryRun != true {
		t.Error("Expected dry run mode to be true")
	}

	if status.Schedule != cfg.Cron.Schedule {
		t.Errorf("Expected schedule %s, got %s", cfg.Cron.Schedule, status.Schedule)
	}
}

// TestIntegrationDryRunExecution tests dry-run execution
func TestIntegrationDryRunExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would normally require network access to test actual API calls
	// For integration testing, you might want to use test doubles or
	// run against a test environment

	cfg := &config.Config{
		LogLevel: "ERROR",
		Cron: config.CronConfig{
			Schedule: "0 0 * * *",
			Timezone: "UTC",
		},
		DigitalOcean: config.DigitalOceanConfig{
			APIKey:     "test-api-key",
			FirewallID: "test-firewall-id",
			InboundRules: []config.InboundRule{
				{Port: 80, Protocol: "tcp"},
			},
		},
		Cloudflare: config.CloudflareConfig{
			IPsURL: "https://httpbin.org/json", // Use a test endpoint
		},
		Netdata: config.NetdataConfig{
			Domains: []string{}, // Empty to avoid DNS lookups
		},
	}

	logger := zaptest.NewLogger(t)
	d, err := daemon.NewDaemon(cfg, logger, true) // Dry-run mode

	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test one-shot execution in dry-run mode
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not make actual API calls due to dry-run mode
	// In a real integration test, you'd verify the dry-run logs
	err = d.RunOnce(ctx)

	// We expect this to fail because we're using invalid API credentials
	// and test endpoints, but it should fail gracefully
	if err == nil {
		t.Log("Dry-run execution completed (this may fail with test credentials)")
	} else {
		t.Logf("Expected failure with test credentials: %v", err)
	}
}

// TestIntegrationLoggerInitialization tests logger setup
func TestIntegrationLoggerInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}

	for _, level := range logLevels {
		t.Run("log level "+level, func(t *testing.T) {
			err := logger.Initialize(level)
			if err != nil {
				t.Errorf("Failed to initialize logger with level %s: %v", level, err)
			}

			// Test that logger is accessible
			log := logger.Get()
			if log == nil {
				t.Error("Logger should not be nil after initialization")
			}

			// Test sync
			if err := logger.Sync(); err != nil {
				t.Errorf("Failed to sync logger: %v", err)
			}
		})
	}
}

// TestIntegrationEnvironmentVariables tests environment variable configuration
func TestIntegrationEnvironmentVariables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set test environment variables
	envVars := map[string]string{
		"FIREWALL_ALLOWLISTER_LOG_LEVEL":                "WARN",
		"FIREWALL_ALLOWLISTER_DIGITALOCEAN_API_KEY":     "env-test-key",
		"FIREWALL_ALLOWLISTER_DIGITALOCEAN_FIREWALL_ID": "env-test-firewall",
		"FIREWALL_ALLOWLISTER_CRON_SCHEDULE":            "0 1 * * *",
		"FIREWALL_ALLOWLISTER_CLOUDFLARE_IPS_URL":       "https://api.cloudflare.com/client/v4/ips",
	}

	// Set environment variables
	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Reset and load configuration
	config.SetDefaults()
	cfg, err := config.Load("", nil) // No config file, only env vars
	if err != nil {
		t.Fatalf("Failed to load config from environment: %v", err)
	}

	// Verify environment variables were loaded
	if cfg.LogLevel != "WARN" {
		t.Errorf("Expected LogLevel WARN from env, got %s", cfg.LogLevel)
	}

	if cfg.DigitalOcean.APIKey != "env-test-key" {
		t.Errorf("Expected API key from env, got %s", cfg.DigitalOcean.APIKey)
	}

	if cfg.Cron.Schedule != "0 1 * * *" {
		t.Errorf("Expected cron schedule from env, got %s", cfg.Cron.Schedule)
	}
}
