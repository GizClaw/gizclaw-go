package cmdhandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCmdMuxHandleRegistersHandler(t *testing.T) {
	mux := NewMux()
	called := false
	handler := HandleFunc(func(context.Context, CommandLine) error {
		called = true
		return nil
	})

	if err := mux.Handle("admin/play", handler); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if err := mux.Execute(context.Background(), CommandLine{Args: []string{"admin", "play"}}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestCmdMuxHandleRejectsNilHandler(t *testing.T) {
	mux := NewMux()
	err := mux.Handle("admin/play", nil)
	if err == nil || !strings.Contains(err.Error(), "nil handler") {
		t.Fatalf("Handle err = %v, want nil handler error", err)
	}
}

func TestCmdMuxHandleRejectsDuplicate(t *testing.T) {
	mux := NewMux()
	handler := HandleFunc(func(context.Context, CommandLine) error { return nil })
	if err := mux.Handle("admin/play", handler); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if err := mux.Handle("admin/play", handler); err == nil {
		t.Fatal("Handle duplicate err = nil, want error")
	}
}

func TestCmdMuxImplementsHandler(t *testing.T) {
	mux := NewMux()
	var gotArgs []string
	var gotFlags []string
	if err := mux.Handle("admin/play", HandleFunc(func(_ context.Context, commandLine CommandLine) error {
		gotArgs = append([]string(nil), commandLine.Args...)
		gotFlags = append([]string(nil), commandLine.Flags...)
		return nil
	})); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	args := []string{"admin", "play"}
	flags := []string{"-force"}
	if err := mux.Execute(context.Background(), CommandLine{Args: args, Flags: flags}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !reflect.DeepEqual(gotArgs, args) {
		t.Fatalf("args = %#v, want %#v", gotArgs, args)
	}
	if !reflect.DeepEqual(gotFlags, flags) {
		t.Fatalf("flags = %#v, want %#v", gotFlags, flags)
	}
}

func TestCmdMuxExecuteMissingHandler(t *testing.T) {
	mux := NewMux()
	err := mux.Execute(context.Background(), CommandLine{Args: []string{"missing"}})
	if !errors.Is(err, ErrHandlerNotFound) {
		t.Fatalf("Execute missing err = %v, want ErrHandlerNotFound", err)
	}
}
