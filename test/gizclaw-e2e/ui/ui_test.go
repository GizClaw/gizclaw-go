package ui_test

import "testing"

func TestUIStories(t *testing.T) {
	RunStories(t, allStories())
}

func allStories() []Story {
	var stories []Story
	for _, group := range [][]Story{
		adminDashboardStories(),
		adminLegacyHashRouteStories(),
		adminPeersListStories(),
		adminPeerDetailStories(),
		adminPeerActionsStories(),
		adminFirmwaresListStories(),
		adminCredentialsListStories(),
		adminMiniMaxTenantsListStories(),
		adminVolcTenantsListStories(),
		adminProviderTenantsListStories(),
		adminVoicesListStories(),
		adminModelsListStories(),
		adminWorkflowsListStories(),
		adminWorkspacesListStories(),
		adminACLStories(),
		adminSidebarNavigationStories(),
		playShellStories(),
		playActionsStories(),
		playAllActionsStories(),
		playActionErrorsStories(),
		realServiceSmokeStories(),
	} {
		stories = append(stories, group...)
	}
	return stories
}
