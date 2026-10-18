package wizard

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Layout constants for the framed view; below the minimum size the wizard
// degrades to a borderless compact layout instead of breaking.
const (
	minFramedWidth  = 72
	minFramedHeight = 20
	sidebarWidth    = 36
	minSidebarWidth = 104
	maxFrameWidth   = 116
	maxFrameHeight  = 32
)

var (
	accentColor        = lipgloss.AdaptiveColor{Light: "#367478", Dark: "#83B5B7"}
	borderColor        = lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#535353"}
	successColor       = lipgloss.AdaptiveColor{Light: "#216643", Dark: "#8ED3AE"}
	accentStyle        = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	headerStyle        = lipgloss.NewStyle().Bold(true)
	dimStyle           = lipgloss.NewStyle().Foreground(borderColor)
	focusedChoiceStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color("#163638")).
				Background(lipgloss.AdaptiveColor{Light: "#B7D4D5", Dark: "#83B5B7"})
	sidebarLabelStyle = lipgloss.NewStyle().Foreground(accentColor)
	// Questions, help, values, placeholders and keyboard instructions are all
	// required reading. Keep their foreground inherited from the terminal.
	helpStyle         = lipgloss.NewStyle()
	errStyle          = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#A83232", Dark: "#ED9292"})
	sidebarValueStyle = lipgloss.NewStyle().Bold(true)
)

// View renders the whole form. A finished wizard renders nothing so the
// alternate screen restores a clean terminal.
func (m *model) View() string {
	if m.done || m.canceled || m.err != nil || m.width == 0 {
		return ""
	}
	m.layout()
	m.refresh()
	if m.compact() {
		return m.compactView()
	}
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, m.framedView())
}

func (m *model) compact() bool {
	return m.width < minFramedWidth || m.height < minFramedHeight
}

// innerWidth is the content width inside the border and horizontal padding.
func (m *model) innerWidth() int {
	return min(m.width, maxFrameWidth) - 6
}

func (m *model) reviewing() bool { return m.curStep() != nil && m.curStep().review }

func (m *model) showSidebar() bool {
	return !m.reviewing() && !m.compact() && m.width >= minSidebarWidth
}

func (m *model) mainWidth() int {
	w := m.innerWidth() - 2 // main area's horizontal padding
	if m.showSidebar() {
		w -= sidebarWidth + 2 // divider and sidebar padding
	}
	return w
}

// layout recomputes the viewport geometry from the terminal size.
func (m *model) layout() {
	if m.compact() {
		m.vp.Width = m.width
		m.vp.Height = m.height - 1 - lipgloss.Height(m.footerView(m.width))
	} else {
		m.vp.Width = m.mainWidth()
		m.vp.Height = min(m.height, maxFrameHeight) - 7 - lipgloss.Height(m.footerView(m.innerWidth()))
	}
	if m.reviewing() {
		m.vp.Height -= lipgloss.Height(m.confirmActions(m.vp.Width)) + 1
	}
	if m.vp.Width < 1 {
		m.vp.Width = 1
	}
	if m.vp.Height < 1 {
		m.vp.Height = 1
	}
}

// framedView renders the bordered window: step header, scrollable main area,
// live summary sidebar, and key hints.
func (m *model) framedView() string {
	header := headerStyle.MaxWidth(m.innerWidth()).Render(m.headerText())
	sep := dimStyle.Render(strings.Repeat("─", m.innerWidth()))
	footer := m.footerView(m.innerWidth())

	main := lipgloss.NewStyle().Width(m.vp.Width).Height(m.vp.Height).Render(m.vp.View())
	if m.reviewing() {
		main += "\n\n" + m.confirmActions(m.vp.Width)
	}
	body := lipgloss.NewStyle().Padding(1, 1).Render(main)
	if m.showSidebar() {
		divider := dimStyle.Render(strings.Repeat("│\n", m.vp.Height+1) + "│")
		sidebar := lipgloss.NewStyle().Width(sidebarWidth).Height(m.vp.Height).Render(m.sidebarView())
		sidebar = lipgloss.NewStyle().Padding(1, 0, 1, 1).Render(sidebar)
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, divider, sidebar)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, sep, body, sep, footer)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 2).
		Render(content)
}

// compactView is the borderless fallback for small terminals.
func (m *model) compactView() string {
	body := m.vp.View()
	if m.reviewing() {
		body += "\n\n" + m.confirmActions(m.vp.Width)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.MaxWidth(m.width).Render(m.headerText()),
		body,
		m.footerView(m.width),
	)
}

func (m *model) headerText() string {
	title := ""
	if s := m.curStep(); s != nil {
		title = s.title
	}
	return fmt.Sprintf("%s Step %d/%d · %s", accentStyle.Render("▸"), m.stepIdx+1, len(m.steps), title)
}

// footerText returns the key hints, tuned to the focused field's control type.
func (m *model) footerText() string {
	nav := "Shift+Tab back · Esc quit"
	if m.reviewing() {
		return "←/→ choose · Enter confirm · PgUp/PgDn scroll · " + nav
	}
	switch f := m.curField(); {
	case f == nil:
		return nav
	case f.kind == kindMultiSelect:
		return "↑/↓ move · Space toggle · Enter next · " + nav
	case f.kind == kindSelect:
		if w, ok := f.widget.(*selectWidget); ok && w.inline {
			return "←/→ choose · Enter next · " + nav
		}
		return "↑/↓ move · Enter next · " + nav
	default:
		return "Enter next · " + nav
	}
}

func (m *model) footerView(width int) string {
	height := max(1, m.height-2)
	if m.reviewing() && m.compact() {
		height = max(1, m.height-3-lipgloss.Height(m.confirmActions(width)))
	}
	return lipgloss.NewStyle().Width(max(1, width)).
		MaxHeight(height).Render(m.footerText())
}

func (m *model) confirmActions(width int) string {
	w, ok := m.curField().widget.(*selectWidget)
	if !ok {
		return ""
	}
	labels := []string{"Create project", "Cancel"}
	if width < 32 {
		labels[0] = "Create"
	}
	items := make([]string, len(labels))
	for i, label := range labels {
		items[i] = "  " + label
		if w.cursor == i {
			items[i] = focusedChoiceStyle.Render("▸ " + label)
		}
	}
	line := strings.Join(items, "  ")
	if lipgloss.Width(line) > width {
		line = strings.Join(items, "\n")
	}
	return lipgloss.NewStyle().Width(max(1, width)).Render(line)
}

// sidebarRow is one entry (label + value) of the summary sidebar.
type sidebarRow struct {
	key         string // matches Field.summaryKey
	label       string
	value       any
	placeholder string
}

// sidebarRows collects the current summary rows.
func (m *model) sidebarRows() []sidebarRow {
	p := m.p
	rows := []sidebarRow{
		{key: "Template", label: "Template", value: p.Template, placeholder: "<required>"},
		{key: "Name", label: "Name", value: p.Name, placeholder: "<required>"},
		{key: "Module", label: "Module", value: p.Module, placeholder: "<required>"},
		{key: "License", label: "License", value: licenseDisplay(p), placeholder: "—"},
	}
	if m.tmpl != nil {
		active := m.activeSummaryKeys()
		for _, o := range m.tmpl.Options {
			if !active[o.Key] {
				continue
			}
			label := o.Prompt
			if short, ok := map[string]string{
				"display_name": "Title", "description": "Desc", "http_framework": "Framework",
				"embed_frontend": "Frontend", "docker_image": "Image", "docker_dev": "Dev image",
				"goreleaser": "Release", "ci_platforms": "CI",
			}[o.Key]; ok {
				label = short
			}
			rows = append(rows, sidebarRow{key: o.Key, label: label, value: optionValue(o, p, m.options), placeholder: "—"})
		}
	}
	return rows
}

// sidebarView keeps stable project rows and only the current page's options.
// Each row occupies exactly one terminal line; Confirm preserves full values.
func (m *model) sidebarView() string {
	blocks := []string{headerStyle.Render("Project")}
	rows := m.sidebarRows()
	focused := ""
	if f := m.curField(); f != nil {
		focused = f.summaryKey
	}
	projectRows := 4
	extra := 3 // project heading and the footer
	if len(rows) > projectRows {
		extra += 2 // current-step heading and separator
	}
	// Drop secondary project rows before clipping current-step options on short
	// terminals. The focused project field is always retained.
	for i := projectRows - 1; len(rows)+extra > m.vp.Height && i >= 0; i-- {
		if rows[i].key != focused {
			rows = append(rows[:i], rows[i+1:]...)
			projectRows--
		}
	}
	for i, row := range rows {
		if i == projectRows {
			title := strings.Join(strings.Fields(m.curStep().title), " ")
			blocks = append(blocks, "", headerStyle.Render(ansi.Truncate(title, sidebarWidth, "…")))
		}
		marker := "  "
		value := formatValue(row.value)
		if row.value == "" {
			value = row.placeholder
		}
		if row.key == focused {
			marker = accentStyle.Render("▸ ")
		}
		plainLabel := ansi.Truncate(strings.Join(strings.Fields(row.label), " ")+":", 11, "…")
		label := sidebarLabelStyle.Width(11).Render(plainLabel)
		value = strings.Join(strings.Fields(value), " ")
		if row.key == "ci_platforms" {
			value = strings.NewReplacer("Ubuntu", "Linux", "Windows", "Win", " · ", "/").Replace(value)
		}
		valueWidth := sidebarWidth - 14
		if row.key == "Module" && lipgloss.Width(value) > valueWidth {
			left := (valueWidth - 1) / 2
			value = ansi.Cut(value, 0, left) + "…" + ansi.Cut(value, lipgloss.Width(value)-(valueWidth-left-1), lipgloss.Width(value))
		} else {
			value = ansi.Truncate(value, valueWidth, "…")
		}
		style := sidebarValueStyle
		if enabled, ok := row.value.(bool); ok && enabled {
			style = style.Foreground(successColor)
		}
		line := marker + label + " " + style.Render(value)
		if row.key == focused {
			line = focusedChoiceStyle.Width(sidebarWidth).Render("▸ " + lipgloss.NewStyle().Width(11).Render(plainLabel) + " " + value)
		}
		blocks = append(blocks, line)
	}
	blocks = append(blocks, "", "Full settings on Confirm")
	lines := strings.Split(lipgloss.JoinVertical(lipgloss.Left, blocks...), "\n")
	if len(lines) > m.vp.Height {
		lines = append(lines[:max(0, m.vp.Height-1)], "  … more on Confirm")
	}
	return strings.Join(lines, "\n")
}

// fieldRange records a field's vertical extent within the step content.
type fieldRange struct{ y, height int }

// stepContent renders the current step's body and visible fields.
func (m *model) stepContent() (string, []fieldRange) {
	s := m.curStep()
	if s == nil {
		return "", nil
	}
	if s.review {
		return m.confirmationView(m.vp.Width), nil
	}
	var blocks []string
	var ranges []fieldRange
	y := 0
	if s.body != nil {
		body := lipgloss.NewStyle().Width(m.vp.Width).Render(s.body())
		blocks = append(blocks, body)
		y += lipgloss.Height(body) + 1
	}
	focused := m.curField()
	for _, f := range s.visibleFields() {
		v := f.view(f == focused, m.vp.Width)
		blocks = append(blocks, v)
		ranges = append(ranges, fieldRange{y, lipgloss.Height(v)})
		y += lipgloss.Height(v) + 1
	}
	return strings.Join(blocks, "\n\n"), ranges
}

// refresh pushes the current step content into the viewport and scrolls it so
// the focused field stays visible.
func (m *model) refresh() {
	content, ranges := m.stepContent()
	m.vp.SetContent(content)
	s := m.curStep()
	if s == nil {
		return
	}
	idx := slices.Index(s.visibleFields(), m.curField())
	if idx < 0 || idx >= len(ranges) {
		return
	}
	r := ranges[idx]
	switch {
	case r.y < m.vp.YOffset:
		m.vp.SetYOffset(r.y)
	case r.y+r.height > m.vp.YOffset+m.vp.Height:
		m.vp.SetYOffset(r.y + r.height - m.vp.Height)
	}
}

// view renders the field: title, optional help, the control, and the
// validation error below it. Text is soft-wrapped to the content width so
// long prompts never overflow the main area.
func (f *Field) view(focused bool, width int) string {
	if width < 1 {
		width = 1
	}
	wrap := lipgloss.NewStyle().Width(width)
	title := "  " + f.title
	if focused {
		title = accentStyle.Render("▸ ") + headerStyle.Render(f.title)
	}
	var b strings.Builder
	b.WriteString(wrap.Render(title))
	if f.help != "" {
		b.WriteString("\n" + wrap.Render(helpStyle.Render(f.help)))
	}
	b.WriteByte('\n')
	b.WriteString(f.widget.view(focused, width))
	if f.err != "" {
		b.WriteString("\n" + wrap.Render(errStyle.Render("! "+f.err)))
	}
	return b.String()
}
