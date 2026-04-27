package playuionly_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/integration/cmd"
)

func TestPlayKeepsOnlyUIEntryUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "706-play-ui-only")

	help := h.RunCLI("play", "--help")
	help.MustSucceed(t)
	if !strings.Contains(help.Stdout, "--listen") {
		t.Fatalf("play help should keep UI listen flag:\n%s", help.Stdout)
	}
	for _, removed := range []string{"register", "config", "ota", "serve"} {
		if strings.Contains(help.Stdout, "\n  "+removed) || strings.Contains(help.Stdout, "\n    "+removed) {
			t.Fatalf("play help should not mention removed %q subcommand:\n%s", removed, help.Stdout)
		}
		result := h.RunCLI("play", removed)
		if result.Err == nil {
			t.Fatalf("play %s should be removed:\nstdout:\n%s", removed, result.Stdout)
		}
	}

	noListen := h.RunCLI("play")
	noListen.MustSucceed(t)
	if !strings.Contains(noListen.Stdout, "--listen") {
		t.Fatalf("play without --listen should print UI help:\n%s", noListen.Stdout)
	}
}

func TestPlayListenAutoRegistersCurrentContext(t *testing.T) {
	h := clitest.NewHarness(t, "706-play-ui-only")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "admin_default", "--sn", "admin-sn").MustSucceed(t)
	h.CreateContext("play-a").MustSucceed(t)

	listenAddr := freeTCPAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, h.BinaryPath, "play", "--context", "play-a", "--listen", listenAddr)
	cmd.Dir = h.SandboxDir
	cmd.Env = append(os.Environ(), "HOME="+h.HomeDir, "XDG_CONFIG_HOME="+h.XDGConfigHome)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start play UI: %v", err)
	}
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	publicKey := h.ContextPublicKey("play-a")
	result, err := h.RunCLIUntilSuccess("admin", "gears", "get", publicKey, "--context", "admin-a")
	if err != nil {
		t.Fatalf("play UI did not auto-register context: %v\nplay stdout:\n%s\nplay stderr:\n%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(result.Stdout, `"auto_registered":true`) {
		t.Fatalf("registered gear should be marked auto_registered:\n%s", result.Stdout)
	}
}

func TestPlayListenDoesNotRegisterWhenListenFails(t *testing.T) {
	h := clitest.NewHarness(t, "706-play-ui-only")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "admin_default", "--sn", "admin-sn").MustSucceed(t)
	h.CreateContext("play-a").MustSucceed(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve occupied listen addr: %v", err)
	}
	defer listener.Close()
	listenAddr := listener.Addr().String()

	result := h.RunCLI("play", "--context", "play-a", "--listen", listenAddr)
	if result.Err == nil {
		t.Fatalf("play should fail when listen addr is occupied:\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}

	publicKey := h.ContextPublicKey("play-a")
	get := h.RunCLI("admin", "gears", "get", publicKey, "--context", "admin-a")
	if get.Err == nil {
		t.Fatalf("play should not auto-register when listen fails:\n%s", get.Stdout)
	}
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate listen addr: %v", err)
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener addr %T", listener.Addr())
	}
	return fmt.Sprintf("127.0.0.1:%d", addr.Port)
}
