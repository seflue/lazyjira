package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/textfuel/lazyjira/v2/pkg/jira"
)

func TestFormatJQLError_MessageFirstThenRequest(t *testing.T) {
	t.Parallel()
	err := &jira.APIError{
		Method:   "GET",
		Path:     "/search/jql?jql=project",
		Status:   400,
		Body:     `{"errorMessages":["Field 'sprintt' does not exist."]}`,
		Messages: []string{"Field 'sprintt' does not exist."},
	}

	got := formatJQLError(err)

	want := []string{
		"Field 'sprintt' does not exist.",
		"",
		"HTTP 400  GET /search/jql?jql=project",
	}
	if len(got) != len(want) {
		t.Fatalf("formatJQLError = %q, want %q", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFormatJQLError_FallsBackToBody(t *testing.T) {
	t.Parallel()
	err := &jira.APIError{
		Method: "GET",
		Path:   "/search/jql",
		Status: 500,
		Body:   "upstream exploded",
	}

	got := formatJQLError(err)

	if got[0] != "upstream exploded" {
		t.Errorf("first line = %q, want the raw body", got[0])
	}
	if !strings.Contains(strings.Join(got, "\n"), "HTTP 500") {
		t.Errorf("formatJQLError = %q, want the status", got)
	}
}

func TestFormatJQLError_PlainError(t *testing.T) {
	t.Parallel()
	got := formatJQLError(errors.New("dial tcp: connection refused"))

	if len(got) != 1 || got[0] != "dial tcp: connection refused" {
		t.Errorf("formatJQLError = %q", got)
	}
}
