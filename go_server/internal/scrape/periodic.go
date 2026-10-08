package scrape

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// PeriodicConfig controls the background scrape loop.
type PeriodicConfig struct {
	Interval time.Duration
}

// PeriodicScheduler runs automatic scrapes on a cooldown-aware schedule.
type PeriodicScheduler struct {
	cfg         PeriodicConfig
	coordinator *Coordinator
}

func NewPeriodicScheduler(cfg PeriodicConfig, coordinator *Coordinator) *PeriodicScheduler {
	return &PeriodicScheduler{cfg: cfg, coordinator: coordinator}
}

// Start launches the loop until ctx is cancelled.
func (s *PeriodicScheduler) Start(ctx context.Context) {
	if s.cfg.Interval <= 0 || s.coordinator == nil {
		return
	}
	go s.loop(ctx)
}

func (s *PeriodicScheduler) loop(ctx context.Context) {
	slog.Info("scrape periodic scheduler started", "interval", s.cfg.Interval.String())
	runIndex := 1
	for {
		if ctx.Err() != nil {
			return
		}
		delay := s.coordinator.DelayUntilNextAutomatic(s.cfg.Interval)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				slog.Info("scrape periodic scheduler stopped")
				return
			case <-timer.C:
			}
		}
		if ctx.Err() != nil {
			return
		}
		s.runAutomatic(ctx, runIndex)
		runIndex++
	}
}

func (s *PeriodicScheduler) runAutomatic(ctx context.Context, runIndex int) {
	err := s.coordinator.Run(ctx, SourceAutomatic)
	if IsScrapeInProgress(err) {
		slog.Info("scrape periodic tick skipped, waiting for active run",
			"run", runIndex,
			"active_source", scrapeInProgressSource(err),
		)
		if waitErr := s.coordinator.WaitUntilIdle(ctx); waitErr != nil {
			return
		}
		return
	}
	if err != nil {
		slog.Warn("scrape periodic run failed", "run", runIndex, "error", err)
		return
	}
	slog.Info("scrape periodic run finished", "run", runIndex)
}

func scrapeInProgressSource(err error) Source {
	var busy *ErrScrapeInProgress
	if errors.As(err, &busy) {
		return busy.Running
	}
	return ""
}
