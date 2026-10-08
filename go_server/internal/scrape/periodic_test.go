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
	scheduler := NewPeriodicScheduler(PeriodicConfig{Interval: 20 * time.Millisecond}, func(ctx context.Context) error {
		runs.Add(1)
		select {
		case <-block:
		case <-ctx.Done():
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	time.Sleep(5 * time.Millisecond)
	if runs.Load() != 1 {
		t.Fatalf("expected first run to start, got %d", runs.Load())
	}
	close(block)
	time.Sleep(35 * time.Millisecond)
	if runs.Load() < 2 {
		t.Fatalf("expected at least two runs, got %d", runs.Load())
	}
}
