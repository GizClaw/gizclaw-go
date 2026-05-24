// User story: As an admin operator with an old bookmarked hash route, I am
// redirected to the current overview route.
package ui_test

import (
	"testing"
)

func adminLegacyHashRouteStories() []Story {
	return []Story{{
		Name: "101-admin-legacy-hash-route",
		Run: func(_ testing.TB, page *Page) {
			page.GotoAdmin("/#/overview")
			page.ExpectURLSuffix("/overview")
			page.ExpectText("Dashboard")
		},
	}}
}
