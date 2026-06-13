// User story: As a Play UI user, I can see a resource loading error when the
// local proxy cannot reach a GizClaw client.
package ui_test

import (
	"testing"
)

func playActionErrorsStories() []Story {
	return []Story{{
		Name: "203-play-action-errors",
		Run: func(_ testing.TB, page *Page) {
			page.GotoErrorPlay("/")
			page.ExpectText("OpenAI Gateway")
			page.ExpectText("no gizclaw client configured for error scenario")
		},
	}}
}
