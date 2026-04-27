// User story: As an admin operator, I can open a firmware depot and inspect
// its seeded metadata, operations, snapshot, and channels.
package ui_test

import (
	"testing"
)

func adminDepotDetailStories() []Story {
	return []Story{{
		Name: "122-admin-depot-detail",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/firmware/" + SeedDepotName)
			page.ExpectText(SeedDepotName)
			page.ExpectText("Depot operations")
			page.ExpectText("Update depot info")
			page.ExpectText("Snapshot")
			page.ExpectText("Testing And Rollback Snapshot")
			page.ExpectText("Stable")
			page.ExpectText("1.0.0")
		},
	}}
}
