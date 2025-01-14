package db

import (
	"sync"
	"time"
)

type Metrics struct {
	mu sync.RWMutex
	// Operation counters
	TotalQueries     int64
	SuccessfulQueries int64
	FailedQueries    int64
	// Latency tracking
	TotalLatency     time.Duration
	MaxLatency       time.Duration
	// Retry metrics
	TotalRetries     int64
	SuccessfulRetries int64
}

func (m *Metrics) RecordQuery(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalQueries++
	m.TotalLatency += duration
	if duration > m.MaxLatency {
		m.MaxLatency = duration
	}

	if err == nil {
		m.SuccessfulQueries++
	} else {
		m.FailedQueries++
	}
}

func (m *Metrics) RecordRetry(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRetries++
	if success {
		m.SuccessfulRetries++
	}
}

func (m *Metrics) GetStats() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := time.Duration(0)
	if m.TotalQueries > 0 {
		avgLatency = time.Duration(int64(m.TotalLatency) / m.TotalQueries)
	}

	return map[string]string{
		"total_queries":       formatInt64(m.TotalQueries),
		"successful_queries": formatInt64(m.SuccessfulQueries),
		"failed_queries":     formatInt64(m.FailedQueries),
		"avg_latency_ms":     formatDuration(avgLatency),
		"max_latency_ms":     formatDuration(m.MaxLatency),
		"total_retries":      formatInt64(m.TotalRetries),
		"successful_retries": formatInt64(m.SuccessfulRetries),
	}
}

func formatInt64(n int64) string {
	return time.Unix(n, 0).Format(time.RFC3339)
}

func formatDuration(d time.Duration) string {
	return d.String()
}
