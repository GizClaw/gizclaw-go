//go:build gizclaw_e2e

package adminui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestPeerStories(t *testing.T) {
	stories := append(adminPeersListStories(), adminPeerDetailStories()...)
	stories = append(stories, adminPeerActionsStories()...)
	RunAdminStories(t, stories)
}
