package discovery

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type ResolutionOutcome string

const (
	OutcomeMatched   ResolutionOutcome = "matched"
	OutcomeNotFound  ResolutionOutcome = "not_found"
	OutcomeAmbiguous ResolutionOutcome = "ambiguous"
	OutcomeError     ResolutionOutcome = "error"
)

type Observer interface {
	ObserveResolution(outcome ResolutionOutcome, kind MatchKind)
	ObserveConflict(operation string)
	ObserveScrapeRegistration(result string)
}

type NoopObserver struct{}

func (NoopObserver) ObserveResolution(ResolutionOutcome, MatchKind)   {}
func (NoopObserver) ObserveConflict(string)                           {}
func (NoopObserver) ObserveScrapeRegistration(string)                 {}

type PrometheusObserver struct {
	resolutionTotal      *prometheus.CounterVec
	conflictTotal        *prometheus.CounterVec
	scrapeRegistrationTotal *prometheus.CounterVec
}

func NewPrometheusObserver(registerer prometheus.Registerer) *PrometheusObserver {
	observer := &PrometheusObserver{
		resolutionTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "discovery",
			Name:      "resolution_total",
			Help:      "Discovery resolution outcomes.",
		}, []string{"outcome", "match_kind"}),
		conflictTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "discovery",
			Name:      "key_conflicts_total",
			Help:      "Typed discovery key conflicts by catalog operation.",
		}, []string{"operation"}),
		scrapeRegistrationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "discovery",
			Name:      "scrape_registration_total",
			Help:      "Scrape registration outcomes (ambiguous matches are counted, not error-logged).",
		}, []string{"result"}),
	}
	registerer.MustRegister(
		observer.resolutionTotal,
		observer.conflictTotal,
		observer.scrapeRegistrationTotal,
	)
	return observer
}

func (o *PrometheusObserver) ObserveResolution(outcome ResolutionOutcome, kind MatchKind) {
	if kind == MatchNone {
		kind = MatchKind("none")
	}
	o.resolutionTotal.WithLabelValues(string(outcome), string(kind)).Inc()
}

func (o *PrometheusObserver) ObserveConflict(operation string) {
	o.conflictTotal.WithLabelValues(operation).Inc()
}

func (o *PrometheusObserver) ObserveScrapeRegistration(result string) {
	o.scrapeRegistrationTotal.WithLabelValues(result).Inc()
}

var (
	defaultObserverOnce sync.Once
	defaultObserver     Observer
)

func DefaultObserver() Observer {
	defaultObserverOnce.Do(func() {
		defaultObserver = NewPrometheusObserver(prometheus.DefaultRegisterer)
	})
	return defaultObserver
}
