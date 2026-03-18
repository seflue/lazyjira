package components

// AdjustOffset ensures cursor is visible within the viewport with a margin.
// Keeps 1 item of context above/below when possible (like lazygit).
func AdjustOffset(cursor, offset, visible, total int) int {
	if total <= visible {
		return 0
	}
	margin := 1
	if visible <= 3 {
		margin = 0
	}

	// Cursor too close to top — scroll up.
	if cursor < offset+margin {
		offset = cursor - margin
	}
	// Cursor too close to bottom — scroll down.
	if cursor >= offset+visible-margin {
		offset = cursor - visible + margin + 1
	}
	// Clamp.
	if offset < 0 {
		offset = 0
	}
	if offset > total-visible {
		offset = total - visible
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}
