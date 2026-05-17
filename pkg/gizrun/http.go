package gizrun

import (
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizrun/internal/metrics"
)

const (
	gizrunHTTPRequestsTotal          = "gizrun_http_requests_total"
	gizrunHTTPRequestDurationSeconds = "gizrun_http_request_duration_seconds"
)

var (
	serveMux    *http.ServeMux
	httpHandler http.Handler
)

func Handler() http.Handler {
	return httpHandler
}

func newHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := TagHTTP(r.Context(),
			HTTPMethod, r.Method,
			HTTPPath, r.URL.Path,
			HTTPHost, r.Host,
		)
		recorder := &statusRecorder{ResponseWriter: w}
		serveMux.ServeHTTP(recorder, r.WithContext(ctx))
		statusCode := recorder.statusCode()
		metricCtx := TagHTTP(ctx, HTTPStatusCode, strconv.Itoa(statusCode))
		if labels, ok := HTTPLabels(metricCtx); ok {
			counter := metrics.Counter(gizrunHTTPRequestsTotal)
			histogram := metrics.Histogram(gizrunHTTPRequestDurationSeconds)
			if counter != nil && histogram != nil {
				promLabels := labels.PrometheusLabels()
				counter.With(promLabels).Inc()
				histogram.With(promLabels).Observe(time.Since(start).Seconds())
			}
		}
	})
}

func pprofHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func registerGizrunMetrics() {
	if _, err := metrics.RegisterCounter(gizrunHTTPRequestsTotal,
		HTTPMethod,
		HTTPPath,
		HTTPHost,
		HTTPStatusCode,
	); err != nil {
		panic(err)
	}
	if _, err := metrics.RegisterHistogram(gizrunHTTPRequestDurationSeconds,
		HTTPMethod,
		HTTPPath,
		HTTPHost,
		HTTPStatusCode,
	); err != nil {
		panic(err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func (w *statusRecorder) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
