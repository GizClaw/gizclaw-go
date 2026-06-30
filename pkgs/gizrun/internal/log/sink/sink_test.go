package sink

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/labelset"
	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestHandlerBatchesAndFlushes(t *testing.T) {
	sink := &recordingSink{}
	handler := NewHandler(sink, &Options{
		Level:         slog.LevelDebug,
		QueueSize:     8,
		BatchSize:     4,
		FlushInterval: time.Hour,
	})
	defer handler.Close(context.Background())

	logger := slog.New(handler)
	logger.Debug("debug")
	logger.Info("info", "name", "gizclaw")

	if err := handler.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}

	records := sink.snapshot()
	if len(records) != 2 {
		t.Fatalf("record count = %d, want 2", len(records))
	}
	if records[0].Message != "debug" || records[1].Message != "info" {
		t.Fatalf("messages = %q, %q", records[0].Message, records[1].Message)
	}
	if got := attrValue(records[1], "name"); got != "gizclaw" {
		t.Fatalf("name attr = %q, want gizclaw", got)
	}
	if stats := handler.Stats(); stats.Enqueued != 2 || stats.Written != 2 || stats.Dropped != 0 {
		t.Fatalf("stats = %+v, want enqueued=2 written=2 dropped=0", stats)
	}
}

func TestHandlerDropsWhenQueueIsFull(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	var drops []Drop
	var dropsMu sync.Mutex

	handler := NewHandler(SinkFunc(func(ctx context.Context, records []slog.Record) error {
		once.Do(func() { close(started) })
		<-release
		return nil
	}), &Options{
		QueueSize:     1,
		BatchSize:     1,
		FlushInterval: time.Hour,
		OnDrop: func(drop Drop) {
			dropsMu.Lock()
			defer dropsMu.Unlock()
			drops = append(drops, drop)
		},
	})

	if err := handler.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "first", 0)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("sink did not start")
	}
	if err := handler.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "queued", 0)); err != nil {
		t.Fatal(err)
	}
	if err := handler.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "dropped", 0)); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("drop err = %v, want %v", err, ErrQueueFull)
	}

	stats := handler.Stats()
	if stats.Enqueued != 2 || stats.Dropped != 1 {
		t.Fatalf("stats = %+v, want enqueued=2 dropped=1", stats)
	}
	dropsMu.Lock()
	gotDrops := append([]Drop(nil), drops...)
	dropsMu.Unlock()
	if len(gotDrops) != 1 || gotDrops[0].Reason != "queue_full" || gotDrops[0].Message != "dropped" {
		t.Fatalf("drops = %+v", gotDrops)
	}

	close(release)
	if err := handler.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestHandlerWithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(NewHandlerSink(slog.NewTextHandler(&buf, nil)), &Options{
		QueueSize:     8,
		BatchSize:     8,
		FlushInterval: time.Hour,
	})
	defer handler.Close(context.Background())

	grouped := handler.WithAttrs([]slog.Attr{slog.String("service", "openai")}).WithGroup("request")
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "served", 0)
	record.AddAttrs(slog.String("id", "req-1"))
	if err := grouped.Handle(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	if err := handler.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	for _, want := range []string{"service=openai", "request.id=req-1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q: %s", want, out)
		}
	}
}

func TestHandlerFlushReturnsWriteError(t *testing.T) {
	wantErr := errors.New("write failed")
	var gotErr error
	handler := NewHandler(SinkFunc(func(ctx context.Context, records []slog.Record) error {
		return wantErr
	}), &Options{
		QueueSize:     4,
		BatchSize:     4,
		FlushInterval: time.Hour,
		OnError: func(err error) {
			gotErr = err
		},
	})
	defer handler.Close(context.Background())

	if err := handler.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "fail", 0)); err != nil {
		t.Fatal(err)
	}
	if err := handler.Flush(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("flush err = %v, want %v", err, wantErr)
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("OnError err = %v, want %v", gotErr, wantErr)
	}
	if stats := handler.Stats(); stats.Failed != 1 || stats.Written != 0 {
		t.Fatalf("stats = %+v, want failed=1 written=0", stats)
	}
}

func TestStartStopGlobalSinkAndMetrics(t *testing.T) {
	defer Stop(context.Background())
	reg := prometheus.NewRegistry()
	metrics.Reset(reg)
	t.Cleanup(func() { metrics.Reset(prometheus.DefaultRegisterer) })
	records := &recordingSink{}
	metricCtx := labelset.Tag(context.Background(), "logsink", testMetricLabelScope, testMetricScopeLogSink)
	metricNamespace, ok := labelset.FromContext(metricCtx, "logsink")
	if !ok {
		t.Fatal("logsink namespace missing")
	}
	if err := Start(records, &Options{
		Level:              slog.LevelDebug,
		QueueSize:          8,
		BatchSize:          8,
		FlushInterval:      time.Hour,
		MetricNamespace:    metricNamespace,
		DisableDefaultPipe: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := Start(records, &Options{DisableDefaultPipe: true}); !errors.Is(err, ErrStarted) {
		t.Fatalf("second start err = %v, want %v", err, ErrStarted)
	}

	logger := slog.New(Handler())
	logger.Debug("debug")
	logger.Info("info")
	if err := Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if stats := CurrentStats(); !stats.Running || stats.Enqueued != 2 || stats.Written != 2 || stats.QueueCapacity != 8 {
		t.Fatalf("stats = %+v", stats)
	}

	if got := metricValue(t, reg, "gizclaw_log_sink_written_total", testMetricScopeLogSink); got != 2 {
		t.Fatalf("written metric = %v, want 2", got)
	}
	if err := Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if CurrentStats().Running {
		t.Fatal("global sink should not be running after Stop")
	}
}

func metricValue(t *testing.T, gatherer prometheus.Gatherer, name string, scope string) float64 {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if metricLabel(metric, testMetricLabelScope) != scope {
				continue
			}
			if metric.Counter != nil {
				return metric.Counter.GetValue()
			}
			if metric.Gauge != nil {
				return metric.Gauge.GetValue()
			}
		}
	}
	t.Fatalf("metric %q not found", name)
	return 0
}

const (
	testMetricLabelScope   = "scope"
	testMetricScopeLogSink = "logsink"
)

func metricLabel(metric *dto.Metric, name string) string {
	for _, label := range metric.GetLabel() {
		if label.GetName() == name {
			return label.GetValue()
		}
	}
	return ""
}

type recordingSink struct {
	mu      sync.Mutex
	records []slog.Record
}

func (s *recordingSink) Write(ctx context.Context, records []slog.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range records {
		s.records = append(s.records, record.Clone())
	}
	return nil
}

func (s *recordingSink) snapshot() []slog.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]slog.Record, len(s.records))
	for i, record := range s.records {
		out[i] = record.Clone()
	}
	return out
}

func attrValue(record slog.Record, key string) string {
	var value string
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == key {
			value = attr.Value.String()
			return false
		}
		return true
	})
	return value
}
