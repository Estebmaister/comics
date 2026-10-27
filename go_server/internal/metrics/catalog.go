package metrics

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	catalogMu     sync.RWMutex
	catalogActive *Catalog
)

// Catalog exposes stable-label catalog, pool, and scrape tail signals.
type Catalog struct {
	httpDuration    *prometheus.HistogramVec
	repoDuration    *prometheus.HistogramVec
	searchPath      *prometheus.CounterVec
	poolWait        *prometheus.HistogramVec
	poolInUse       prometheus.Gauge
	scrapePhase     *prometheus.HistogramVec
	scrapeNoop      *prometheus.CounterVec
	scrapeRunning   prometheus.Gauge
	schedulerDelay  *prometheus.HistogramVec
}

func NewCatalog(registerer prometheus.Registerer) (*Catalog, error) {
	c := &Catalog{
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "comics",
			Subsystem: "catalog",
			Name:      "http_duration_seconds",
			Help:      "Catalog HTTP handler latency by normalized route and outcome.",
			Buckets:   durationBuckets,
		}, []string{"route", "outcome"}),
		repoDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "comics",
			Subsystem: "catalog",
			Name:      "repo_duration_seconds",
			Help:      "Catalog repository list/search latency by phase.",
			Buckets:   durationBuckets,
		}, []string{"phase", "search_path"}),
		searchPath: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "catalog",
			Name:      "search_path_total",
			Help:      "Title search executions by indexed path.",
		}, []string{"path"}),
		poolWait: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "comics",
			Subsystem: "catalog",
			Name:      "pool_wait_seconds",
			Help:      "SQLite reader pool wait before query execution.",
			Buckets:   durationBuckets,
		}, []string{"role"}),
		poolInUse: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "comics",
			Subsystem: "catalog",
			Name:      "pool_connections_in_use",
			Help:      "Open catalog DB connections (writer + readers).",
		}),
		scrapePhase: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "comics",
			Subsystem: "scrape",
			Name:      "phase_duration_seconds",
			Help:      "Scrape phase latency (fetch, register, run).",
			Buckets:   durationBuckets,
		}, []string{"phase", "outcome"}),
		scrapeNoop: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "scrape",
			Name:      "noop_writes_skipped_total",
			Help:      "Scrape registrations skipped because comic row was unchanged.",
		}, []string{"outcome"}),
		scrapeRunning: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "comics",
			Subsystem: "scrape",
			Name:      "running",
			Help:      "Whether a scrape run is active (1) or not (0).",
		}),
		schedulerDelay: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "comics",
			Subsystem: "scrape",
			Name:      "scheduler_startup_delay_seconds",
			Help:      "Delay before the first automatic scrape after server start.",
			Buckets:   durationBuckets,
		}, []string{"outcome"}),
	}
	if err := registerAll(registerer,
		c.httpDuration,
		c.repoDuration,
		c.searchPath,
		c.poolWait,
		c.poolInUse,
		c.scrapePhase,
		c.scrapeNoop,
		c.scrapeRunning,
		c.schedulerDelay,
	); err != nil {
		return nil, err
	}
	c.searchPath.WithLabelValues("none")
	c.scrapeNoop.WithLabelValues("skipped")
	return c, nil
}

// SetCatalog installs the process-wide catalog metrics handle.
func SetCatalog(c *Catalog) {
	catalogMu.Lock()
	catalogActive = c
	catalogMu.Unlock()
}

func activeCatalog() *Catalog {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	return catalogActive
}

func (c *Catalog) ObserveHTTP(route string, outcome string, elapsed time.Duration) {
	c.httpDuration.WithLabelValues(normalizeCatalogRoute(route), normalizeHTTPOutcome(outcome)).
		Observe(elapsed.Seconds())
}

func (c *Catalog) ObserveRepo(phase, searchPath string, elapsed time.Duration) {
	c.repoDuration.WithLabelValues(normalizeRepoPhase(phase), normalizeSearchPath(searchPath)).
		Observe(elapsed.Seconds())
}

func (c *Catalog) IncSearchPath(path string) {
	c.searchPath.WithLabelValues(normalizeSearchPath(path)).Inc()
}

func (c *Catalog) ObservePoolWait(role string, elapsed time.Duration) {
	c.poolWait.WithLabelValues(normalizePoolRole(role)).Observe(elapsed.Seconds())
}

func (c *Catalog) SetPoolInUse(count int) {
	c.poolInUse.Set(float64(count))
}

func (c *Catalog) ObserveScrapePhase(phase string, elapsed time.Duration, err error) {
	outcome := shutdownOutcome(err)
	c.scrapePhase.WithLabelValues(normalizeScrapePhase(phase), outcome).Observe(elapsed.Seconds())
}

func (c *Catalog) IncScrapeNoopSkipped() {
	c.scrapeNoop.WithLabelValues("skipped").Inc()
}

func (c *Catalog) SetScrapeRunning(running bool) {
	if running {
		c.scrapeRunning.Set(1)
		return
	}
	c.scrapeRunning.Set(0)
}

func (c *Catalog) ObserveSchedulerStartupDelay(elapsed time.Duration) {
	c.schedulerDelay.WithLabelValues("applied").Observe(elapsed.Seconds())
}

// ObserveCatalogHTTP is nil-safe.
func ObserveCatalogHTTP(route, outcome string, elapsed time.Duration) {
	if c := activeCatalog(); c != nil {
		c.ObserveHTTP(route, outcome, elapsed)
	}
}

// ObserveCatalogRepo is nil-safe.
func ObserveCatalogRepo(phase, searchPath string, elapsed time.Duration) {
	if c := activeCatalog(); c != nil {
		c.ObserveRepo(phase, searchPath, elapsed)
	}
}

// IncCatalogSearchPath is nil-safe.
func IncCatalogSearchPath(path string) {
	if c := activeCatalog(); c != nil {
		c.IncSearchPath(path)
	}
}

// IncScrapeNoopSkipped is nil-safe.
func IncScrapeNoopSkipped() {
	if c := activeCatalog(); c != nil {
		c.IncScrapeNoopSkipped()
	}
}

// SetScrapeRunningMetric is nil-safe.
func SetScrapeRunningMetric(running bool) {
	if c := activeCatalog(); c != nil {
		c.SetScrapeRunning(running)
	}
}

// ObserveScrapePhaseMetric is nil-safe.
func ObserveScrapePhaseMetric(phase string, elapsed time.Duration, err error) {
	if c := activeCatalog(); c != nil {
		c.ObserveScrapePhase(phase, elapsed, err)
	}
}

// ObserveSchedulerStartupDelay is nil-safe.
func ObserveSchedulerStartupDelay(elapsed time.Duration) {
	if c := activeCatalog(); c != nil {
		c.ObserveSchedulerStartupDelay(elapsed)
	}
}

func normalizeCatalogRoute(route string) string {
	route = strings.TrimSpace(strings.Split(route, "?")[0])
	switch route {
	case "/comics", "/comics/search/:title", "/comics/:id", "/health/db":
		return route
	default:
		return "other"
	}
}

func normalizeHTTPOutcome(outcome string) string {
	switch strings.TrimSpace(outcome) {
	case "success", "error", "client_error":
		return outcome
	default:
		return "error"
	}
}

func normalizeRepoPhase(phase string) string {
	switch strings.TrimSpace(phase) {
	case "count", "page", "list":
		return phase
	default:
		return "list"
	}
}

func normalizeSearchPath(path string) string {
	switch strings.TrimSpace(path) {
	case "trigram", "scan", "none":
		return path
	default:
		return "scan"
	}
}

func normalizePoolRole(role string) string {
	switch strings.TrimSpace(role) {
	case "writer", "reader":
		return role
	default:
		return "reader"
	}
}

func normalizeScrapePhase(phase string) string {
	switch strings.TrimSpace(phase) {
	case "fetch", "register", "run":
		return phase
	default:
		return "run"
	}
}
