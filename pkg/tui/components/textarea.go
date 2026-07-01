package components

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/textfuel/lazyjira/v2/pkg/tui/theme"
)

// TextArea is a multi-line readline-style input. The value is a flat string with
// embedded newlines; the cursor is a flat rune index. Line structure is derived
// on demand, so the Highlighter and any consumer working on a flat rune index
// (autocomplete) keep operating unchanged. Only hard, user-typed newlines are
// supported; there is no soft-wrapping.
type TextArea struct {
	value       string
	cursor      int // flat rune index
	desiredCol  int // remembered column for vertical moves; -1 = unset
	width       int
	Highlighter func(text []rune) []StyledSegment
}

func NewTextArea() TextArea {
	return TextArea{desiredCol: -1}
}

func (t *TextArea) SetValue(s string) {
	t.value = s
	t.cursor = utf8.RuneCountInString(s)
	t.desiredCol = -1
}

func (t *TextArea) Value() string { return t.value }

func (t *TextArea) SetWidth(w int) { t.width = w }

func (t *TextArea) CursorPos() int { return t.cursor }

func (t *TextArea) setCursor(pos int) {
	n := len([]rune(t.value))
	if pos < 0 {
		pos = 0
	}
	if pos > n {
		pos = n
	}
	t.cursor = pos
}

// LineCount reports the number of visual lines (hard newlines + 1).
func (t *TextArea) LineCount() int { return strings.Count(t.value, "\n") + 1 }

func (t *TextArea) InsertAtCursor(s string) {
	runes := []rune(t.value)
	inserted := []rune(s)
	newRunes := make([]rune, 0, len(runes)+len(inserted))
	newRunes = append(newRunes, runes[:t.cursor]...)
	newRunes = append(newRunes, inserted...)
	newRunes = append(newRunes, runes[t.cursor:]...)
	t.value = string(newRunes)
	t.cursor += len(inserted)
}

// lineBounds returns the [start,end) rune range of the line containing pos,
// where end is the index of the terminating newline (or len for the last line).
func lineBounds(runes []rune, pos int) (start, end int) {
	start = 0
	for i := pos - 1; i >= 0; i-- {
		if runes[i] == '\n' {
			start = i + 1
			break
		}
	}
	end = len(runes)
	for i := pos; i < len(runes); i++ {
		if runes[i] == '\n' {
			end = i
			break
		}
	}
	return start, end
}

// Update handles key events and reports whether the value changed.
func (t *TextArea) Update(msg tea.Msg) (TextArea, bool) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return *t, false
	}

	runes := []rune(t.value)
	n := len(runes)

	switch km.Type {
	case tea.KeyLeft:
		t.desiredCol = -1
		if t.cursor > 0 {
			t.cursor--
		}
		return *t, false

	case tea.KeyRight:
		t.desiredCol = -1
		if t.cursor < n {
			t.cursor++
		}
		return *t, false

	case tea.KeyUp:
		t.moveVertical(runes, -1)
		return *t, false

	case tea.KeyDown:
		t.moveVertical(runes, +1)
		return *t, false

	case tea.KeyHome, tea.KeyCtrlA:
		t.desiredCol = -1
		s, _ := lineBounds(runes, t.cursor)
		t.cursor = s
		return *t, false

	case tea.KeyEnd, tea.KeyCtrlE:
		t.desiredCol = -1
		_, e := lineBounds(runes, t.cursor)
		t.cursor = e
		return *t, false

	case tea.KeyEnter:
		t.desiredCol = -1
		t.InsertAtCursor("\n")
		return *t, true

	case tea.KeyBackspace:
		t.desiredCol = -1
		if t.cursor > 0 {
			t.value = string(runes[:t.cursor-1]) + string(runes[t.cursor:])
			t.cursor--
			return *t, true
		}
		return *t, false

	case tea.KeyDelete:
		t.desiredCol = -1
		if t.cursor < n {
			t.value = string(runes[:t.cursor]) + string(runes[t.cursor+1:])
			return *t, true
		}
		return *t, false

	case tea.KeyCtrlW:
		return t.deleteWord(runes)

	case tea.KeyCtrlK:
		return t.killToLineEnd(runes, n)

	case tea.KeyCtrlU:
		return t.killToLineStart(runes)

	case tea.KeySpace:
		t.desiredCol = -1
		t.InsertAtCursor(" ")
		return *t, true

	case tea.KeyRunes:
		t.desiredCol = -1
		t.InsertAtCursor(km.String())
		return *t, true

	default:
		return *t, false
	}
}

// moveVertical moves the cursor one line up (dir=-1) or down (dir=+1), keeping
// the desired column across shorter intervening lines.
func (t *TextArea) moveVertical(runes []rune, dir int) {
	start, end := lineBounds(runes, t.cursor)

	if t.desiredCol < 0 {
		t.desiredCol = t.cursor - start
	}
	col := t.desiredCol

	if dir < 0 {
		if start == 0 {
			t.cursor = 0
			t.desiredCol = -1
			return
		}
		ps, pe := lineBounds(runes, start-1)
		if plen := pe - ps; col > plen {
			col = plen
		}
		t.cursor = ps + col
		return
	}

	if end >= len(runes) {
		t.cursor = len(runes)
		t.desiredCol = -1
		return
	}
	ns := end + 1
	_, ne := lineBounds(runes, ns)
	if nlen := ne - ns; col > nlen {
		col = nlen
	}
	t.cursor = ns + col
}

func (t *TextArea) deleteWord(runes []rune) (TextArea, bool) {
	t.desiredCol = -1
	if t.cursor == 0 {
		return *t, false
	}
	i := t.cursor
	for i > 0 && runes[i-1] == ' ' {
		i--
	}
	for i > 0 && runes[i-1] != ' ' && runes[i-1] != '\n' {
		i--
	}
	t.value = string(runes[:i]) + string(runes[t.cursor:])
	t.cursor = i
	return *t, true
}

func (t *TextArea) killToLineEnd(runes []rune, n int) (TextArea, bool) {
	t.desiredCol = -1
	_, e := lineBounds(runes, t.cursor)
	if t.cursor < e {
		t.value = string(runes[:t.cursor]) + string(runes[e:])
		return *t, true
	}
	if e < n { // at line end: swallow the newline, joining the next line
		t.value = string(runes[:e]) + string(runes[e+1:])
		return *t, true
	}
	return *t, false
}

func (t *TextArea) killToLineStart(runes []rune) (TextArea, bool) {
	t.desiredCol = -1
	s, _ := lineBounds(runes, t.cursor)
	if t.cursor > s {
		t.value = string(runes[:s]) + string(runes[t.cursor:])
		t.cursor = s
		return *t, true
	}
	return *t, false
}

// cursorLineCol returns the visual (line, column) of the flat cursor.
func (t *TextArea) cursorLineCol() (int, int) {
	runes := []rune(t.value)
	line, col := 0, 0
	for i := 0; i < t.cursor && i < len(runes); i++ {
		if runes[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

// View renders all lines with a block cursor on the cursor's line.
func (t *TextArea) View() string {
	lines := strings.Split(t.value, "\n")
	curLine, curCol := t.cursorLineCol()

	var b strings.Builder
	for i, line := range lines {
		if i == curLine {
			b.WriteString(t.renderLine([]rune(line), curCol, true))
		} else {
			b.WriteString(t.renderLine([]rune(line), 0, false))
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (t *TextArea) renderLine(line []rune, cursorCol int, showCursor bool) string {
	cursorStyle := lipgloss.NewStyle().Foreground(theme.ColorCyan)
	cursorBlock := cursorStyle.Render("█")

	if t.Highlighter != nil {
		return renderHighlightedLine(t.Highlighter(line), cursorCol, showCursor, cursorStyle, cursorBlock)
	}

	if !showCursor {
		return string(line)
	}
	var b strings.Builder
	if cursorCol >= 0 && cursorCol < len(line) {
		b.WriteString(string(line[:cursorCol]))
		b.WriteString(cursorStyle.Render(string(line[cursorCol])))
		b.WriteString(string(line[cursorCol+1:]))
	} else {
		b.WriteString(string(line))
		b.WriteString(cursorBlock)
	}
	return b.String()
}

func renderHighlightedLine(segments []StyledSegment, cursorCol int, showCursor bool, cursorStyle lipgloss.Style, cursorBlock string) string {
	var b strings.Builder
	pos := 0
	for _, seg := range segments {
		segRunes := []rune(seg.Text)
		segEnd := pos + len(segRunes)

		if showCursor && cursorCol >= pos && cursorCol < segEnd {
			before := segRunes[:cursorCol-pos]
			at := segRunes[cursorCol-pos]
			after := segRunes[cursorCol-pos+1:]
			if len(before) > 0 {
				b.WriteString(seg.Style.Render(string(before)))
			}
			b.WriteString(cursorStyle.Render(string(at)))
			if len(after) > 0 {
				b.WriteString(seg.Style.Render(string(after)))
			}
		} else {
			b.WriteString(seg.Style.Render(seg.Text))
		}
		pos = segEnd
	}
	if showCursor && cursorCol >= pos {
		b.WriteString(cursorBlock)
	}
	return b.String()
}
