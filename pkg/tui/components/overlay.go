package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Overlay places fg centered on top of bg, preserving bg content around fg.
// width and height define the visible area dimensions.
func Overlay(bg, fg string, width, height int) string {
	fgLines := strings.Split(fg, "\n")
	fgW := lipgloss.Width(fg)

	x := (width - fgW) / 2
	y := (height - len(fgLines)) / 2

	return overlayLines(bg, fgLines, x, y, width, height)
}

// OverlayAt places fg at position (x, y) on top of bg.
func OverlayAt(bg, fg string, x, y, width, height int) string {
	return overlayLines(bg, strings.Split(fg, "\n"), x, y, width, height)
}

func overlayLines(bg string, fgLines []string, x, y, width, height int) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bgLines := strings.Split(bg, "\n")

	// Ensure bg has enough lines.
	for len(bgLines) < height {
		bgLines = append(bgLines, "")
	}

	for i, fgLine := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		bgLines[row] = overlayLine(bgLines[row], fgLine, x, width)
	}

	return strings.Join(bgLines[:height], "\n")
}

// overlayLine places fgLine at column x over bgLine, preserving bg on both sides.
func overlayLine(bgLine, fgLine string, x, totalW int) string {
	bgW := ansi.StringWidth(bgLine)
	fgW := ansi.StringWidth(fgLine)

	// Left part: first x columns of bg.
	var left string
	if x > 0 {
		left = ansi.Truncate(bgLine, x, "")
		// Pad if bg is shorter than x.
		leftW := ansi.StringWidth(left)
		if leftW < x {
			left += strings.Repeat(" ", x-leftW)
		}
	}

	// Right part: bg content after x+fgW.
	rightStart := x + fgW
	var right string
	if rightStart < bgW {
		right = ansi.TruncateLeft(bgLine, rightStart, "")
	}
	rightW := ansi.StringWidth(right)
	// Pad right to fill totalW if needed.
	used := x + fgW + rightW
	if used < totalW {
		right += strings.Repeat(" ", totalW-used)
	}

	return left + fgLine + right
}
