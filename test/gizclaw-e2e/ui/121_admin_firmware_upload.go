// User story: As an admin operator, I can upload a firmware release and apply
// depot metadata through the real Admin UI proxy.
package ui_test

import (
	"fmt"
	"testing"
)

func adminFirmwareUploadStories() []Story {
	return []Story{{
		Name: "121-admin-firmware-upload",
		Run: func(t testing.TB, page *Page) {
			page.GotoAdmin("/firmware/new")
			page.ExpectText("Add firmware")
			page.ExpectText("Upload firmware")
			page.ExpectText("Update depot info")

			page.FillNth(`input[placeholder="demo-main"]`, 0, SeedDepotName)
			page.SetInputFiles(0, "release.tar", "application/octet-stream", FirmwareReleaseTar(t, "stable", "1.0.0"))
			page.ClickRole("button", "Upload Release")
			page.ExpectText(fmt.Sprintf("Uploaded release.tar to %s/stable.", SeedDepotName))

			page.FillNth(`input[placeholder="demo-main"]`, 1, SeedDepotName)
			page.SetInputFiles(1, "info.json", "application/json", DepotInfoJSON(t))
			page.ClickRole("button", "Apply Manifest")
			page.ExpectText(fmt.Sprintf("Updated metadata for depot %s.", SeedDepotName))
		},
	}}
}
