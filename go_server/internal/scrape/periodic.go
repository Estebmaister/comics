package scrape

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// PeriodicConfig controls the background scrape loop (Python py-daemon scrape thread).
type PeriodicConfig struct {
	Interval time.Duration
}

// PeriodicScheduler runs scrape on a fixed interval without overlapping runs.
type PeriodicScheduler struct {
	cfg PeriodicConfig
	run func(context.Context) error
	mu  sync.Mutex
}

func NewPeriodicScheduler(cfg PeriodicConfig, run func(context.Context) error) *PeriodicScheduler {
	return &PeriodicScheduler{cfg: cfg, run: run}
}

// Start launches the loop until ctx is cancelled. The first run starts immediately.
func (s *PeriodicScheduler) Start(ctx context.Context) {
	if s.cfg.Interval <= 0 {
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
		s.runOnce(ctx, runIndex)
		runIndex++
		timer := time.NewTimer(s.cfg.Interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			slog.Info("scrape periodic scheduler stopped")
			return
		case <-timer.C:
		}
	}
}

func (s *PeriodicScheduler) runOnce(ctx context.Context, runIndex int) {
	if !s.mu.TryLock() {
		slog.Warn("scrape periodic run skipped, previous pass still running", "run", runIndex)
		return
	}
	defer s.mu.Unlock()

	started := time.Now()
	err := s.run(ctx)
	elapsed := time.Since(started).Round(time.Millisecond)
	if err != nil {
		slog.Warn("scrape periodic run failed", "run", runIndex, "elapsed", elapsed.String(), "error", err)
		return
	}
	slog.Info("scrape periodic run finished", "run", runIndex, "elapsed", elapsed.String())
}
