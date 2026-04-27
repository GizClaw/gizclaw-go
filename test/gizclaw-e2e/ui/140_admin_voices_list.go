// User story: As an admin operator, I can browse seeded AI voices and confirm
// the voice provider, tenant, and capability metadata.
package ui_test

import (
	"testing"
)

func adminVoicesListStories() []Story {
	return []Story{{
		Name: "140-admin-voices-list",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/ai/voices")
			page.ExpectText("Voices")
			page.ExpectText(SeedVoiceID)
			page.ExpectText("Seeded UI Voice")
			page.ExpectText("manual")
			page.ExpectText(SeedMiniMaxTenantName)
			page.ExpectText("provider-ui-seed-voice")
			page.ExpectText("voice_cloning")
		},
	}}
}
