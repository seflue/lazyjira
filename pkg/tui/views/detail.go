package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cockroach-eater/lazyjira/pkg/jira"
	"github.com/cockroach-eater/lazyjira/pkg/tui/components"
	"github.com/cockroach-eater/lazyjira/pkg/tui/theme"
)

type DetailTab int

const (
	TabDetails  DetailTab = iota
	TabSubtasks
	TabComments
	TabLinks
	TabInfo
	TabHistory
	tabCount = 6
)

// MainMode controls what the right panel displays.
type MainMode int

const (
	ModeIssue   MainMode = iota
	ModeSplash
	ModeProject
)

// SplashInfo holds data for the splash/status screen.
type SplashInfo struct {
	AuthMethod string // "API Token" or "OAuth" or "env vars"
	Host       string
	Email      string
	Project    string
}

type DetailView struct {
	issue     *jira.Issue
	project   *jira.Project
	splash    SplashInfo
	mode      MainMode
	activeTab DetailTab
	scrollY   int
	width     int
	height    int
	focused   bool
	theme     *theme.Theme
}

func NewDetailView() *DetailView {
	return &DetailView{theme: theme.DefaultTheme(), mode: ModeIssue}
}

func (d *DetailView) SetIssue(issue *jira.Issue) {
	prevKey := ""
	if d.issue != nil {
		prevKey = d.issue.Key
	}
	d.issue = issue
	d.mode = ModeIssue
	// Only reset tab/scroll when switching to a different issue.
	if issue == nil || issue.Key != prevKey {
		d.scrollY = 0
		d.activeTab = TabDetails
	}
}

// UpdateIssueData stores issue data without changing mode (for background updates).
func (d *DetailView) UpdateIssueData(issue *jira.Issue) {
	prevKey := ""
	if d.issue != nil {
		prevKey = d.issue.Key
	}
	d.issue = issue
	if issue != nil && issue.Key != prevKey {
		d.scrollY = 0
		d.activeTab = TabDetails
	}
}

func (d *DetailView) SetProject(project *jira.Project) {
	d.project = project
	d.mode = ModeProject
	d.scrollY = 0
}

func (d *DetailView) SetSplash(info SplashInfo) {
	d.splash = info
	d.mode = ModeSplash
	d.scrollY = 0
}

func (d *DetailView) SetSize(w, h int)       { d.width = w; d.height = h }
func (d *DetailView) SetFocused(focused bool) { d.focused = focused }
func (d *DetailView) Init() tea.Cmd           { return nil }

func (d *DetailView) NextTab() {
	vt := d.visibleTabs()
	for i, t := range vt {
		if t == d.activeTab {
			d.activeTab = vt[(i+1)%len(vt)]
			d.scrollY = 0
			return
		}
	}
	if len(vt) > 0 {
		d.activeTab = vt[0]
		d.scrollY = 0
	}
}

func (d *DetailView) PrevTab() {
	vt := d.visibleTabs()
	for i, t := range vt {
		if t == d.activeTab {
			d.activeTab = vt[(i+len(vt)-1)%len(vt)]
			d.scrollY = 0
			return
		}
	}
	if len(vt) > 0 {
		d.activeTab = vt[0]
		d.scrollY = 0
	}
}

func (d *DetailView) visibleTabs() []DetailTab {
	labels := d.tabLabels()
	tabs := make([]DetailTab, len(labels))
	for i, l := range labels {
		tabs[i] = l.tab
	}
	return tabs
}

// ClickTab switches tab based on x position in the title bar.
func (d *DetailView) ClickTab(x int) {
	if d.issue == nil {
		return
	}
	// Reconstruct tab labels and their positions in the title.
	type tabPos struct {
		tab   DetailTab
		start int
		end   int
	}
	labels := d.tabLabels()
	prefix := fmt.Sprintf("[0] %s", d.issue.Key)
	sep := " - "
	pos := len(prefix) + len(sep)

	var positions []tabPos
	for i, tl := range labels {
		end := pos + len(tl.label)
		positions = append(positions, tabPos{tab: tl.tab, start: pos, end: end})
		if i < len(labels)-1 {
			pos = end + len(sep)
		}
	}

	for _, p := range positions {
		if x >= p.start && x < p.end {
			d.activeTab = p.tab
			d.scrollY = 0
			return
		}
	}
}

func (d *DetailView) ScrollBy(delta int) {
	d.scrollY += delta
	if d.scrollY < 0 {
		d.scrollY = 0
	}
}

func (d *DetailView) Update(msg tea.Msg) (*DetailView, tea.Cmd) {
	if !d.focused {
		return d, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			d.scrollY++
		case "k", "up":
			if d.scrollY > 0 {
				d.scrollY--
			}
		case "tab":
			d.activeTab = (d.activeTab + 1) % tabCount
			d.scrollY = 0
		case "i":
			d.activeTab = TabInfo
			d.scrollY = 0
		case "ctrl+d":
			d.scrollY += d.visibleRows() / 2
		case "ctrl+u":
			d.scrollY -= d.visibleRows() / 2
			if d.scrollY < 0 {
				d.scrollY = 0
			}
		}
	}
	return d, nil
}

func (d *DetailView) visibleRows() int {
	// Total height = innerHeight + 2 (borders). Tabs are in the title now.
	rows := d.height - 2
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (d *DetailView) View() string {
	contentWidth := d.width - 2
	if contentWidth < 10 {
		contentWidth = 10
	}
	innerH := d.height - 2
	if innerH < 1 {
		innerH = 1
	}

	// Splash mode.
	if d.mode == ModeSplash {
		return d.renderSplash(contentWidth, innerH)
	}

	// Project mode.
	if d.mode == ModeProject && d.project != nil {
		return d.renderProjectView(contentWidth, innerH)
	}

	visible := d.visibleRows()

	// Issue mode.
	if d.issue == nil {
		title := "[0] Detail"
		placeholder := lipgloss.NewStyle().Foreground(theme.ColorGray).Render("Select an issue to view details")
		return components.RenderPanel(title, placeholder, d.width, innerH, d.focused)
	}

	// Build title: [0] KEY - Tab - Tab - Tab
	title := d.buildTitle(contentWidth)

	// Content lines.
	var contentLines []string
	switch d.activeTab {
	case TabDetails:
		contentLines = d.renderDescription(contentWidth)
	case TabSubtasks:
		contentLines = d.renderSubtasks(contentWidth)
	case TabComments:
		contentLines = d.renderComments(contentWidth)
	case TabLinks:
		contentLines = d.renderLinks(contentWidth)
	case TabInfo:
		contentLines = d.renderInfo(contentWidth)
	case TabHistory:
		contentLines = d.renderHistory(contentWidth)
	}

	// Apply scroll.
	if d.scrollY > len(contentLines) {
		d.scrollY = len(contentLines)
	}
	if d.scrollY < 0 {
		d.scrollY = 0
	}
	scrolled := contentLines
	if d.scrollY < len(scrolled) {
		scrolled = scrolled[d.scrollY:]
	} else {
		scrolled = nil
	}
	if len(scrolled) > visible {
		scrolled = scrolled[:visible]
	}

	body := strings.Join(scrolled, "\n")

	return components.RenderPanel(title, body, d.width, innerH, d.focused)
}

type tabLabel struct {
	tab   DetailTab
	label string
}

func (d *DetailView) tabLabels() []tabLabel {
	var tabs []tabLabel
	tabs = append(tabs, tabLabel{TabDetails, "Body"})
	if d.issue != nil {
		if n := len(d.issue.Subtasks); n > 0 {
			tabs = append(tabs, tabLabel{TabSubtasks, fmt.Sprintf("Sub(%d)", n)})
		}
		if n := len(d.issue.Comments); n > 0 {
			tabs = append(tabs, tabLabel{TabComments, fmt.Sprintf("Cmt(%d)", n)})
		}
		if n := len(d.issue.IssueLinks); n > 0 {
			tabs = append(tabs, tabLabel{TabLinks, fmt.Sprintf("Lnk(%d)", n)})
		}
	}
	tabs = append(tabs, tabLabel{TabInfo, "Info"})
	if d.issue != nil && len(d.issue.Changelog) > 0 {
		tabs = append(tabs, tabLabel{TabHistory, fmt.Sprintf("Hist(%d)", len(d.issue.Changelog))})
	}
	return tabs
}

func (d *DetailView) buildTitle(maxWidth int) string {
	tabs := d.tabLabels()

	activeStyle := lipgloss.NewStyle().Foreground(theme.ColorGreen).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)
	sepStyle := lipgloss.NewStyle().Foreground(theme.ColorGray)

	prefix := fmt.Sprintf("[0] %s", d.issue.Key)

	var tabParts []string
	for _, t := range tabs {
		if t.tab == d.activeTab {
			tabParts = append(tabParts, activeStyle.Render(t.label))
		} else {
			tabParts = append(tabParts, inactiveStyle.Render(t.label))
		}
	}

	sep := sepStyle.Render(" - ")
	return prefix + sep + strings.Join(tabParts, sep)
}

func (d *DetailView) renderDescription(width int) []string {
	valStyle := d.theme.ValueStyle
	var lines []string

	desc := d.issue.Description
	if desc == "" {
		desc = "(no description)"
	}
	for _, line := range wrapText(desc, width-2) {
		lines = append(lines, " "+colorMentions(valStyle.Render(line)))
	}
	return lines
}

// colorMentions replaces \x00MENTION:@Name\x00 markers with colored author names.
func colorMentions(s string) string {
	const prefix = "\x00MENTION:"
	const suffix = "\x00"
	result := s
	for {
		start := strings.Index(result, prefix)
		if start == -1 {
			break
		}
		rest := result[start+len(prefix):]
		end := strings.Index(rest, suffix)
		if end == -1 {
			break
		}
		name := rest[:end]
		colored := theme.AuthorRender(name)
		result = result[:start] + colored + rest[end+len(suffix):]
	}
	return result
}

func (d *DetailView) renderSubtasks(width int) []string {
	if d.issue == nil || len(d.issue.Subtasks) == 0 {
		return []string{" No subtasks."}
	}
	var lines []string
	for _, sub := range d.issue.Subtasks {
		emoji := statusEmoji(sub.Status)
		lines = append(lines, fmt.Sprintf(" %s %s: %s", emoji, sub.Key, sub.Summary))
	}
	return lines
}

func (d *DetailView) renderInfo(width int) []string {
	issue := d.issue
	valStyle := d.theme.ValueStyle

	var lines []string

	statusName := "Unknown"
	if issue.Status != nil {
		statusName = theme.StatusColor(issue.Status.CategoryKey).Render(issue.Status.Name)
	}
	priorityName := "None"
	if issue.Priority != nil {
		priorityName = d.priorityStyled(issue.Priority.Name)
	}
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Status:", statusName))
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Priority:", priorityName))

	assignee := "Unassigned"
	if issue.Assignee != nil {
		assignee = theme.AuthorRender(issue.Assignee.DisplayName)
	}
	reporter := "Unknown"
	if issue.Reporter != nil {
		reporter = theme.AuthorRender(issue.Reporter.DisplayName)
	}
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Assignee:", assignee))
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Reporter:", reporter))

	typeName := "Unknown"
	if issue.IssueType != nil {
		typeName = issue.IssueType.Name
	}
	sprintName := "None"
	if issue.Sprint != nil {
		sprintName = issue.Sprint.Name
	}
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Type:", valStyle.Render(typeName)))
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Sprint:", valStyle.Render(sprintName)))

	if len(issue.Labels) > 0 {
		lines = append(lines, fmt.Sprintf(" %-11s %s", "Labels:", valStyle.Render(strings.Join(issue.Labels, ", "))))
	}

	if len(issue.Components) > 0 {
		var names []string
		for _, c := range issue.Components {
			names = append(names, c.Name)
		}
		lines = append(lines, fmt.Sprintf(" %-11s %s", "Components:", valStyle.Render(strings.Join(names, ", "))))
	}

	return lines
}

// renderEntry renders a single author+time header + content block + separator.
func renderEntry(author string, created time.Time, content []string, width int, last bool) []string {
	gray := lipgloss.NewStyle().Foreground(theme.ColorGray)
	var lines []string
	lines = append(lines, " "+theme.AuthorRender(author)+" "+gray.Render(timeAgo(created)))
	lines = append(lines, content...)
	if !last {
		lines = append(lines, " "+strings.Repeat("─", width-2))
	}
	return lines
}

func (d *DetailView) renderHistory(width int) []string {
	if d.issue == nil || len(d.issue.Changelog) == 0 {
		return []string{" No history."}
	}

	gray := lipgloss.NewStyle().Foreground(theme.ColorGray)
	var lines []string

	// Reverse order: newest first.
	for i := len(d.issue.Changelog) - 1; i >= 0; i-- {
		entry := d.issue.Changelog[i]
		author := "Unknown"
		if entry.Author != nil {
			author = entry.Author.DisplayName
		}

		var content []string
		for _, item := range entry.Items {
			from := cleanWikiMarkup(item.FromString)
			to := cleanWikiMarkup(item.ToString)
			if from == "" {
				from = "none"
			}
			if to == "" {
				to = "none"
			}

			field := strings.ToLower(item.Field)

			if field == "description" || field == "comment" || field == "environment" {
				content = append(content, "   "+gray.Render(item.Field)+gray.Render(":"))
				content = append(content, renderDiff(from, to, width-4)...)
				continue
			}

			if field == "assignee" || field == "reviewer" || field == "reporter" {
				if from != "none" {
					from = theme.AuthorRender(from)
				}
				if to != "none" {
					to = theme.AuthorRender(to)
				}
			}
			changeLine := fmt.Sprintf("   %s: %s → %s", gray.Render(item.Field), from, to)
			for _, wl := range wrapText(changeLine, width-2) {
				content = append(content, wl)
			}
		}

		lines = append(lines, renderEntry(author, entry.Created, content, width, i == 0)...)
	}
	return lines
}

func (d *DetailView) renderComments(width int) []string {
	if d.issue == nil {
		return []string{" No issue selected."}
	}
	if len(d.issue.Comments) == 0 {
		return []string{" No comments."}
	}

	valStyle := d.theme.ValueStyle
	var lines []string
	for i, c := range d.issue.Comments {
		author := "Unknown"
		if c.Author != nil {
			author = c.Author.DisplayName
		}

		var content []string
		for _, wl := range wrapText(c.Body, width-2) {
			content = append(content, " "+colorMentions(valStyle.Render(wl)))
		}

		lines = append(lines, renderEntry(author, c.Created, content, width, i == len(d.issue.Comments)-1)...)
	}
	return lines
}

func (d *DetailView) renderLinks(width int) []string {
	if d.issue == nil {
		return []string{" No issue selected."}
	}
	if len(d.issue.IssueLinks) == 0 {
		return []string{" No links."}
	}

	keyStyle := d.theme.KeyStyle
	valStyle := d.theme.ValueStyle
	var lines []string
	for _, link := range d.issue.IssueLinks {
		if link.Type == nil {
			continue
		}
		if link.OutwardIssue != nil {
			lines = append(lines, " "+
				keyStyle.Render(link.Type.Outward)+" "+
				valStyle.Render(fmt.Sprintf("%s: %s", link.OutwardIssue.Key, link.OutwardIssue.Summary)))
		}
		if link.InwardIssue != nil {
			lines = append(lines, " "+
				keyStyle.Render(link.Type.Inward)+" "+
				valStyle.Render(fmt.Sprintf("%s: %s", link.InwardIssue.Key, link.InwardIssue.Summary)))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, " No links.")
	}
	return lines
}

func (d *DetailView) renderSplash(contentWidth, innerH int) string {
	green := lipgloss.NewStyle().Foreground(theme.ColorGreen).Bold(true)
	gray := lipgloss.NewStyle().Foreground(theme.ColorGray)
	label := lipgloss.NewStyle().Foreground(theme.ColorGreen)
	val := lipgloss.NewStyle()

	ascii := `   _                  _ _
  | |                (_|_)
  | | __ _ _____   _  _ _ _ __ __ _
  | |/ _` + "`" + ` |_  / | | || | | '__/ _` + "`" + ` |
  | | (_| |/ /| |_| || | | | | (_| |
  |_|\__,_/___|\__, || |_|_|  \__,_|
                __/ |/ |
               |___/__/`

	var lines []string
	for _, l := range strings.Split(ascii, "\n") {
		lines = append(lines, green.Render(l))
	}
	lines = append(lines, "")
	lines = append(lines, gray.Render("  lazyjira — Jira TUI"))
	lines = append(lines, gray.Render("  (c) 2026 Andrey Kondratev"))

	// Connection info.
	s := d.splash
	lines = append(lines, "")
	lines = append(lines, "  "+strings.Repeat("─", 30))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s", label.Render("Auth:"), val.Render(s.AuthMethod)))
	lines = append(lines, fmt.Sprintf("  %s  %s", label.Render("Host:"), val.Render(s.Host)))
	lines = append(lines, fmt.Sprintf("  %s %s", label.Render("Email:"), val.Render(s.Email)))
	if s.Project != "" {
		lines = append(lines, fmt.Sprintf("  %s  %s", label.Render("Project:"), val.Render(s.Project)))
	}

	content := strings.Join(lines, "\n")
	return components.RenderPanel("[0] lazyjira", content, d.width, innerH, d.focused)
}

func (d *DetailView) renderProjectView(contentWidth, innerH int) string {
	p := d.project
	valStyle := d.theme.ValueStyle

	var lines []string
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Key:", valStyle.Render(p.Key)))
	lines = append(lines, fmt.Sprintf(" %-11s %s", "Name:", valStyle.Render(p.Name)))
	if p.Lead != nil {
		lines = append(lines, fmt.Sprintf(" %-11s %s", "Lead:", valStyle.Render(p.Lead.DisplayName)))
	}

	content := strings.Join(lines, "\n")
	title := fmt.Sprintf("[0] Project: %s", p.Name)
	title = truncateRunes(title, contentWidth-2)
	return components.RenderPanel(title, content, d.width, innerH, d.focused)
}

func (d *DetailView) priorityStyled(name string) string {
	switch strings.ToLower(name) {
	case "highest", "high", "critical", "blocker":
		return d.theme.PriorityHigh.Render(name)
	case "medium":
		return d.theme.PriorityMedium.Render(name)
	default:
		return d.theme.PriorityLow.Render(name)
	}
}

// renderDiff shows removed lines in red and added lines in green.
func renderDiff(from, to string, maxWidth int) []string {
	redStyle := lipgloss.NewStyle().Foreground(theme.ColorRed)
	greenStyle := lipgloss.NewStyle().Foreground(theme.ColorGreen)

	fromLines := strings.Split(strings.TrimSpace(from), "\n")
	toLines := strings.Split(strings.TrimSpace(to), "\n")

	// Build sets for simple diff.
	fromSet := make(map[string]bool)
	toSet := make(map[string]bool)
	for _, l := range fromLines {
		fromSet[strings.TrimSpace(l)] = true
	}
	for _, l := range toLines {
		toSet[strings.TrimSpace(l)] = true
	}

	var lines []string

	// Show removed lines (in from but not in to).
	for _, l := range fromLines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || trimmed == "none" {
			continue
		}
		if !toSet[trimmed] {
			for _, wl := range wrapText("- "+trimmed, maxWidth) {
				lines = append(lines, "    "+redStyle.Render(wl))
			}
		}
	}

	// Show added lines (in to but not in from).
	for _, l := range toLines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || trimmed == "none" {
			continue
		}
		if !fromSet[trimmed] {
			for _, wl := range wrapText("+ "+trimmed, maxWidth) {
				lines = append(lines, "    "+greenStyle.Render(wl))
			}
		}
	}

	if len(lines) == 0 {
		lines = append(lines, "    "+lipgloss.NewStyle().Foreground(theme.ColorGray).Render("(content changed)"))
	}

	return lines
}

// cleanWikiMarkup strips Jira wiki markup from changelog values.
// Handles: [~accountid:...], {code:lang}...{code}, [text|url], etc.
func cleanWikiMarkup(s string) string {
	if s == "" {
		return s
	}
	result := s

	// [~accountid:UUID] → replace with @user (unresolved mentions)
	for {
		start := strings.Index(result, "[~accountid:")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "]")
		if end == -1 {
			break
		}
		result = result[:start] + "@user" + result[start+end+1:]
	}

	// {code:lang}...{code} → just the content
	for {
		start := strings.Index(result, "{code")
		if start == -1 {
			break
		}
		// Find closing }
		endOpen := strings.Index(result[start:], "}")
		if endOpen == -1 {
			break
		}
		// Find {code} closing tag
		closeTag := strings.Index(result[start+endOpen+1:], "{code}")
		if closeTag == -1 {
			// No closing tag, just strip the opening
			result = result[:start] + result[start+endOpen+1:]
			continue
		}
		content := result[start+endOpen+1 : start+endOpen+1+closeTag]
		result = result[:start] + strings.TrimSpace(content) + result[start+endOpen+1+closeTag+6:]
	}

	// [text|url] → text
	for {
		start := strings.Index(result, "[")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "]")
		if end == -1 {
			break
		}
		inner := result[start+1 : start+end]
		if pipe := strings.Index(inner, "|"); pipe != -1 {
			inner = inner[:pipe]
		}
		result = result[:start] + inner + result[start+end+1:]
	}

	return strings.TrimSpace(result)
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		width = 80
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		if len(paragraph) <= width {
			lines = append(lines, paragraph)
			continue
		}
		for len(paragraph) > width {
			cut := width
			for cut > 0 && paragraph[cut] != ' ' {
				cut--
			}
			if cut == 0 {
				cut = width
			}
			lines = append(lines, paragraph[:cut])
			paragraph = strings.TrimLeft(paragraph[cut:], " ")
		}
		if paragraph != "" {
			lines = append(lines, paragraph)
		}
	}
	return lines
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	}
}
