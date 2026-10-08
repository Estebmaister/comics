package scrape

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoordinatorRejectsOverlappingManual(t *testing.T) {
	var active atomic.Int32
	coord := NewCoordinatorWithRunner(func(ctx context.Context) error {
		active.Add(1)
		time.Sleep(50 * time.Millisecond)
		active.Add(-1)
		return nil
	})

	ctx := context.Background()
	go func() {
		_ = coord.Run(ctx, SourceAutomatic)
	}()

	time.Sleep(10 * time.Millisecond)
	err := coord.Run(ctx, SourceManual)
	if !IsScrapeInProgress(err) {
		t.Fatalf("expected in progress, got %v", err)
	}
	var busyErr *ErrScrapeInProgress
	if !errors.As(err, &busyErr) || busyErr.Running != SourceAutomatic {
		t.Fatalf("expected automatic running, got %v", err)
	}
}

func TestCoordinatorManualResetsAutomaticDelay(t *testing.T) {
	coord := NewCoordinatorWithRunner(func(context.Context) error { return nil })
	interval := time.Minute

	before := time.Now()
	_ = coord.Run(context.Background(), SourceManual)
	delay := coord.DelayUntilNextAutomatic(interval)
	if delay < interval-time.Second || delay > interval {
		t.Fatalf("expected ~%s delay, got %s", interval, delay)
	}
	lastAt, _ := coord.LastCompleted()
	if lastAt.Before(before) {
		t.Fatal("expected last completed to be updated")
	}
}
