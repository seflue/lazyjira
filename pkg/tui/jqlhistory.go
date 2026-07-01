package tui

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/textfuel/lazyjira/v2/pkg/config"
)

const (
	jqlHistoryFile    = "jql_history"
	jqlHistoryMaxSize = 50
)

// LoadJQLHistory loads JQL queries from the history file.
// The current format is a YAML list of strings; legacy newline-delimited files
// are read via the fallback below and migrated on the next SaveJQLHistory.
// Returns nil on error, missing or empty file.
func LoadJQLHistory() []string {
	path := filepath.Join(config.ConfigDir(), jqlHistoryFile)
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil
	}
	var queries []string
	if err := yaml.Unmarshal(data, &queries); err == nil {
		return queries
	}
	// Legacy format: a plain newline-delimited scalar, not a YAML sequence.
	lines := strings.Split(string(data), "\n")
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// SaveJQLHistory writes queries to the history file as a YAML list, so entries
// may contain embedded newlines (multi-line JQL).
func SaveJQLHistory(queries []string) error {
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(queries)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, jqlHistoryFile)
	return os.WriteFile(path, data, 0o644)
}

// AddToHistory prepends a new query to the history, deduplicating.
// Returns the updated history capped at 50 entries
func AddToHistory(history []string, newQuery string) []string {
	newQuery = strings.TrimSpace(newQuery)
	if newQuery == "" {
		return history
	}
	var result []string
	result = append(result, newQuery)
	for _, q := range history {
		if q != newQuery {
			result = append(result, q)
		}
	}
	if len(result) > jqlHistoryMaxSize {
		result = result[:jqlHistoryMaxSize]
	}
	return result
}
