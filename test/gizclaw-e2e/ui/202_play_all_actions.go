// User story: As a Play UI user, I can run all exposed actions and see seeded
// server, device, configuration, and OTA data from the real service.
package ui_test

import (
	"testing"
)

func playAllActionsStories() []Story {
	return []Story{{
		Name: "202-play-all-actions",
		Run: func(_ testing.TB, page *Page) {
			page.GotoPlay("/")
			page.ClickRole("button", "Run Server Info")
			page.ExpectText("Server Info loaded successfully.")
			page.ExpectText("build_commit")

			page.ClickRole("button", "Run Device Info")
			page.ExpectText("Device Info loaded successfully.")
			page.ExpectText("Seeded UI Device")

			page.ClickRole("button", "Run Configuration")
			page.ExpectText("Configuration loaded successfully.")
			page.ExpectText("stable")

			page.ClickRole("button", "Run OTA Summary")
			page.ExpectText("OTA Summary loaded successfully.")
			page.ExpectText(SeedDepotName)
		},
	}}
}
