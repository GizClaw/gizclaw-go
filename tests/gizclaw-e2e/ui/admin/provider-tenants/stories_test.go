//go:build gizclaw_e2e

package adminui_test

import (
	. "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/ui/internal/harness"
	"testing"
)

func TestProviderTenantStories(t *testing.T) {
	stories := append(adminMiniMaxTenantsListStories(), adminVolcTenantsListStories()...)
	stories = append(stories, adminProviderTenantsListStories()...)
	RunAdminStories(t, stories)
}
