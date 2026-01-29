package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/onelineai/hana-news-api/internal/service"
)

// Scheduler manages background batch jobs
type Scheduler struct {
	scheduler    gocron.Scheduler
	batchService *service.BatchService
	logger       *slog.Logger
	interval     time.Duration
}

func New(batchService *service.BatchService, interval time.Duration, logger *slog.Logger) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		scheduler:    s,
		batchService: batchService,
		logger:       logger,
		interval:     interval,
	}, nil
}

// Start begins the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	// Define the batch sync job
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(s.interval),
		gocron.NewTask(s.runBatchSync, ctx),
		gocron.WithSingletonMode(gocron.LimitModeReschedule), // Prevent overlapping runs
		gocron.WithName("batch-sync"),
	)
	if err != nil {
		return err
	}

	// Run initial sync immediately
	go func() {
		s.logger.Info("running initial batch sync")
		s.runBatchSync(ctx)
	}()

	// Start the scheduler
	s.scheduler.Start()
	s.logger.Info("scheduler started", "interval", s.interval)

	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() error {
	s.logger.Info("stopping scheduler")
	return s.scheduler.Shutdown()
}

func (s *Scheduler) runBatchSync(ctx context.Context) {
	s.logger.Info("batch sync job triggered")
	if err := s.batchService.SyncAll(ctx); err != nil {
		s.logger.Error("batch sync failed", "error", err)
	}
}
