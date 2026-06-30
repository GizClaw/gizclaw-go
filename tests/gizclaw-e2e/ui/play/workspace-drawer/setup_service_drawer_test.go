//go:build gizclaw_e2e

package playui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestSetupServicePlayWorkspaceDrawer(t *testing.T) {
	RunPlayStories(t, []Story{{
		Name: "setup-service-play-workspace-drawer",
		Run: func(t testing.TB, page *Page) {
			ensurePlayWorkspace(t, page, SeedWorkspaceName)
			page.GotoPlay("/")
			page.ExpectText("OpenAI Gateway")
			page.ClickRole("button", "Workspace")
			page.ExpectText("Conversation")
			page.ExpectText("Push to talk")
			page.ExpectText("History")
			page.ExpectText("Memory")
			page.ExpectText("Recall")
			page.ExpectText("flowcraft")
		},
	}})
}
