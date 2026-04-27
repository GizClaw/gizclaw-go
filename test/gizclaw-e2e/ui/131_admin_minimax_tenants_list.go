// User story: As an admin operator, I can browse seeded MiniMax tenants and
// verify their credential and provider metadata.
package ui_test

import (
	"testing"
)

func adminMiniMaxTenantsListStories() []Story {
	return []Story{{
		Name: "131-admin-minimax-tenants-list",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/providers/minimax-tenants")
			page.ExpectText("MiniMax Tenants")
			page.ExpectText(SeedMiniMaxTenantName)
			page.ExpectText("ui-seed-app")
			page.ExpectText("ui-seed-group")
			page.ExpectText(SeedCredentialName)
			page.ExpectText("https://example.invalid")
		},
	}}
}
