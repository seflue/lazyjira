package components

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/textfuel/lazyjira/pkg/tui/theme"
)

// ScrollInfo provides data for rendering a scrollbar in the right border.
type ScrollInfo struct {
	Total   int // total items/lines
	Visible int // visible items/lines
	Offset  int // scroll offset (first visible item index)
}

// RenderPanel draws a bordered panel with title in the top border.
func RenderPanel(title, content string, width, innerHeight int, focused bool) string {
	return RenderPanelFull(title, "", content, width, innerHeight, focused, nil)
}

// RenderPanelWithFooter draws a panel with title and footer.
func RenderPanelWithFooter(title, footer, content string, width, innerHeight int, focused bool) string {
	return RenderPanelFull(title, footer, content, width, innerHeight, focused, nil)
}

// RenderPanelFull draws a panel with title, footer, and optional scrollbar.
func RenderPanelFull(title, footer, content string, width, innerHeight int, focused bool, scroll *ScrollInfo) string {
	th := theme.DefaultTheme()

	borderColor := theme.ColorNone
	if focused {
		borderColor = theme.ColorGreen
	}

	var styledTitle string
	if focused {
		styledTitle = th.Title.Render(title)
	} else {
		styledTitle = lipgloss.NewStyle().Foreground(borderColor).Render(title)
	}

	contentWidth := width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Top border.
	titleLen := lipgloss.Width(styledTitle)
	topPadding := contentWidth - titleLen
	if topPadding < 0 {
		topPadding = 0
	}
	topLine := borderStyle.Render("╭") +
		styledTitle +
		borderStyle.Render(strings.Repeat("─", topPadding)+"╮")

	// Content lines.
	lines := strings.Split(content, "\n")
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}

	// Compute scrollbar — same algorithm as lazygit's gocui.
	showScroll := scroll != nil && scroll.Total > scroll.Visible && innerHeight > 0
	var thumbStart, thumbEnd int
	if showScroll {
		listSize := scroll.Total
		pageSize := scroll.Visible
		position := scroll.Offset
		scrollArea := innerHeight

		// Thumb height proportional to visible/total.
		thumbH := int(float64(pageSize) / float64(listSize) * float64(scrollArea))
		if thumbH < 1 {
			thumbH = 1
		}

		// Thumb position — snap to bottom at end.
		maxPos := listSize - pageSize
		if maxPos <= 0 {
			thumbStart = 0
		} else if position >= maxPos {
			thumbStart = scrollArea - thumbH
		} else {
			thumbStart = int(math.Ceil(float64(position) / float64(maxPos) * float64(scrollArea-thumbH-1)))
		}

		thumbEnd = thumbStart + thumbH
		if thumbEnd > scrollArea {
			thumbEnd = scrollArea
		}
	}

	borderVert := borderStyle.Render("│")
	thumbChar := borderStyle.Render("▐")
	var body strings.Builder
	for i, line := range lines {
		rendered := line
		lineWidth := lipgloss.Width(rendered)
		if lineWidth < contentWidth {
			rendered += strings.Repeat(" ", contentWidth-lineWidth)
		}
		// Right border: scrollbar or normal.
		rightBorder := borderVert
		if showScroll && i >= thumbStart && i < thumbEnd {
			rightBorder = thumbChar
		}
		body.WriteString(borderVert + rendered + rightBorder + "\n")
	}

	// Bottom border.
	var bottomLine string
	if footer != "" {
		styledFooter := borderStyle.Render(footer)
		footerLen := lipgloss.Width(styledFooter)
		padding := contentWidth - footerLen
		if padding < 0 {
			padding = 0
		}
		bottomLine = borderStyle.Render("╰"+strings.Repeat("─", padding)) +
			styledFooter +
			borderStyle.Render("╯")
	} else {
		bottomLine = borderStyle.Render("╰" + strings.Repeat("─", contentWidth) + "╯")
	}

	return topLine + "\n" + body.String() + bottomLine
}
