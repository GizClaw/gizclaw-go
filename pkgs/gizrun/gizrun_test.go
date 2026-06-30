package gizrun

import (
	"context"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/gizrun/internal/cmdhandler"
)

func TestHandleCmd(t *testing.T) {
	resetDefaultCmdMuxForTest(t)

	handler := CmdHandleFunc(func(context.Context, CommandLine) error { return nil })
	if err := HandleCmd("admin/play", handler); err != nil {
		t.Fatalf("HandleCmd failed: %v", err)
	}

	if err := runCtx.cmdMux.Execute(context.Background(), CommandLine{Args: []string{"admin", "play"}}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestServeMissingCommandReturnsError(t *testing.T) {
	resetRuntimeForTest(t)
	resetInitHooksForTest(t)
	resetDefaultCmdMuxForTest(t)
	resetFlagSetForTest(t)

	previousArgs := os.Args
	os.Args = []string{"gizrun", "missing"}
	t.Cleanup(func() {
		os.Args = previousArgs
	})

	exitRan := false
	PostInitAt(0, func(*RunContext) error {
		return nil
	})
	ExitAt(0, func(*RunContext) error {
		exitRan = true
		return nil
	})

	err := Serve()
	if err == nil || !strings.Contains(err.Error(), "command handler not found") {
		t.Fatalf("Serve error = %v, want command handler not found", err)
	}
	if !exitRan {
		t.Fatal("exit hook did not run after post-init startup")
	}
}

func TestStripRegisteredFlags(t *testing.T) {
	fs := flag.NewFlagSet("gizrun-test", flag.ContinueOnError)
	var config string
	var verbose bool
	fs.StringVar(&config, "config", "", "config path")
	fs.BoolVar(&verbose, "verbose", false, "verbose logging")

	got, err := stripRegisteredFlags(fs, []string{
		"-config", "app.yaml",
		"chat",
		"-model=gpt",
		"--verbose",
		"--",
		"-literal",
	})
	if err != nil {
		t.Fatalf("stripRegisteredFlags failed: %v", err)
	}
	if config != "app.yaml" {
		t.Fatalf("config = %q, want app.yaml", config)
	}
	if !verbose {
		t.Fatal("verbose = false, want true")
	}
	if want := []string{"chat", "-model=gpt", "--", "-literal"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %#v, want %#v", got, want)
	}
}

func TestStripRegisteredFlagsMissingValue(t *testing.T) {
	fs := flag.NewFlagSet("gizrun-test", flag.ContinueOnError)
	var config string
	fs.StringVar(&config, "config", "", "config path")

	if _, err := stripRegisteredFlags(fs, []string{"-config"}); err == nil {
		t.Fatal("stripRegisteredFlags error = nil, want missing value error")
	}
}

func resetDefaultCmdMuxForTest(t *testing.T) {
	t.Helper()
	previous := runCtx.cmdMux
	runCtx.cmdMux = cmdhandler.NewMux()
	t.Cleanup(func() {
		runCtx.cmdMux = previous
	})
}
