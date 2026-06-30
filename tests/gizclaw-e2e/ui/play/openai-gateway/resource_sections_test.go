//go:build gizclaw_e2e

// User story: As a Play UI user, I can switch between OpenAI gateway resource
// views.
package playui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func playActionsStories() []Story {
	return []Story{{
		Name: "201-play-actions",
		Run: func(_ testing.TB, page *Page) {
			page.GotoPlay("/")
			page.ClickRoleLike("button", "Models")
			page.ExpectText(SeedModelID)
			page.ClickRoleLike("button", "Voices")
			page.ExpectText(SeedVoiceID)
			page.ExpectText("MiniMax Narrator Clone")
			page.ClickRoleLike("button", "Pets")
			page.ExpectText("No pets")
			page.ClickRoleLike("button", "Transactions")
			page.ExpectText("No transactions")
			page.ClickRoleLike("button", "Rewards")
			page.ExpectText("No rewards")
			page.ClickRoleLike("button", "Models")
			page.ExpectText(SeedModelID)
		},
	}}
}
