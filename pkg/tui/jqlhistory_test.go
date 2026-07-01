package tui

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/textfuel/lazyjira/v2/pkg/internal/testkit"
)

func TestAddToHistory_PrependsDeduplicated(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		existing []string
		newQuery string
		want     []string
	}{
		{
			name:     "empty history gains one entry",
			existing: nil,
			newQuery: "project = X",
			want:     []string{"project = X"},
		},
		{
			name:     "new query prepended",
			existing: []string{"project = A", "project = B"},
			newQuery: "project = C",
			want:     []string{"project = C", "project = A", "project = B"},
		},
		{
			name:     "duplicate moved to front",
			existing: []string{"project = A", "project = B"},
			newQuery: "project = B",
			want:     []string{"project = B", "project = A"},
		},
		{
			name:     "blank query is no-op",
			existing: []string{"project = A"},
			newQuery: "   ",
			want:     []string{"project = A"},
		},
		{
			name:     "whitespace trimmed before dedup",
			existing: []string{"project = X"},
			newQuery: "  project = X  ",
			want:     []string{"project = X"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := AddToHistory(testCase.existing, testCase.newQuery)
			testkit.AssertSliceEqual(t, "history", got, testCase.want)
		})
	}
}

func TestAddToHistory_CapsAt50(t *testing.T) {
	t.Parallel()
	existing := make([]string, 50)
	for i := range existing {
		existing[i] = "query"
	}
	existing[0] = "first"
	existing[49] = "last"

	result := AddToHistory(existing, "brand-new")

	if len(result) != 50 {
		t.Errorf("len = %d, want 50", len(result))
	}
	if result[0] != "brand-new" {
		t.Errorf("result[0] = %q, want brand-new", result[0])
	}
}

func TestSaveAndLoadJQLHistory_RoundTrip(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())

	queries := []string{"project = X ORDER BY updated DESC", "assignee = currentUser()"}

	if err := SaveJQLHistory(queries); err != nil {
		t.Fatalf("SaveJQLHistory: %v", err)
	}

	loaded := LoadJQLHistory()
	testkit.AssertSliceEqual(t, "loaded queries", loaded, queries)
}

func TestLoadJQLHistory_MissingFileReturnsNil(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())

	result := LoadJQLHistory()
	if result != nil {
		t.Errorf("expected nil for missing file, got %v", result)
	}
}

func TestSaveAndLoadJQLHistory_MultilineRoundTrip(t *testing.T) {
	t.Setenv("LAZYJIRA_CONFIG_DIR", t.TempDir())

	queries := []string{
		"project = FOO\nAND status = Open",
		"assignee = currentUser()",
	}

	if err := SaveJQLHistory(queries); err != nil {
		t.Fatalf("SaveJQLHistory: %v", err)
	}

	loaded := LoadJQLHistory()
	testkit.AssertSliceEqual(t, "loaded queries", loaded, queries)
}

func TestLoadJQLHistory_LegacyNewlineFormat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LAZYJIRA_CONFIG_DIR", dir)

	// Old on-disk format: newline-delimited, one entry per line, with a
	// trailing blank line to exercise empty-line dropping.
	legacy := "project = A\nproject = B\n"
	if err := os.WriteFile(filepath.Join(dir, jqlHistoryFile), []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	loaded := LoadJQLHistory()
	testkit.AssertSliceEqual(t, "loaded queries", loaded, []string{"project = A", "project = B"})
}

func TestLoadJQLHistory_EmptyFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LAZYJIRA_CONFIG_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, jqlHistoryFile), nil, 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	if result := LoadJQLHistory(); result != nil {
		t.Errorf("expected nil for empty file, got %v", result)
	}
}

func TestLoadJQLHistory_MigratesLegacyToYAMLOnSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LAZYJIRA_CONFIG_DIR", dir)

	legacy := "project = A\nproject = B\n"
	path := filepath.Join(dir, jqlHistoryFile)
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	loaded := LoadJQLHistory()
	if err := SaveJQLHistory(loaded); err != nil {
		t.Fatalf("SaveJQLHistory: %v", err)
	}

	// File is now valid YAML sequence...
	data, err := os.ReadFile(path) //nolint:gosec // test reads a temp file it just wrote
	if err != nil {
		t.Fatalf("read migrated file: %v", err)
	}
	var asYAML []string
	if err := yaml.Unmarshal(data, &asYAML); err != nil {
		t.Fatalf("migrated file is not a YAML sequence: %v", err)
	}

	// ...and re-loads identically.
	reloaded := LoadJQLHistory()
	testkit.AssertSliceEqual(t, "reloaded queries", reloaded, []string{"project = A", "project = B"})
}
