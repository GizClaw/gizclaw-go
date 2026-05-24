// User story: As a maintainer, I can run one smoke path that proves Admin and
// Play both serve against the same seeded test service.
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

			page.GotoPlay("/")
			page.ClickRole("button", "Start Video Call")
			page.ClickRole("button", "Logs")
			page.ExpectText("rpc.response")
			page.ExpectText("peer.ping")
			page.ClickRole("button", "Close RPC logs")
			page.ClickRole("button", "Get Info")
			page.ClickRole("button", "Logs")
			page.ExpectText("Seeded UI Device")
		},
	}}
}
