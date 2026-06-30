package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsRegisterAndHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg)
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gizclaw",
		Name:      "requests_total",
		Help:      "Total requests.",
	}, []string{"method"})

	if err := m.Register(counter); err != nil {
		t.Fatal(err)
	}
	counter.With(prometheus.Labels{"method": "GET"}).Inc()

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, `gizclaw_requests_total{method="GET"} 1`) {
		t.Fatalf("metrics output missing counter:\n%s", got)
	}
}

func TestMetricsRegisterIgnoresAlreadyRegistered(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg)
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_registered_gauge", Help: "test gauge"})

	if err := m.Register(gauge); err != nil {
		t.Fatal(err)
	}
	if err := m.Register(gauge); err != nil {
		t.Fatal(err)
	}
}

func TestMetricsRegisterRejectsConflictingCollector(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg)
	first := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_conflict_metric", Help: "first"})
	second := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_conflict_metric", Help: "second"})

	if err := m.Register(first); err != nil {
		t.Fatal(err)
	}
	if err := m.Register(second); err == nil {
		t.Fatal("Register conflicting collector err = nil, want error")
	}
}

func TestMetricsNoopsAndUnavailableGatherer(t *testing.T) {
	m := New(registererOnly{Registerer: prometheus.NewRegistry()})
	if err := m.Register(nil); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

type registererOnly struct {
	prometheus.Registerer
}
