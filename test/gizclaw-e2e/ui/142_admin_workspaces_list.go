// User story: As an admin operator, I can browse seeded workspaces and verify
// their associated workflow.
package ui_test

import (
	"testing"
)

func adminWorkspacesListStories() []Story {
	return []Story{{
		Name: "142-admin-workspaces-list",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/ai/workspaces")
			page.ExpectText("Workspaces")
			page.ExpectText(SeedWorkspaceName)
			page.ExpectText(SeedWorkflowName)
			page.ExpectText("Refresh")
		},
	}}
}
