package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestDefaultMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	Reset(reg)
	t.Cleanup(func() { Reset(prometheus.DefaultRegisterer) })

	counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "test_default_counter_total", Help: "test counter"})
	if err := Register(counter); err != nil {
		t.Fatal(err)
	}
	counter.Inc()

	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == "test_default_counter_total" {
			return
		}
	}
	t.Fatal("default counter was not gathered")
}

func TestDefaultHandler(t *testing.T) {
	if Handler() == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestDefaultRegisterMetricHelpers(t *testing.T) {
	reg := prometheus.NewRegistry()
	Reset(reg)
	t.Cleanup(func() { Reset(prometheus.DefaultRegisterer) })

	counter, err := RegisterCounter("default_helper_counter_total", "status")
	if err != nil {
		t.Fatal(err)
	}
	gauge, err := RegisterGauge("default_helper_gauge", "status")
	if err != nil {
		t.Fatal(err)
	}
	histogram, err := RegisterHistogram("default_helper_duration_seconds", "status")
	if err != nil {
		t.Fatal(err)
	}
	summary, err := RegisterSummary("default_helper_summary_seconds", "status")
	if err != nil {
		t.Fatal(err)
	}
	if Counter("default_helper_counter_total") != counter {
		t.Fatal("Counter did not return registered counter")
	}
	if Gauge("default_helper_gauge") != gauge {
		t.Fatal("Gauge did not return registered gauge")
	}
	if Histogram("default_helper_duration_seconds") != histogram {
		t.Fatal("Histogram did not return registered histogram")
	}
	if Summary("default_helper_summary_seconds") != summary {
		t.Fatal("Summary did not return registered summary")
	}

	labels := prometheus.Labels{"status": "ok"}
	counter.With(labels).Inc()
	gauge.With(labels).Set(1)
	histogram.With(labels).Observe(0.1)
	summary.With(labels).Observe(0.2)

	for _, want := range []string{
		"gizclaw_default_helper_counter_total",
		"gizclaw_default_helper_gauge",
		"gizclaw_default_helper_duration_seconds",
		"gizclaw_default_helper_summary_seconds",
	} {
		if !hasMetric(t, reg, want) {
			t.Fatalf("metric %q was not gathered", want)
		}
	}
}

func TestResetClearsRegisteredMetricHelpers(t *testing.T) {
	Reset(prometheus.NewRegistry())
	if _, err := RegisterCounter("reset_helper_counter_total"); err != nil {
		t.Fatal(err)
	}
	if Counter("reset_helper_counter_total") == nil {
		t.Fatal("Counter missing before Reset")
	}

	Reset(prometheus.NewRegistry())
	t.Cleanup(func() { Reset(prometheus.DefaultRegisterer) })
	if Counter("reset_helper_counter_total") != nil {
		t.Fatal("Counter returned stale collector after Reset")
	}
}

func TestRegisterMetricHelperDuplicateReturnsExisting(t *testing.T) {
	reg := prometheus.NewRegistry()
	Reset(reg)
	t.Cleanup(func() { Reset(prometheus.DefaultRegisterer) })

	first, err := RegisterCounter("helper_duplicate_total", "method")
	if err != nil {
		t.Fatal(err)
	}
	second, err := RegisterCounter("helper_duplicate_total", "method")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("duplicate RegisterCounter returned a different collector")
	}
}

func TestRegisterMetricHelperRejectsWrongKind(t *testing.T) {
	reg := prometheus.NewRegistry()
	Reset(reg)
	t.Cleanup(func() { Reset(prometheus.DefaultRegisterer) })

	if _, err := RegisterCounter("helper_wrong_kind_total"); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterGauge("helper_wrong_kind_total"); err == nil {
		t.Fatal("RegisterGauge after RegisterCounter err = nil, want error")
	}
}

func TestRegisterMetricHelperRejectsEmptyName(t *testing.T) {
	checks := map[string]error{
		"RegisterCounter":   emptyCounterName(),
		"RegisterGauge":     emptyGaugeName(),
		"RegisterHistogram": emptyHistogramName(),
		"RegisterSummary":   emptySummaryName(),
	}
	for name, err := range checks {
		if err == nil {
			t.Fatalf("%s empty name err = nil, want error", name)
		}
	}
}

func emptyCounterName() error {
	_, err := RegisterCounter("")
	return err
}

func emptyGaugeName() error {
	_, err := RegisterGauge("")
	return err
}

func emptyHistogramName() error {
	_, err := RegisterHistogram("")
	return err
}

func emptySummaryName() error {
	_, err := RegisterSummary("")
	return err
}

func hasMetric(t *testing.T, gatherer prometheus.Gatherer, name string) bool {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == name {
			return true
		}
	}
	return false
}
