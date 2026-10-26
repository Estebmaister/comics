package discovery

import (
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
	if err := registerer.Register(observer.resolutionTotal); err != nil {
		panic(err)
	}
	if err := registerer.Register(observer.conflictTotal); err != nil {
		panic(err)
	}
	if err := registerer.Register(observer.scrapeRegistrationTotal); err != nil {
		panic(err)
	}
	observer.resolutionTotal.WithLabelValues(string(OutcomeNotFound), string(MatchKind("none")))
	observer.conflictTotal.WithLabelValues("unknown")
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

func DefaultObserver() Observer {
	return NoopObserver{}
}
