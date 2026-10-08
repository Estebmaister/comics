package scrape

import (
	"context"
	"errors"
	"sync"
	"time"

	"comics/domain"
)

// Source identifies who started a scrape run.
type Source string

const (
	SourceManual    Source = "manual"
	SourceAutomatic Source = "automatic"
	SourceCLI       Source = "cli"
)

// ErrScrapeInProgress is returned when a second scrape is requested while one is active.
type ErrScrapeInProgress struct {
	Running Source
}

func (e *ErrScrapeInProgress) Error() string {
	return "scrape already in progress (source=" + string(e.Running) + ")"
}

// Coordinator ensures a single scrape run and records completion time for scheduling.
type Coordinator struct {
	run func(context.Context) error

	mu                  sync.Mutex
	busy                bool
	busySource          Source
	lastCompletedAt     time.Time
	lastCompletedSource Source
}

func NewCoordinator(repo domain.ComicRepository) *Coordinator {
	return NewCoordinatorWithRunner(func(ctx context.Context) error {
		return RunAll(ctx, repo, nil)
	})
}

// NewCoordinatorWithRunner is used in tests or custom entry points.
func NewCoordinatorWithRunner(run func(context.Context) error) *Coordinator {
	return &Coordinator{run: run}
}

// Run starts a scrape unless another source is already running.
func (c *Coordinator) Run(ctx context.Context, source Source) error {
	if err := c.begin(source); err != nil {
		return err
	}
	err := c.run(ctx)
	c.end(source)
	return err
}

func (c *Coordinator) begin(source Source) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.busy {
		return &ErrScrapeInProgress{Running: c.busySource}
	}
	c.busy = true
	c.busySource = source
	return nil
}

func (c *Coordinator) end(source Source) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.busy = false
	c.lastCompletedAt = time.Now()
	c.lastCompletedSource = source
}

// Running reports whether a scrape is active and which source started it.
func (c *Coordinator) Running() (bool, Source) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.busy, c.busySource
}

// LastCompleted returns the end time and source of the most recent scrape (zero if none).
func (c *Coordinator) LastCompleted() (time.Time, Source) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastCompletedAt, c.lastCompletedSource
}

// DelayUntilNextAutomatic returns how long to wait before the next periodic tick.
func (c *Coordinator) DelayUntilNextAutomatic(interval time.Duration) time.Duration {
	c.mu.Lock()
	last := c.lastCompletedAt
	c.mu.Unlock()
	if last.IsZero() {
		return 0
	}
	wait := time.Until(last.Add(interval))
	if wait < 0 {
		return 0
	}
	return wait
}

// WaitUntilIdle blocks until no scrape is running or ctx is cancelled.
func (c *Coordinator) WaitUntilIdle(ctx context.Context) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		busy, _ := c.Running()
		if !busy {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func IsScrapeInProgress(err error) bool {
	var inProgress *ErrScrapeInProgress
	return errors.As(err, &inProgress)
}

// RunStatus is a point-in-time view of coordinator state (for /scrape/status).
type RunStatus struct {
	Running             bool
	ActiveSource        Source
	LastCompletedAt     time.Time
	LastCompletedSource Source
}

func (c *Coordinator) Snapshot() RunStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return RunStatus{
		Running:             c.busy,
		ActiveSource:        c.busySource,
		LastCompletedAt:     c.lastCompletedAt,
		LastCompletedSource: c.lastCompletedSource,
	}
}
