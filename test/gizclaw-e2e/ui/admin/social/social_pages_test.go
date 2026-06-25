//go:build gizclaw_e2e

// User story: As an admin operator, I can manage social Friend and Friend
// Group resources from top-level Admin UI pages.
package adminui_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"

	. "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/ui/internal/harness"
)

func adminSocialStories() []Story {
	return []Story{
		{
			Name: "160-admin-social-sidebar-links",
			Run: func(_ testing.TB, page *Page) {
				page.GotoAdmin("/overview")
				page.ClickNavigationLink("Friends")
				page.ExpectURLSuffix("/social/friends")
				page.ExpectText("Friends")
				page.ExpectText("Create Friend")

				page.ClickNavigationLink("Friend Groups")
				page.ExpectURLSuffix("/social/friend-groups")
				page.ExpectText("Friend Groups")
				page.ExpectText("Create Friend Group")
			},
		},
		{
			Name: "161-admin-social-friends-list-and-detail",
			Run: func(t testing.TB, page *Page) {
				suffix := time.Now().UnixNano()
				owner := fmt.Sprintf("000-ui-social-owner-%d", suffix)
				peer := fmt.Sprintf("000-ui-social-peer-%d", suffix)

				page.GotoAdmin("/social/friends")
				page.Fill(`input[placeholder="Owner peer public key"]`, owner)
				page.Fill(`input[placeholder="Friend peer public key"]`, peer)
				page.ClickRole("button", "Create")
				page.ExpectText(peer)
				page.ExpectText("Workspace History")
				page.ExpectText("No history entries")

				page.GotoAdmin("/social/friends")
				page.ExpectText(peer)
				clickFriendOpenLink(t, page, owner, peer)
				page.ExpectText(owner)
				page.ExpectText(peer)
				page.ExpectText("Friend Row")
			},
		},
		{
			Name: "162-admin-social-friend-groups-detail",
			Run: func(t testing.TB, page *Page) {
				suffix := time.Now().UnixNano()
				groupName := fmt.Sprintf("ui-social-group-%d", suffix)
				member := fmt.Sprintf("000-ui-social-member-%d", suffix)
				token := fmt.Sprintf("ui-social-token-%d", suffix)

				page.GotoAdmin("/social/friend-groups")
				page.Fill(`input[placeholder="story-club"]`, groupName)
				page.Fill(`textarea[placeholder="Group description"]`, "Created from Admin UI social e2e")
				page.ClickRole("button", "Create")
				page.ExpectText(groupName)
				page.ExpectText("Info")
				page.ExpectText("Members")
				page.ExpectText("Invite Token")
				page.ExpectText("History")

				page.ClickRole("tab", "Members")
				page.Fill(`input[placeholder="Peer public key"]`, member)
				page.ClickRole("button", "Add")
				page.ExpectText("Member added.")
				page.ExpectText(member)

				clickFirstTableCombobox(t, page)
				if err := page.Raw().GetByRole(playwright.AriaRole("option"), playwright.PageGetByRoleOptions{
					Name:  "admin",
					Exact: playwright.Bool(true),
				}).Click(); err != nil {
					t.Fatalf("select admin role: %v", err)
				}
				page.ExpectText("Member role updated.")

				page.ClickRole("button", "Remove")
				page.ClickRole("button", "Delete")
				page.ExpectText("Member removed.")

				page.ClickRole("tab", "Invite Token")
				page.Fill(`input[placeholder="Invite token"]`, token)
				page.ClickRole("button", "Save Token")
				page.ExpectText("Invite token saved.")
				expectInputValue(t, page, `input[placeholder="Invite token"]`, token)
				page.ClickRole("button", "Clear")
				page.ClickRole("button", "Delete")
				page.ExpectText("Invite token cleared.")

				page.ClickRole("tab", "History")
				page.ExpectText("Workspace History")
				page.ExpectText("No history entries")
			},
		},
	}
}

func clickFirstTableCombobox(t testing.TB, page *Page) {
	t.Helper()
	if err := page.Raw().Locator(`table [role="combobox"]`).First().Click(); err != nil {
		t.Fatalf("click first member role combobox: %v", err)
	}
}

func clickFriendOpenLink(t testing.TB, page *Page, owner, peer string) {
	t.Helper()
	relationID := fmt.Sprintf("%s:%s", owner, peer)
	href := fmt.Sprintf("/social/friends/%s/%s", url.QueryEscape(owner), url.QueryEscape(relationID))
	if err := page.Raw().Locator(fmt.Sprintf(`a[href="%s"]`, href)).Click(); err != nil {
		t.Fatalf("click friend open link %q: %v", href, err)
	}
}

func expectInputValue(t testing.TB, page *Page, selector, want string) {
	t.Helper()
	if err := WaitUntil(10*time.Second, func() error {
		got, err := page.Raw().Locator(selector).InputValue()
		if err != nil {
			return err
		}
		if got == want {
			return nil
		}
		return fmt.Errorf("%s value = %q, want %q", selector, got, want)
	}); err != nil {
		t.Fatal(err)
	}
}
