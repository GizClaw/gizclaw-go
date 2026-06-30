//go:build gizclaw_e2e

// User story: As an admin operator, I can browse shared AI voices and confirm
// the voice provider, tenant, and capability metadata.
package adminui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
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
			page.ExpectText("MiniMax Narrator Clone")
			page.ExpectText("manual")
			page.ExpectText(SeedMiniMaxTenantName)
		},
	}, {
		Name: "140-admin-volc-voice-detail-cli",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/ai/voices/" + url.PathEscape(SeedVolcVoiceID))
			page.ExpectText("Vivi")
			page.ClickRole("tab", "CLI")
			page.ExpectText("Voice Resource Spec")
			page.ExpectText(`"kind": "Voice"`)
			page.ExpectText(`"name": "volc-tenant:volc-main:zh_female_vv_mars_bigtts"`)
			page.ExpectText("gizclaw admin --context <admin-cli-context> show Voice 'volc-tenant:volc-main:zh_female_vv_mars_bigtts'")
			page.ExpectText("gizclaw admin --context <admin-cli-context> show VolcTenant 'volc-main'")
			page.ExpectText("gizclaw admin volc-tenants --context <admin-cli-context> sync-voices 'volc-main'")
		},
	}}
}
