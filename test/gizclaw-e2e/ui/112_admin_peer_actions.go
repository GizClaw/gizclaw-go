// User story: As an admin operator, I can perform peer actions against real
// seeded registrations, including approve, refresh, block, and reset.
package ui_test

import (
	"net/url"
	"testing"
)

func adminPeerActionsStories() []Story {
	return []Story{
		{
			Name: "112-admin-peer-actions",
			Run: func(_ testing.TB, page *Page) {
				page.GotoAdmin("/peers/" + url.PathEscape(page.Seed.ActionDevicePublicKey))
				page.ClickRole("button", "Refresh Peer")
				page.ExpectText("Peer refreshed.")
				page.ClickRole("tab", "Edit")
				page.ClickRole("button", "Approve")
				page.ExpectText("Peer approved as gear.")
				page.ClickRole("tab", "Edit")
				page.ClickRole("button", "Block")
				page.ExpectText("Peer blocked.")
			},
		},
		{
			Name: "112-admin-peer-delete",
			Run: func(_ testing.TB, page *Page) {
				page.GotoAdmin("/peers/" + url.PathEscape(page.Seed.DeleteDevicePublicKey))
				page.ClickRole("tab", "Edit")
				page.ClickRole("button", "Reset")
				page.ExpectURLSuffix("/peers")
				page.ExpectText("Peers")
			},
		},
	}
}
