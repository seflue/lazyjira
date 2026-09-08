package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// APIError is a non-2xx response from the Jira REST API. It keeps the raw body
// for the unchanged Error() string while exposing the parsed, user-facing
// messages from the error envelope via Messages.
type APIError struct {
	Method   string
	Path     string
	Status   int
	Body     string
	Messages []string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request %s %s returned status %d: %s",
		e.Method, e.Path, e.Status, e.Body)
}

func newAPIError(method, path string, status int, body string) *APIError {
	return &APIError{
		Method:   method,
		Path:     path,
		Status:   status,
		Body:     body,
		Messages: parseErrorMessages(body),
	}
}

// parseErrorMessages extracts the human-readable messages from a Jira REST
// error envelope ({"errorMessages": [...], "errors": {...}}). Field errors are
// appended in key order for determinism. Returns nil when the body is not such
// an envelope.
func parseErrorMessages(body string) []string {
	var envelope struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return nil
	}

	messages := envelope.ErrorMessages

	keys := make([]string, 0, len(envelope.Errors))
	for k := range envelope.Errors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		messages = append(messages, envelope.Errors[k])
	}

	return messages
}

// ErrorDetail is a failed request broken into its display parts. Messages holds
// the user-facing text Jira returned and is empty when the response carried no
// error envelope; Request and Body describe the exchange itself.
type ErrorDetail struct {
	Messages []string
	Request  string
	Status   int
	Body     string
}

// DescribeError splits err into its display parts. Anything that is not an
// *APIError becomes a single message, since its text is already user-facing.
func DescribeError(err error) ErrorDetail {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return ErrorDetail{Messages: []string{err.Error()}}
	}
	return ErrorDetail{
		Messages: apiErr.Messages,
		Request:  apiErr.Method + " " + apiErr.Path,
		Status:   apiErr.Status,
		Body:     apiErr.Body,
	}
}
