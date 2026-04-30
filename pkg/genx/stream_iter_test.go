package genx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestIterDoesNotDropEventsUnderBackpressure(t *testing.T) {
	const n = 20
	chunks := make([]*MessageChunk, 0, n)
	for i := range n {
		chunks = append(chunks, &MessageChunk{
			Role: RoleModel,
			Name: "assistant",
			ToolCall: &ToolCall{
				ID: fmt.Sprintf("id-%d", i),
				FuncCall: &FuncCall{
					Name:      "tool",
					Arguments: "{}",
				},
			},
		})
	}

	itr := Iter(&sliceStream{chunks: chunks, doneErr: ErrDone})

	// 先让生产者跑起来，模拟消费者慢导致的背压场景。
	time.Sleep(30 * time.Millisecond)

	count := 0
	for {
		el, err := itr.Next()
		if err != nil {
			if errors.Is(err, ErrDone) {
				break
			}
			t.Fatalf("iter next failed: %v", err)
		}
		if _, ok := el.(*ToolCallElement); !ok {
			t.Fatalf("unexpected element type: %T", el)
		}
		count++
	}

	if count != n {
		t.Fatalf("expected %d events, got %d", n, count)
	}
}

func TestIterTreatsEOFAsDone(t *testing.T) {
	itr := Iter(&sliceStream{chunks: []*MessageChunk{{Part: Text("x")}}, doneErr: io.EOF})

	if _, err := itr.Next(); err != nil {
		t.Fatalf("first next failed: %v", err)
	}
	if _, err := itr.Next(); !errors.Is(err, ErrDone) {
		t.Fatalf("expected ErrDone after EOF source, got: %v", err)
	}
}

func TestStreamIterWhereFirstWhereAndWriteTo(t *testing.T) {
	chunks := []*MessageChunk{
		{Role: RoleModel, ToolCall: &ToolCall{ID: "c1", FuncCall: &FuncCall{Name: "fn", Arguments: "{}"}}},
		{Role: RoleModel, Part: Text("hello")},
		{Role: RoleModel, Part: &Blob{MIMEType: "audio/opus", Data: []byte{1, 2}}},
	}

	itr := Iter(&sliceStream{chunks: chunks, doneErr: ErrDone})

	var got []IterElement
	for el, err := range itr.Where(func(el IterElement) bool {
		_, ok := el.(*ToolCallElement)
		return ok
	}) {
		if err != nil {
			t.Fatalf("where returned error: %v", err)
		}
		got = append(got, el)
	}
	if len(got) != 1 {
		t.Fatalf("unexpected where result length: %d", len(got))
	}

	itr2 := Iter(&sliceStream{chunks: []*MessageChunk{
		{Role: RoleModel, Part: Text("hello")},
		{Role: RoleModel, Part: &Blob{MIMEType: "audio/opus", Data: []byte{1, 2}}},
	}, doneErr: ErrDone})
	el, err := itr2.FirstWhere(func(el IterElement) bool {
		s, ok := el.(*StreamElement)
		return ok && s.MIMEType == "text/plain"
	})
	if err != nil {
		t.Fatalf("FirstWhere failed: %v", err)
	}
	if el == nil {
		t.Fatal("expected FirstWhere to return stream element")
	}

	itr3 := Iter(&sliceStream{chunks: []*MessageChunk{{Role: RoleModel, Part: Text("hello")}}, doneErr: ErrDone})
	var out bytes.Buffer
	n, err := itr3.WriteTo("text/plain", &out)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	if n == 0 || out.String() != "hello" {
		t.Fatalf("unexpected WriteTo output: n=%d text=%q", n, out.String())
	}

	itr4 := Iter(&sliceStream{chunks: []*MessageChunk{}, doneErr: ErrDone})
	el, err = itr4.FirstWhere(func(IterElement) bool { return true })
	if err != nil {
		t.Fatalf("FirstWhere on empty should not fail: %v", err)
	}
	if el != nil {
		t.Fatalf("expected nil element on empty iterator, got: %#v", el)
	}
}

func TestStreamIterPullInvalidChunkAndPart(t *testing.T) {
	itr := Iter(&sliceStream{chunks: []*MessageChunk{{Role: RoleModel}}, doneErr: ErrDone})
	if _, err := itr.Next(); err == nil || !strings.Contains(err.Error(), "invalid message chunk") {
		t.Fatalf("expected invalid chunk error, got: %v", err)
	}

	itr2 := Iter(&sliceStream{chunks: []*MessageChunk{{Role: RoleModel, Part: customPart{}}}, doneErr: ErrDone})
	if _, err := itr2.Next(); err == nil || !strings.Contains(err.Error(), "invalid part type") {
		t.Fatalf("expected invalid part type error, got: %v", err)
	}
}

func TestStreamIterWherePropagatesErrors(t *testing.T) {
	want := errors.New("read failed")
	itr := Iter(&sliceStream{doneErr: want})

	count := 0
	for _, err := range itr.Where(func(IterElement) bool { return true }) {
		count++
		if !errors.Is(err, want) {
			t.Fatalf("expected propagated error %v, got %v", want, err)
		}
	}
	if count != 1 {
		t.Fatalf("expected one yielded error, got %d", count)
	}
}
