// User story: As an admin operator, I can browse seeded workspace templates and
// inspect the template kind, API version, and description.
package ui_test

import (
	"testing"
)

func adminWorkspaceTemplatesListStories() []Story {
	return []Story{{
		Name: "141-admin-workspace-templates-list",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/ai/workspace-templates")
			page.ExpectText("Workspace Templates")
			page.ExpectText(SeedWorkspaceTemplateName)
			page.ExpectText("SingleAgentGraphWorkflowTemplate")
			page.ExpectText("gizclaw.flowcraft/v1alpha1")
			page.ExpectText("Seeded workspace template for UI real-service tests")
		},
	}}
}
