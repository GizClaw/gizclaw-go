package clientpublicretryablereads_test

import (
	"context"
	"testing"
	"time"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestClientPublicRetryableReadsUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "302-client-public-retryable-reads")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("device-a").MustSucceed(t)
	h.RegisterContext("device-a", "--sn", "device-a-sn").MustSucceed(t)

	for i := range 4 {
		c := h.ConnectClientFromContext("device-a")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		info, err := c.GetPeerInfo(ctx, "peer.info.get")
		cancel()
		_ = c.Close()
		if err != nil {
			t.Fatalf("get device info on iteration %d: %v", i, err)
		}
		if info == nil || info.Sn == nil || *info.Sn != "device-a-sn" {
			t.Fatalf("expected device info response on iteration %d, got %+v", i, info)
		}
		if _, err := h.RunCLIUntilSuccess("peer", "ping", "--context", "device-a"); err != nil {
			t.Fatal(err)
		}
	}
}
