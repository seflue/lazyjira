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

func TestWrapLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  []visualLine
	}{
		{
			name:  "word break at last space",
			value: "hello world",
			width: 8,
			want:  []visualLine{{0, 6}, {6, 11}},
		},
		{
			name:  "overlong token breaks hard",
			value: "abcdefghij",
			width: 4,
			want:  []visualLine{{0, 4}, {4, 8}, {8, 10}},
		},
		{
			name:  "width zero disables wrap",
			value: "aaaaaaaa\nbb",
			width: 0,
			want:  []visualLine{{0, 8}, {9, 11}},
		},
		{
			name:  "hard newline plus soft wrap",
			value: "hi\nhello world",
			width: 8,
			want:  []visualLine{{0, 2}, {3, 9}, {9, 14}},
		},
		{
			name:  "trailing newline yields empty final line",
			value: "a\n",
			width: 0,
			want:  []visualLine{{0, 1}, {2, 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := wrapLines([]rune(tt.value), tt.width)
			testkit.AssertSliceEqual(t, "lines", got, tt.want)
		})
	}
}

func TestVisualPos(t *testing.T) {
	t.Parallel()

	// "hello world" @ width 8 -> {0,6} "hello ", {6,11} "world"
	lines := wrapLines([]rune("hello world"), 8)

	tests := []struct {
		name    string
		pos     int
		wantRow int
		wantCol int
	}{
		{name: "start of first fragment", pos: 0, wantRow: 0, wantCol: 0},
		{name: "inside first fragment", pos: 4, wantRow: 0, wantCol: 4},
		{name: "soft boundary maps to next row start", pos: 6, wantRow: 1, wantCol: 0},
		{name: "inside second fragment", pos: 8, wantRow: 1, wantCol: 2},
		{name: "end of value", pos: 11, wantRow: 1, wantCol: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			row, col := visualPos(lines, tt.pos)
			testkit.AssertEqual(t, "row", row, tt.wantRow)
			testkit.AssertEqual(t, "col", col, tt.wantCol)
		})
	}
}

func TestTextArea_View_SoftWrap(t *testing.T) {
	t.Parallel()

	ta := NewTextArea()
	ta.SetWidth(8)
	ta.SetValue("hello world") // wraps to "hello " / "world"
	// cursor sits at end (row 1); first row renders without a cursor block

	out := ta.View()
	testkit.AssertEqual(t, "visual line count", strings.Count(out, "\n")+1, 2)
	if !strings.Contains(out, "hello ") {
		t.Errorf("view missing first fragment: %q", out)
	}
	if !strings.Contains(out, "world") {
		t.Errorf("view missing second fragment: %q", out)
	}
}

func TestTextArea_VerticalMove_Visual(t *testing.T) {
	t.Parallel()

	ta := NewTextArea()
	ta.SetWidth(8)
	// "hello world foobar" @ width 8 ->
	//   {0,6}  "hello "
	//   {6,12} "world "
	//   {12,18} "foobar"
	ta.SetValue("hello world foobar")
	ta.setCursor(2) // row 0, col 2

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown})
	testkit.AssertEqual(t, "down into second fragment", ta.CursorPos(), 8)

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown})
	testkit.AssertEqual(t, "down into third fragment", ta.CursorPos(), 14)

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyUp})
	testkit.AssertEqual(t, "up back into second fragment", ta.CursorPos(), 8)
}

func TestTextArea_DesiredColumnAcrossShortVisualFragment(t *testing.T) {
	t.Parallel()

	ta := NewTextArea()
	ta.SetWidth(10)
	// "aaaaaaaaa bb\nZ\naaaaaaaaa cc" @ width 10 ->
	//   {0,10}  "aaaaaaaaa "
	//   {10,12} "bb"
	//   {13,14} "Z"
	//   {15,25} "aaaaaaaaa "
	//   {25,27} "cc"
	ta.SetValue("aaaaaaaaa bb\nZ\naaaaaaaaa cc")
	ta.setCursor(8) // row 0, col 8

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown}) // row1 "bb", clamp col 8 -> 2
	testkit.AssertEqual(t, "clamped on bb", ta.CursorPos(), 12)

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown}) // row2 "Z", clamp col 8 -> 1
	testkit.AssertEqual(t, "clamped on Z", ta.CursorPos(), 14)

	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown}) // row3, desired col 8 restored
	testkit.AssertEqual(t, "restored column", ta.CursorPos(), 23)
}

func TestTextArea_HeightWindow(t *testing.T) {
	t.Parallel()

	ta := NewTextArea()
	ta.SetWidth(0)
	ta.SetValue("l0\nl1\nl2\nl3\nl4") // 5 visual lines
	ta.SetHeight(3)
	// cursor at end -> row 4 (bottom)

	out := ta.View()
	testkit.AssertEqual(t, "rendered line count", strings.Count(out, "\n")+1, 3)
	if !strings.Contains(out, "l4") {
		t.Errorf("window missing cursor line: %q", out)
	}
	if strings.Contains(out, "l0") {
		t.Errorf("window should have scrolled past first line: %q", out)
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
