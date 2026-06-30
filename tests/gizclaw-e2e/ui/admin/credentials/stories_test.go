//go:build gizclaw_e2e

package adminui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestCredentialStories(t *testing.T) {
	RunAdminStories(t, adminCredentialsListStories())
}
