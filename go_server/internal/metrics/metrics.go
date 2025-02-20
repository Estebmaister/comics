package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	statusSuccess = "success"
	statusError   = "error"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	RecordQuery(duration time.Duration, operation string, err error)
	RecordRetry(operation string, success bool)
	RecordConnection(connectionTime time.Duration, err error)
	CloseConnection(connectionTime time.Duration, err error)
	ReleaseConnection()
	RetrieveConnection()
	GetStats() map[string]string
}

var _ MetricsCollector = &Metrics{}

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
	totalQueries      uint64
	successfulQueries uint64
	// Error tracking
	lastErrorTime time.Time
	lastConnError error
	// Retry tracking
	totalRetries      uint64
	successfulRetries uint64
	// Latency tracking
	maxLatency   time.Duration
	totalLatency time.Duration
	// Connection tracking
	activeConnections       uint64
	idleConnections         uint64
	totalCreatedConnections uint64
	totalClosedConnections  uint64
}

// NewMetrics creates a new instance of Metrics
func NewMetrics(namespace, subsystem string) *Metrics {
	m := &Metrics{}

	// Initialize Prometheus metrics
	m.queryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "query_duration_seconds",
		Help:      "Duration of database queries in seconds",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
	})

	m.queryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "queries_total",
		Help:      "Total number of database queries",
	}, []string{"operation", "status"})

	m.retryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "retries_total",
		Help:      "Total number of query retries",
	}, []string{"operation", "status"})

	m.activeRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "active_requests",
		Help:      "Number of active database requests",
	})

	m.errorRate = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "error_rate",
		Help:      "Rate of database errors",
	})

	return m
}

// RecordQuery records a database query
func (m *Metrics) RecordQuery(duration time.Duration, operation string, err error) {
	log.Print("recording query")
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

func (m *Metrics) RecordRetry(operation string, success bool) {
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

func (m *Metrics) RetrieveConnection() {
	log.Print("retrieving connection")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeRequests.Inc()
	m.activeConnections++
	if m.idleConnections > 0 {
		m.idleConnections--
	}
}

func (m *Metrics) ReleaseConnection() {
	log.Print("releasing connection")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeRequests.Dec()
	m.activeConnections--
	m.idleConnections++
}

func (m *Metrics) CloseConnection(duration time.Duration, err error) {
	log.Print("closing connection")
	if err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalClosedConnections++
	if m.activeConnections > 0 {
		m.activeConnections--
	} else {
		m.idleConnections--
	}
}

func (m *Metrics) RecordConnection(duration time.Duration, err error) {
	log.Print("recording connection")
	if err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalCreatedConnections++
}

func (m *Metrics) GetStats() map[string]string {
	// Unmarshal the JSON to a map[string]interface{}
	var result map[string]interface{}
	_ = json.Unmarshal(m.GetSnapshot().ToJSON(), &result)

	// Convert the map values to strings and return it
	stringMap := make(map[string]string)
	for key, value := range result {
		stringMap[key] = fmt.Sprintf("%v", value)
	}
	return stringMap
}

func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate average latency time
	avgLatency := time.Duration(0)
	if m.totalQueries > 0 {
		avgLatency = time.Duration(uint64(m.totalLatency) / m.totalQueries)
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

	return &MetricsSnapshot{
		// Operation counters
		TotalQueries:      m.totalQueries,
		SuccessfulQueries: m.successfulQueries,

		// Latency tracking
		TotalLatency:   stringDuration(m.totalLatency),
		AverageLatency: stringDuration(avgLatency),
		MaxLatency:     stringDuration(m.maxLatency),

		// Connection tracking
		ActiveConnections:       m.activeConnections,
		IdleConnections:         m.idleConnections,
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

func (s *MetricsSnapshot) ToJSON() []byte {
	b, _ := json.Marshal(s)
	return b
}

// MetricsSnapshot represents a snapshot of metrics data.
type MetricsSnapshot struct {
	// Operation counters
	TotalQueries      uint64 `json:"queries_count"`
	SuccessfulQueries uint64 `json:"queries_successful"`

	// Latency tracking
	TotalLatency   stringDuration `json:"latency_total"`
	AverageLatency stringDuration `json:"latency_avg"`
	MaxLatency     stringDuration `json:"latency_max"`

	// Connection tracking
	ActiveConnections       uint64 `json:"connection_active_count"`
	IdleConnections         uint64 `json:"connection_idle_count"`
	TotalCreatedConnections uint64 `json:"connection_total_created"`
	TotalClosedConnections  uint64 `json:"connection_total_closed"`

	// Retry tracking
	TotalRetries      uint64 `json:"retries_count"`
	SuccessfulRetries uint64 `json:"retries_successful"`

	// Error tracking
	ErrorRate     float64   `json:"error_rate"`
	LastErrorTime time.Time `json:"last_error_time"`
	LastConnError string    `json:"last_connection_error"`
}
