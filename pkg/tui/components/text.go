package components

import (
	"slices"
	"strings"
)

import "github.com/charmbracelet/lipgloss"

// TruncateEnd truncates s to fit within maxWidth display columns,
// appending "…" if needed. Respects multi-byte UTF-8 characters.
func TruncateEnd(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i])
		if lipgloss.Width(candidate)+1 <= maxWidth { // +1 for "…"
			return candidate + "…"
		}
	}
	return "…"
}

// TruncateMiddle truncates keeping start and end visible: "abcdef...xyz"
// Uses display width (not byte count) so multi-byte chars like → are handled correctly.
func TruncateMiddle(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth < 5 {
		runes := []rune(s)
		if len(runes) > maxWidth {
			return string(runes[:maxWidth])
		}
		return s
	}
	runes := []rune(s)
	ellipsis := "..."
	ellipsisW := 3
	budget := maxWidth - ellipsisW
	startBudget := (budget + 1) / 2 // start gets the extra column on odd budget
	endBudget := budget - startBudget

	// Build start: runes from the beginning.
	var start []rune
	w := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > startBudget {
			break
		}
		start = append(start, r)
		w += rw
	}

	// Build end: runes from the end.
	var end []rune
	w = 0
	for _, v := range slices.Backward(runes) {
		rw := lipgloss.Width(string(v))
		if w+rw > endBudget {
			break
		}
		end = append([]rune{v}, end...)
		w += rw
	}

	return string(start) + ellipsis + string(end)
}

// Truncate truncates s to max bytes, appending "…" if needed.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}

// PanelDimensions computes usable content width and inner height from total panel dimensions.
func PanelDimensions(width, height int) (contentWidth, innerHeight int) {
	return max(width-2, 10), max(height-2, 1)
}

// WrapSegments breaks s into lines of at most width columns. Breaks are placed
// at spaces and at URL separators -- before '?' and '&', after ',' -- so query
// parameters and field lists stay readable instead of being cut mid-token. A
// segment longer than width is split at width, never inside a percent-escape.
func WrapSegments(s string, width int) []string {
	if width < 1 || s == "" {
		return nil
	}

	var lines []string
	for _, seg := range splitSegments(s) {
		if len(lines) > 0 {
			last := lines[len(lines)-1]
			if len([]rune(last))+len([]rune(seg)) <= width {
				lines[len(lines)-1] = last + seg
				continue
			}
		}
		lines = append(lines, splitAtWidth(strings.TrimLeft(seg, " "), width)...)
	}
	return lines
}

// splitSegments cuts s before '?' and '&' and after ',' and ' ', keeping the
// separators attached so joining the segments reproduces s.
func splitSegments(s string) []string {
	runes := []rune(s)
	var segs []string
	start := 0
	for i, r := range runes {
		switch r {
		case '?', '&':
			if i > start {
				segs = append(segs, string(runes[start:i]))
				start = i
			}
		case ',', ' ':
			segs = append(segs, string(runes[start:i+1]))
			start = i + 1
		}
	}
	if start < len(runes) {
		segs = append(segs, string(runes[start:]))
	}
	return segs
}

// splitAtWidth chops s into width-sized chunks, backing off so a cut never
// lands inside a percent-escape.
func splitAtWidth(s string, width int) []string {
	runes := []rune(s)
	var out []string
	for len(runes) > width {
		cut := width
		for cut > 1 && (runes[cut-1] == '%' || (cut > 2 && runes[cut-2] == '%')) {
			cut--
		}
		out = append(out, string(runes[:cut]))
		runes = runes[cut:]
	}
	if len(runes) > 0 {
		out = append(out, string(runes))
	}
	return out
}
