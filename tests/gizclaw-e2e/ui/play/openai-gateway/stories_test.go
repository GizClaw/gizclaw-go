//go:build gizclaw_e2e

package playui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestOpenAIGatewayStories(t *testing.T) {
	stories := append(playShellStories(), playActionsStories()...)
	stories = append(stories, playAllActionsStories()...)
	stories = append(stories, playActionErrorsStories()...)
	RunPlayStories(t, stories)
}
