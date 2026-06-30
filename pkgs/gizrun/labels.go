package gizrun

import (
	"context"
	"log/slog"

	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/labelset"
)

const (
	nsHTTP    = "http"
	nsGenx    = "genx"
	nsLogSink = "logsink"
)

const (
	httpMethod     = "method"
	httpPath       = "path"
	httpHost       = "host"
	httpStatusCode = "status_code"
)

const (
	genxProvider  = "provider"
	genxMethod    = "method"
	genxModel     = "model"
	genxStatus    = "status"
	genxTokenType = "token_type"
)

const (
	tokenCached    = "cached"
	tokenGenerated = "generated"
	tokenPrompt    = "prompt"
)

const (
	GenxProvider  = genxProvider
	GenxMethod    = genxMethod
	GenxModel     = genxModel
	GenxStatus    = genxStatus
	GenxTokenType = genxTokenType
)

func tagHTTP(ctx context.Context, kvs ...string) context.Context {
	return labelset.Tag(ctx, nsHTTP, kvs...)
}

func TagGenx(ctx context.Context, kvs ...string) context.Context {
	return tagGenx(ctx, kvs...)
}

func GenxAttr(ctx context.Context) (slog.Attr, bool) {
	labels, ok := genxLabels(ctx)
	if !ok {
		return slog.Attr{}, false
	}
	return labels.Attr(), true
}

func tagGenx(ctx context.Context, kvs ...string) context.Context {
	return labelset.Tag(ctx, nsGenx, kvs...)
}

func tagLogSink(ctx context.Context, kvs ...string) context.Context {
	return labelset.Tag(ctx, nsLogSink, kvs...)
}

func tag(ctx context.Context, name string, kvs ...string) context.Context {
	return labelset.Tag(ctx, name, kvs...)
}

func httpLabels(ctx context.Context) (labelset.LabelSet, bool) {
	return labelset.FromContext(ctx, nsHTTP)
}

func genxLabels(ctx context.Context) (labelset.LabelSet, bool) {
	return labelset.FromContext(ctx, nsGenx)
}

func logSinkLabels(ctx context.Context) (labelset.LabelSet, bool) {
	return labelset.FromContext(ctx, nsLogSink)
}

func labels(ctx context.Context, namespace string) (labelset.LabelSet, bool) {
	return labelset.FromContext(ctx, namespace)
}
