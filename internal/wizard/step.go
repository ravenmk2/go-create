package wizard

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

// fieldKind identifies the control type of a Field (mirrors the template
// option types plus free-form inputs).
type fieldKind int

const (
	kindInput fieldKind = iota
	kindSelect
	kindMultiSelect
)

// Field is one labeled control inside a step. It renders itself, handles keys
// through its widget, validates on submit, and commits its value into the
// bound parameter.
type Field struct {
	kind       fieldKind
	title      string
	help       string
	summaryKey string // sidebar row this field maps to ("" = none)
	widget     widget
	commit     func()        // writes the widget value into the bound parameter
	validate   func() string // "" when valid
	visible    func() bool   // nil means always visible
	err        string        // latest validation error, shown under the field
}

func (f *Field) isVisible() bool { return f.visible == nil || f.visible() }

// Step is one wizard page: a title plus the fields asked together.
type Step struct {
	title  string
	fields []*Field
	body   func() string // optional static content rendered above the fields
}

// visibleFields returns the fields currently shown (dynamic visibility, e.g.
// the copyright holder only applies to real licenses).
func (s *Step) visibleFields() []*Field {
	out := make([]*Field, 0, len(s.fields))
	for _, f := range s.fields {
		if f.isVisible() {
			out = append(out, f)
		}
	}
	return out
}

const (
	ungroupedTitle      = "Options"
	maxUngroupedPerPage = 5
)

// optionPage is one wizard page of template options.
type optionPage struct {
	title   string
	grouped bool // true when the page comes from a declared option group
	options []template.Option
}

// paginateOptions splits options into wizard pages: options declaring the same
// group share one page (first-appearance order, group name as the title);
// ungrouped options are merged into pages of at most maxUngroupedPerPage.
func paginateOptions(options []template.Option) []optionPage {
	var pages []optionPage
	groupIdx := map[string]int{}
	for _, o := range options {
		if o.Group != "" {
			i, ok := groupIdx[o.Group]
			if !ok {
				pages = append(pages, optionPage{title: o.Group, grouped: true})
				i = len(pages) - 1
				groupIdx[o.Group] = i
			}
			pages[i].options = append(pages[i].options, o)
			continue
		}
		// Ungrouped options fill the most recent ungrouped page while it has
		// room, otherwise they open a new page.
		if n := len(pages); n > 0 && !pages[n-1].grouped && len(pages[n-1].options) < maxUngroupedPerPage {
			pages[n-1].options = append(pages[n-1].options, o)
		} else {
			pages = append(pages, optionPage{title: ungroupedTitle, options: []template.Option{o}})
		}
	}
	return pages
}

// buildSteps constructs the static prefix of the wizard flow: the steps that
// do not depend on the selected template. Steps whose values were already
// provided on the command line are skipped (mirrors the huh fallback).
func buildSteps(templates []*template.Template, p *generate.Params) []*Step {
	var steps []*Step
	if p.Template == "" {
		steps = append(steps, templateSelectStep(templates, p))
	}
	if s := commonStep(p); s != nil {
		steps = append(steps, s)
	}
	if s := licenseStep(p); s != nil {
		steps = append(steps, s)
	}
	return steps
}

// templateSelectStep asks for the project template.
func templateSelectStep(templates []*template.Template, p *generate.Params) *Step {
	labels := make([]string, 0, len(templates))
	values := make([]string, 0, len(templates))
	for _, t := range templates {
		labels = append(labels, fmt.Sprintf("%s - %s", t.Name, t.Description))
		values = append(values, t.Name)
	}
	w := newSelectWidget(labels, values, p.Template, func(v string) { p.Template = v })
	return &Step{title: "Template", fields: []*Field{{
		kind:       kindSelect,
		title:      "Select a project template",
		summaryKey: "Template",
		widget:     w,
		commit:     func() { p.Template = w.value() },
	}}}
}

// commonStep asks for project name and module path (only those not preset).
func commonStep(p *generate.Params) *Step {
	var fields []*Field
	if p.Name == "" {
		w := newTextWidget(p.Name, "", func(v string) { p.Name = v })
		fields = append(fields, &Field{
			kind:       kindInput,
			title:      "Project name",
			help:       "Used for the directory and binary name.",
			summaryKey: "Name",
			widget:     w,
			commit:     func() { p.Name = w.value() },
			validate: func() string {
				if generate.PkgName(w.value()) == "" {
					return fmt.Sprintf("%q cannot be converted into a valid Go package name", w.value())
				}
				return ""
			},
		})
	}
	if p.Module == "" {
		w := newTextWidget(p.Module, "github.com/user/project", func(v string) { p.Module = v })
		fields = append(fields, &Field{
			kind:       kindInput,
			title:      "Module path",
			summaryKey: "Module",
			widget:     w,
			commit:     func() { p.Module = w.value() },
			validate: func() string {
				if !strings.Contains(w.value(), "/") {
					return "module path must contain / (e.g. github.com/user/project)"
				}
				return ""
			},
		})
	}
	if len(fields) == 0 {
		return nil
	}
	return &Step{title: "Project", fields: fields}
}

// licenseStep asks for the license, with the copyright holder as a conditional
// field on the same page. A preset license only needs the holder follow-up.
func licenseStep(p *generate.Params) *Step {
	if p.License != "" {
		if p.License != "none" && p.Holder == "" {
			return holderStep(p)
		}
		return nil
	}

	var holder *Field
	if p.Holder == "" {
		w := newTextWidget("", "", func(v string) { p.Holder = v })
		holder = &Field{
			kind:       kindInput,
			title:      "License copyright holder",
			summaryKey: "License",
			widget:     w,
			commit:     func() { p.Holder = w.value() },
			visible:    func() bool { return p.License != "" && p.License != "none" },
			validate: func() string {
				if strings.TrimSpace(w.value()) == "" {
					return "copyright holder cannot be empty"
				}
				return ""
			},
		}
	}
	w := newSelectWidget(
		[]string{"None", "MIT", "Apache-2.0", "GPL-3.0"},
		[]string{"none", "mit", "apache-2.0", "gpl-3.0"},
		p.License,
		func(v string) {
			p.License = v
			if v != "none" && p.Holder == "" && holder != nil {
				p.Holder = gitUserName()
				holder.widget.(*textWidget).setValue(p.Holder)
			}
		},
	)
	license := &Field{
		kind:       kindSelect,
		title:      "License",
		summaryKey: "License",
		widget:     w,
		commit:     func() { p.License = w.value() },
	}
	fields := []*Field{license}
	if holder != nil {
		fields = append(fields, holder)
	}
	return &Step{title: "License", fields: fields}
}

// holderStep asks only for the copyright holder (license preset via CLI).
func holderStep(p *generate.Params) *Step {
	p.Holder = gitUserName()
	w := newTextWidget(p.Holder, "", func(v string) { p.Holder = v })
	return &Step{title: "License", fields: []*Field{{
		kind:       kindInput,
		title:      "License copyright holder",
		summaryKey: "License",
		widget:     w,
		commit:     func() { p.Holder = w.value() },
		validate: func() string {
			if strings.TrimSpace(w.value()) == "" {
				return "copyright holder cannot be empty"
			}
			return ""
		},
	}}}
}

// optionSteps builds one step per option page; field values are prefilled from
// the coerced current values (defaults merged with --set values).
func optionSteps(tmpl *template.Template, p *generate.Params, current map[string]any) []*Step {
	var steps []*Step
	for _, page := range paginateOptions(tmpl.Options) {
		s := &Step{title: page.title}
		for _, o := range page.options {
			s.fields = append(s.fields, optionField(o, p, current[o.Key]))
		}
		steps = append(steps, s)
	}
	return steps
}

// optionField creates the field matching an option's type.
func optionField(o template.Option, p *generate.Params, current any) *Field {
	f := &Field{title: o.Prompt, help: o.Help, summaryKey: o.Key}
	switch o.Type {
	case template.OptionBoolean:
		v, _ := current.(bool)
		w := newSelectWidget([]string{"Enabled", "Disabled"}, []string{"true", "false"}, strconv.FormatBool(v),
			func(s string) { p.Options[o.Key] = s == "true" })
		w.inline = true
		f.kind = kindSelect
		f.widget = w
		f.commit = func() { p.Options[o.Key] = w.value() == "true" }
	case template.OptionString:
		v, _ := current.(string)
		w := newTextWidget(v, "", func(s string) { p.Options[o.Key] = s })
		var re *regexp.Regexp
		if o.Pattern != "" {
			re = regexp.MustCompile(o.Pattern) // validated at template load
		}
		f.kind = kindInput
		f.widget = w
		f.commit = func() { p.Options[o.Key] = w.value() }
		f.validate = func() string {
			if s := w.value(); s != "" && re != nil && !re.MatchString(s) {
				return fmt.Sprintf("must match %s", o.Pattern)
			}
			return ""
		}
	case template.OptionChoice:
		v, _ := current.(string)
		labels, values := choiceLists(o.Choices)
		w := newSelectWidget(labels, values, v, func(s string) { p.Options[o.Key] = s })
		f.kind = kindSelect
		f.widget = w
		f.commit = func() { p.Options[o.Key] = w.value() }
	case template.OptionMultiChoice:
		v, _ := current.([]string)
		labels, values := choiceLists(o.Choices)
		w := newMultiWidget(labels, values, v, func(list []string) { p.Options[o.Key] = list })
		f.kind = kindMultiSelect
		f.widget = w
		f.commit = func() { p.Options[o.Key] = w.value() }
	}
	return f
}

// choiceLists splits choices into parallel label/value lists for the widgets.
func choiceLists(choices []template.Choice) (labels, values []string) {
	for _, c := range choices {
		labels = append(labels, c.Label)
		values = append(values, c.Value)
	}
	return labels, values
}

// confirmStep is the final page: a live summary plus a Create/Cancel choice.
// The choice is written to confirmed on change and on commit; the model turns
// a false value into the canceled result.
func confirmStep(tmpl *template.Template, p *generate.Params, current map[string]any, confirmed *bool) *Step {
	*confirmed = true // the cursor starts on Create
	w := newSelectWidget([]string{"Create project", "Cancel"}, []string{"create", "cancel"}, "create",
		func(v string) { *confirmed = v == "create" })
	return &Step{
		title: "Confirm",
		body:  func() string { return summary(tmpl, p, current) },
		fields: []*Field{{
			kind:   kindSelect,
			title:  "Create project?",
			widget: w,
			commit: func() { *confirmed = w.value() == "create" },
		}},
	}
}
