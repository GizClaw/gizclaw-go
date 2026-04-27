// User story: As an admin operator, I can list registered devices and filter
// the current inventory page using real device registrations.
package ui_test

import (
	"testing"
)

func adminDevicesListStories() []Story {
	return []Story{{
		Name: "110-admin-devices-list",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/devices")
			page.ExpectText("Devices")
			page.ExpectText("Device Inventory")
			page.ExpectText(page.Seed.DevicePublicKey)
			page.ExpectText("device")
			page.ExpectText("Active")

			page.Fill(`input[placeholder="Filter current page by key, role, or status"]`, "missing")
			page.ExpectText("No matching devices")
			page.Fill(`input[placeholder="Filter current page by key, role, or status"]`, page.Seed.DevicePublicKey[:12])
			page.ExpectText(page.Seed.DevicePublicKey)
		},
	}}
}
