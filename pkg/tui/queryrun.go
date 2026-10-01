package tui

import (
	"bytes"
	"context"
	"errors"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/textfuel/lazyjira/v2/pkg/config"
	"github.com/textfuel/lazyjira/v2/pkg/jira"
)

// runOrigin decides where a query run's result goes.
type runOrigin int

const (
	originSearch runOrigin = iota
	originEdit
	originTab
)

// queryRun is one execution of a query. It is fixed when the run starts and
// travels with its result, so routing never reads state that changed while the
// request was in flight.
type queryRun struct {
	origin     runOrigin
	query      string
	managedTab int
	tab        int
	epoch      int
	modalRun   int
}

type queryResultMsg struct {
	run    queryRun
	issues []jira.Issue
	err    error
}

var errNoProject = errors.New("no active project")

// queryVars are the template variables a query may use.
type queryVars struct{ project, email string }

// ProjectKey fails without an active project instead of yielding project = "".
func (v queryVars) ProjectKey() (string, error) {
	if v.project == "" {
		return "", errNoProject
	}
	return `"` + v.project + `"`, nil
}

func (v queryVars) UserEmail() string { return v.email }

func resolveQuery(query string, vars queryVars) (string, error) {
	tmpl, err := template.New("jql").Parse(query)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// startRun resolves the query, picks the page size and searches Jira. A query
// that does not resolve comes back as a failed result without a request.
func (a *App) startRun(r queryRun) tea.Cmd {
	if r.origin != originTab {
		a.modalRun++
		r.modalRun = a.modalRun
		a.jqlModal.SetLoading(true)
	}
	*a.logFlag = true
	jql, err := resolveQuery(r.query, queryVars{project: a.projectKey, email: a.cfg.Jira.Email})
	if err != nil {
		return func() tea.Msg { return queryResultMsg{run: r, err: err} }
	}
	client, pageSize := a.client, a.runPageSize(r)
	return func() tea.Msg {
		result, err := client.SearchIssues(context.Background(), jql, 0, pageSize)
		if err != nil {
			return queryResultMsg{run: r, err: err}
		}
		return queryResultMsg{run: r, issues: result.Issues}
	}
}

func (a *App) runPageSize(r queryRun) int {
	switch r.origin {
	case originEdit:
		return a.cfg.ResolveMaxResults(config.IssueTabConfig{MaxResults: a.savedTabs[r.managedTab].MaxResults})
	case originTab:
		return a.cfg.ResolveMaxResults(a.issuesList.ActiveTab())
	default:
		return a.cfg.ResolveGlobalMaxResults()
	}
}

// finishRun routes a result by the origin of its run. Modal results are
// dropped once the modal was cancelled or resubmitted after the run started.
func (a *App) finishRun(m queryResultMsg) (tea.Model, tea.Cmd) {
	if m.run.origin == originTab {
		if m.err != nil {
			return a.Update(errorMsg{err: m.err})
		}
		return a.handleTabIssues(m.run, m.issues)
	}
	if m.run.modalRun != a.modalRun {
		return a, nil
	}
	*a.logFlag = false
	if m.err != nil {
		a.jqlModal.SetError(formatJQLError(m.err))
		return a, nil
	}
	a.jqlModal.Hide()
	history := LoadJQLHistory()
	history = AddToHistory(history, m.run.query)
	_ = SaveJQLHistory(history)
	if idx := m.run.managedTab; m.run.origin == originEdit && idx < len(a.savedTabs) {
		a.savedTabs[idx].JQL = m.run.query
		if err := config.SaveSavedTabs(a.savedTabs); err != nil {
			a.helpBar.SetStatusMsg("save tabs: " + err.Error())
		}
		a.issuesList.SetSavedTabs(a.savedTabs)
		a.jumpToManagedTab(a.savedTabs[idx].Name)
		a.issuesList.SetIssues(m.issues)
		a.editingManagedTab = -1
	} else {
		a.issuesList.AddJQLTab(m.run.query)
		a.issuesList.SetIssues(m.issues)
	}
	a.side = sideLeft
	a.leftFocus = focusIssues
	a.updateFocusState()
	cmds := make([]tea.Cmd, 0, len(m.issues))
	for _, issue := range m.issues {
		cmds = append(cmds, prefetchIssue(a.client, issue.Key))
	}
	return a, tea.Batch(cmds...)
}
