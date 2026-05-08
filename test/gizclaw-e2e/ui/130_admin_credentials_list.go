// User story: As an admin operator, I can browse seeded provider credentials
// and confirm the credential metadata shown by the Admin UI.
package ui_test

import (
	"net/url"
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
	}, {
		Name: "130-admin-credential-detail-cli",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/providers/credentials/" + url.PathEscape(SeedCredentialName))
			page.ExpectText(SeedCredentialName)
			page.ExpectText("Credential Body")
			page.ClickRole("tab", "CLI")
			page.ExpectText("Credential Resource Spec")
			page.ExpectText(`"kind": "Credential"`)
			page.ExpectText("gizclaw admin credentials --context <admin-cli-context> get '" + SeedCredentialName + "'")
			page.ExpectText("gizclaw admin --context <admin-cli-context> show Credential '" + SeedCredentialName + "'")
		},
	}}
}
