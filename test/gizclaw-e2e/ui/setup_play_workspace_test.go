package ui_test

import (
	"os"
	"testing"
)

func TestSetupServicePlayWorkspaceDrawer(t *testing.T) {
	playURL := os.Getenv("GIZCLAW_E2E_PLAY_URL")
	if playURL == "" {
		t.Skip("set GIZCLAW_E2E_PLAY_URL to run against a setup-prepared gizclaw play service")
	}

	runner := newBrowserRunner(t)
	defer runner.close(t)
	ctx, err := runner.browser.NewContext()
	if err != nil {
		t.Fatalf("create browser context: %v", err)
	}
	defer func() {
		if err := ctx.Close(); err != nil {
			t.Fatalf("close browser context: %v", err)
		}
	}()
	page, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	defer page.Close()

	p := &Page{t: t, page: page, Seed: Seed{PlayURL: playURL}}
	p.GotoPlay("/")
	p.ExpectText("OpenAI Gateway")
	p.ClickRole("button", "Workspace")
	p.ExpectText("Realtime Chat")
	p.ExpectText("Push To Talk")
	p.ExpectText("History")
	p.ExpectText("Memory")
	p.ExpectText("Recall")
	p.ExpectText("flowcraft-voice")
}
