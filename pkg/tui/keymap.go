package tui

import "slices"

// Action represents a user-triggerable action.
type Action string

// Actions — each can be remapped to different keys in the future via config.
const (
	ActQuit        Action = "quit"
	ActHelp        Action = "help"
	ActSearch      Action = "search"
	ActSwitchPanel Action = "switchPanel"
	ActFocusRight  Action = "focusRight"
	ActFocusLeft   Action = "focusLeft"
	ActSelect      Action = "select"      // primary: mark active + open
	ActOpen        Action = "open"        // secondary: open/preview without marking
	ActPrevTab     Action = "prevTab"
	ActNextTab     Action = "nextTab"
	ActFocusDetail Action = "focusDetail"
	ActFocusStatus Action = "focusStatus"
	ActFocusIssues Action = "focusIssues"
	ActFocusProj   Action = "focusProjects"
	ActCopyURL     Action = "copyURL"
	ActBrowser     Action = "browser"
	ActURLPicker   Action = "urlPicker"
	ActTransition  Action = "transition"
	ActRefresh     Action = "refresh"
	ActRefreshAll  Action = "refreshAll"
	ActInfoTab     Action = "infoTab"
)

// Keymap maps actions to key strings. Multiple keys can trigger the same action.
type Keymap map[Action][]string

// DefaultKeymap returns the default key bindings.
func DefaultKeymap() Keymap {
	return Keymap{
		ActQuit:        {"q", "ctrl+c"},
		ActHelp:        {"?"},
		ActSearch:      {"/"},
		ActSwitchPanel: {"tab"},
		ActFocusRight:  {"l", "right"},
		ActFocusLeft:   {"h", "left", "esc"},
		ActSelect:      {" "},
		ActOpen:        {"enter"},
		ActPrevTab:     {"["},
		ActNextTab:     {"]"},
		ActFocusDetail: {"0"},
		ActFocusStatus: {"1"},
		ActFocusIssues: {"2"},
		ActFocusProj:   {"3"},
		ActCopyURL:     {"y"},
		ActBrowser:     {"o"},
		ActURLPicker:   {"u"},
		ActTransition:  {"t"},
		ActRefresh:     {"r"},
		ActRefreshAll:  {"R"},
		ActInfoTab:     {"i"},
	}
}

// Match returns the action for the given key, or "" if none.
func (km Keymap) Match(key string) Action {
	for action, keys := range km {
		if slices.Contains(keys, key) {
			return action
		}
	}
	return ""
}

// Keys returns the first key bound to the action (for display in help).
func (km Keymap) Keys(action Action) string {
	if keys, ok := km[action]; ok && len(keys) > 0 {
		k := keys[0]
		if k == " " {
			return "space"
		}
		return k
	}
	return ""
}
