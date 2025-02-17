package repo

import (
	"fmt"
	"sync"
	"time"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	RecordQuery(duration time.Duration, err error)
	RecordRetry(success bool)
	RecordConnection(connectionTime time.Duration, err error)
	CloseConnection(connectionTime time.Duration, err error)
	ReleaseConnection()
	RetrieveConnection()
	Reset()
	GetStats() map[string]string
}

// Metrics represents the metrics for a database connection
type Metrics struct {
	// Mutex for thread-safe access
	mu sync.RWMutex

	// Operation counters
	TotalQueries      int64
	SuccessfulQueries int64
	FailedQueries     int64

	// Latency tracking
	TotalLatency time.Duration
	MaxLatency   time.Duration

	// Connection tracking
	TotalConnectionTime   time.Duration
	AverageConnectionTime time.Duration
	LastConnectionTime    time.Time

	TotalCreatedConnections uint64
	TotalClosedConnections  uint64
	ActiveConnections       int64
	IdleConnections         int64

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

	m.TotalCreatedConnections++

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

// CloseConnection updates metrics when a connection is closed
func (m *Metrics) CloseConnection(connectionTime time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalClosedConnections++
	if m.ActiveConnections > 0 {
		m.ActiveConnections--
	} else {
		m.IdleConnections--
	}
	m.TotalConnectionTime += connectionTime

	// Handle close error
	if err != nil {
		m.ErrorCount++
		m.LastErrorTime = time.Now()
		m.LastConnError = err
	}
}

// ReleaseConnection updates metrics when a connection is released from the pool
func (m *Metrics) ReleaseConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ActiveConnections--
	m.IdleConnections++
}

// RetrieveConnection updates metrics when a connection is retrieved from the pool
func (m *Metrics) RetrieveConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ActiveConnections++
	m.IdleConnections--
}

// Reset clears all accumulated metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Operation counters
	m.TotalQueries = 0
	m.SuccessfulQueries = 0
	m.FailedQueries = 0

	// Latency tracking
	m.TotalLatency = 0
	m.MaxLatency = 0

	// Connection tracking
	m.TotalConnectionTime = 0
	m.AverageConnectionTime = 0
	m.LastConnectionTime = time.Time{}

	m.TotalCreatedConnections = 0
	m.TotalClosedConnections = 0
	m.ActiveConnections = 0
	m.IdleConnections = 0

	// Retry tracking
	m.TotalRetries = 0
	m.SuccessfulRetries = 0

	// Error tracking
	m.ErrorCount = 0
	m.LastErrorTime = time.Time{}
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
	if m.TotalCreatedConnections > 0 {
		m.AverageConnectionTime =
			m.TotalConnectionTime / time.Duration(m.TotalCreatedConnections)
	}

	return map[string]string{
		// Operation counters
		"queries_count":      formatNumber(m.TotalQueries),
		"queries_successful": formatNumber(m.SuccessfulQueries),
		"queries_failed":     formatNumber(m.FailedQueries),

		// Latency tracking
		"latency_total": formatTime(m.TotalLatency),
		"latency_avg":   formatTime(avgLatency),
		"latency_max":   formatTime(m.MaxLatency),

		// Connection tracking
		"connection_time_total": formatTime(m.TotalConnectionTime),
		"connection_time_avg":   formatTime(m.AverageConnectionTime),
		"connection_last_time":  formatTime(m.LastConnectionTime),

		"connections_created": formatNumber(m.TotalCreatedConnections),
		"connections_closed":  formatNumber(m.TotalClosedConnections),
		"connections_active":  formatNumber(m.ActiveConnections),
		"connections_idle":    formatNumber(m.IdleConnections),

		// Retry metrics
		"retries_count":      formatNumber(m.TotalRetries),
		"retries_successful": formatNumber(m.SuccessfulRetries),

		// Error metrics
		"error_count":             formatNumber(m.ErrorCount),
		"error_last_time":         formatTime(m.LastErrorTime),
		"error_conn_at_last_time": formatError(m.LastConnError),
	}
}

type stringer interface {
	String() string
}

func formatTime[T stringer](t T) string {
	return t.String()
}

func formatNumber[T int64 | uint64](n T) string {
	return fmt.Sprintf("%d", n)
}

func formatError(e error) string {
	if e != nil {
		return e.Error()
	} else {
		return "No errors recorded"
	}
}
