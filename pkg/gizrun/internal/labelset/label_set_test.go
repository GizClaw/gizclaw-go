package labelset

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
)

func TestTag(t *testing.T) {
	ctx := Tag(nil, "http", "method", "GET", "path", "/v1")

	ns, ok := FromContext(ctx, "http")
	if !ok {
		t.Fatalf("FromContext(http) ok = false, want true")
	}
	if got := ns.Name(); got != "http" {
		t.Fatalf("Name() = %q, want %q", got, "http")
	}
	if got, ok := ns.Value("method"); !ok || got != "GET" {
		t.Fatalf("Value(method) = (%q, %v), want (%q, true)", got, ok, "GET")
	}
	if got, ok := ns.Value("missing"); ok || got != "" {
		t.Fatalf("Value(missing) = (%q, %v), want (%q, false)", got, ok, "")
	}

	if got := collect(ns.Keys()); !reflect.DeepEqual(got, []string{"method", "path"}) {
		t.Fatalf("Keys() = %#v, want method/path order", got)
	}
	if got := collect(ns.Values()); !reflect.DeepEqual(got, []string{"GET", "/v1"}) {
		t.Fatalf("Values() = %#v, want GET/path values", got)
	}
	if got := collectLabels(ns.Labels()); !reflect.DeepEqual(got, [][2]string{{"method", "GET"}, {"path", "/v1"}}) {
		t.Fatalf("Labels() = %#v, want method/path entries", got)
	}
	attr := ns.Attr()
	if attr.Key != "http" || attr.Value.Kind() != slog.KindGroup {
		t.Fatalf("Attr() = %#v, want http group", attr)
	}
}

func TestTagMergesLabelSet(t *testing.T) {
	ctx := Tag(context.Background(), "http", "method", "GET", "path", "/v1")
	ctx = Tag(ctx, "http", "method", "POST", "status_code", "200")

	ns, ok := FromContext(ctx, "http")
	if !ok {
		t.Fatalf("FromContext(http) ok = false, want true")
	}
	if got := collect(ns.Keys()); !reflect.DeepEqual(got, []string{"method", "path", "status_code"}) {
		t.Fatalf("Keys() = %#v, want method/path/status_code order", got)
	}
}

func TestTagIgnoresEmptyLabelSetAndIncompletePair(t *testing.T) {
	ctx := Tag(context.Background(), "", "method", "GET")
	if _, ok := FromContext(ctx, ""); ok {
		t.Fatalf("FromContext(empty) ok = true, want false")
	}

	ctx = Tag(ctx, "http", "method")
	ns, ok := FromContext(ctx, "http")
	if ok {
		t.Fatalf("FromContext(http) = (%#v, true), want missing", ns)
	}
}

func TestTagDoesNotMutateParentContext(t *testing.T) {
	parent := Tag(context.Background(), "http", "method", "GET")
	child := Tag(parent, "peer", "id", "peer-1")

	if _, ok := FromContext(parent, "peer"); ok {
		t.Fatalf("parent FromContext(peer) ok = true, want false")
	}
	if ns, ok := FromContext(child, "peer"); !ok || ns.Name() != "peer" || len(collect(ns.Keys())) != 1 {
		t.Fatalf("child FromContext(peer) = (%#v, %v), want injected label set", ns, ok)
	}
}

func TestTagDoesNotMutateParentLabelSet(t *testing.T) {
	parent := Tag(context.Background(), "http", "method", "GET")
	child := Tag(parent, "http", "method", "POST", "path", "/v1")

	parentNS, ok := FromContext(parent, "http")
	if !ok {
		t.Fatalf("parent FromContext(http) ok = false, want true")
	}
	if got, ok := parentNS.Value("method"); !ok || got != "GET" {
		t.Fatalf("parent method = (%q, %v), want (%q, true)", got, ok, "GET")
	}
	if _, ok := parentNS.Value("path"); ok {
		t.Fatalf("parent path ok = true, want false")
	}

	childNS, ok := FromContext(child, "http")
	if !ok {
		t.Fatalf("child FromContext(http) ok = false, want true")
	}
	if got, ok := childNS.Value("method"); !ok || got != "POST" {
		t.Fatalf("child method = (%q, %v), want (%q, true)", got, ok, "POST")
	}
	if got, ok := childNS.Value("path"); !ok || got != "/v1" {
		t.Fatalf("child path = (%q, %v), want (%q, true)", got, ok, "/v1")
	}
}

func TestLabelSetIteratorsStopWhenYieldReturnsFalse(t *testing.T) {
	ctx := Tag(context.Background(), "http", "method", "GET")
	ns, ok := FromContext(ctx, "http")
	if !ok {
		t.Fatalf("FromContext(http) ok = false, want true")
	}

	var got []string
	for key := range ns.Keys() {
		got = append(got, key)
		break
	}
	if len(got) != 1 || got[0] != "method" {
		t.Fatalf("first key = %#v, want method", got)
	}
}

func TestLabelSetMissingReturnsFalse(t *testing.T) {
	ns, ok := FromContext(context.Background(), "http")
	if ok {
		t.Fatalf("FromContext(http) ok = true, want false")
	}
	if got := ns.Name(); got != "" {
		t.Fatalf("missing label set name = %q, want empty", got)
	}
}

func collect(seq func(func(string) bool)) []string {
	var values []string
	for value := range seq {
		values = append(values, value)
	}
	return values
}

func collectLabels(seq func(func(string, string) bool)) [][2]string {
	var values [][2]string
	for key, value := range seq {
		values = append(values, [2]string{key, value})
	}
	return values
}
