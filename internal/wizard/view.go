package wizard

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Layout constants for the framed view; below the minimum size the wizard
// degrades to a borderless compact layout instead of breaking.
const (
	minFramedWidth  = 72
	minFramedHeight = 20
	sidebarWidth    = 28
)

var (
	accentStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	borderColor = lipgloss.Color("240")
	// Sidebar labels and placeholders use a subdued adaptive gray (the charm
	// default-theme convention), readable on both light and dark terminals;
	// values keep the terminal's default foreground for maximum contrast.
	sidebarLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})
	sidebarValueStyle = lipgloss.NewStyle().Bold(true)
)

// View renders the whole form. A finished wizard renders nothing so the
// alternate screen restores a clean terminal.
func (m *model) View() string {
	if m.done || m.canceled || m.err != nil || m.width == 0 {
		return ""
	}
	m.refresh()
	if m.compact() {
		return m.compactView()
	}
	return m.framedView()
}

func (m *model) compact() bool {
	return m.width < minFramedWidth || m.height < minFramedHeight
}

// innerWidth is the content width inside the border and horizontal padding.
func (m *model) innerWidth() int {
	return m.width - 4
}

func (m *model) mainWidth() int {
	w := m.innerWidth() - sidebarWidth - 1 // sidebar + divider
	if w < 16 {
		w = 16
	}
	return w
}

// layout recomputes the viewport geometry from the terminal size.
func (m *model) layout() {
	if m.compact() {
		m.vp.Width = m.width
		m.vp.Height = m.height - 2 // header + footer
	} else {
		m.vp.Width = m.mainWidth()
		m.vp.Height = m.height - 6 // border + header + separators + footer
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
	footer := dimStyle.MaxWidth(m.innerWidth()).Render(" " + m.footerText())

	divider := dimStyle.Render(strings.Repeat("│\n", m.vp.Height-1) + "│")
	main := lipgloss.NewStyle().Width(m.mainWidth()).Height(m.vp.Height).Render(m.vp.View())
	sidebar := lipgloss.NewStyle().Width(sidebarWidth).Height(m.vp.Height).Render(m.sidebarView())
	body := lipgloss.JoinHorizontal(lipgloss.Top, main, divider, sidebar)

	content := lipgloss.JoinVertical(lipgloss.Left, header, sep, body, sep, footer)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(content)
}

// compactView is the borderless fallback for small terminals.
func (m *model) compactView() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.MaxWidth(m.width).Render(m.headerText()),
		m.vp.View(),
		dimStyle.MaxWidth(m.width).Render(m.footerText()),
	)
}

func (m *model) headerText() string {
	title := ""
	if s := m.curStep(); s != nil {
		title = s.title
	}
	return fmt.Sprintf(" Step %d/%d · %s", m.stepIdx+1, len(m.steps), title)
}

// footerText returns the key hints, tuned to the focused field's control type.
func (m *model) footerText() string {
	nav := "shift+tab back · ctrl+c quit"
	switch f := m.curField(); {
	case f == nil:
		return nav
	case f.kind == kindMultiSelect:
		return "space toggle · enter next · " + nav
	case f.kind == kindSelect:
		return "↑/↓ move · enter next · " + nav
	default:
		return "enter next · " + nav
	}
}

// sidebarRow is one entry (label + value) of the summary sidebar.
type sidebarRow struct {
	key         string // matches Field.summaryKey
	label       string
	value       string
	placeholder string // shown dimmed while the value is empty
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
		for _, o := range m.tmpl.Options {
			v, ok := p.Options[o.Key]
			if !ok {
				v = m.options[o.Key]
			}
			rows = append(rows, sidebarRow{key: o.Key, label: o.Prompt, value: formatValue(v), placeholder: "—"})
		}
	}
	return rows
}

// sidebarView renders the live parameter summary with visual hierarchy:
// dimmed labels, bright values, dimmed placeholders for empty values, and an
// accent marker plus bold face on the rows owned by the current step.
func (m *model) sidebarView() string {
	active := m.activeSummaryKeys()
	wrap := lipgloss.NewStyle().Width(sidebarWidth)
	blocks := []string{accentStyle.Render("Summary")}
	for _, row := range m.sidebarRows() {
		marker, label := "  ", sidebarLabelStyle.Render(row.label+":")
		value := row.value
		if value == "" {
			value = sidebarLabelStyle.Render(row.placeholder)
		}
		if active[row.key] {
			marker = accentStyle.Render("▸ ")
			label = lipgloss.NewStyle().Bold(true).Render(row.label + ":")
			if row.value != "" {
				value = sidebarValueStyle.Render(value)
			}
		}
		blocks = append(blocks, wrap.Render(marker+label+" "+value))
	}
	lines := strings.Split(lipgloss.JoinVertical(lipgloss.Left, blocks...), "\n")
	if len(lines) > m.vp.Height {
		lines = lines[:m.vp.Height]
	}
	return strings.Join(lines, "\n")
}

// formatValue renders an option value for the sidebar.
func formatValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return strings.Join(t, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// fieldRange records a field's vertical extent within the step content.
type fieldRange struct{ y, height int }

// stepContent renders the current step's body and visible fields.
func (m *model) stepContent() (string, []fieldRange) {
	s := m.curStep()
	if s == nil {
		return "", nil
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
	title := f.title
	if focused {
		title = accentStyle.Render(title)
	}
	var b strings.Builder
	b.WriteString(wrap.Render(title))
	if f.help != "" {
		b.WriteString("\n" + wrap.Render(helpStyle.Render(f.help)))
	}
	b.WriteByte('\n')
	b.WriteString(f.widget.view(focused, width))
	if f.err != "" {
		b.WriteString("\n" + wrap.Render(errStyle.Render(f.err)))
	}
	return b.String()
}
