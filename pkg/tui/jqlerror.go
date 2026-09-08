package tui

import (
	"fmt"

	"github.com/textfuel/lazyjira/v2/pkg/jira"
)

// formatJQLError turns a failed search into the logical lines shown in the JQL
// modal's error panel: Jira's own messages first, the raw body when it sent
// none, then the request that produced them, separated by a blank line.
// Wrapping to the panel width happens at render time.
func formatJQLError(err error) []string {
	detail := jira.DescribeError(err)

	var lines []string
	switch {
	case len(detail.Messages) > 0:
		lines = append(lines, detail.Messages...)
	case detail.Body != "":
		lines = append(lines, detail.Body)
	}

	if detail.Request == "" {
		return lines
	}

	if len(lines) > 0 {
		lines = append(lines, "")
	}
	if detail.Status > 0 {
		return append(lines, fmt.Sprintf("HTTP %d  %s", detail.Status, detail.Request))
	}
	return append(lines, detail.Request)
}
