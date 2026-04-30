// User story: As a Play UI user, I can invoke server and device info actions
// through the real local proxy and see their successful responses.
package ui_test

import (
	"testing"
)

func playActionsStories() []Story {
	return []Story{{
		Name: "201-play-actions",
		Run: func(_ testing.TB, page *Page) {
			page.GotoPlay("/")
			page.ClickRole("button", "Run Server Info")
			page.ExpectText("Server Info loaded successfully.")
			page.ExpectText("build_commit")

			page.ClickRole("button", "Run Device Info")
			page.ExpectText("Device Info loaded successfully.")
			page.ExpectText("Seeded UI Device")
		},
	}}
}
