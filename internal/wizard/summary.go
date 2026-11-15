package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

type summaryRow struct {
	label string
	value any
}

type summaryGroup struct {
	title string
	rows  []summaryRow
}

func optionValue(o template.Option, p *generate.Params, defaults map[string]any) any {
	v, ok := p.Options[o.Key]
	if !ok {
		v = defaults[o.Key]
	}
	label := func(value string) string {
		for _, c := range o.Choices {
			if c.Value == value {
				return c.Label
			}
		}
		return value
	}
	switch value := v.(type) {
	case string:
		return label(value)
	case []string:
		labels := make([]string, 0, len(value))
		for _, item := range value {
			labels = append(labels, label(item))
		}
		return labels
	default:
		return v
	}
}

func summaryGroups(t *template.Template, p *generate.Params, defaults map[string]any) []summaryGroup {
	dir := p.Dir
	if dir == "" {
		dir = "./" + p.Name
	}
	groups := []summaryGroup{{title: "Project", rows: []summaryRow{
		{"Template", t.Name}, {"Name", p.Name}, {"Module", p.Module},
		{"Directory", dir}, {"License", licenseDisplay(p)},
	}}}
	for _, o := range t.Options {
		title := o.Group
		if title == "" {
			title = "Options"
		}
		group := -1
		for i := 1; i < len(groups); i++ {
			if groups[i].title == title {
				group = i
				break
			}
		}
		if group < 0 {
			groups = append(groups, summaryGroup{title: title})
			group = len(groups) - 1
		}
		groups[group].rows = append(groups[group].rows, summaryRow{o.Prompt, optionValue(o, p, defaults)})
	}
	return groups
}

func formatValue(v any) string {
	switch value := v.(type) {
	case bool:
		if value {
			return "✓ On"
		}
		return "– Off"
	case string:
		if value != "" {
			return value
		}
	case []string:
		if len(value) > 0 {
			return strings.Join(value, " · ")
		}
	case nil:
	default:
		return fmt.Sprint(value)
	}
	return "—"
}

// confirmationView preserves every value and wraps continuation lines beneath
// the value column. Narrow terminals put labels above their values instead.
func (m *model) confirmationView(width int) string {
	groups := summaryGroups(m.tmpl, m.p, m.options)
	labelWidth := 0
	for _, group := range groups {
		for _, row := range group.rows {
			labelWidth = max(labelWidth, lipgloss.Width(row.label))
		}
	}
	labelWidth = min(labelWidth, 22)
	stacked := width < labelWidth+24
	var sections []string
	if m.previewErr != nil {
		sections = append(sections, lipgloss.NewStyle().Width(width).Render(errStyle.Render("! "+m.previewErr.Error()+"\nPress E to edit settings.")))
	} else if m.preview != nil {
		title := "Ready to create"
		if m.p.DryRun {
			title = "Preview only — no files will be written"
		}
		style := helpStyle
		if len(m.preview.Overwrites()) > 0 && !m.p.DryRun {
			title = "Existing files will be overwritten"
			style = errStyle
		}
		sections = append(sections, lipgloss.NewStyle().Width(width).Render(headerStyle.Render(title)+"\n"+style.Render(m.preview.Summary())))
	}
	for _, group := range groups {
		lines := []string{lipgloss.NewStyle().Width(width).Render(accentStyle.Render(group.title))}
		for _, row := range group.rows {
			valueStyle := sidebarValueStyle
			if enabled, ok := row.value.(bool); ok && enabled {
				valueStyle = valueStyle.Foreground(successColor)
			}
			value := formatValue(row.value)
			if stacked {
				lines = append(lines, lipgloss.NewStyle().Width(width).Render(sidebarLabelStyle.Render(row.label)),
					lipgloss.NewStyle().Width(width).Render(valueStyle.Render(value)))
				continue
			}
			label := lipgloss.NewStyle().Width(labelWidth).Render(sidebarLabelStyle.Render(row.label))
			values := strings.Split(lipgloss.NewStyle().Width(width-labelWidth-2).Render(valueStyle.Render(value)), "\n")
			lines = append(lines, label+"  "+values[0])
			for _, continuation := range values[1:] {
				lines = append(lines, strings.Repeat(" ", labelWidth+2)+continuation)
			}
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n\n")
}
