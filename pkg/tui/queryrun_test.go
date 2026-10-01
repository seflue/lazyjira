package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/textfuel/lazyjira/v2/pkg/config"
	"github.com/textfuel/lazyjira/v2/pkg/internal/testkit"
	"github.com/textfuel/lazyjira/v2/pkg/jira"
	"github.com/textfuel/lazyjira/v2/pkg/jira/jiratest"
	"github.com/textfuel/lazyjira/v2/pkg/tui/components"
)

func searchFake(t *testing.T, issues ...jira.Issue) *jiratest.FakeClient {
	t.Helper()
	return &jiratest.FakeClient{T: t, SearchIssuesFunc: func(context.Context, string, int, int) (*jira.SearchResult, error) {
		return &jira.SearchResult{Issues: issues}, nil
	}}
}

// submitJQL submits query from the JQL modal and delivers the run's result.
func submitJQL(app *App, query string) {
	_, cmd := app.Update(components.JQLSubmitMsg{Query: query})
	app.Update(cmd())
}

func modalShows(app *App, text string) bool {
	app.jqlModal.SetSize(80, 24)
	return strings.Contains(app.jqlModal.View(), text)
}

func TestResolveQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   string
		vars    queryVars
		want    string
		wantErr bool
	}{
		{name: "project placeholder is quoted", query: "project = {{.ProjectKey}} ORDER BY updated DESC", vars: queryVars{project: testProject}, want: `project = "PLAT" ORDER BY updated DESC`},
		{name: "user email placeholder is raw", query: "assignee = {{.UserEmail}}", vars: queryVars{email: "user@example.com"}, want: "assignee = user@example.com"},
		{name: "query without placeholders is unchanged", query: "assignee = currentUser()", want: "assignee = currentUser()"},
		{name: "broken placeholder fails", query: "project = {{.Bad", vars: queryVars{project: testProject}, wantErr: true},
		{name: "unknown placeholder fails", query: "project = {{.ProjectKy}}", vars: queryVars{project: testProject}, wantErr: true},
		{name: "project placeholder without project fails", query: "project = {{.ProjectKey}}", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveQuery(tt.query, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			testkit.AssertEqual(t, "jql", got, tt.want)
		})
	}
}

func TestSearchRun_opensJQLTabWithTypedQuery(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())
	const query = "project = {{.ProjectKey}}"
	app := jqlApp(t)
	app.projectKey = testProject
	fake := searchFake(t, jira.Issue{Key: testKey, Summary: testSummary})
	app.client = fake
	app.jqlModal.Show("", nil)

	submitJQL(app, query)

	call := fake.SearchIssuesCalls[0]
	testkit.AssertEqual(t, "sent JQL", call.JQL, `project = "PLAT"`)
	testkit.AssertEqual(t, "sent maxResults", call.MaxResults, app.cfg.ResolveGlobalMaxResults())
	if !app.issuesList.IsJQLTab() {
		t.Fatal("JQL tab should open after a search")
	}
	testkit.AssertEqual(t, "JQL tab query", app.issuesList.JQLQuery(), query)
	testkit.AssertEqual(t, "leftFocus", app.leftFocus, focusIssues)
	if h := LoadJQLHistory(); len(h) == 0 || h[0] != query {
		t.Errorf("history = %v, want %q first", h, query)
	}
	if app.jqlModal.IsVisible() {
		t.Error("modal should close after a successful search")
	}
}

func TestSearchRun_resultAfterCancelIsDropped(t *testing.T) {
	t.Parallel()
	app := jqlApp(t)
	app.client = searchFake(t, jira.Issue{Key: testKey})
	app.jqlModal.Show("", nil)

	_, cmd := app.Update(components.JQLSubmitMsg{Query: "project = X"})
	app.Update(components.JQLCancelMsg{})
	app.Update(cmd())

	if app.issuesList.IsJQLTab() {
		t.Error("a result arriving after Esc must not open the JQL tab")
	}
}

func TestSearchRun_unresolvableQueryShowsErrorWithoutRequest(t *testing.T) {
	t.Parallel()
	app := jqlApp(t) // its fake fails the test on any SearchIssues call
	app.jqlModal.Show("", nil)

	submitJQL(app, "project = {{.ProjectKey}}")

	if !app.jqlModal.IsVisible() {
		t.Fatal("modal should stay open on a resolution error")
	}
	if !modalShows(app, errNoProject.Error()) {
		t.Errorf("modal should show %q", errNoProject)
	}
}

func TestSearchRun_jiraErrorShowsInModal(t *testing.T) {
	t.Parallel()
	app := jqlApp(t)
	app.client = &jiratest.FakeClient{T: t, SearchIssuesFunc: func(context.Context, string, int, int) (*jira.SearchResult, error) {
		return nil, errors.New("bad jql")
	}}
	app.jqlModal.Show("", nil)

	submitJQL(app, "project = X")

	if !app.jqlModal.IsVisible() || !modalShows(app, "bad jql") {
		t.Error("modal should stay open and show the Jira error")
	}
}

func TestTabRun_resolvesQueryWithTabPageSize(t *testing.T) {
	t.Parallel()
	pageSize := 7
	fake := searchFake(t, jira.Issue{Key: testKey})
	app := newAppWithFake(t, fake)
	app.leftFocus = focusProjects // skip preview side effects
	app.projectKey = testProject
	app.issuesList.SetTabs([]config.IssueTabConfig{{Name: "Mine", JQL: "project = {{.ProjectKey}}", MaxResults: &pageSize}})

	app.Update(app.fetchActiveTab()())

	call := fake.SearchIssuesCalls[0]
	testkit.AssertEqual(t, "sent JQL", call.JQL, `project = "PLAT"`)
	testkit.AssertEqual(t, "sent maxResults", call.MaxResults, pageSize)
	if !app.issuesList.HasCachedTab() {
		t.Error("tab should hold the fetched issues")
	}
}

func TestTabRun_unresolvableQueryReportsError(t *testing.T) {
	t.Parallel()
	app := newAppWithFake(t, &jiratest.FakeClient{T: t})
	app.projectKey = testProject
	app.issuesList.SetTabs([]config.IssueTabConfig{{Name: "Broken", JQL: "project = {{.ProjectKy}}"}})

	app.Update(app.fetchActiveTab()())

	if !strings.Contains(app.statusPanel.ErrorMessage(), "ProjectKy") {
		t.Errorf("status panel error = %q, want the resolution error", app.statusPanel.ErrorMessage())
	}
}
