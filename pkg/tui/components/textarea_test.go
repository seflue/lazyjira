package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/textfuel/lazyjira/v2/pkg/internal/testkit"
)

func TestTextArea_Editing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		initial     string
		cursor      int
		key         tea.KeyType
		runes       string
		wantValue   string
		wantCursor  int
		wantChanged bool
	}{
		{name: "enter inserts newline", initial: "ab", cursor: 1, key: tea.KeyEnter, wantValue: "a\nb", wantCursor: 2, wantChanged: true},
		{name: "backspace at line start joins lines", initial: "a\nb", cursor: 2, key: tea.KeyBackspace, wantValue: "ab", wantCursor: 1, wantChanged: true},
		{name: "home jumps to line start", initial: "ab\ncde", cursor: 5, key: tea.KeyHome, wantValue: "ab\ncde", wantCursor: 3},
		{name: "end jumps to line end", initial: "ab\ncde", cursor: 3, key: tea.KeyEnd, wantValue: "ab\ncde", wantCursor: 6},
		{name: "ctrl+k kills to line end", initial: "ab\ncde", cursor: 4, key: tea.KeyCtrlK, wantValue: "ab\nc", wantCursor: 4, wantChanged: true},
		{name: "ctrl+k at line end swallows newline", initial: "ab\ncde", cursor: 2, key: tea.KeyCtrlK, wantValue: "abcde", wantCursor: 2, wantChanged: true},
		{name: "ctrl+u kills to line start", initial: "ab\ncde", cursor: 5, key: tea.KeyCtrlU, wantValue: "ab\ne", wantCursor: 3, wantChanged: true},
		{name: "ctrl+w stops at newline", initial: "ab\ncde", cursor: 6, key: tea.KeyCtrlW, wantValue: "ab\n", wantCursor: 3, wantChanged: true},
		{name: "rune inserts at cursor", initial: "a\nb", cursor: 2, key: tea.KeyRunes, runes: "X", wantValue: "a\nXb", wantCursor: 3, wantChanged: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ta := NewTextArea()
			ta.SetValue(tt.initial)
			ta.setCursor(tt.cursor)

			msg := tea.KeyMsg{Type: tt.key}
			if tt.runes != "" {
				msg.Runes = []rune(tt.runes)
			}
			got, changed := ta.Update(msg)

			testkit.AssertEqual(t, "value", got.Value(), tt.wantValue)
			testkit.AssertEqual(t, "cursor", got.CursorPos(), tt.wantCursor)
			testkit.AssertEqual(t, "changed", changed, tt.wantChanged)
		})
	}
}

func TestTextArea_VerticalMove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    string
		cursor     int
		key        tea.KeyType
		wantCursor int
	}{
		// "abc\nde" : line0 = abc (0..3), '\n'=3, line1 = de (4..6)
		{name: "down keeps column", initial: "abc\ndef", cursor: 2, key: tea.KeyDown, wantCursor: 6},
		{name: "down clamps to shorter line", initial: "abc\nd", cursor: 3, key: tea.KeyDown, wantCursor: 5},
		{name: "up keeps column", initial: "abc\ndef", cursor: 6, key: tea.KeyUp, wantCursor: 2},
		{name: "up on first line goes to start", initial: "abc\ndef", cursor: 2, key: tea.KeyUp, wantCursor: 0},
		{name: "down on last line goes to end", initial: "abc\ndef", cursor: 5, key: tea.KeyDown, wantCursor: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ta := NewTextArea()
			ta.SetValue(tt.initial)
			ta.setCursor(tt.cursor)
			got, _ := ta.Update(tea.KeyMsg{Type: tt.key})
			testkit.AssertEqual(t, "cursor", got.CursorPos(), tt.wantCursor)
		})
	}
}

// desiredCol must survive a short intervening line: long -> short -> long
// returns to the original column, not the clamped one.
func TestTextArea_DesiredColumnAcrossShortLine(t *testing.T) {
	t.Parallel()

	// line0 "hello" (col 0..5), line1 "x" (len 1), line2 "world" (col 0..5)
	ta := NewTextArea()
	ta.SetValue("hello\nx\nworld")
	ta.setCursor(4) // "hell|o" on line0, column 4

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown}) // to line1 "x", clamped to col 1
	testkit.AssertEqual(t, "cursor on short line", ta.CursorPos(), 7)

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown}) // to line2, desired col 4 restored
	// line2 starts at index 8 ("world"), col 4 -> index 12
	testkit.AssertEqual(t, "cursor restored column", ta.CursorPos(), 12)
}

func TestTextArea_View_MultiLine(t *testing.T) {
	t.Parallel()

	ta := NewTextArea()
	ta.SetWidth(40)
	ta.SetValue("project = FOO\nAND status = Open")
	// cursor sits at end (line 1), so line 0 renders without a cursor block

	out := ta.View()
	testkit.AssertEqual(t, "line count", strings.Count(out, "\n"), 1)
	if !strings.Contains(out, "project = FOO") {
		t.Errorf("view missing first line: %q", out)
	}
}

func TestTextArea_View_WithHighlighterDoesNotPanic(t *testing.T) {
	t.Parallel()

	ta := NewTextArea()
	ta.SetWidth(40)
	ta.Highlighter = HighlightJQL
	ta.SetValue("project = FOO\nAND status = Open")
	ta.setCursor(20) // somewhere on the second line

	_ = ta.View() // must not panic; cursor placed within a highlighted segment
}
