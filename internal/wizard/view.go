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
	accentColor        = lipgloss.AdaptiveColor{Light: "#007A8A", Dark: "#22D3EE"}
	borderColor        = lipgloss.AdaptiveColor{Light: "#7B8794", Dark: "#64748B"}
	successColor       = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	accentStyle        = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	headerStyle        = lipgloss.NewStyle().Bold(true)
	dimStyle           = lipgloss.NewStyle().Foreground(borderColor)
	focusedChoiceStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#082F49"}).
				Background(accentColor)
	sidebarLabelStyle = lipgloss.NewStyle().Foreground(accentColor)
	// Keep body text inherited; use a readable secondary color for help.
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#475569", Dark: "#B8C5D6"})
	errStyle          = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#BE123C", Dark: "#FB7185"})
	sidebarValueStyle = lipgloss.NewStyle()
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
		w -= sidebarWidth + 2 // gutter and sidebar padding
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
		m.vp.Height = min(m.height, maxFrameHeight) - 6 - lipgloss.Height(m.footerView(m.innerWidth()))
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
		sidebar := lipgloss.NewStyle().Width(sidebarWidth).Height(m.vp.Height).Render(m.sidebarView())
		sidebar = lipgloss.NewStyle().Padding(1, 0, 1, 1).Render(sidebar)
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, " ", sidebar)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, sep, body, footer)
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
	if m.editMenu != nil {
		return "Edit settings"
	}
	if !m.compact() {
		var parts []string
		for i, s := range m.steps {
			label := s.title
			if i < m.stepIdx {
				label = "✓ " + label
			}
			if i == m.stepIdx {
				label = accentStyle.Render("▸ " + label)
			}
			parts = append(parts, label)
		}
		line := strings.Join(parts, "  ·  ")
		if lipgloss.Width(line) <= m.innerWidth() {
			return line
		}
	}
	title := ""
	if s := m.curStep(); s != nil {
		title = s.title
	}
	return fmt.Sprintf("%s Step %d/%d · %s", accentStyle.Render("▸"), m.stepIdx+1, len(m.steps), title)
}

// footerText returns the key hints, tuned to the focused field's control type.
func (m *model) footerText() string {
	if m.editMenu != nil {
		return "↑/↓ choose section · Enter edit · Esc return"
	}
	nav := "Shift+Tab back · Esc quit"
	if m.reviewing() {
		return "←/→ choose · Enter confirm · E edit · PgUp/PgDn scroll · " + nav
	}
	switch f := m.curField(); {
	case f == nil:
		return nav
	case f.kind == kindMultiSelect:
		return "↑/↓ move · Space toggle · Tab/Enter next · " + nav
	case f.kind == kindSelect:
		if w, ok := f.widget.(*selectWidget); ok && w.inline {
			return "←/→ choose · Tab/Enter next · " + nav
		}
		return "↑/↓ move · Tab/Enter next · " + nav
	default:
		return "Tab/Enter next · " + nav
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
	if m.editMenu != nil {
		return ""
	}
	w, ok := m.curField().widget.(*selectWidget)
	if !ok {
		return ""
	}
	labels := []string{"Create project", "Cancel"}
	if width < 32 {
		labels[0] = "Create"
	}
	if m.p.DryRun {
		labels[0] = "Preview"
	} else if m.preview != nil && len(m.preview.Overwrites()) > 0 {
		labels[0] = "Overwrite & create"
		if width < 32 {
			labels[0] = "Overwrite"
		}
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
	return rows
}

// sidebarView keeps project context and explains the focused field. The active
// control owns the only solid highlight; the sidebar uses a small marker.
func (m *model) sidebarView() string {
	blocks := []string{headerStyle.Render("Project")}
	focused := ""
	if f := m.curField(); f != nil {
		focused = f.summaryKey
	}
	for _, row := range m.sidebarRows() {
		value := formatValue(row.value)
		if row.value == "" {
			value = row.placeholder
		}
		value = strings.Join(strings.Fields(value), " ")
		valueWidth := sidebarWidth - 14
		if row.key == "Module" && lipgloss.Width(value) > valueWidth {
			left := (valueWidth - 1) / 2
			value = ansi.Cut(value, 0, left) + "…" + ansi.Cut(value, lipgloss.Width(value)-(valueWidth-left-1), lipgloss.Width(value))
		} else {
			value = ansi.Truncate(value, valueWidth, "…")
		}
		marker := "  "
		if row.key == focused {
			marker = accentStyle.Render("▸ ")
		}
		label := sidebarLabelStyle.Width(11).Render(row.label + ":")
		blocks = append(blocks, marker+label+" "+sidebarValueStyle.Render(value))
	}
	if f := m.curField(); f != nil {
		title := strings.Join(strings.Fields(f.title), " ")
		help := f.help
		if help == "" {
			switch f.kind {
			case kindInput:
				help = "Enter a value, then press Tab or Enter to continue."
			case kindMultiSelect:
				help = "Choose any combination. Space toggles an item."
			default:
				help = "Use the arrow keys to choose, then Tab or Enter to continue."
			}
		}
		blocks = append(blocks, "", headerStyle.Render("About this setting"),
			lipgloss.NewStyle().Width(sidebarWidth).Render(sidebarLabelStyle.Render(title)),
			lipgloss.NewStyle().Width(sidebarWidth).Render(helpStyle.Render(help)))
	}
	lines := strings.Split(strings.Join(blocks, "\n"), "\n")
	if len(lines) > m.vp.Height {
		lines = append(lines[:max(0, m.vp.Height-1)], helpStyle.Render("… widen or enlarge to read more"))
	}
	return strings.Join(lines, "\n")
}

// fieldRange records a field's vertical extent within the step content.
type fieldRange struct{ y, height int }

// stepContent renders the current step's body and visible fields.
func (m *model) stepContent() (string, []fieldRange) {
	if m.editMenu != nil {
		return m.editMenu.view(true, m.vp.Width), nil
	}
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
		field := *f
		if m.showSidebar() {
			field.help = ""
		}
		v := field.view(f == focused, m.vp.Width)
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
	if m.editMenu != nil {
		y := m.editMenu.cursor
		if y < m.vp.YOffset {
			m.vp.SetYOffset(y)
		}
		if y >= m.vp.YOffset+m.vp.Height {
			m.vp.SetYOffset(y - m.vp.Height + 1)
		}
		return
	}
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
	if f.compactBool && width >= lipgloss.Width(title)+24 {
		controlWidth := 22
		control := f.widget.view(focused, controlWidth)
		line := lipgloss.NewStyle().Width(width-controlWidth).Render(title) + control
		if f.help != "" {
			line += "\n" + wrap.Render(helpStyle.Render("  "+f.help))
		}
		if f.err != "" {
			line += "\n" + wrap.Render(errStyle.Render("! "+f.err))
		}
		return line
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
