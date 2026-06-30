//go:build gizclaw_e2e

package smoke_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestRealServiceSmokeStories(t *testing.T) {
	RunSmokeStories(t, realServiceSmokeStories())
}
