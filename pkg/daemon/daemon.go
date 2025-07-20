package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kholisrag/do-firewall-allowlister/pkg/config"
	"github.com/kholisrag/do-firewall-allowlister/pkg/scheduler"
	"github.com/kholisrag/do-firewall-allowlister/pkg/service"
	"go.uber.org/zap"
)

// Daemon manages the long-running service
type Daemon struct {
	config    *config.Config
	service   *service.Service
	scheduler *scheduler.Scheduler
	logger    *zap.Logger
	dryRun    bool
}

// NewDaemon creates a new daemon instance
func NewDaemon(cfg *config.Config, logger *zap.Logger, dryRun bool) (*Daemon, error) {
	// Create service
	svc := service.NewService(cfg, logger, dryRun)

	// Create scheduler
	sched, err := scheduler.NewScheduler(cfg.Cron.Timezone, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	return &Daemon{
		config:    cfg,
		service:   svc,
		scheduler: sched,
		logger:    logger.Named("daemon"),
		dryRun:    dryRun,
	}, nil
}

// Start starts the daemon with graceful shutdown handling
func (d *Daemon) Start(ctx context.Context) error {
	d.logger.Info("Starting daemon",
		zap.String("schedule", d.config.Cron.Schedule),
		zap.String("timezone", d.config.Cron.Timezone),
		zap.Bool("dry_run", d.dryRun))

	// Validate configuration before starting
	if err := d.service.ValidateConfiguration(ctx); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Add the firewall update job to scheduler
	jobFunc := func(ctx context.Context) error {
		return d.service.UpdateFirewallRules(ctx)
	}

	if err := d.scheduler.AddJob(d.config.Cron.Schedule, "firewall-update", jobFunc); err != nil {
		return fmt.Errorf("failed to add scheduled job: %w", err)
	}

	// Start the scheduler
	d.scheduler.Start()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	d.logger.Info("Daemon started successfully, waiting for signals or context cancellation")

	// Wait for shutdown signal or context cancellation
	select {
	case sig := <-sigChan:
		d.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		d.logger.Info("Context cancelled, shutting down")
	}

	// Graceful shutdown
	d.logger.Info("Initiating graceful shutdown")
	d.shutdown()

	d.logger.Info("Daemon stopped")
	return nil
}

// shutdown performs graceful shutdown
func (d *Daemon) shutdown() {
	// Stop the scheduler
	d.scheduler.Stop()

	d.logger.Info("Graceful shutdown completed")
}

// RunOnce runs the firewall update job once and exits
func (d *Daemon) RunOnce(ctx context.Context) error {
	d.logger.Info("Running firewall update once", zap.Bool("dry_run", d.dryRun))

	// Validate configuration
	if err := d.service.ValidateConfiguration(ctx); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Run the update job
	if err := d.service.UpdateFirewallRules(ctx); err != nil {
		return fmt.Errorf("firewall update failed: %w", err)
	}

	d.logger.Info("One-shot execution completed successfully")
	return nil
}

// GetStatus returns the current status of the daemon and its services
func (d *Daemon) GetStatus(ctx context.Context) (*DaemonStatus, error) {
	status := &DaemonStatus{
		IsRunning: d.scheduler.IsRunning(),
		DryRun:    d.dryRun,
		Schedule:  d.config.Cron.Schedule,
		Timezone:  d.config.Cron.Timezone,
	}

	// Get scheduler entries
	entries := d.scheduler.GetEntries()
	for _, entry := range entries {
		status.ScheduledJobs = append(status.ScheduledJobs, ScheduledJobInfo{
			ID:       int(entry.ID),
			Next:     entry.Next,
			Previous: entry.Prev,
		})
	}

	// Get service status
	serviceStatus, err := d.service.GetStatus(ctx)
	if err != nil {
		d.logger.Error("Failed to get service status", zap.Error(err))
		status.ServiceStatus = nil
		status.StatusError = err.Error()
	} else {
		status.ServiceStatus = serviceStatus
	}

	return status, nil
}

// ValidateSchedule validates the cron schedule
func (d *Daemon) ValidateSchedule() error {
	return scheduler.ValidateSchedule(d.config.Cron.Schedule)
}

// GetNextRunTime returns the next scheduled run time
func (d *Daemon) GetNextRunTime() (time.Time, error) {
	return scheduler.GetNextRunTime(d.config.Cron.Schedule, d.config.Cron.Timezone)
}

// DaemonStatus represents the current status of the daemon
type DaemonStatus struct {
	IsRunning     bool               `json:"is_running"`
	DryRun        bool               `json:"dry_run"`
	Schedule      string             `json:"schedule"`
	Timezone      string             `json:"timezone"`
	ScheduledJobs []ScheduledJobInfo `json:"scheduled_jobs"`
	ServiceStatus *service.Status    `json:"service_status,omitempty"`
	StatusError   string             `json:"status_error,omitempty"`
}

// ScheduledJobInfo contains information about a scheduled job
type ScheduledJobInfo struct {
	ID       int       `json:"id"`
	Next     time.Time `json:"next"`
	Previous time.Time `json:"previous"`
}

// StartWithTimeout starts the daemon with a timeout for testing
func (d *Daemon) StartWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return d.Start(ctx)
}

// Health performs a health check of the daemon and its dependencies
func (d *Daemon) Health(ctx context.Context) error {
	d.logger.Debug("Performing health check")

	// Check if scheduler is running (if daemon is started)
	if d.scheduler.IsRunning() {
		d.logger.Debug("Scheduler is running")
	}

	// Validate configuration
	if err := d.service.ValidateConfiguration(ctx); err != nil {
		d.logger.Error("Health check failed: configuration validation error", zap.Error(err))
		return fmt.Errorf("health check failed: %w", err)
	}

	d.logger.Debug("Health check passed")
	return nil
}
