package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler manages cron-based scheduling
type Scheduler struct {
	cron     *cron.Cron
	logger   *zap.Logger
	timezone *time.Location
}

// JobFunc represents a function that can be scheduled
type JobFunc func(ctx context.Context) error

// NewScheduler creates a new scheduler with the specified timezone
func NewScheduler(timezone string, logger *zap.Logger) (*Scheduler, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", timezone, err)
	}

	c := cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(), // Enable seconds precision
		cron.WithLogger(cron.DefaultLogger),
	)

	return &Scheduler{
		cron:     c,
		logger:   logger.Named("scheduler"),
		timezone: loc,
	}, nil
}

// AddJob adds a job to the scheduler with the specified cron expression
func (s *Scheduler) AddJob(schedule string, jobName string, job JobFunc) error {
	s.logger.Info("Adding scheduled job",
		zap.String("job_name", jobName),
		zap.String("schedule", schedule),
		zap.String("timezone", s.timezone.String()))

	wrappedJob := s.wrapJob(jobName, job)

	_, err := s.cron.AddFunc(schedule, wrappedJob)
	if err != nil {
		s.logger.Error("Failed to add scheduled job",
			zap.String("job_name", jobName),
			zap.String("schedule", schedule),
			zap.Error(err))
		return fmt.Errorf("failed to add job %s with schedule %s: %w", jobName, schedule, err)
	}

	s.logger.Info("Successfully added scheduled job",
		zap.String("job_name", jobName),
		zap.String("schedule", schedule))

	return nil
}

// wrapJob wraps a JobFunc with logging and error handling
func (s *Scheduler) wrapJob(jobName string, job JobFunc) func() {
	return func() {
		ctx := context.Background()

		s.logger.Info("Starting scheduled job execution", zap.String("job_name", jobName))
		startTime := time.Now()

		err := job(ctx)
		duration := time.Since(startTime)

		if err != nil {
			s.logger.Error("Scheduled job failed",
				zap.String("job_name", jobName),
				zap.Duration("duration", duration),
				zap.Error(err))
		} else {
			s.logger.Info("Scheduled job completed successfully",
				zap.String("job_name", jobName),
				zap.Duration("duration", duration))
		}
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.logger.Info("Starting scheduler", zap.String("timezone", s.timezone.String()))
	s.cron.Start()
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping scheduler")
	ctx := s.cron.Stop()

	// Wait for running jobs to complete
	select {
	case <-ctx.Done():
		s.logger.Info("Scheduler stopped gracefully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Scheduler stop timeout, some jobs may have been interrupted")
	}
}

// GetEntries returns information about scheduled jobs
func (s *Scheduler) GetEntries() []EntryInfo {
	entries := s.cron.Entries()
	var info []EntryInfo

	for _, entry := range entries {
		info = append(info, EntryInfo{
			ID:       entry.ID,
			Schedule: entry.Schedule.Next(time.Now()).Format(time.RFC3339),
			Next:     entry.Next,
			Prev:     entry.Prev,
		})
	}

	return info
}

// EntryInfo contains information about a scheduled job
type EntryInfo struct {
	ID       cron.EntryID `json:"id"`
	Schedule string       `json:"schedule"`
	Next     time.Time    `json:"next"`
	Prev     time.Time    `json:"prev"`
}

// IsRunning returns true if the scheduler is running
func (s *Scheduler) IsRunning() bool {
	// Check if any entries exist and the cron is started
	entries := s.cron.Entries()
	return len(entries) > 0
}

// ValidateSchedule validates a cron schedule expression
func ValidateSchedule(schedule string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(schedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule %s: %w", schedule, err)
	}
	return nil
}

// GetNextRunTime returns the next scheduled run time for a given schedule
func GetNextRunTime(schedule string, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %s: %w", timezone, err)
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, err := parser.Parse(schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron schedule %s: %w", schedule, err)
	}

	now := time.Now().In(loc)
	next := sched.Next(now)

	return next, nil
}

// RunOnce executes a job immediately (for testing or one-shot execution)
func (s *Scheduler) RunOnce(jobName string, job JobFunc) error {
	s.logger.Info("Running job once", zap.String("job_name", jobName))

	ctx := context.Background()
	startTime := time.Now()

	err := job(ctx)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("One-shot job failed",
			zap.String("job_name", jobName),
			zap.Duration("duration", duration),
			zap.Error(err))
		return fmt.Errorf("one-shot job %s failed: %w", jobName, err)
	}

	s.logger.Info("One-shot job completed successfully",
		zap.String("job_name", jobName),
		zap.Duration("duration", duration))

	return nil
}
