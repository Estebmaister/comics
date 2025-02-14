package repo

import (
	"fmt"
	"sync"
	"time"
)

// Metrics represents the metrics for a database connection
type Metrics struct {
	// Mutex for thread-safe access
	mu sync.RWMutex

	// Operation counters
	TotalQueries      int64
	SuccessfulQueries int64
	FailedQueries     int64

	// Latency tracking
	TotalLatency   time.Duration
	MaxLatency     time.Duration
	AverageLatency time.Duration

	// Connection metrics
	TotalConnectionTime   time.Duration
	AverageConnectionTime time.Duration
	LastConnectionTime    time.Time
	ConnectionCount       int64
	ActiveConnections     int64

	// Retry metrics
	TotalRetries      int64
	SuccessfulRetries int64

	// Error metrics
	ErrorCount    int64
	LastErrorTime time.Time
	LastConnError error
}

// RecordQuery updates metrics for a database query
func (m *Metrics) RecordQuery(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalQueries++

	if err != nil {
		m.FailedQueries++
		m.ErrorCount++
		m.LastErrorTime = time.Now()
		m.LastConnError = err
	} else {
		m.SuccessfulQueries++
	}

	m.TotalLatency += duration

	if duration > m.MaxLatency {
		m.MaxLatency = duration
	}
}

// RecordRetry updates metrics for a retry attempt
func (m *Metrics) RecordRetry(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRetries++

	if success {
		m.SuccessfulRetries++
	}
}

// RecordConnection updates metrics for a database connection
func (m *Metrics) RecordConnection(connectionTime time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ConnectionCount++

	if err != nil {
		// Handle connection error
		m.ErrorCount++
		m.LastErrorTime = time.Now()
		m.LastConnError = err
	} else {
		// Successful connection
		m.ActiveConnections++
		m.TotalConnectionTime += connectionTime
		m.LastConnectionTime = time.Now()
	}
}

// ReleaseConnection updates metrics when a connection is released
func (m *Metrics) ReleaseConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ActiveConnections--
}

// Reset clears all accumulated metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalQueries = 0
	m.SuccessfulQueries = 0
	m.FailedQueries = 0
	m.TotalLatency = 0
	m.MaxLatency = 0
	m.ConnectionCount = 0
	m.ActiveConnections = 0
	m.TotalConnectionTime = 0
	m.AverageConnectionTime = 0
	m.TotalRetries = 0
	m.SuccessfulRetries = 0
	m.ErrorCount = 0
	m.LastErrorTime = time.Time{}
	m.LastConnectionTime = time.Time{}
	m.LastConnError = nil
}

func (m *Metrics) GetStats() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := time.Duration(0)
	if m.TotalQueries > 0 {
		avgLatency = time.Duration(int64(m.TotalLatency) / m.TotalQueries)
	}
	// Calculate average connection time
	if m.ConnectionCount > 0 {
		m.AverageConnectionTime =
			m.TotalConnectionTime / time.Duration(m.ConnectionCount)
	}

	return map[string]string{
		// Operation counters
		"total_queries":      formatInt64(m.TotalQueries),
		"successful_queries": formatInt64(m.SuccessfulQueries),
		"failed_queries":     formatInt64(m.FailedQueries),

		// Latency tracking
		"total_latency": formatDuration(m.TotalLatency),
		"avg_latency":   formatDuration(avgLatency),
		"max_latency":   formatDuration(m.MaxLatency),

		// Connection metrics
		"total_connection_time": formatDuration(m.TotalConnectionTime),
		"avg_connection_time":   formatDuration(m.AverageConnectionTime),
		"last_connection_time":  m.LastConnectionTime.String(),
		"connection_count":      formatInt64(m.ConnectionCount),
		"active_connections":    formatInt64(m.ActiveConnections),

		// Error metrics
		"error_count":     formatInt64(m.ErrorCount),
		"last_error_time": m.LastErrorTime.String(),
		"last_connection_error": func() string {
			if m.LastConnError != nil {
				return m.LastConnError.Error()
			} else {
				return "No errors recorded"
			}
		}(),

		// Retry metrics
		"total_retries":      formatInt64(m.TotalRetries),
		"successful_retries": formatInt64(m.SuccessfulRetries),
	}
}

func formatInt64(n int64) string {
	return fmt.Sprintf("%d", n)
}

func formatDuration(d time.Duration) string {
	return d.String()
}
