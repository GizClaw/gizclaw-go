//go:build gizclaw_e2e

package playui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestWorkspacesSection(t *testing.T) {
	RunPlayStories(t, []Story{{
		Name: "play-workspaces-section",
		Run: func(_ testing.TB, page *Page) {
			page.GotoPlay("/")
			page.ClickRoleLike("button", "Workspaces")
			page.ExpectText(SeedWorkspaceName)
		},
	}})
}
