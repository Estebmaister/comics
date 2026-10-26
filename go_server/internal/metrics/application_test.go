package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestApplicationNormalizesAuthorizationDenials(t *testing.T) {
	registry := prometheus.NewRegistry()
	app, err := NewApplication(registry)
	if err != nil {
		t.Fatal(err)
	}
	app.ObserveAuthorizationDenial("/comics/123?token=secret", "token-secret")

	if got := testutil.ToFloat64(
		app.authorizationDenials.WithLabelValues("unknown", "unknown"),
	); got != 1 {
		t.Fatalf("unknown authorization denial = %v, want 1", got)
	}
	assertNoForbiddenLabelNames(t, registry)
}

func TestApplicationDuplicateRegistrationFails(t *testing.T) {
	registry := prometheus.NewRegistry()
	if _, err := NewApplication(registry); err != nil {
		t.Fatal(err)
	}
	if _, err := NewApplication(registry); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestApplicationRecordsAllFamilies(t *testing.T) {
	registry := prometheus.NewRegistry()
	app, err := NewApplication(registry)
	if err != nil {
		t.Fatal(err)
	}
	app.SetDiscoveryKeylessActive(346)
	app.ObserveDiscoveryKeylessRefresh("success")
	app.ObserveReadiness("catalog", "ready", 0)
	app.SetReadinessStatus("catalog", true)
	app.ObserveShutdownPhase("http", 0, nil)
	app.ObserveShutdown(0, nil)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) < 8 {
		t.Fatalf("expected collectors, got %d metric families", len(metrics))
	}
	assertNoForbiddenLabelNames(t, registry)
}

var forbiddenLabelSubstrings = []string{
	"request_id", "run_id", "trace_id", "span_id", "user_id",
	"comic_id", "title", "url", "token", "error",
}

func assertNoForbiddenLabelNames(t *testing.T, registry *prometheus.Registry) {
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range metrics {
		for _, metric := range family.Metric {
			for _, pair := range metric.Label {
				name := strings.ToLower(pair.GetName())
				for _, forbidden := range forbiddenLabelSubstrings {
					if strings.Contains(name, forbidden) {
						t.Fatalf("metric %s has forbidden label name %q", family.GetName(), pair.GetName())
					}
				}
			}
		}
	}
}
