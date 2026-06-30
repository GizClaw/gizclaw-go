package servecmd

import (
	"context"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/gizrun"
)

func TestHandlerRejectsMissingWorkspace(t *testing.T) {
	err := Handler{}.Execute(context.Background(), gizrun.CommandLine{Args: []string{"serve"}})
	if err == nil || !strings.Contains(err.Error(), "expected workspace dir") {
		t.Fatalf("Execute error = %v, want workspace error", err)
	}
}

func TestHandlerRejectsUnexpectedFlagArgs(t *testing.T) {
	err := Handler{}.Execute(context.Background(), gizrun.CommandLine{
		Args:  []string{"serve", t.TempDir()},
		Flags: []string{"--force", "extra"},
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected flags") {
		t.Fatalf("Execute error = %v, want unexpected flags error", err)
	}
}

func TestHandlerRejectsDirectServeWithoutForce(t *testing.T) {
	err := Handler{}.Execute(context.Background(), gizrun.CommandLine{
		Args: []string{"serve", t.TempDir()},
	})
	if err == nil || !strings.Contains(err.Error(), "direct serve is disabled") {
		t.Fatalf("Execute error = %v, want direct serve disabled error", err)
	}
}

func TestHandlerForceAllowsServePath(t *testing.T) {
	err := Handler{}.Execute(context.Background(), gizrun.CommandLine{
		Args:  []string{"serve", t.TempDir()},
		Flags: []string{"--force"},
	})
	if err == nil || !strings.Contains(err.Error(), "load config") {
		t.Fatalf("Execute error = %v, want workspace load error", err)
	}
}

func TestNewCmdRejectsMissingWorkspace(t *testing.T) {
	cmd := NewCmd()
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute succeeded, want argument error")
	}
}
