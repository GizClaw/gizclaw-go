package adminapi

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/gizclaw/api/apitypes"
)

func TestCollectAllPagesAggregatesUntilDone(t *testing.T) {
	var seen []string
	items, err := collectAllPages(func(cursor *string, limit *int32) (pagedItems[string], error) {
		if limit == nil || *limit != 200 {
			t.Fatalf("limit = %v, want 200", limit)
		}
		label := "<nil>"
		if cursor != nil {
			label = string(*cursor)
		}
		seen = append(seen, label)
		switch label {
		case "<nil>":
			next := "page-1"
			return pagedItems[string]{HasNext: true, Items: []string{"a", "b"}, NextCursor: &next}, nil
		case "page-1":
			next := "page-2"
			return pagedItems[string]{HasNext: true, Items: []string{"c"}, NextCursor: &next}, nil
		case "page-2":
			return pagedItems[string]{HasNext: false, Items: []string{"d"}}, nil
		default:
			t.Fatalf("unexpected cursor %q", label)
			return pagedItems[string]{}, nil
		}
	})
	if err != nil {
		t.Fatalf("collectAllPages error = %v", err)
	}
	if !slices.Equal(seen, []string{"<nil>", "page-1", "page-2"}) {
		t.Fatalf("seen cursors = %v", seen)
	}
	if !slices.Equal(items, []string{"a", "b", "c", "d"}) {
		t.Fatalf("items = %v", items)
	}
}

func TestCollectAllPagesStopsOnMissingNextCursor(t *testing.T) {
	calls := 0
	items, err := collectAllPages(func(cursor *string, limit *int32) (pagedItems[string], error) {
		calls++
		return pagedItems[string]{HasNext: true, Items: []string{"a"}}, nil
	})
	if err != nil {
		t.Fatalf("collectAllPages error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if !slices.Equal(items, []string{"a"}) {
		t.Fatalf("items = %v", items)
	}
}

func TestCollectAllPagesRejectsRepeatedCursor(t *testing.T) {
	repeated := "same"
	_, err := collectAllPages(func(cursor *string, limit *int32) (pagedItems[string], error) {
		if cursor == nil {
			return pagedItems[string]{HasNext: true, Items: []string{"a"}, NextCursor: &repeated}, nil
		}
		return pagedItems[string]{HasNext: true, Items: []string{"b"}, NextCursor: &repeated}, nil
	})
	if err == nil {
		t.Fatal("collectAllPages succeeded with repeated cursor")
	}
	if !strings.Contains(err.Error(), "cursor did not advance") {
		t.Fatalf("collectAllPages error = %v", err)
	}
}

func TestCollectAllPagesPropagatesErrors(t *testing.T) {
	want := errors.New("boom")
	_, err := collectAllPages(func(cursor *string, limit *int32) (pagedItems[string], error) {
		return pagedItems[string]{}, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestResponseErrorUsesStructuredError(t *testing.T) {
	err := responseError(400, nil, &apitypes.ErrorResponse{
		Error: apitypes.ErrorPayload{Code: "bad_request", Message: "missing field"},
	})
	if err == nil || err.Error() != "bad_request: missing field" {
		t.Fatalf("responseError() = %v", err)
	}
}

func TestResponseErrorUsesResponseBody(t *testing.T) {
	err := responseError(502, []byte(" upstream failed \n"))
	if err == nil || !strings.Contains(err.Error(), "unexpected status 502: upstream failed") {
		t.Fatalf("responseError() = %v", err)
	}
}

func TestResponseErrorUsesFallbackStatus(t *testing.T) {
	err := responseError(500, nil)
	if err == nil || err.Error() != "unexpected status 500" {
		t.Fatalf("responseError() = %v", err)
	}
}

func TestResponseErrorHandlesEmptyResponse(t *testing.T) {
	err := responseError(0, nil)
	if err == nil || err.Error() != "unexpected empty response" {
		t.Fatalf("responseError() = %v", err)
	}
}
