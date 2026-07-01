package tui

import (
	"testing"

	"github.com/textfuel/lazyjira/v2/pkg/config"
	"github.com/textfuel/lazyjira/v2/pkg/jira"
	"github.com/textfuel/lazyjira/v2/pkg/jira/jiratest"
	"github.com/textfuel/lazyjira/v2/pkg/tui/components"
)

// newManagedTabApp builds an app focused on a managed tab in project ABC.
func newManagedTabApp(t *testing.T, jql string) *App {
	t.Helper()
	app := newAppWithFake(t, &jiratest.FakeClient{T: t})
	app.demoMode = true
	app.keymap = DefaultKeymap()
	app.projectKey = testProjectABC
	app.issuesList.SetTabs([]config.IssueTabConfig{{Name: "All", JQL: "x"}})
	app.savedTabs = []config.ManagedTab{{Name: testManagedName, JQL: jql, Project: testProjectABC}}
	app.issuesList.SetSavedTabs(app.savedTabs)
	app.issuesList.RebuildTabs(testProjectABC)
	app.issuesList.SetTabIndex(0) // managed Bugs
	return app
}

func TestEditManagedTab_updatesQueryInPlace(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())
	app := newManagedTabApp(t, "old query")

	if _, _, ok := app.handleTabAction(ActJQLSearch); !ok {
		t.Fatal("ActJQLSearch not dispatched")
	}
	if app.editingManagedTab != 0 {
		t.Fatalf("editingManagedTab = %d, want 0", app.editingManagedTab)
	}
	if got := app.jqlModal.InputValue(); got != "old query" {
		t.Errorf("prefill = %q, want %q", got, "old query")
	}

	app.handleJQLSearchResult(jqlSearchResultMsg{
		jql:    "new query",
		issues: []jira.Issue{{Key: "ABC-9", Summary: "s"}},
	})

	if app.savedTabs[0].JQL != "new query" {
		t.Errorf("store JQL = %q, want new query", app.savedTabs[0].JQL)
	}
	saved := config.LoadSavedTabs()
	if len(saved) != 1 || saved[0].JQL != "new query" {
		t.Errorf("persisted = %+v, want one entry with new query", saved)
	}
	if app.issuesList.IsJQLTab() {
		t.Error("no transient JQL tab should be created on in-place edit")
	}
	if !app.issuesList.IsManagedTab() || app.issuesList.ActiveTab().Name != testManagedName {
		t.Errorf("active tab should remain managed %q, got managed=%v name=%q",
			testManagedName, app.issuesList.IsManagedTab(), app.issuesList.ActiveTab().Name)
	}
	if app.editingManagedTab != -1 {
		t.Errorf("editingManagedTab should reset to -1, got %d", app.editingManagedTab)
	}
}

func TestEditManagedTab_multilineQueryPreserved(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())
	want := "project = ABC\nAND status = Open\nORDER BY updated DESC"
	app := newManagedTabApp(t, want)

	if _, _, ok := app.handleTabAction(ActJQLSearch); !ok {
		t.Fatal("ActJQLSearch not dispatched")
	}

	if got := app.jqlModal.InputValue(); got != want {
		t.Errorf("multi-line query must prefill with newlines intact: got %q, want %q", got, want)
	}
}

func TestPromote_multilineConfigQueryStoredMultiline(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())
	app := newAppWithFake(t, &jiratest.FakeClient{T: t})
	app.demoMode = true
	app.keymap = DefaultKeymap()
	app.projectKey = testProjectABC
	want := "project = ABC\nAND status = Open\nORDER BY updated DESC"
	app.issuesList.SetTabs([]config.IssueTabConfig{
		{Name: "Multi", JQL: want},
	})
	app.issuesList.RebuildTabs(testProjectABC)
	app.issuesList.SetTabIndex(0) // config Multi

	app.handleTabAction(ActManageTab) // promote
	app.handleInputConfirmed(components.InputConfirmedMsg{Text: "Multi"})

	if len(app.savedTabs) != 1 {
		t.Fatalf("want 1 saved tab, got %+v", app.savedTabs)
	}
	if app.savedTabs[0].JQL != want {
		t.Errorf("stored JQL must keep newlines:\n got %q\nwant %q", app.savedTabs[0].JQL, want)
	}
}

func TestEditManagedTab_onlyManagedTabs(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())
	app := newManagedTabApp(t, "managed")
	app.issuesList.SetTabIndex(app.issuesList.HomeTabIndex()) // config "All"
	if app.issuesList.IsManagedTab() {
		t.Fatal("setup: expected a config tab, not managed")
	}

	if _, _, ok := app.handleTabAction(ActJQLSearch); !ok {
		t.Fatal("ActJQLSearch not dispatched")
	}
	if app.editingManagedTab != -1 {
		t.Errorf("config tab must not target the store, editingManagedTab = %d", app.editingManagedTab)
	}
}

func TestJQLSaveTab_fromModal_savesManagedTab(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())
	app := newAppWithFake(t, &jiratest.FakeClient{T: t})
	app.demoMode = true
	app.projectKey = testProjectABC
	app.issuesList.SetTabs([]config.IssueTabConfig{{Name: "All", JQL: "x"}})
	app.issuesList.SetSavedTabs(nil)
	app.savedTabs = nil
	app.issuesList.RebuildTabs(testProjectABC)
	app.jqlModal.Show("project = X", nil)

	app.Update(components.JQLSaveTabMsg{Query: "project = X"})
	if app.jqlModal.IsVisible() {
		t.Error("JQL modal should hide before the name prompt")
	}
	app.handleInputConfirmed(components.InputConfirmedMsg{Text: "Saved Q"})

	saved := config.LoadSavedTabs()
	if len(saved) != 1 || saved[0].Name != "Saved Q" || saved[0].JQL != "project = X" {
		t.Errorf("persisted = %+v, want one Saved Q / project = X", saved)
	}
}

func TestHelpBar_jqlModal_hasSaveTabHint(t *testing.T) {
	t.Parallel()
	app := appForKeybindings(t)
	app.jqlModal.Show("", nil)

	found := false
	for _, it := range app.helpBarItems() {
		if it.Description == "save tab" {
			found = true
		}
	}
	if !found {
		t.Errorf("jqlModal help bar must include a 'save tab' hint, got %+v", app.helpBarItems())
	}
}
