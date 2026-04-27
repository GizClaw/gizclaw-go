// User story: As an admin operator, I can open a firmware channel, upload a
// release tarball, and see invalid channel routes rejected.
package ui_test

import (
	"fmt"
	"testing"
)

func adminChannelDetailStories() []Story {
	return []Story{
		{
			Name: "124-admin-channel-detail",
			Run: func(t testing.TB, page *Page) {
				page.GotoAdmin("/firmware/" + SeedDepotName + "/stable")
				page.ExpectText(SeedDepotName + " / stable")
				page.ExpectText("Current release")
				page.ExpectText("firmware.bin")
				page.ExpectText("firmware_semver")
				page.SetInputFiles(0, "release.tar", "application/octet-stream", FirmwareReleaseTar(t, "stable", "1.0.0"))
				page.ClickRole("button", "Upload release")
				page.ExpectText(fmt.Sprintf("Uploaded release.tar to %s/stable.", SeedDepotName))
			},
		},
		{
			Name: "124-admin-channel-invalid-route",
			Run: func(_ testing.TB, page *Page) {
				page.GotoAdmin("/firmware/" + SeedDepotName + "/rollback")
				page.ExpectText("Invalid channel")
				page.ExpectText("Supported channels are stable, beta, and testing.")
			},
		},
	}
}
