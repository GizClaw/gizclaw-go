//go:build gizclaw_e2e

package connect_test

import (
	"sync"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestMultiClientConcurrentPingUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "400-multi-client-concurrent-ping")

	contexts := []string{"alpha", "beta", "gamma"}
	for _, name := range contexts {
		h.CreateContext(name).MustSucceed(t)
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(contexts))
	for _, name := range contexts {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			_, err := h.RunCLIUntilSuccess("connect", "ping", "--context", name)
			errs <- err
		}(name)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	if _, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "alpha"); err != nil {
		t.Fatal(err)
	}
}
