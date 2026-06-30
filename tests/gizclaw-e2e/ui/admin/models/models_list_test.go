//go:build gizclaw_e2e

// User story: As an admin operator, I can inspect model resource entries and
// copy the matching resource commands from the detail page.
package adminui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"net/url"
	"testing"
)

func adminModelsListStories() []Story {
	return []Story{{
		Name: "143-admin-models-list-and-detail-cli",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/ai/models")
			page.ExpectText("Models")
			page.ExpectText("Fake OpenAI chat model fake-openai-chat-000")
			page.ExpectText("openai-tenant")

			page.GotoAdmin("/ai/models/" + url.PathEscape(SeedModelID))
			page.ExpectText(SeedModelID)
			page.ExpectText("Fake OpenAI chat model fake-openai-chat-000")
			page.ClickRole("tab", "CLI")
			page.ExpectText("Model Resource Spec")
			page.ExpectText("gizclaw admin models --context <admin-cli-context> get '" + SeedModelID + "'")
		},
	}}
}
