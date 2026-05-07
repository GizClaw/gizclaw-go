// User story: As an admin operator, I can browse seeded AI voices and confirm
// the voice provider, tenant, and capability metadata.
package ui_test

import (
	"net/url"
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
		},
	}, {
		Name: "140-admin-volc-voice-detail-cli",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/ai/voices/" + url.PathEscape(SeedVolcVoiceID))
			page.ExpectText("Seeded Volc UI Voice")
			page.ClickRole("tab", "CLI")
			page.ExpectText("Voice Resource Spec")
			page.ExpectText(`"kind": "Voice"`)
			page.ExpectText(`"name": "volc-tenant:ui-seed-volc-tenant:ICL_ui_seed_voice"`)
			page.ExpectText("gizclaw admin --context <admin-cli-context> show Voice 'volc-tenant:ui-seed-volc-tenant:ICL_ui_seed_voice'")
			page.ExpectText("gizclaw admin --context <admin-cli-context> show VolcTenant 'ui-seed-volc-tenant'")
			page.ExpectText("gizclaw admin volc-tenants --context <admin-cli-context> sync-voices 'ui-seed-volc-tenant'")
		},
	}}
}
