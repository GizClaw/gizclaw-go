//go:build gizclaw_e2e

package playui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestWorkspaceDrawerStories(t *testing.T) {
	RunPlayStories(t, playWorkspaceDrawerStories())
}
