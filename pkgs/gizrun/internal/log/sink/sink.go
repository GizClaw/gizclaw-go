package sink

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/labelset"
	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrClosed     = errors.New("log sink closed")
	ErrQueueFull  = errors.New("log sink queue full")
	ErrStarted    = errors.New("log sink already started")
	ErrNotStarted = errors.New("log sink not started")
)

type Sink interface {
	Write(context.Context, []slog.Record) error
}

type SinkFunc func(context.Context, []slog.Record) error

func (f SinkFunc) Write(ctx context.Context, records []slog.Record) error {
	if f == nil {
		return nil
	}
	return f(ctx, records)
}

type Drop struct {
	Level   slog.Level
	Message string
	Dropped uint64
	Reason  string
}

type Stats struct {
	Running       bool
	Enqueued      uint64
	Dropped       uint64
	Written       uint64
	Failed        uint64
	QueueDepth    int
	QueueCapacity int
}

type Options struct {
	Level              slog.Leveler
	QueueSize          int
	BatchSize          int
	FlushInterval      time.Duration
	WriteTimeout       time.Duration
	OnDrop             func(Drop)
	OnError            func(error)
	MetricNamespace    labelset.LabelSet
	DisableDefaultPipe bool
}

type SlogHandler struct {
	dispatcher func() *dispatcher
	attrs      []slog.Attr
	groups     []string
}

type dispatcher struct {
	sink          Sink
	level         slog.Leveler
	batchSize     int
	flushInterval time.Duration
	writeTimeout  time.Duration
	onDrop        func(Drop)
	onError       func(error)

	ch     chan item
	closed atomic.Bool
	wg     sync.WaitGroup

	enqueued atomic.Uint64
	dropped  atomic.Uint64
	written  atomic.Uint64
	failed   atomic.Uint64

	metricNamespace labelset.LabelSet
}

type item struct {
	record    slog.Record
	hasRecord bool
	done      chan error
	close     bool
}

var global = struct {
	sync.Mutex
	dispatcher *dispatcher
}{}

const (
	metricLogSinkEnqueuedTotal = "log_sink_enqueued_total"
	metricLogSinkDroppedTotal  = "log_sink_dropped_total"
	metricLogSinkWrittenTotal  = "log_sink_written_total"
	metricLogSinkFailedTotal   = "log_sink_failed_total"
	metricLogSinkQueueDepth    = "log_sink_queue_depth"
	metricLogSinkQueueCapacity = "log_sink_queue_capacity"
	metricLogSinkRunning       = "log_sink_running"
)

const (
	metricLogSinkLabelScope = "scope"
	metricLogSinkScope      = "default"
)

var (
	logSinkEnqueuedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gizclaw",
		Name:      metricLogSinkEnqueuedTotal,
		Help:      "Total log records accepted by the async log sink.",
	}, []string{metricLogSinkLabelScope})
	logSinkDroppedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gizclaw",
		Name:      metricLogSinkDroppedTotal,
		Help:      "Total log records dropped by the async log sink.",
	}, []string{metricLogSinkLabelScope})
	logSinkWrittenTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gizclaw",
		Name:      metricLogSinkWrittenTotal,
		Help:      "Total log records written by the async log sink.",
	}, []string{metricLogSinkLabelScope})
	logSinkFailedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gizclaw",
		Name:      metricLogSinkFailedTotal,
		Help:      "Total log records that failed to write through the async log sink.",
	}, []string{metricLogSinkLabelScope})
	logSinkQueueDepth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gizclaw",
		Name:      metricLogSinkQueueDepth,
		Help:      "Current async log sink queue depth.",
	}, []string{metricLogSinkLabelScope})
	logSinkQueueCapacity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gizclaw",
		Name:      metricLogSinkQueueCapacity,
		Help:      "Configured async log sink queue capacity.",
	}, []string{metricLogSinkLabelScope})
	logSinkRunning = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gizclaw",
		Name:      metricLogSinkRunning,
		Help:      "Whether the async log sink dispatcher is running.",
	}, []string{metricLogSinkLabelScope})
	logSinkCounters = map[string]*prometheus.CounterVec{
		metricLogSinkEnqueuedTotal: logSinkEnqueuedTotal,
		metricLogSinkDroppedTotal:  logSinkDroppedTotal,
		metricLogSinkWrittenTotal:  logSinkWrittenTotal,
		metricLogSinkFailedTotal:   logSinkFailedTotal,
	}
	logSinkGauges = map[string]*prometheus.GaugeVec{
		metricLogSinkQueueDepth:    logSinkQueueDepth,
		metricLogSinkQueueCapacity: logSinkQueueCapacity,
		metricLogSinkRunning:       logSinkRunning,
	}
)

func Start(s Sink, opts *Options) error {
	if s == nil {
		s = SinkFunc(nil)
	}
	options := optionsOf(opts)
	d := newDispatcher(s, options)

	global.Lock()
	if global.dispatcher != nil && !global.dispatcher.closed.Load() {
		global.Unlock()
		return ErrStarted
	}
	global.dispatcher = d
	global.Unlock()

	if !options.DisableDefaultPipe {
		logger := slog.New(Handler())
		slog.SetDefault(logger)
	}
	d.start()
	return nil
}

func Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	global.Lock()
	d := global.dispatcher
	if d != nil {
		global.dispatcher = nil
	}
	global.Unlock()
	if d == nil {
		return nil
	}
	return d.close(ctx)
}

func Flush(ctx context.Context) error {
	d := currentDispatcher()
	if d == nil {
		return ErrNotStarted
	}
	return d.flush(ctx)
}

func Handler() slog.Handler {
	return &SlogHandler{dispatcher: currentDispatcher}
}

func NewHandler(s Sink, opts *Options) *SlogHandler {
	if s == nil {
		s = SinkFunc(nil)
	}
	d := newDispatcher(s, optionsOf(opts))
	d.start()
	return &SlogHandler{dispatcher: func() *dispatcher { return d }}
}

func NewHandlerSink(handler slog.Handler) Sink {
	return handlerSink{handler: handler}
}

func CurrentStats() Stats {
	d := currentDispatcher()
	if d == nil {
		return Stats{}
	}
	return d.stats()
}

func (h *SlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	d := h.current()
	if d == nil || d.closed.Load() {
		return false
	}
	return level >= d.level.Level()
}

func (h *SlogHandler) Handle(ctx context.Context, record slog.Record) error {
	d := h.current()
	if d == nil {
		return ErrNotStarted
	}
	if !h.Enabled(ctx, record.Level) {
		return nil
	}
	return d.enqueue(h.prepareRecord(record))
}

func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h == nil {
		return (*SlogHandler)(nil)
	}
	next := h.clone()
	next.attrs = append(next.attrs, groupAttrs(next.groups, cloneAttrs(attrs))...)
	return next
}

func (h *SlogHandler) WithGroup(name string) slog.Handler {
	if h == nil {
		return (*SlogHandler)(nil)
	}
	if name == "" {
		return h
	}
	next := h.clone()
	next.groups = append(next.groups, name)
	return next
}

func (h *SlogHandler) Flush(ctx context.Context) error {
	d := h.current()
	if d == nil {
		return ErrNotStarted
	}
	return d.flush(ctx)
}

func (h *SlogHandler) Close(ctx context.Context) error {
	d := h.current()
	if d == nil {
		return nil
	}
	return d.close(ctx)
}

func (h *SlogHandler) Stats() Stats {
	d := h.current()
	if d == nil {
		return Stats{}
	}
	return d.stats()
}

func (h *SlogHandler) current() *dispatcher {
	if h == nil || h.dispatcher == nil {
		return nil
	}
	return h.dispatcher()
}

func (h *SlogHandler) clone() *SlogHandler {
	return &SlogHandler{
		dispatcher: h.dispatcher,
		attrs:      cloneAttrs(h.attrs),
		groups:     append([]string(nil), h.groups...),
	}
}

func (h *SlogHandler) prepareRecord(record slog.Record) slog.Record {
	out := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	out.AddAttrs(h.attrs...)
	var attrs []slog.Attr
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	out.AddAttrs(groupAttrs(h.groups, attrs)...)
	return out
}

func newDispatcher(s Sink, opts Options) *dispatcher {
	return &dispatcher{
		sink:            s,
		level:           opts.Level,
		batchSize:       opts.BatchSize,
		flushInterval:   opts.FlushInterval,
		writeTimeout:    opts.WriteTimeout,
		onDrop:          opts.OnDrop,
		onError:         opts.OnError,
		ch:              make(chan item, opts.QueueSize),
		metricNamespace: opts.MetricNamespace,
	}
}

func (d *dispatcher) start() {
	d.registerMetrics()
	d.setGauge(metricLogSinkQueueCapacity, float64(cap(d.ch)))
	d.setGauge(metricLogSinkRunning, 1)
	d.wg.Add(1)
	go d.run()
}

func (d *dispatcher) enqueue(record slog.Record) error {
	if d == nil {
		return ErrNotStarted
	}
	if d.closed.Load() {
		return ErrClosed
	}
	select {
	case d.ch <- item{record: record, hasRecord: true}:
		d.enqueued.Add(1)
		d.counter(metricLogSinkEnqueuedTotal, 1)
		d.gauge(metricLogSinkQueueDepth, 1)
		return nil
	default:
		dropped := d.dropped.Add(1)
		d.counter(metricLogSinkDroppedTotal, 1)
		if d.onDrop != nil {
			d.onDrop(Drop{Level: record.Level, Message: record.Message, Dropped: dropped, Reason: "queue_full"})
		}
		return ErrQueueFull
	}
}

func (d *dispatcher) flush(ctx context.Context) error {
	if d == nil {
		return ErrNotStarted
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if d.closed.Load() {
		return ErrClosed
	}
	done := make(chan error, 1)
	select {
	case d.ch <- item{done: done}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *dispatcher) close(ctx context.Context) error {
	if d == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !d.closed.CompareAndSwap(false, true) {
		return nil
	}
	done := make(chan error, 1)
	select {
	case d.ch <- item{done: done, close: true}:
	case <-ctx.Done():
		d.closed.Store(false)
		return ctx.Err()
	}
	select {
	case err := <-done:
		d.wg.Wait()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *dispatcher) stats() Stats {
	if d == nil {
		return Stats{}
	}
	return Stats{
		Running:       !d.closed.Load(),
		Enqueued:      d.enqueued.Load(),
		Dropped:       d.dropped.Load(),
		Written:       d.written.Load(),
		Failed:        d.failed.Load(),
		QueueDepth:    len(d.ch),
		QueueCapacity: cap(d.ch),
	}
}

func (d *dispatcher) run() {
	defer d.wg.Done()
	defer d.setGauge(metricLogSinkRunning, 0)
	ticker := time.NewTicker(d.flushInterval)
	defer ticker.Stop()
	batch := make([]slog.Record, 0, d.batchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := d.write(batch)
		batch = batch[:0]
		return err
	}
	for {
		select {
		case item := <-d.ch:
			if item.hasRecord {
				d.gauge(metricLogSinkQueueDepth, -1)
				batch = append(batch, item.record.Clone())
				if len(batch) >= d.batchSize {
					_ = flush()
				}
			}
			if item.done != nil {
				item.done <- flush()
			}
			if item.close {
				return
			}
		case <-ticker.C:
			_ = flush()
		}
	}
}

func (d *dispatcher) write(records []slog.Record) error {
	ctx := context.Background()
	var cancel context.CancelFunc
	if d.writeTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, d.writeTimeout)
		defer cancel()
	}
	batch := cloneRecords(records)
	if err := d.sink.Write(ctx, batch); err != nil {
		d.failed.Add(uint64(len(batch)))
		d.counter(metricLogSinkFailedTotal, float64(len(batch)))
		if d.onError != nil {
			d.onError(err)
		}
		return err
	}
	d.written.Add(uint64(len(batch)))
	d.counter(metricLogSinkWrittenTotal, float64(len(batch)))
	return nil
}

func (d *dispatcher) counter(name string, value float64) {
	if d == nil || value <= 0 {
		return
	}
	counter := logSinkCounters[name]
	if counter == nil {
		return
	}
	counter.With(d.metricLabels()).Add(value)
}

func (d *dispatcher) gauge(name string, value float64) {
	if d == nil {
		return
	}
	gauge := logSinkGauges[name]
	if gauge == nil {
		return
	}
	gauge.With(d.metricLabels()).Add(value)
}

func (d *dispatcher) setGauge(name string, value float64) {
	if d == nil {
		return
	}
	gauge := logSinkGauges[name]
	if gauge == nil {
		return
	}
	gauge.With(d.metricLabels()).Set(value)
}

func (d *dispatcher) registerMetrics() {
	if d == nil {
		return
	}
	for _, collector := range []prometheus.Collector{
		logSinkEnqueuedTotal,
		logSinkDroppedTotal,
		logSinkWrittenTotal,
		logSinkFailedTotal,
		logSinkQueueDepth,
		logSinkQueueCapacity,
		logSinkRunning,
	} {
		if err := metrics.Register(collector); err != nil {
			panic(err)
		}
	}
}

func (d *dispatcher) metricLabels() prometheus.Labels {
	labels := d.metricNamespace.PrometheusLabels()
	scope := labels[metricLogSinkLabelScope]
	if scope == "" {
		scope = metricLogSinkScope
	}
	return prometheus.Labels{metricLogSinkLabelScope: scope}
}

type handlerSink struct {
	handler slog.Handler
}

func (s handlerSink) Write(ctx context.Context, records []slog.Record) error {
	if s.handler == nil {
		return nil
	}
	var errs []error
	for _, record := range records {
		if !s.handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := s.handler.Handle(ctx, record.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func currentDispatcher() *dispatcher {
	global.Lock()
	defer global.Unlock()
	return global.dispatcher
}

func optionsOf(opts *Options) Options {
	out := Options{
		Level:         slog.LevelInfo,
		QueueSize:     1024,
		BatchSize:     64,
		FlushInterval: time.Second,
	}
	if opts == nil {
		return out
	}
	if opts.Level != nil {
		out.Level = opts.Level
	}
	if opts.QueueSize > 0 {
		out.QueueSize = opts.QueueSize
	}
	if opts.BatchSize > 0 {
		out.BatchSize = opts.BatchSize
	}
	if opts.FlushInterval > 0 {
		out.FlushInterval = opts.FlushInterval
	}
	out.WriteTimeout = opts.WriteTimeout
	out.OnDrop = opts.OnDrop
	out.OnError = opts.OnError
	out.MetricNamespace = opts.MetricNamespace
	out.DisableDefaultPipe = opts.DisableDefaultPipe
	return out
}

func groupAttrs(groups []string, attrs []slog.Attr) []slog.Attr {
	if len(attrs) == 0 {
		return nil
	}
	attrs = cloneAttrs(attrs)
	if len(groups) == 0 {
		return attrs
	}
	group := slog.Attr{Key: groups[len(groups)-1], Value: slog.GroupValue(attrs...)}
	for i := len(groups) - 2; i >= 0; i-- {
		group = slog.Attr{Key: groups[i], Value: slog.GroupValue(group)}
	}
	return []slog.Attr{group}
}

func cloneAttrs(attrs []slog.Attr) []slog.Attr {
	if len(attrs) == 0 {
		return nil
	}
	out := make([]slog.Attr, len(attrs))
	copy(out, attrs)
	return out
}

func cloneRecords(records []slog.Record) []slog.Record {
	out := make([]slog.Record, len(records))
	for i, record := range records {
		out[i] = record.Clone()
	}
	return out
}
