package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	mu sync.RWMutex

	// Prometheus metrics
	queryDuration   prometheus.Histogram
	queryTotal      *prometheus.CounterVec
	retryTotal     *prometheus.CounterVec
	activeRequests prometheus.Gauge
	errorRate      prometheus.Gauge

	// Internal metrics for quick access
	totalQueries      int64
	successfulQueries int64
	failedQueries     int64
	totalRetries      int64
	successfulRetries int64
	maxLatency        time.Duration
	totalLatency      time.Duration
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{}

	// Initialize Prometheus metrics
	m.queryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "query_duration_seconds",
		Help:      "Duration of database queries in seconds",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
	})

	m.queryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "queries_total",
		Help:      "Total number of database queries",
	}, []string{"operation", "status"})

	m.retryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "retries_total",
		Help:      "Total number of query retries",
	}, []string{"operation", "status"})

	m.activeRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "active_requests",
		Help:      "Number of active database requests",
	})

	m.errorRate = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "error_rate",
		Help:      "Rate of database errors",
	})

	return m
}

func (m *Metrics) RecordQuery(duration float64, operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update Prometheus metrics
	m.queryDuration.Observe(duration)
	status := "success"
	if err != nil {
		status = "error"
	}
	m.queryTotal.WithLabelValues(operation, status).Inc()

	// Update internal metrics
	m.totalQueries++
	m.totalLatency += time.Duration(duration * float64(time.Second))
	if time.Duration(duration*float64(time.Second)) > m.maxLatency {
		m.maxLatency = time.Duration(duration * float64(time.Second))
	}

	if err == nil {
		m.successfulQueries++
	} else {
		m.failedQueries++
	}

	// Update error rate
	if m.totalQueries > 0 {
		m.errorRate.Set(float64(m.failedQueries) / float64(m.totalQueries))
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

func (m *Metrics) IncActiveRequests() {
	m.activeRequests.Inc()
}

func (m *Metrics) DecActiveRequests() {
	m.activeRequests.Dec()
}

type MetricsSnapshot struct {
	TotalQueries      int64
	SuccessfulQueries int64
	FailedQueries     int64
	TotalRetries      int64
	SuccessfulRetries int64
	ErrorRate         float64
	AverageLatency    time.Duration
	MaxLatency        time.Duration
}

func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := time.Duration(0)
	if m.totalQueries > 0 {
		avgLatency = time.Duration(int64(m.totalLatency) / m.totalQueries)
	}

	errorRate := float64(0)
	if m.totalQueries > 0 {
		errorRate = float64(m.failedQueries) / float64(m.totalQueries)
	}

	return &MetricsSnapshot{
		TotalQueries:      m.totalQueries,
		SuccessfulQueries: m.successfulQueries,
		FailedQueries:     m.failedQueries,
		TotalRetries:      m.totalRetries,
		SuccessfulRetries: m.successfulRetries,
		ErrorRate:         errorRate,
		AverageLatency:    avgLatency,
		MaxLatency:        m.maxLatency,
	}
}
