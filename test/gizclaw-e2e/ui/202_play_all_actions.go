// User story: As a Play UI user, I can start a WebRTC call, open the RPC data
// channel, and run gear-service RPC commands through the local proxy.
package ui_test

import (
	"testing"
)

func playAllActionsStories() []Story {
	return []Story{{
		Name: "202-play-all-actions",
		Run: func(_ testing.TB, page *Page) {
			page.GotoPlay("/")
			page.ClickRole("button", "Start Video Call")
			page.ExpectText("Connected")
			page.ClickRole("button", "Logs")
			page.ExpectText("webrtc.state")
			page.ExpectText("peer.ping")
			page.ExpectText("rpc.response")
			page.ClickRole("button", "Close RPC logs")
			page.ClickRole("button", "Get Info")
			page.ClickRole("button", "Logs")
			page.ExpectText("gear.info.get")
			page.ExpectText("Seeded UI Device")
			page.ClickRole("button", "Close RPC logs")
			page.ClickRole("button", "Get Config")
			page.ClickRole("button", "Logs")
			page.ExpectText("gear.config.get")
			page.ExpectText("under-12")
			page.ClickRole("button", "Close RPC logs")
			page.ClickRole("button", "Get Runtime")
			page.ClickRole("button", "Logs")
			page.ExpectText("gear.runtime.get")
			page.ExpectText("online")
		},
	}}
}
