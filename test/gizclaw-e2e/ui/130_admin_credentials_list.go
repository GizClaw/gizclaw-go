// User story: As an admin operator, I can browse seeded provider credentials
// and confirm the credential metadata shown by the Admin UI.
package ui_test

import (
	"testing"
)

func adminCredentialsListStories() []Story {
	return []Story{{
		Name: "130-admin-credentials-list",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/providers/credentials")
			page.ExpectText("Credentials")
			page.ExpectText(SeedCredentialName)
			page.ExpectText("minimax")
			page.ExpectText("api_key")
			page.ExpectText("Refresh")
		},
	}}
}
