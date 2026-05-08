// User story: As an admin operator, I can browse seeded MiniMax tenants and
// verify their credential and provider metadata.
package ui_test

import (
	"net/url"
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
	}, {
		Name: "131-admin-minimax-tenant-detail-cli",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/providers/minimax-tenants/" + url.PathEscape(SeedMiniMaxTenantName))
			page.ExpectText(SeedMiniMaxTenantName)
			page.ClickRole("tab", "CLI")
			page.ExpectText("MiniMaxTenant Resource Spec")
			page.ExpectText(`"kind": "MiniMaxTenant"`)
			page.ExpectText("gizclaw admin minimax-tenants --context <admin-cli-context> get '" + SeedMiniMaxTenantName + "'")
			page.ExpectText("gizclaw admin minimax-tenants --context <admin-cli-context> sync-voices '" + SeedMiniMaxTenantName + "'")
		},
	}}
}
