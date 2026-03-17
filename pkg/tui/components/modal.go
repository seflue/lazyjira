package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModalItem is one option in the modal.
type ModalItem struct {
	ID    string
	Label string
}

// ModalSelectedMsg is sent when user picks an item.
type ModalSelectedMsg struct {
	Item ModalItem
}

// ModalCancelledMsg is sent when user presses Esc.
type ModalCancelledMsg struct{}

// Modal is a centered popup list for picking an option (transitions, etc).
type Modal struct {
	title   string
	items   []ModalItem
	cursor  int
	visible bool
	width   int
	height  int
}

func NewModal() Modal {
	return Modal{}
}

func (m *Modal) Show(title string, items []ModalItem) {
	m.title = title
	m.items = items
	m.cursor = 0
	m.visible = true
}

func (m *Modal) Hide()          { m.visible = false }
func (m *Modal) IsVisible() bool { return m.visible }

func (m *Modal) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Modal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if !m.visible {
		return *m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.items) {
				selected := m.items[m.cursor]
				m.visible = false
				return *m, func() tea.Msg { return ModalSelectedMsg{Item: selected} }
			}
		case "esc", "q":
			m.visible = false
			return *m, func() tea.Msg { return ModalCancelledMsg{} }
		}
	}
	return *m, nil
}

func (m *Modal) View() string {
	if !m.visible || len(m.items) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("4"))

	normalStyle := lipgloss.NewStyle()

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	var lines []string
	lines = append(lines, " "+titleStyle.Render(m.title))
	lines = append(lines, "")

	for i, item := range m.items {
		label := fmt.Sprintf(" %s", item.Label)
		if i == m.cursor {
			lines = append(lines, selectedStyle.Render(label))
		} else {
			lines = append(lines, normalStyle.Render(label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, " "+dimStyle.Render("enter: select | esc: cancel"))

	content := strings.Join(lines, "\n")

	popupW := 40
	for _, item := range m.items {
		if len(item.Label)+4 > popupW {
			popupW = len(item.Label) + 4
		}
	}
	if popupW > m.width-4 {
		popupW = m.width - 4
	}
	popupH := len(lines)
	if popupH > m.height-4 {
		popupH = m.height - 4
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Width(popupW).
		Height(popupH).
		Render(content)
}
