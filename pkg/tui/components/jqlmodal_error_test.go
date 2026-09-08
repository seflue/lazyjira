package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/textfuel/lazyjira/v2/pkg/internal/testkit"
)

func errModal(t *testing.T, lines []string) JQLModal {
	t.Helper()
	m := NewJQLModal()
	m.SetSize(80, 24)
	m.Show("", []string{testHistoryItem1, testHistoryItem2})
	m.SetError(lines)
	return m
}

func longError(n int) []string {
	lines := make([]string, 0, n)
	for i := range n {
		lines = append(lines, strings.Repeat("x", 10)+string(rune('a'+i%26)))
	}
	return lines
}

func TestJQLModal_ErrorPanelShowsMessageBeforeRequest(t *testing.T) {
	t.Parallel()
	m := errModal(t, []string{
		"Field 'sprintt' does not exist or you do not have permission to view it.",
		"",
		"HTTP 400  GET /search/jql?jql=project%20%3D%20FOO&startAt=0",
	})

	plain := stripANSI(m.View())
	for _, want := range []string{"Error", "Field 'sprintt' does not exist", "HTTP 400", "/search/jql"} {
		if !strings.Contains(plain, want) {
			t.Errorf("view missing %q, got:\n%s", want, plain)
		}
	}
	if strings.Index(plain, "sprintt") > strings.Index(plain, "HTTP 400") {
		t.Error("message should render above the request line")
	}
}

func TestJQLModal_ErrorPanelShrinksList(t *testing.T) {
	t.Parallel()
	m := NewJQLModal()
	m.SetSize(80, 24)
	m.Show("", nil)
	before := m.listHeight()

	m.SetError([]string{"a", "b", "c"})
	if got, want := m.listHeight(), before-(m.errorHeight()+2); got != want {
		t.Errorf("listHeight = %d, want %d", got, want)
	}
}

func TestJQLModal_ErrorPanelHeightCapped(t *testing.T) {
	t.Parallel()
	m := errModal(t, longError(40))
	if got, want := m.errorHeight(), 8; got != want {
		t.Errorf("errorHeight = %d, want %d (a third of 24)", got, want)
	}
	if !strings.Contains(stripANSI(m.View()), "of 40") {
		t.Error("clipped panel should report the hidden lines in its footer")
	}
}

func TestJQLModal_TabCyclesThroughErrorPanel(t *testing.T) {
	t.Parallel()
	m := errModal(t, []string{"boom"})

	testkit.AssertEqual(t, "starts on input", m.focus, jqlFocusInput)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	testkit.AssertEqual(t, "error next", m.focus, jqlFocusError)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	testkit.AssertEqual(t, "list next", m.focus, jqlFocusList)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	testkit.AssertEqual(t, "back to input", m.focus, jqlFocusInput)
}

func TestJQLModal_TabSkipsErrorPanelWhenAbsent(t *testing.T) {
	t.Parallel()
	m := NewJQLModal()
	m.SetSize(80, 24)
	m.Show("", []string{testHistoryItem1})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	testkit.AssertEqual(t, "straight to list", m.focus, jqlFocusList)
}

func TestJQLModal_EscFromErrorReturnsToInput(t *testing.T) {
	t.Parallel()
	m := errModal(t, []string{"boom"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	testkit.AssertEqual(t, "focus back on input", m.focus, jqlFocusInput)
	testkit.AssertEqual(t, "modal stays open", m.IsVisible(), true)
	testkit.AssertEqual(t, "error still shown", m.HasError(), true)
	if cmd != nil {
		t.Error("esc from error panel should not emit a command")
	}
}

func TestJQLModal_ErrorPanelScrolls(t *testing.T) {
	t.Parallel()
	m := errModal(t, longError(20))
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	testkit.AssertEqual(t, "scrolled down", m.errOffset, 1)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	testkit.AssertEqual(t, "jumped to bottom", m.errOffset, m.maxErrorOffset())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	testkit.AssertEqual(t, "jumped to top", m.errOffset, 0)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	testkit.AssertEqual(t, "clamped at top", m.errOffset, 0)
}

func TestJQLModal_ErrorClearedOnSubmit(t *testing.T) {
	t.Parallel()
	m := NewJQLModal()
	m.SetSize(80, 24)
	m.Show("project = FOO", nil)
	m.SetError([]string{"boom"})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	testkit.AssertEqual(t, "error cleared", m.HasError(), false)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	if _, ok := cmd().(JQLSubmitMsg); !ok {
		t.Errorf("expected JQLSubmitMsg, got %T", cmd())
	}
}

func TestJQLModal_ErrorClearedOnHide(t *testing.T) {
	t.Parallel()
	m := errModal(t, []string{"boom"})
	m.Hide()
	testkit.AssertEqual(t, "error dropped with the modal", m.HasError(), false)
}

func TestJQLModal_ClickSelectsHistoryRowBelowErrorPanel(t *testing.T) {
	t.Parallel()
	m := errModal(t, []string{"a", "b", "c"})

	listTop := 5 + m.errorHeight() + 2
	m, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      5,
		Y:      listTop + 1,
	})
	testkit.AssertEqual(t, "second history row selected", m.cursor, 1)
	testkit.AssertEqual(t, "focus moved to list", m.focus, jqlFocusList)
}

func TestJQLModal_ClickInsideErrorPanelFocusesIt(t *testing.T) {
	t.Parallel()
	m := errModal(t, []string{"a", "b", "c"})

	m, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      5,
		Y:      5,
	})
	testkit.AssertEqual(t, "focus moved to error", m.focus, jqlFocusError)
}

func TestJQLModal_ErrorRewrapsOnResize(t *testing.T) {
	t.Parallel()
	m := errModal(t, []string{strings.Repeat("word ", 30)})
	wide := len(m.errWrapped)

	m.SetSize(40, 24)
	if len(m.errWrapped) <= wide {
		t.Errorf("narrower modal should wrap into more lines, got %d then %d", wide, len(m.errWrapped))
	}
}
