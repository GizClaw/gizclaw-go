// User story: As an admin operator, I can inspect a seeded peer across its
// info, edit, and CLI views.
package ui_test

import (
	"net/url"
	"testing"
)

func adminPeerDetailStories() []Story {
	return []Story{{
		Name: "111-admin-peer-detail",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/peers/" + url.PathEscape(page.Seed.DevicePublicKey))
			page.ExpectText("Seeded UI Device")
			page.ExpectText(page.Seed.DevicePublicKey)
			page.ExpectText("ui-device-sn")
			page.ExpectText("Configuration")
			page.ExpectText("ui-cert")
			page.ExpectText("Last Address")
			page.ExpectText("Online")
			page.ExpectText("firmware_semver")

			page.ClickRole("tab", "Edit")
			page.ExpectText("Peer Actions")
			page.ExpectText("Firmware Policy")

			page.ClickRole("tab", "CLI")
			page.ExpectText("PeerConfig Resource Spec")
			page.ExpectText("gizclaw admin peers")
		},
	}}
}
