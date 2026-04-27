// User story: As a Play UI user, I can see useful feedback when a proxied gear
// action fails because no GizClaw client is available.
package ui_test

import (
	"testing"
)

func playActionErrorsStories() []Story {
	return []Story{{
		Name: "203-play-action-errors",
		Run: func(_ testing.TB, page *Page) {
			page.GotoErrorPlay("/")
			page.ClickRole("button", "Run Configuration")
			page.ExpectText("no gizclaw client configured for error scenario")
			page.ExpectText("No response available")
		},
	}}
}
