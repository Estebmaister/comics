package metrics

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

const (
	statusSuccess = "success"
	statusError   = "error"
)

// Collector defines the interface for collecting metrics
type Collector interface {
	RecordQuery(duration time.Duration, operation string, err error)
	RecordRetry(operation string, success bool)
	RecordConnection(err error)
	CloseConnection(err error)
	ReleaseConnection()
	RetrieveConnection()
	GetStats() map[string]string
}

var _ Collector = &Metrics{}

// Metrics represents the metrics for a database connection
type Metrics struct {
	mu sync.RWMutex

	// Prometheus metrics
	queryDuration  prometheus.Histogram
	queryTotal     *prometheus.CounterVec
	retryTotal     *prometheus.CounterVec
	activeRequests prometheus.Gauge
	errorRate      prometheus.Gauge

	// Internal metrics for quick access
	// Operation counters
	totalQueries      int64
	successfulQueries int64
	// Error tracking
	lastErrorTime time.Time
	lastConnError error
	// Retry tracking
	totalRetries      int64
	successfulRetries int64
	// Latency tracking
	maxLatency   time.Duration
	totalLatency time.Duration
	// Connection tracking
	activeConnections       int64
	totalCreatedConnections int64
	totalClosedConnections  int64
}

// NewMetrics creates a new instance of Metrics
func NewMetrics(serviceName, namespace string) *Metrics {
	// recover form panic creating duplicated metrics
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("panic recovered: %s", r)
		}
	}()
	m := &Metrics{}

	// Initialize Prometheus metrics
	m.queryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: serviceName,
		Subsystem: namespace,
		Name:      "query_duration_seconds",
		Help:      "Duration of database queries in mili seconds",
		Buckets:   prometheus.ExponentialBuckets(0.06, 1.3, 12), // From 60ms to ~1s
	})

	m.queryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: serviceName,
		Subsystem: namespace,
		Name:      "queries_total",
		Help:      "Total number of database queries",
	}, []string{"operation", "status"})

	m.retryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: serviceName,
		Subsystem: namespace,
		Name:      "retries_total",
		Help:      "Total number of query retries",
	}, []string{"operation", "status"})

	m.activeRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: serviceName,
		Subsystem: namespace,
		Name:      "active_requests",
		Help:      "Number of active database requests",
	})

	m.errorRate = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: serviceName,
		Subsystem: namespace,
		Name:      "error_rate",
		Help:      "Rate of database errors",
	})

	return m
}

// RecordQuery records a database query
func (m *Metrics) RecordQuery(duration time.Duration, operation string, err error) {
	log.Trace().Msgf("record db query: %s", operation)
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update Prometheus metrics
	m.queryDuration.Observe(duration.Seconds())
	status := statusSuccess
	if err != nil {
		status = statusError
	}
	m.queryTotal.WithLabelValues(operation, status).Inc()

	// Update internal metrics
	m.totalQueries++
	m.totalLatency += duration
	if duration > m.maxLatency {
		m.maxLatency = duration
	}

	if err != nil {
		m.lastConnError = err
		m.lastErrorTime = time.Now()
	} else {
		m.successfulQueries++
	}

	// Update error rate
	if m.totalQueries > 0 {
		m.errorRate.Set(float64(m.totalQueries-m.successfulQueries) / float64(m.totalQueries))
	}
}

// RecordRetry records a database retry
func (m *Metrics) RecordRetry(operation string, success bool) {
	log.Trace().Msgf("record db retry: %s", operation)
	m.mu.Lock()
	defer m.mu.Unlock()

	status := "success"
	if !success {
		status = "error"
	}
	m.retryTotal.WithLabelValues(operation, status).Inc()

	m.totalRetries++
	if success {
		m.successfulRetries++
	}
}

// RetrieveConnection records a database connection retrieval
func (m *Metrics) RetrieveConnection() {
	log.Trace().Msg("retriving db connection")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeRequests.Inc()
	m.activeConnections++
}

// ReleaseConnection records a database connection release
func (m *Metrics) ReleaseConnection() {
	log.Trace().Msg("releasing db connection")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeRequests.Dec()
	m.activeConnections--
}

// CloseConnection records a database connection closure
func (m *Metrics) CloseConnection(err error) {
	log.Trace().Msg("close db connection")
	if err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalClosedConnections++
}

// RecordConnection records a database connection creation
func (m *Metrics) RecordConnection(err error) {
	log.Trace().Msg("record db connection")
	if err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalCreatedConnections++
}

// GetStats returns a map of metrics
func (m *Metrics) GetStats() map[string]string {
	var result map[string]any
	_ = json.Unmarshal(m.GetSnapshot().ToJSON(), &result)

	// Convert the map values to strings and return it
	stringMap := make(map[string]string)
	for key, value := range result {
		stringMap[key] = fmt.Sprintf("%v", value)
	}
	return stringMap
}

// GetSnapshot returns a snapshot of the metrics
func (m *Metrics) GetSnapshot() *Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate average latency time
	avgLatency := time.Duration(0)
	if m.totalQueries > 0 {
		avgLatency = time.Duration(int64(m.totalLatency) / m.totalQueries)
	}

	// Calculate error rate
	errorRate := float64(0)
	if m.totalQueries > 0 {
		errorRate = float64(m.totalQueries-m.successfulQueries) / float64(m.totalQueries)
	}

	// Last connection error
	lastConnError := "No errors recorded"
	if m.lastConnError != nil {
		lastConnError = m.lastConnError.Error()
	}

	return &Snapshot{
		// Operation counters
		TotalQueries:      m.totalQueries,
		SuccessfulQueries: m.successfulQueries,

		// Latency tracking
		TotalLatency:   stringDuration(m.totalLatency),
		AverageLatency: stringDuration(avgLatency),
		MaxLatency:     stringDuration(m.maxLatency),

		// Connection tracking
		ActiveConnections:       m.activeConnections,
		TotalCreatedConnections: m.totalCreatedConnections,
		TotalClosedConnections:  m.totalClosedConnections,

		// Retry tracking
		TotalRetries:      m.totalRetries,
		SuccessfulRetries: m.successfulRetries,

		// Error tracking
		ErrorRate:     errorRate,
		LastErrorTime: m.lastErrorTime,
		LastConnError: lastConnError,
	}
}

type stringDuration time.Duration

// MarshalJSON converts time.Duration to a quoted string (e.g., "1s")
func (d stringDuration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", time.Duration(d).String())), nil
}

// ToJSON converts the Snapshot to JSON
func (s *Snapshot) ToJSON() []byte {
	b, _ := json.Marshal(s)
	return b
}

// Snapshot represents a snapshot of metrics data.
type Snapshot struct {
	// Operation counters
	TotalQueries      int64 `json:"queries_count"`
	SuccessfulQueries int64 `json:"queries_successful"`

	// Latency tracking
	TotalLatency   stringDuration `json:"latency_total"`
	AverageLatency stringDuration `json:"latency_avg"`
	MaxLatency     stringDuration `json:"latency_max"`

	// Connection tracking
	ActiveConnections       int64 `json:"connection_active_count"`
	TotalCreatedConnections int64 `json:"connection_total_created"`
	TotalClosedConnections  int64 `json:"connection_total_closed"`

	// Retry tracking
	TotalRetries      int64 `json:"retries_count"`
	SuccessfulRetries int64 `json:"retries_successful"`

	// Error tracking
	ErrorRate     float64   `json:"error_rate"`
	LastErrorTime time.Time `json:"last_error_time"`
	LastConnError string    `json:"last_connection_error"`
}
