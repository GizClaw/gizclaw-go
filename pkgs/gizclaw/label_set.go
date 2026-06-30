package gizclaw

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"maps"
)

var (
	_ LabelSet = (*HTTPLabelSet)(nil)
	_ LabelSet = (*GenxLabelSet)(nil)
)

type LabelSet interface {
	Namespace() string
	MergeWith(LabelSet) error
	Keys() iter.Seq[string]
	Values() iter.Seq[string]
	KeyValues() iter.Seq2[string, string]
}

type HTTPLabelSet struct {
	Method     string
	Path       string
	Host       string
	StatusCode string
	TraceID    string
}

type GenxLabelSet struct {
	HTTP      HTTPLabelSet
	Provider  string
	Method    string
	Model     string
	Status    string
	TokenType string
}

type labelSetStore map[string]LabelSet

type labelSetStoreContextKey struct{}

func Tag(ctx context.Context, labels LabelSet) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	store, _ := ctx.Value(labelSetStoreContextKey{}).(labelSetStore)
	next := labelSetStore{}
	maps.Copy(next, store)
	if current := next[labels.Namespace()]; current != nil {
		if err := current.MergeWith(labels); err != nil {
			return nil, err
		}
		next[current.Namespace()] = current
	} else {
		next[labels.Namespace()] = labels
	}
	return context.WithValue(ctx, labelSetStoreContextKey{}, next), nil
}

func LogAttr(labels LabelSet) slog.Attr {
	attrs := []slog.Attr{}
	for key, value := range labels.KeyValues() {
		attrs = append(attrs, slog.String(key, value))
	}
	return slog.Attr{Key: labels.Namespace(), Value: slog.GroupValue(attrs...)}
}

func labelSet(ctx context.Context, namespace string) (LabelSet, bool) {
	store, _ := ctx.Value(labelSetStoreContextKey{}).(labelSetStore)
	labels, ok := store[namespace]
	return labels, ok
}

func (*HTTPLabelSet) Namespace() string {
	return "http"
}

func (s *HTTPLabelSet) MergeWith(next LabelSet) error {
	value, ok := next.(*HTTPLabelSet)
	if !ok {
		return fmt.Errorf("gizclaw: merge http labels with %T", next)
	}
	if value.Method != "" {
		s.Method = value.Method
	}
	if value.Path != "" {
		s.Path = value.Path
	}
	if value.Host != "" {
		s.Host = value.Host
	}
	if value.StatusCode != "" {
		s.StatusCode = value.StatusCode
	}
	if value.TraceID != "" {
		s.TraceID = value.TraceID
	}
	return nil
}

func (s *HTTPLabelSet) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, key := range []string{"method", "path", "host", "status_code", "trace_id"} {
			if !yield(key) {
				return
			}
		}
	}
}

func (s *HTTPLabelSet) Values() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, value := range []string{s.Method, s.Path, s.Host, s.StatusCode, s.TraceID} {
			if value != "" && !yield(value) {
				return
			}
		}
	}
}

func (s *HTTPLabelSet) KeyValues() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for key, value := range map[string]string{
			"method":      s.Method,
			"path":        s.Path,
			"host":        s.Host,
			"status_code": s.StatusCode,
			"trace_id":    s.TraceID,
		} {
			if value != "" && !yield(key, value) {
				return
			}
		}
	}
}

func (*GenxLabelSet) Namespace() string {
	return "genx"
}

func (s *GenxLabelSet) MergeWith(next LabelSet) error {
	value, ok := next.(*GenxLabelSet)
	if !ok {
		return fmt.Errorf("gizclaw: merge genx labels with %T", next)
	}
	if err := s.HTTP.MergeWith(&value.HTTP); err != nil {
		return err
	}
	if value.Provider != "" {
		s.Provider = value.Provider
	}
	if value.Method != "" {
		s.Method = value.Method
	}
	if value.Model != "" {
		s.Model = value.Model
	}
	if value.Status != "" {
		s.Status = value.Status
	}
	if value.TokenType != "" {
		s.TokenType = value.TokenType
	}
	return nil
}

func (s *GenxLabelSet) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for key := range s.HTTP.Keys() {
			if !yield(key) {
				return
			}
		}
		for _, key := range []string{"provider", "genx_method", "model", "status", "token_type"} {
			if !yield(key) {
				return
			}
		}
	}
}

func (s *GenxLabelSet) Values() iter.Seq[string] {
	return func(yield func(string) bool) {
		for value := range s.HTTP.Values() {
			if !yield(value) {
				return
			}
		}
		for _, value := range []string{s.Provider, s.Method, s.Model, s.Status, s.TokenType} {
			if value != "" && !yield(value) {
				return
			}
		}
	}
}

func (s *GenxLabelSet) KeyValues() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for key, value := range s.HTTP.KeyValues() {
			if !yield(key, value) {
				return
			}
		}
		for key, value := range map[string]string{
			"provider":    s.Provider,
			"genx_method": s.Method,
			"model":       s.Model,
			"status":      s.Status,
			"token_type":  s.TokenType,
		} {
			if value != "" && !yield(key, value) {
				return
			}
		}
	}
}
