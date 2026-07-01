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
// (autocomplete) keep operating unchanged. Only hard, user-typed newlines live in
// the value; long lines are soft-wrapped at width for display and navigation, but
// no soft break ever inserts a real newline into the value.
type TextArea struct {
	value       string
	cursor      int // flat rune index
	desiredCol  int // remembered visual column for vertical moves; -1 = unset
	width       int
	height      int // visible visual-line window; <=0 renders every line
	offset      int // first visible visual line, follows the cursor
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

func (t *TextArea) SetHeight(h int) { t.height = h }

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

// VisualLineCount reports the number of on-screen rows after soft-wrapping at the
// current width (always at least 1).
func (t *TextArea) VisualLineCount() int { return len(wrapLines([]rune(t.value), t.width)) }

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

// visualLine is one on-screen row: the half-open flat rune range [start,end) it
// displays. Rows partition the value's runes; a hard newline separating two
// logical lines is not part of any row's range.
type visualLine struct {
	start int
	end   int
}

// wrapLines splits runes into visual rows. Each logical line (delimited by '\n')
// is word-wrapped at width: the break falls after the last space within width, or
// hard at the char boundary for a token longer than width. width <= 0 disables
// wrapping, so each logical line becomes exactly one row. Every logical line
// yields at least one row, including empty ones.
func wrapLines(runes []rune, width int) []visualLine {
	var lines []visualLine
	n := len(runes)
	lineStart := 0
	for i := 0; i <= n; i++ {
		if i == n || runes[i] == '\n' {
			lines = append(lines, wrapLogicalLine(runes, lineStart, i, width)...)
			lineStart = i + 1
		}
	}
	return lines
}

// wrapLogicalLine wraps the single logical line runes[start:end] into rows.
func wrapLogicalLine(runes []rune, start, end, width int) []visualLine {
	if width <= 0 {
		return []visualLine{{start, end}}
	}
	var out []visualLine
	segStart := start
	for {
		if end-segStart <= width {
			out = append(out, visualLine{segStart, end})
			return out
		}
		breakAt := segStart + width // hard break unless a space is found within width
		for j := segStart + width - 1; j > segStart; j-- {
			if runes[j] == ' ' {
				breakAt = j + 1 // keep the space on the current row
				break
			}
		}
		out = append(out, visualLine{segStart, breakAt})
		segStart = breakAt
	}
}

// visualPos maps a flat rune index to its (row, col). At a soft-wrap boundary the
// position belongs to the following row's column 0; at a hard newline or value end
// it stays on the current row past its last rune.
func visualPos(lines []visualLine, pos int) (row, col int) {
	for i, vl := range lines {
		if pos < vl.end {
			return i, pos - vl.start
		}
		if pos == vl.end {
			if i+1 < len(lines) && lines[i+1].start == vl.end {
				continue // soft continuation owns this boundary as its column 0
			}
			return i, pos - vl.start
		}
	}
	if len(lines) == 0 {
		return 0, 0
	}
	last := len(lines) - 1
	return last, pos - lines[last].start
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

// moveVertical moves the cursor one visual row up (dir=-1) or down (dir=+1),
// keeping the desired visual column across shorter intervening rows. Moving above
// the first row snaps to the value start; below the last row snaps to its end.
func (t *TextArea) moveVertical(runes []rune, dir int) {
	lines := wrapLines(runes, t.width)
	row, col := visualPos(lines, t.cursor)

	if t.desiredCol < 0 {
		t.desiredCol = col
	}

	target := row + dir
	if target < 0 {
		t.cursor = 0
		t.desiredCol = -1
		return
	}
	if target >= len(lines) {
		t.cursor = len(runes)
		t.desiredCol = -1
		return
	}

	vl := lines[target]
	c := t.desiredCol
	if maxCol := vl.end - vl.start; c > maxCol {
		c = maxCol
	}
	t.cursor = vl.start + c
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

// View renders the visible visual-line window with a block cursor on the cursor's
// row. Soft-wrapping and the height window are display-only; the value is
// untouched.
func (t *TextArea) View() string {
	runes := []rune(t.value)
	lines := wrapLines(runes, t.width)
	curRow, curCol := visualPos(lines, t.cursor)
	from, to := t.window(lines, curRow)

	var b strings.Builder
	for i := from; i < to; i++ {
		vl := lines[i]
		seg := runes[vl.start:vl.end]
		if i == curRow {
			b.WriteString(t.renderLine(seg, curCol, true))
		} else {
			b.WriteString(t.renderLine(seg, 0, false))
		}
		if i < to-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// window returns the [from,to) range of visible rows, updating offset so the
// cursor's row stays in view. height <= 0 (or a window larger than the content)
// shows everything.
func (t *TextArea) window(lines []visualLine, curRow int) (from, to int) {
	if t.height <= 0 || t.height >= len(lines) {
		t.offset = 0
		return 0, len(lines)
	}
	if curRow < t.offset {
		t.offset = curRow
	} else if curRow >= t.offset+t.height {
		t.offset = curRow - t.height + 1
	}
	if maxOffset := len(lines) - t.height; t.offset > maxOffset {
		t.offset = maxOffset
	}
	if t.offset < 0 {
		t.offset = 0
	}
	return t.offset, t.offset + t.height
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
