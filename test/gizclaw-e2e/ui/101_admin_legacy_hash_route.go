// User story: As an admin operator with an old bookmarked hash route, I am
// redirected to the current firmware route without losing seeded depot data.
package ui_test

import (
	"testing"
)

func adminLegacyHashRouteStories() []Story {
	return []Story{{
		Name: "101-admin-legacy-hash-route",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/#/firmware")
			page.ExpectURLSuffix("/firmware")
			page.ExpectText("Depots")
			page.ExpectText(SeedDepotName)
		},
	}}
}
