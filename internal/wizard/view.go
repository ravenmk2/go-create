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
	minSidebarWidth = 96
	maxFrameWidth   = 116
	maxFrameHeight  = 32
)

var (
	accentColor        = lipgloss.AdaptiveColor{Light: "#367478", Dark: "#83B5B7"}
	borderColor        = lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#535353"}
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

func (m *model) showSidebar() bool { return !m.compact() && m.width >= minSidebarWidth }

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
	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.MaxWidth(m.width).Render(m.headerText()),
		m.vp.View(),
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
	return lipgloss.NewStyle().Width(max(1, width)).
		MaxHeight(max(1, m.height-2)).Render(m.footerText())
}

// sidebarRow is one entry (label + value) of the summary sidebar.
type sidebarRow struct {
	key         string // matches Field.summaryKey
	label       string
	value       string
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
		for _, o := range m.tmpl.Options {
			v, ok := p.Options[o.Key]
			if !ok {
				v = m.options[o.Key]
			}
			label := o.Prompt
			if lipgloss.Width(label) > 18 {
				label = strings.ReplaceAll(o.Key, "_", " ")
				label = strings.ToUpper(label[:1]) + label[1:]
			}
			rows = append(rows, sidebarRow{key: o.Key, label: label, value: formatValue(v), placeholder: "—"})
		}
	}
	return rows
}

// sidebarView puts the current page first so long option lists cannot hide it.
// Labels use readable teal; values retain the terminal foreground in bold.
func (m *model) sidebarView() string {
	active := m.activeSummaryKeys()
	wrap := lipgloss.NewStyle().Width(sidebarWidth)
	blocks := []string{headerStyle.Render("Summary"), ""}
	rows := m.sidebarRows()
	slices.SortStableFunc(rows, func(a, b sidebarRow) int {
		if active[a.key] == active[b.key] {
			return 0
		}
		if active[a.key] {
			return -1
		}
		return 1
	})
	for _, row := range rows {
		marker, label := "  ", sidebarLabelStyle.Render(row.label+":")
		value := row.value
		if value == "" {
			value = row.placeholder
		}
		if active[row.key] {
			marker = accentStyle.Render("▸ ")
		}
		blocks = append(blocks, wrap.Render(marker+label+" "+sidebarValueStyle.Render(value)))
	}
	lines := strings.Split(lipgloss.JoinVertical(lipgloss.Left, blocks...), "\n")
	if len(lines) > m.vp.Height {
		lines = append(lines[:max(0, m.vp.Height-1)], "  … more on Confirm")
	}
	return strings.Join(lines, "\n")
}

// formatValue renders an option value for the sidebar.
func formatValue(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "Enabled"
		}
		return "Disabled"
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
