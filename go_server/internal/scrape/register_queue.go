package scrape

import (
	"log/slog"
	"sync"
)

const (
	defaultRegisterQueueDepth = 64
)

type registerJob struct {
	fn func() error
}

// registerQueue serializes scrape writes without blocking fetch goroutines on DB I/O.
type registerQueue struct {
	jobs chan registerJob
	wg   sync.WaitGroup
}

func newRegisterQueue(depth int) *registerQueue {
	if depth <= 0 {
		depth = defaultRegisterQueueDepth
	}
	q := &registerQueue{jobs: make(chan registerJob, depth)}
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for job := range q.jobs {
			if err := job.fn(); err != nil {
				slog.Warn("scrape register failed", "error", err)
			}
		}
	}()
	return q
}

func (q *registerQueue) submit(fn func() error) error {
	q.jobs <- registerJob{fn: fn}
	return nil
}

func (q *registerQueue) closeAndWait() {
	close(q.jobs)
	q.wg.Wait()
}
