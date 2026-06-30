//go:build gizclaw_e2e

package adminui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestSocialStories(t *testing.T) {
	RunAdminStories(t, adminSocialStories())
}
