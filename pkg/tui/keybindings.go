package tui

import "github.com/textfuel/lazyjira/pkg/tui/components"

// Binding represents a single keybinding with context.
type Binding struct {
	Key         string
	Description string
}

func (a *App) bind(action Action, desc string) Binding {
	return Binding{a.keymap.Keys(action), desc}
}

// ContextBindings returns keybindings for the current focus context.
func (a *App) ContextBindings() []Binding {
	km := a.keymap
	global := []Binding{
		{km.Keys(ActQuit), "quit"},
		{km.Keys(ActSwitchPanel), "switch left/right panels"},
		{km.Keys(ActFocusStatus), "focus status panel"},
		{km.Keys(ActFocusIssues), "focus issues panel"},
		{km.Keys(ActFocusProj), "focus projects panel"},
		{km.Keys(ActSearch), "search / filter current list"},
		{km.Keys(ActRefresh), "refresh data from Jira"},
		{km.Keys(ActHelp), "show all keybindings"},
	}

	switch {
	case a.side == sideLeft && a.leftFocus == focusIssues:
		return append(global,
			Binding{"j/k", "navigate up/down"},
			Binding{"g/G", "go to top/bottom"},
			Binding{"ctrl+d/u", "half-page down/up"},
			a.bind(ActSelect, "select issue (mark active + open)"),
			a.bind(ActOpen, "open issue detail"),
			a.bind(ActFocusRight, "open issue detail"),
			a.bind(ActTransition, "transition issue status"),
			a.bind(ActBrowser, "open issue in browser"),
			a.bind(ActURLPicker, "open URL picker"),
			Binding{"[/]", "switch All/Assigned"},
		)

	case a.side == sideLeft && a.leftFocus == focusProjects:
		return append(global,
			Binding{"j/k", "navigate up/down"},
			Binding{"g/G", "go to top/bottom"},
			Binding{"ctrl+d/u", "half-page down/up"},
			a.bind(ActSelect, "select project and load issues"),
			a.bind(ActOpen, "preview project"),
			a.bind(ActFocusRight, "switch to detail panel"),
		)

	case a.side == sideLeft && a.leftFocus == focusStatus:
		return append(global,
			a.bind(ActFocusRight, "switch to detail panel"),
		)

	case a.side == sideRight:
		return append(global,
			Binding{"j/k", "scroll up/down"},
			Binding{"ctrl+d/u", "half-page down/up"},
			Binding{"[/]", "previous/next tab"},
			a.bind(ActFocusLeft, "back to left panel"),
			a.bind(ActInfoTab, "jump to info tab"),
			a.bind(ActBrowser, "open in browser"),
			a.bind(ActURLPicker, "open URL picker"),
		)
	}

	return global
}

func (a *App) helpBarItems() []components.HelpItem {
	km := a.keymap
	switch {
	case a.side == sideLeft && a.leftFocus == focusIssues:
		return []components.HelpItem{
			{Key: "j/k", Description: "navigate"},
			{Key: km.Keys(ActSelect), Description: "select"},
			{Key: km.Keys(ActOpen), Description: "open"},
			{Key: km.Keys(ActTransition), Description: "transition"},
			{Key: km.Keys(ActHelp), Description: "help"},
		}
	case a.side == sideLeft && a.leftFocus == focusProjects:
		return []components.HelpItem{
			{Key: "j/k", Description: "navigate"},
			{Key: km.Keys(ActSelect), Description: "select"},
			{Key: km.Keys(ActOpen), Description: "preview"},
			{Key: km.Keys(ActHelp), Description: "help"},
		}
	case a.side == sideLeft && a.leftFocus == focusStatus:
		return []components.HelpItem{
			{Key: km.Keys(ActSwitchPanel) + "/" + km.Keys(ActFocusRight), Description: "detail"},
			{Key: km.Keys(ActHelp), Description: "help"},
		}
	case a.side == sideRight:
		return []components.HelpItem{
			{Key: "j/k", Description: "scroll"},
			{Key: "[/]", Description: "tabs"},
			{Key: km.Keys(ActFocusLeft), Description: "back"},
			{Key: km.Keys(ActInfoTab), Description: "info"},
			{Key: km.Keys(ActHelp), Description: "help"},
		}
	}
	return nil
}
