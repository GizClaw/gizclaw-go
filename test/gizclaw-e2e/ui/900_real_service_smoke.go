// User story: As a maintainer, I can run one smoke path that proves Admin and
// Play both talk to the same running GizClaw service and seeded data set.
package ui_test

import (
	"testing"
)

func realServiceSmokeStories() []Story {
	return []Story{{
		Name: "900-real-service-smoke",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/")
			page.ExpectText("Dashboard")
			page.ExpectText(SeedDepotName)

			page.GotoPlay("/")
			page.ClickRole("button", "Run Device Info")
			page.ExpectText("Seeded UI Device")
		},
	}}
}
