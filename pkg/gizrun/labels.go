package gizrun

import (
	"context"

	"github.com/GizClaw/gizclaw-go/pkg/gizrun/internal/labelset"
)

type LabelSet = labelset.LabelSet

const (
	nsHTTP    = "http"
	nsGenx    = "genx"
	nsLogSink = "logsink"
)

const (
	HTTPMethod     = "method"
	HTTPRoute      = "route"
	HTTPPath       = "path"
	HTTPHost       = "host"
	HTTPStatusCode = "status_code"
)

const (
	GenxProvider  = "provider"
	GenxMethod    = "method"
	GenxModel     = "model"
	GenxStatus    = "status"
	GenxTokenType = "token_type"
)

const (
	TokenCached    = "cached"
	TokenGenerated = "generated"
	TokenPrompt    = "prompt"
)

func TagHTTP(ctx context.Context, keyValues ...string) context.Context {
	return labelset.Tag(ctx, nsHTTP, keyValues...)
}

func TagGenx(ctx context.Context, keyValues ...string) context.Context {
	return labelset.Tag(ctx, nsGenx, keyValues...)
}

func TagLogSink(ctx context.Context, keyValues ...string) context.Context {
	return labelset.Tag(ctx, nsLogSink, keyValues...)
}

func Tag(ctx context.Context, name string, keyValues ...string) context.Context {
	return labelset.Tag(ctx, name, keyValues...)
}

func HTTPLabels(ctx context.Context) (LabelSet, bool) {
	return labelset.FromContext(ctx, nsHTTP)
}

func GenxLabels(ctx context.Context) (LabelSet, bool) {
	return labelset.FromContext(ctx, nsGenx)
}

func LogSinkLabels(ctx context.Context) (LabelSet, bool) {
	return labelset.FromContext(ctx, nsLogSink)
}

func Labels(ctx context.Context, namespace string) (LabelSet, bool) {
	return labelset.FromContext(ctx, namespace)
}
