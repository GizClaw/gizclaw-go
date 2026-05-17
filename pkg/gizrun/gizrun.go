package gizrun

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"sort"

	"github.com/GizClaw/gizclaw-go/pkg/gizrun/internal/log/sink"
	"github.com/GizClaw/gizclaw-go/pkg/gizrun/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var flags = struct {
	enablePprof bool
	logLevel    slog.LevelVar
	registered  bool
}{}

type initHook struct {
	seq   int
	order int
	fn    func() error
}

var initHooks = struct {
	next  int
	hooks []initHook
}{}

func init() {
	InitAt(0, func() error {
		if flags.registered {
			return nil
		}
		flag.BoolVar(&flags.enablePprof, "pprof", false, "enable pprof handlers")
		flags.logLevel.Set(slog.LevelInfo)
		flag.Var(logLevelFlag{target: &flags.logLevel}, "log-level", "set slog level: debug, info, warn, or error")
		flags.registered = true
		return nil
	})
}

func InitAt(seq int, fn func() error) {
	if fn == nil {
		return
	}
	initHooks.hooks = append(initHooks.hooks, initHook{seq: seq, order: initHooks.next, fn: fn})
	initHooks.next++
}

func Start() {
	metrics.Reset(prometheus.DefaultRegisterer)
	hooks := append([]initHook(nil), initHooks.hooks...)

	sort.SliceStable(hooks, func(i, j int) bool {
		if hooks[i].seq != hooks[j].seq {
			return hooks[i].seq < hooks[j].seq
		}
		return hooks[i].order < hooks[j].order
	})
	for _, hook := range hooks {
		if err := hook.fn(); err != nil {
			panic(err)
		}
	}

	flag.Parse()
	serveMux = http.NewServeMux()
	if flags.enablePprof {
		serveMux.Handle("/debug/pprof/", pprofHandler())
	}
	serveMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})
	httpHandler = newHandler()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: &flags.logLevel})
	if err := sink.Start(sink.NewHandlerSink(handler), nil); err != nil {
		panic(err)
	}
	registerGizrunMetrics()
}

func Stop() {
	_ = sink.Stop(context.Background())
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	metrics.Reset(prometheus.DefaultRegisterer)
}
