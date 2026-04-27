// User story: As an admin operator, I can perform device actions against real
// seeded registrations, including approve, refresh, channel save, block, and delete.
package ui_test

import (
	"net/url"
	"testing"
)

func adminDeviceActionsStories() []Story {
	return []Story{
		{
			Name: "112-admin-device-actions",
			Run: func(_ testing.TB, page *Page) {
				page.GotoAdmin("/devices/" + url.PathEscape(page.Seed.ActionDevicePublicKey))
				page.ClickRole("button", "Approve")
				page.ExpectText("Device approved as device.")
				page.ClickRole("button", "Refresh")
				page.ExpectText("Device refreshed.")
				page.ClickRole("button", "Save Channel")
				page.ExpectText("Desired channel updated to stable.")
				page.ClickRole("button", "Block")
				page.ExpectText("Device blocked.")
			},
		},
		{
			Name: "112-admin-device-delete",
			Run: func(_ testing.TB, page *Page) {
				page.GotoAdmin("/devices/" + url.PathEscape(page.Seed.DeleteDevicePublicKey))
				page.ClickRole("button", "Delete")
				page.ExpectURLSuffix("/devices")
				page.ExpectText("Devices")
			},
		},
	}
}
