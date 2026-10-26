package metrics

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var durationBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30,
}

type Application struct {
	authorizationDenials    *prometheus.CounterVec
	discoveryKeylessActive  prometheus.Gauge
	discoveryKeylessRefresh *prometheus.CounterVec
	readinessChecks         *prometheus.CounterVec
	readinessDuration       *prometheus.HistogramVec
	readinessStatus         *prometheus.GaugeVec
	shutdownPhases          *prometheus.CounterVec
	shutdownPhaseDuration   *prometheus.HistogramVec
	shutdowns               *prometheus.CounterVec
	shutdownDuration        prometheus.Histogram
}

func NewApplication(registerer prometheus.Registerer) (*Application, error) {
	app := &Application{
		authorizationDenials: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Name:      "authorization_denials_total",
			Help:      "Authorization denials by route and stable reason.",
		}, []string{"route", "reason"}),
		discoveryKeylessActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "comics",
			Subsystem: "discovery",
			Name:      "keyless_active",
			Help:      "Active comics without discovery title keys.",
		}),
		discoveryKeylessRefresh: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "discovery",
			Name:      "keyless_refresh_total",
			Help:      "Keyless catalog refresh outcomes.",
		}, []string{"outcome"}),
		readinessChecks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Name:      "readiness_checks_total",
			Help:      "Readiness probe results by dependency.",
		}, []string{"dependency", "outcome"}),
		readinessDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "comics",
			Name:      "readiness_duration_seconds",
			Help:      "Readiness probe latency by dependency.",
			Buckets:   durationBuckets,
		}, []string{"dependency", "outcome"}),
		readinessStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "comics",
			Name:      "readiness_status",
			Help:      "Readiness status by dependency (1=ready, 0=not ready).",
		}, []string{"dependency"}),
		shutdownPhases: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Name:      "shutdown_phases_total",
			Help:      "Shutdown phase results.",
		}, []string{"phase", "outcome"}),
		shutdownPhaseDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "comics",
			Name:      "shutdown_phase_duration_seconds",
			Help:      "Shutdown phase latency.",
			Buckets:   durationBuckets,
		}, []string{"phase", "outcome"}),
		shutdowns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Name:      "shutdown_total",
			Help:      "Application shutdown results.",
		}, []string{"outcome"}),
		shutdownDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "comics",
			Name:      "shutdown_duration_seconds",
			Help:      "Application shutdown latency.",
			Buckets:   durationBuckets,
		}),
	}
	if err := registerAll(registerer,
		app.authorizationDenials,
		app.discoveryKeylessActive,
		app.discoveryKeylessRefresh,
		app.readinessChecks,
		app.readinessDuration,
		app.readinessStatus,
		app.shutdownPhases,
		app.shutdownPhaseDuration,
		app.shutdowns,
		app.shutdownDuration,
	); err != nil {
		return nil, err
	}
	app.authorizationDenials.WithLabelValues("unknown", "unknown")
	app.discoveryKeylessRefresh.WithLabelValues("success")
	app.readinessChecks.WithLabelValues("aggregate", "ready")
	app.readinessStatus.WithLabelValues("aggregate").Set(0)
	app.shutdownPhases.WithLabelValues("unknown", "success")
	app.shutdowns.WithLabelValues("success")
	return app, nil
}

func registerAll(registerer prometheus.Registerer, collectors ...prometheus.Collector) error {
	for _, collector := range collectors {
		if err := registerer.Register(collector); err != nil {
			return err
		}
	}
	return nil
}

func (m *Application) ObserveAuthorizationDenial(route, reason string) {
	m.authorizationDenials.WithLabelValues(
		normalizeAuthRoute(route),
		normalizeAuthReason(reason),
	).Inc()
}

func (m *Application) SetDiscoveryKeylessActive(count int) {
	m.discoveryKeylessActive.Set(float64(count))
}

func (m *Application) ObserveDiscoveryKeylessRefresh(outcome string) {
	m.discoveryKeylessRefresh.WithLabelValues(normalizeKeylessOutcome(outcome)).Inc()
}

func (m *Application) ObserveReadiness(dependency, outcome string, elapsed time.Duration) {
	dep := normalizeReadinessDependency(dependency)
	out := normalizeReadinessOutcome(outcome)
	m.readinessChecks.WithLabelValues(dep, out).Inc()
	m.readinessDuration.WithLabelValues(dep, out).Observe(elapsed.Seconds())
}

func (m *Application) SetReadinessStatus(dependency string, ready bool) {
	value := 0.0
	if ready {
		value = 1
	}
	m.readinessStatus.WithLabelValues(normalizeReadinessDependency(dependency)).Set(value)
}

func (m *Application) ObserveShutdownPhase(phase string, elapsed time.Duration, err error) {
	outcome := shutdownOutcome(err)
	m.shutdownPhases.WithLabelValues(normalizeShutdownPhase(phase), outcome).Inc()
	m.shutdownPhaseDuration.WithLabelValues(normalizeShutdownPhase(phase), outcome).Observe(elapsed.Seconds())
}

func (m *Application) ObserveShutdown(elapsed time.Duration, err error) {
	outcome := shutdownOutcome(err)
	m.shutdowns.WithLabelValues(outcome).Inc()
	m.shutdownDuration.Observe(elapsed.Seconds())
}

func shutdownOutcome(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "error"
}

func normalizeAuthRoute(route string) string {
	route = strings.TrimSpace(strings.Split(route, "?")[0])
	switch route {
	case "/comics", "/scrape", "/scrape/status", "/scrape/history", "/protected/profile", "/metrics",
		"/comics/:id", "/comics/:id/cover-visibility", "/comics/:id/:merging_id":
		return route
	default:
		return "unknown"
	}
}

func normalizeAuthReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "missing_token", "malformed_token", "invalid_token", "inactive_user",
		"missing_role", "forbidden_role", "metrics_disabled", "invalid_metrics_token":
		return reason
	default:
		return "unknown"
	}
}

func normalizeKeylessOutcome(outcome string) string {
	switch strings.TrimSpace(outcome) {
	case "success", "error":
		return outcome
	default:
		return "error"
	}
}

func normalizeReadinessDependency(dependency string) string {
	switch strings.TrimSpace(dependency) {
	case "users", "catalog", "aggregate":
		return dependency
	default:
		return "aggregate"
	}
}

func normalizeReadinessOutcome(outcome string) string {
	switch strings.TrimSpace(outcome) {
	case "ready", "not_ready":
		return outcome
	default:
		return "not_ready"
	}
}

func normalizeShutdownPhase(phase string) string {
	switch strings.TrimSpace(phase) {
	case "readiness", "http", "scheduler", "scrape", "application", "catalog",
		"users", "tracing", "logging", "instance_lock":
		return phase
	default:
		return "unknown"
	}
}
