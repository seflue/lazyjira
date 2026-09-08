package jira

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/textfuel/lazyjira/v2/pkg/internal/testkit"
)

func TestParseErrorMessages_ErrorMessagesArray(t *testing.T) {
	t.Parallel()

	body := `{"errorMessages":["You cannot create issues in this project."],"errors":{}}`
	got := parseErrorMessages(body)

	want := []string{"You cannot create issues in this project."}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("parseErrorMessages(%q) = %v, want %v", body, got, want)
	}
}

func TestParseErrorMessages_FieldErrorsMap(t *testing.T) {
	t.Parallel()

	body := `{"errorMessages":[],"errors":{"parent":"Issue does not exist or you do not have permission to see it."}}`
	got := parseErrorMessages(body)

	want := []string{"Issue does not exist or you do not have permission to see it."}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("parseErrorMessages(%q) = %v, want %v", body, got, want)
	}
}

func TestParseErrorMessages_NonEnvelopeReturnsNil(t *testing.T) {
	t.Parallel()

	if got := parseErrorMessages("plain text, not json"); got != nil {
		t.Errorf("parseErrorMessages(non-json) = %v, want nil", got)
	}
}

func TestClient_HTTPError_ExposesAPIErrorWithMessages(t *testing.T) {
	t.Parallel()

	body := `{"errorMessages":["You cannot create issues in this project."]}`
	client, _ := newRecordingClient(t, cloudOpts(),
		testkit.StubResponse{Status: http.StatusNotFound, Body: body})

	_, err := client.GetCreateMeta(t.Context(), errorTestProjectKey, "10001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %v is not unwrappable to *APIError", err)
	}
	if len(apiErr.Messages) != 1 ||
		apiErr.Messages[0] != "You cannot create issues in this project." {
		t.Errorf("apiErr.Messages = %v", apiErr.Messages)
	}

	// Backward compatibility: the Error() string keeps status and wrapper.
	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("Error() %q missing 'status 404'", err.Error())
	}
	if !strings.Contains(err.Error(), "get create meta") {
		t.Errorf("Error() %q missing wrapper prefix", err.Error())
	}
}

func TestDescribeError_SplitsAPIError(t *testing.T) {
	t.Parallel()
	body := `{"errorMessages":["Field 'sprintt' does not exist or you do not have permission to view it."],"errors":{}}`
	err := fmt.Errorf("search issues: %w",
		newAPIError("GET", "/search/jql?jql=project%20%3D%20FOO", 400, body))

	got := DescribeError(err)

	if len(got.Messages) != 1 {
		t.Fatalf("Messages = %q, want one entry", got.Messages)
	}
	if !strings.Contains(got.Messages[0], "Field 'sprintt' does not exist") {
		t.Errorf("Messages[0] = %q, want Jira's own text", got.Messages[0])
	}
	if strings.Contains(got.Messages[0], "/search/jql") {
		t.Errorf("Messages[0] = %q, should not carry the request URL", got.Messages[0])
	}
	if got.Status != 400 {
		t.Errorf("Status = %d, want 400", got.Status)
	}
	if got.Request != "GET /search/jql?jql=project%20%3D%20FOO" {
		t.Errorf("Request = %q", got.Request)
	}
	if got.Body != body {
		t.Errorf("Body = %q, want the raw response", got.Body)
	}
}

func TestDescribeError_NoEnvelopeKeepsBody(t *testing.T) {
	t.Parallel()
	err := newAPIError("GET", "/search/jql", 500, "upstream exploded")

	got := DescribeError(err)

	if len(got.Messages) != 0 {
		t.Errorf("Messages = %q, want none for a non-envelope body", got.Messages)
	}
	if got.Body != "upstream exploded" {
		t.Errorf("Body = %q", got.Body)
	}
}

func TestDescribeError_PlainErrorBecomesMessage(t *testing.T) {
	t.Parallel()
	got := DescribeError(errors.New("dial tcp: connection refused"))

	if len(got.Messages) != 1 || got.Messages[0] != "dial tcp: connection refused" {
		t.Errorf("Messages = %q", got.Messages)
	}
	if got.Request != "" {
		t.Errorf("Request = %q, want empty", got.Request)
	}
}
