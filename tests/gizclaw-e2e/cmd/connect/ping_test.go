//go:build gizclaw_e2e

package connect_test

import (
	"sync"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestPingUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "000-ping")

	h.CreateContext("client-a").MustSucceed(t)
	h.CreateContext("client-b").MustSucceed(t)
	h.WaitForPing("client-a")
	h.WaitForPing("client-b")

	t.Run("single client can ping repeatedly", func(t *testing.T) {
		for range 3 {
			result, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "client-a")
			if err != nil {
				t.Fatal(err)
			}
			assertPingOutput(t, result.Stdout)
		}
	})

	t.Run("multiple clients can ping concurrently", func(t *testing.T) {
		contexts := []string{"client-a", "client-b"}
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
	})
}
