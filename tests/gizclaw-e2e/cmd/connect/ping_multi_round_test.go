//go:build gizclaw_e2e

package connect_test

import (
	"sync"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestPingMultiRoundUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "002-ping-multi-round")

	contexts := []string{"client-a", "client-b", "client-c"}
	for _, name := range contexts {
		h.CreateContext(name).MustSucceed(t)
		h.WaitForPing(name)
	}

	for round := range 3 {
		t.Run("sequential-round-"+itoa(round), func(t *testing.T) {
			for _, name := range contexts {
				result, err := h.RunCLIUntilSuccess("connect", "ping", "--context", name)
				if err != nil {
					t.Fatal(err)
				}
				assertPingOutput(t, result.Stdout)
			}
		})

		t.Run("concurrent-round-"+itoa(round), func(t *testing.T) {
			runConcurrentPings(t, h, []string{"client-a", "client-b"})
		})
	}

	finalCheck, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "client-c")
	if err != nil {
		t.Fatal(err)
	}
	assertPingOutput(t, finalCheck.Stdout)
}

func runConcurrentPings(t *testing.T, h *clitest.Harness, contexts []string) {
	t.Helper()

	type outcome struct {
		result clitest.Result
		err    error
	}
	results := make(chan outcome, len(contexts))

	var wg sync.WaitGroup
	for _, ctxName := range contexts {
		ctxName := ctxName
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := h.RunCLIUntilSuccess("connect", "ping", "--context", ctxName)
			results <- outcome{result: result, err: err}
		}()
	}

	wg.Wait()
	close(results)

	for outcome := range results {
		if outcome.err != nil {
			t.Fatal(outcome.err)
		}
		assertPingOutput(t, outcome.result.Stdout)
	}
}
