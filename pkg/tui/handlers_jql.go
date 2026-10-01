package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/textfuel/lazyjira/v2/pkg/tui/components"
)

// handleJQLSubmit runs the modal's query, as an in-place edit when the modal
// was opened on a managed tab.
func (a *App) handleJQLSubmit(msg components.JQLSubmitMsg) (tea.Model, tea.Cmd) {
	run := queryRun{origin: originSearch, query: msg.Query}
	if idx := a.editingManagedTab; idx >= 0 && idx < len(a.savedTabs) {
		run.origin, run.managedTab = originEdit, idx
	}
	return a, a.startRun(run)
}

// handleJQLSaveTab hides the JQL modal and opens the name prompt to persist the
// current query as a new managed tab, reusing the editTabName confirm path.
func (a *App) handleJQLSaveTab(msg components.JQLSaveTabMsg) (tea.Model, tea.Cmd) {
	prev := a.editingManagedTab
	a.jqlModal.Hide()
	a.editingManagedTab = -1
	a.inputModal.Show("Save tab", "")
	a.editContext = editCtx{kind: editTabName, tabJQL: msg.Query, returnToJQLModal: true, prevEditingManagedTab: prev}
	return a, nil
}

// handleJQLInputChanged processes JQL autocomplete context.
func (a *App) handleJQLInputChanged(msg components.JQLInputChangedMsg) (tea.Model, tea.Cmd) {
	ctx := parseJQLContext(msg.Text, msg.CursorPos)
	a.jqlModal.SetPartialLen(ctx.PartialLen)
	switch ctx.Mode {
	case jqlCtxField:
		if a.jqlFields != nil {
			suggestions := matchFieldSuggestions(a.jqlFields, ctx.Partial)
			if len(suggestions) > 0 {
				a.jqlModal.SetSuggestions(suggestions)
			} else {
				a.jqlModal.SetHistory(LoadJQLHistory())
			}
		}
	case jqlCtxValue:
		a.jqlModal.SetACLoading(true)
		return a, fetchJQLSuggestions(a.client, ctx.FieldName, ctx.Partial)
	default:
		a.jqlModal.SetHistory(LoadJQLHistory())
	}
	return a, nil
}

// handleJQLFieldsLoaded caches JQL autocomplete field data.
func (a *App) handleJQLFieldsLoaded(msg jqlFieldsLoadedMsg) (tea.Model, tea.Cmd) {
	a.jqlFields = msg.fields
	return a, nil
}

// handleJQLSuggestions updates JQL suggestion list.
func (a *App) handleJQLSuggestions(msg jqlSuggestionsMsg) (tea.Model, tea.Cmd) {
	if a.jqlModal.IsVisible() {
		items := make([]string, 0, len(msg.suggestions))
		for _, s := range msg.suggestions {
			items = append(items, s.Value)
		}
		if len(items) > 0 {
			a.jqlModal.SetSuggestions(items)
		} else {
			a.jqlModal.SetHistory(LoadJQLHistory())
		}
	}
	return a, nil
}
