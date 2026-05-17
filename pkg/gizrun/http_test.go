package gizrun

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizrun/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestHTTPHandlerTagsRequestAndRecordsMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics.Reset(reg)
	t.Cleanup(func() { metrics.Reset(prometheus.DefaultRegisterer) })
	registerGizrunMetrics()

	var labels LabelSet
	serveMux = http.NewServeMux()
	serveMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})
	serveMux.HandleFunc("/v1/test", func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		labels, ok = HTTPLabels(r.Context())
		if !ok {
			t.Fatal("HTTPLabels missing")
		}
		w.WriteHeader(http.StatusCreated)
	})
	httpHandler = newHandler()

	req := httptest.NewRequest(http.MethodPost, "/v1/test", nil)
	req.Host = "gizclaw.test"
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	for key, want := range map[string]string{
		HTTPMethod: http.MethodPost,
		HTTPPath:   "/v1/test",
		HTTPHost:   "gizclaw.test",
	} {
		if got, ok := labels.Value(key); !ok || got != want {
			t.Fatalf("HTTP label %q = (%q, %v), want (%q, true)", key, got, ok, want)
		}
	}
	assertGatheredMetric(t, reg, "gizclaw_gizrun_http_requests_total")
	assertGatheredMetric(t, reg, "gizclaw_gizrun_http_request_duration_seconds")
}

func TestHandlerServesMetricsAndPprofDisabledByDefault(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics.Reset(reg)
	t.Cleanup(func() { metrics.Reset(prometheus.DefaultRegisterer) })
	flags.enablePprof = false
	serveMux = http.NewServeMux()
	serveMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})
	httpHandler = newHandler()

	counter, err := metrics.RegisterCounter("http_test_total")
	if err != nil {
		t.Fatal(err)
	}
	counter.With(nil).Inc()

	metricsRec := httptest.NewRecorder()
	Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("/metrics status = %d, want 200: %s", metricsRec.Code, metricsRec.Body.String())
	}
	if !strings.Contains(metricsRec.Body.String(), "gizclaw_http_test_total") {
		t.Fatalf("/metrics output missing test counter:\n%s", metricsRec.Body.String())
	}

	pprofRec := httptest.NewRecorder()
	Handler().ServeHTTP(pprofRec, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if pprofRec.Code != http.StatusNotFound {
		t.Fatalf("/debug/pprof/ status = %d, want 404", pprofRec.Code)
	}
}

func TestHandlerServesPprofWhenEnabled(t *testing.T) {
	flags.enablePprof = true
	serveMux = http.NewServeMux()
	serveMux.Handle("/debug/pprof/", pprofHandler())
	serveMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})
	httpHandler = newHandler()
	t.Cleanup(func() {
		flags.enablePprof = false
		serveMux = http.NewServeMux()
		serveMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			metrics.Handler().ServeHTTP(w, r)
		})
		httpHandler = newHandler()
	})

	pprofRec := httptest.NewRecorder()
	Handler().ServeHTTP(pprofRec, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if pprofRec.Code != http.StatusOK {
		t.Fatalf("/debug/pprof/ status = %d, want 200", pprofRec.Code)
	}
}

func assertGatheredMetric(t *testing.T, gatherer prometheus.Gatherer, name string) {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == name {
			return
		}
	}
	t.Fatalf("metric %q was not gathered", name)
}
