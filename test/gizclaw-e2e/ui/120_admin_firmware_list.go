// User story: As an admin operator, I can browse firmware depots and see seeded
// channel releases from the real service.
package ui_test

import (
	"testing"
)

func adminFirmwareListStories() []Story {
	return []Story{{
		Name: "120-admin-firmware-list",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/firmware")
			page.ExpectText("Depots")
			page.ExpectText("Add firmware")
			page.ExpectText(SeedDepotName)
			page.ExpectText("1.0.0")
			page.ExpectText("1.0.1")
			page.ExpectText("Readiness")
		},
	}}
}
