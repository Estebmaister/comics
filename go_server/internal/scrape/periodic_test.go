package scrape

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPeriodicSchedulerRunsWithoutOverlap(t *testing.T) {
	var runs atomic.Int32
	block := make(chan struct{})
	coord := NewCoordinatorWithRunner(func(ctx context.Context) error {
		runs.Add(1)
		select {
		case <-block:
		case <-ctx.Done():
		}
		return nil
	})
	scheduler := NewPeriodicScheduler(PeriodicConfig{Interval: 20 * time.Millisecond}, coord)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	time.Sleep(5 * time.Millisecond)
	if runs.Load() != 1 {
		t.Fatalf("expected first run to start, got %d", runs.Load())
	}
	close(block)
	time.Sleep(45 * time.Millisecond)
	if runs.Load() < 2 {
		t.Fatalf("expected at least two runs, got %d", runs.Load())
	}
}
