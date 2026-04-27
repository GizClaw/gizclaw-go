package clientpublicreadsequence_test

import (
	"context"
	"strings"
	"testing"
	"time"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestClientPublicReadSequenceUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "300-client-public-read-sequence")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("device-a").MustSucceed(t)
	h.RegisterContext(
		"device-a",
		"device_default",
		"--name", "device-a",
		"--sn", "device-a-sn",
		"--manufacturer", "Acme",
		"--model", "Model-A",
		"--depot", "demo",
		"--firmware-semver", "1.0.0",
	).MustSucceed(t)

	c := h.ConnectClientFromContext("device-a")
	defer c.Close()
	api, err := c.GearServiceClient()
	if err != nil {
		t.Fatalf("create gear service client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	config, err := api.GetConfigWithResponse(ctx)
	if err != nil {
		t.Fatalf("get device config: %v", err)
	}
	if config.JSON200 == nil {
		t.Fatalf("expected public config response, got status %d: %s", config.StatusCode(), strings.TrimSpace(string(config.Body)))
	}

	if _, err := h.RunCLIUntilSuccess("ping", "--context", "device-a"); err != nil {
		t.Fatal(err)
	}
}
