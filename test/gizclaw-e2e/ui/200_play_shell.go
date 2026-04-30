// User story: As a Play UI user, I can open the shell and see the available
// real-service actions for server info, device info, config, and OTA.
package ui_test

import (
	"testing"
)

func playShellStories() []Story {
	return []Story{{
		Name: "200-play-shell",
		Run: func(_ testing.TB, page *Page) {
			page.GotoPlay("/")
			page.ExpectText("GizClaw Play")
			page.ExpectText("Current State")
			page.ExpectText("Run Server Info")
			page.ExpectText("Run Device Info")
		},
	}}
}
