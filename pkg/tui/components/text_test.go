package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/textfuel/lazyjira/v2/pkg/internal/testkit"
)

func TestTruncateMiddle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string // exact match (empty = skip exact check)
	}{
		{"short no truncation", "hello", 10, "hello"},
		{"exact fit", "hello", 5, "hello"},
		{"basic truncation", "Start Progress to In Progress", 20, ""},
		{"unicode arrow no truncation", "Start Progress → In Progress", 30, "Start Progress → In Progress"},
		{"unicode arrow truncated", "Start Progress → In Progress", 20, ""},
		{"very small", "hello world", 4, ""},
		{"emoji", "Bug 🐛 fix needed", 12, ""},
		{"cyrillic", "Проверка кириллицы тут", 15, ""},
		{"empty", "", 10, ""},
		{"maxWidth 0", "hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TruncateMiddle(tt.input, tt.maxWidth)

			if tt.want != "" && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}

			// Must fit within maxWidth display columns.
			if tt.maxWidth > 0 {
				w := lipgloss.Width(got)
				if w > tt.maxWidth {
					t.Errorf("got %q (width %d), exceeds max %d", got, w, tt.maxWidth)
				}
			}

			// Must never contain replacement character (broken UTF-8).
			for _, r := range got {
				if r == '\uFFFD' {
					t.Errorf("got %q, contains replacement char U+FFFD", got)
					break
				}
			}
		})
	}
}

func TestTruncateEnd(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		maxWidth int
	}{
		{"short", "short", 10},
		{"needs truncation", "a longer string here", 10},
		{"unicode arrow", "→ arrow", 5},
		{"wide chars", "日本語テスト", 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TruncateEnd(tt.input, tt.maxWidth)
			w := lipgloss.Width(got)
			if w > tt.maxWidth {
				t.Errorf("got %q (width %d), exceeds max %d", got, w, tt.maxWidth)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"non positive max returns input", "abcdef", 0, "abcdef"},
		{"shorter than max unchanged", "abc", 5, "abc"},
		{"equal to max unchanged", "abc", 3, "abc"},
		{"longer than max gets ellipsis", "abcdef", 4, "abc…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testkit.AssertEqual(t, "Truncate", Truncate(tt.input, tt.maxLen), tt.want)
		})
	}
}

func TestPanelDimensions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		width            int
		height           int
		wantContentWidth int
		wantInnerHeight  int
	}{
		{"typical panel", 40, 20, 38, 18},
		{"narrow floors width at ten", 5, 5, 10, 3},
		{"short floors height at one", 12, 1, 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			contentWidth, innerHeight := PanelDimensions(tt.width, tt.height)
			testkit.AssertEqual(t, "contentWidth", contentWidth, tt.wantContentWidth)
			testkit.AssertEqual(t, "innerHeight", innerHeight, tt.wantInnerHeight)
		})
	}
}

func TestWrapSegments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		width int
		want  []string
	}{
		{
			name:  "prose wraps at spaces",
			input: "Field 'sprintt' does not exist",
			width: 16,
			want:  []string{"Field 'sprintt' ", "does not exist"},
		},
		{
			name:  "query params break before ampersand",
			input: "GET /search?jql=a&startAt=0&maxResults=50",
			width: 20,
			want:  []string{"GET /search?jql=a", "&startAt=0", "&maxResults=50"},
		},
		{
			name:  "field list breaks after comma",
			input: "&fields=summary,status,priority",
			width: 16,
			want:  []string{"&fields=summary,", "status,priority"},
		},
		{
			name:  "short input stays on one line",
			input: "boom",
			width: 20,
			want:  []string{"boom"},
		},
		{
			name:  "empty input yields nothing",
			input: "",
			width: 20,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WrapSegments(tt.input, tt.width)
			if len(got) != len(tt.want) {
				t.Fatalf("WrapSegments(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWrapSegments_RespectsWidth(t *testing.T) {
	t.Parallel()
	long := "?jql=" + strings.Repeat("a", 100)
	for _, line := range WrapSegments(long, 20) {
		if len([]rune(line)) > 20 {
			t.Errorf("line %q exceeds width 20", line)
		}
	}
}

func TestWrapSegments_NeverSplitsPercentEscape(t *testing.T) {
	t.Parallel()
	// A single unbreakable segment of percent-escaped spaces.
	input := "?jql=" + strings.Repeat("%20", 40)
	for _, line := range WrapSegments(input, 22) {
		if strings.HasSuffix(line, "%") || strings.HasSuffix(line, "%2") {
			t.Errorf("line %q ends inside a percent-escape", line)
		}
	}
}
