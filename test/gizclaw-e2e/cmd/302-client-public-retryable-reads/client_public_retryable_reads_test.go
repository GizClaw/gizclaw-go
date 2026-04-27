package clientpublicretryablereads_test

import (
	"context"
	"strings"
	"testing"
	"time"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestClientPublicRetryableReadsUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "302-client-public-retryable-reads")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("device-a").MustSucceed(t)
	h.RegisterContext("device-a", "device_default", "--sn", "device-a-sn").MustSucceed(t)

	for i := 0; i < 4; i++ {
		c := h.ConnectClientFromContext("device-a")
		api, err := c.GearServiceClient()
		if err != nil {
			_ = c.Close()
			t.Fatalf("create gear service client on iteration %d: %v", i, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		config, err := api.GetConfigWithResponse(ctx)
		cancel()
		_ = c.Close()
		if err != nil {
			t.Fatalf("get device config on iteration %d: %v", i, err)
		}
		if config.JSON200 == nil {
			t.Fatalf("expected public config response on iteration %d, got status %d: %s", i, config.StatusCode(), strings.TrimSpace(string(config.Body)))
		}
		if _, err := h.RunCLIUntilSuccess("ping", "--context", "device-a"); err != nil {
			t.Fatal(err)
		}
	}
}
