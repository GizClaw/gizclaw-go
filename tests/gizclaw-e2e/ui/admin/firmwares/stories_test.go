//go:build gizclaw_e2e

package adminui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestFirmwareStories(t *testing.T) {
	RunAdminStories(t, adminFirmwaresListStories())
}
