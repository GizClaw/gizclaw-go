// User story: As an admin operator, I can release, roll back, update depot
// metadata, and see validation feedback for invalid depot manifests.
package ui_test

import (
	"fmt"
	"testing"
)

func adminDepotActionsStories() []Story {
	return []Story{
		{
			Name: "123-admin-depot-actions",
			Run: func(t testing.TB, page *Page) {
				page.GotoAdmin("/firmware/" + SeedDepotName)
				page.ClickRole("button", "Release")
				page.ExpectText(fmt.Sprintf("Released depot %s.", SeedDepotName))
				page.ClickRole("button", "Rollback")
				page.ExpectText(fmt.Sprintf("Rolled back depot %s.", SeedDepotName))
				page.SetInputFiles(0, "info.json", "application/json", DepotInfoJSON(t))
				page.ClickRole("button", "Apply manifest")
				page.ExpectText(fmt.Sprintf("Updated metadata for depot %s.", SeedDepotName))
			},
		},
		{
			Name: "123-admin-depot-action-errors",
			Run: func(_ testing.TB, page *Page) {
				page.GotoAdmin("/firmware/" + SeedDepotName)
				page.SetInputFiles(0, "info.json", "application/json", []byte("not-json"))
				page.ClickRole("button", "Apply manifest")
				page.ExpectText("not valid JSON")
			},
		},
	}
}
