package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/textfuel/lazyjira/v2/pkg/jira"
	"github.com/textfuel/lazyjira/v2/pkg/jira/jiratest"
	"github.com/textfuel/lazyjira/v2/pkg/tui/components"
)

// newFocusedTestApp returns a test App with a fake Jira client, the default
// keymap installed (so handleKeyMsg-level tests can resolve actions), and
// the left side focused on the given panel -- the setup nearly every search/
// filter test in this file needs.
func newFocusedTestApp(t *testing.T, focus focusPanel) *App {
	t.Helper()
	app := newAppWithFake(t, &jiratest.FakeClient{T: t})
	app.keymap = DefaultKeymap()
	app.side = sideLeft
	app.leftFocus = focus
	return app
}

func TestHandleSearchChanged_FiltersFocusedPanel(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusIssues)
	app.issuesList.SetIssues([]jira.Issue{{Key: testKey, Summary: "alpha"}, {Key: "PLAT-2", Summary: "beta"}})

	_, _ = app.handleSearchChanged(components.SearchChangedMsg{Query: "alpha"})

	if sel := app.issuesList.SelectedIssue(); sel == nil || sel.Key != testKey {
		t.Errorf("filtered selection = %v, want %s", sel, testKey)
	}
}

func TestHandleSearchConfirmed_IssuesSelectsAndFetches(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusIssues)
	app.issuesList.SetIssues([]jira.Issue{{Key: testKey}})

	_, cmd := app.handleSearchConfirmed()

	if cmd == nil {
		t.Error("expected a fetch command for the selected issue")
	}
}

func TestHandleSearchConfirmed_UpdatesPreviewToLandedIssue(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusIssues)
	app.issuesList.SetIssues([]jira.Issue{{Key: testKey, Summary: "alpha"}, {Key: "PLAT-2", Summary: "beta"}})
	app.previewKey = testKey

	app.issuesList.SetFilter("beta")
	_, _ = app.handleSearchConfirmed()

	if selected := app.issuesList.SelectedIssue(); selected == nil || selected.Key != "PLAT-2" {
		t.Fatalf("cursor landed on %v, want PLAT-2", selected)
	}
	if current := app.currentIssue(); current == nil || current.Key != "PLAT-2" {
		t.Errorf("currentIssue() = %v, want PLAT-2 so actions target the searched ticket", current)
	}
}

func TestHandleSearchConfirmed_IssuesKeepsFilterActive(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusIssues)
	app.issuesList.SetIssues([]jira.Issue{{Key: testKey, Summary: "alpha"}, {Key: "PLAT-2", Summary: "beta"}})

	app.issuesList.SetFilter("beta")
	_, _ = app.handleSearchConfirmed()

	if got := app.issuesList.ItemCount(); got != 1 {
		t.Errorf("ItemCount() = %d after confirming, want 1 (filter should stay active, not reveal all issues)", got)
	}
}

func TestHandleSearchConfirmed_ProjectsKeepsFilterActiveWithoutSwitching(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusProjects)
	app.projectList.SetProjects([]jira.Project{{Key: testProject, ID: "1"}, {Key: "OTHER", ID: "2"}})
	app.projectList.SetFilter(testProject)

	_, _ = app.handleSearchConfirmed()

	if got := app.projectList.ItemCount(); got != 1 {
		t.Errorf("ItemCount() = %d after confirming, want 1 (filter should stay active)", got)
	}
	if app.projectKey == testProject {
		t.Errorf("projectKey = %q, confirming the filter should not switch the project -- that's a separate open action", app.projectKey)
	}
}

func TestHandleKeyMsg_OpenAfterSearchConfirmSelectsProject(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusProjects)
	app.projectList.SetProjects([]jira.Project{{Key: testProject, ID: "1"}, {Key: "OTHER", ID: "2"}})
	app.projectList.SetFilter(testProject)
	_, _ = app.handleSearchConfirmed()

	_, _ = app.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})

	if app.projectKey != testProject {
		t.Errorf("projectKey = %q, want %s after a normal Enter following filter-confirm", app.projectKey, testProject)
	}
}

func TestHandleSearchConfirmed_InfoKeepsFilterActive(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusInfo)
	app.infoPanel.SetFilter("beta")

	_, _ = app.handleSearchConfirmed()

	if !app.infoPanel.HasFilter() {
		t.Error("infoPanel filter should stay active after confirming (persistent filter)")
	}
}

func TestHandleKeyMsg_EscClearsActiveFilterInsteadOfChangingFocus(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusIssues)
	app.issuesList.SetIssues([]jira.Issue{{Key: testKey, Summary: "alpha"}, {Key: "PLAT-2", Summary: "beta"}})
	app.issuesList.SetFilter("beta")

	_, _ = app.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})

	if got := app.issuesList.ItemCount(); got != 2 {
		t.Errorf("ItemCount() = %d after Esc, want 2 (Esc should clear the filter)", got)
	}
	if app.leftFocus != focusIssues {
		t.Errorf("leftFocus = %v, want focusIssues unchanged (Esc should clear the filter, not move focus)", app.leftFocus)
	}
}

func TestHandleKeyMsg_SearchActivateResetsPreviousFilter(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusIssues)
	app.issuesList.SetIssues([]jira.Issue{{Key: testKey, Summary: "alpha"}, {Key: "PLAT-2", Summary: "beta"}})
	app.issuesList.SetFilter("beta")

	_, _ = app.handleKeyMsg(runeKey('/'))

	if got := app.issuesList.ItemCount(); got != 2 {
		t.Errorf("ItemCount() = %d after reopening search, want 2 (starting the filter again resets it, lazygit-style)", got)
	}
	if !app.searchBar.IsActive() {
		t.Error("search bar should be active after pressing /")
	}
	if q := app.searchBar.Query(); q != "" {
		t.Errorf("search bar query = %q, want empty on fresh activation", q)
	}
}

func TestHandleSearchCancelled_ClearsFilters(t *testing.T) {
	t.Parallel()
	app := newFocusedTestApp(t, focusIssues)
	app.issuesList.SetIssues([]jira.Issue{{Key: testKey, Summary: "alpha"}, {Key: "PLAT-2", Summary: "beta"}})
	app.issuesList.SetFilter("alpha")

	_, cmd := app.handleSearchCancelled()

	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleAutoFetch_SchedulesNextTick(t *testing.T) {
	t.Parallel()
	app := newAppWithFake(t, &jiratest.FakeClient{T: t})

	_, cmd := app.handleAutoFetch()

	if cmd == nil {
		t.Error("auto fetch should always schedule the next tick")
	}
}

func TestRouteToPanel_ForwardsToFocusedPanel(t *testing.T) {
	t.Parallel()

	t.Run("left issues panel receives input", func(t *testing.T) {
		t.Parallel()
		app := newFocusedTestApp(t, focusIssues)
		app.issuesList.ResolveNav = app.keymap.MatchNav
		app.issuesList.SetFocused(true)
		app.issuesList.SetIssues([]jira.Issue{{Key: testKey}, {Key: "PLAT-2"}})

		_ = app.routeToPanel(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

		if sel := app.issuesList.SelectedIssue(); sel == nil || sel.Key != "PLAT-2" {
			t.Errorf("expected cursor to move down to PLAT-2, got %v", sel)
		}
	})

	t.Run("right detail panel receives input", func(t *testing.T) {
		t.Parallel()
		app := newAppWithFake(t, &jiratest.FakeClient{T: t})
		app.side = sideRight

		app.routeToPanel(tea.KeyMsg{Type: tea.KeyDown})

		if app.side != sideRight {
			t.Errorf("side = %v, want right after routing to detail panel", app.side)
		}
	})
}
