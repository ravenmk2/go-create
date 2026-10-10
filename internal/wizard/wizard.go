// Package wizard interactively fills in generation parameters with huh dynamic forms.
package wizard

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

// Run runs the wizard: select a template first when p.Template is empty,
// then fill in missing common parameters, dynamically build the option form from the
// template descriptor, and finally confirm with a summary.
// It returns the selected template; cancellation by the user returns an error.
func Run(templates []*template.Template, p *generate.Params) (*template.Template, error) {
	if len(templates) == 0 {
		return nil, errors.New("no templates available")
	}

	// 1. Template selection
	if p.Template == "" {
		opts := make([]huh.Option[string], 0, len(templates))
		for _, t := range templates {
			opts = append(opts, huh.NewOption(fmt.Sprintf("%s - %s", t.Name, t.Description), t.Name))
		}
		if err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().Title("Select a project template").Options(opts...).Value(&p.Template),
		)).Run(); err != nil {
			return nil, err
		}
	}
	var tmpl *template.Template
	for _, t := range templates {
		if t.Name == p.Template {
			tmpl = t
		}
	}
	if tmpl == nil {
		return nil, fmt.Errorf("template %q does not exist", p.Template)
	}

	// 2. Common parameters: project name, module path
	var common []huh.Field
	if p.Name == "" {
		common = append(common, huh.NewInput().
			Title("Project name (directory / binary name)").
			Value(&p.Name).
			Validate(func(s string) error {
				if generate.PkgName(s) == "" {
					return fmt.Errorf("%q cannot be converted into a valid Go package name", s)
				}
				return nil
			}))
	}
	if p.Module == "" {
		common = append(common, huh.NewInput().
			Title("Module path").
			Placeholder("github.com/user/project").
			Value(&p.Module).
			Validate(func(s string) error {
				if !strings.Contains(s, "/") {
					return errors.New("module path must contain / (e.g. github.com/user/project)")
				}
				return nil
			}))
	}
	if len(common) > 0 {
		if err := huh.NewForm(huh.NewGroup(common...)).Run(); err != nil {
			return nil, err
		}
	}

	// 3. License (a pre-creation decision point; the wizard always asks)
	if p.License == "" {
		if err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("License").
				Options(
					huh.NewOption("None", "none"),
					huh.NewOption("MIT", "mit"),
					huh.NewOption("Apache-2.0", "apache-2.0"),
					huh.NewOption("GPL-3.0", "gpl-3.0"),
				).
				Value(&p.License),
		)).Run(); err != nil {
			return nil, err
		}
	}
	if p.License != "" && p.License != "none" && p.Holder == "" {
		p.Holder = gitUserName()
		if err := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("License copyright holder").
				Value(&p.Holder).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("copyright holder cannot be empty")
					}
					return nil
				}),
		)).Run(); err != nil {
			return nil, err
		}
	}

	// 4. Template options (built dynamically from the descriptor)
	if p.Options == nil {
		p.Options = map[string]any{}
	}
	current, err := tmpl.CoerceOptions(p.Options)
	if err != nil {
		return nil, err
	}
	var optFields []huh.Field
	type binder struct {
		key    string
		commit func()
	}
	var binders []binder
	for _, o := range tmpl.Options {
		o := o
		switch o.Type {
		case template.OptionBoolean:
			v, _ := current[o.Key].(bool)
			binders = append(binders, binder{o.Key, func() { p.Options[o.Key] = v }})
			field := huh.NewConfirm().Title(o.Prompt).Description(o.Help).Value(&v)
			optFields = append(optFields, field)
		case template.OptionString:
			v, _ := current[o.Key].(string)
			binders = append(binders, binder{o.Key, func() { p.Options[o.Key] = v }})
			field := huh.NewInput().Title(o.Prompt).Description(o.Help).Value(&v)
			if o.Pattern != "" {
				re := regexp.MustCompile(o.Pattern)
				field.Validate(func(s string) error {
					if s != "" && !re.MatchString(s) {
						return fmt.Errorf("must match %s", o.Pattern)
					}
					return nil
				})
			}
			optFields = append(optFields, field)
		case template.OptionChoice:
			v, _ := current[o.Key].(string)
			binders = append(binders, binder{o.Key, func() { p.Options[o.Key] = v }})
			opts := make([]huh.Option[string], 0, len(o.Choices))
			for _, c := range o.Choices {
				opts = append(opts, huh.NewOption(c.Label, c.Value))
			}
			optFields = append(optFields, huh.NewSelect[string]().Title(o.Prompt).Description(o.Help).Options(opts...).Value(&v))
		case template.OptionMultiChoice:
			v, _ := current[o.Key].([]string)
			binders = append(binders, binder{o.Key, func() { p.Options[o.Key] = v }})
			opts := make([]huh.Option[string], 0, len(o.Choices))
			for _, c := range o.Choices {
				opts = append(opts, huh.NewOption(c.Label, c.Value))
			}
			optFields = append(optFields, huh.NewMultiSelect[string]().Title(o.Prompt).Description(o.Help).Options(opts...).Value(&v))
		}
	}
	if len(optFields) > 0 {
		if err := huh.NewForm(huh.NewGroup(optFields...)).Run(); err != nil {
			return nil, err
		}
		for _, b := range binders {
			b.commit()
		}
	}

	// 5. Summary confirmation
	var ok bool
	if err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(fmt.Sprintf("Create project?\n\n%s", summary(tmpl, p, current))).
			Affirmative("Create").
			Negative("Cancel").
			Value(&ok),
	)).Run(); err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("canceled")
	}
	return tmpl, nil
}

func summary(t *template.Template, p *generate.Params, options map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Template: %s\nName: %s\nModule: %s\nLicense: %s\n", t.Name, p.Name, p.Module, licenseDisplay(p))
	// Show the latest option values (binders have committed to p.Options and the summary
	// is built after commit; merge here: p.Options first, defaults as fallback).
	for _, o := range t.Options {
		v, present := p.Options[o.Key]
		if !present {
			v = options[o.Key]
		}
		fmt.Fprintf(&b, "%s: %v\n", o.Prompt, v)
	}
	return b.String()
}

func licenseDisplay(p *generate.Params) string {
	if p.License == "" || p.License == "none" {
		return "none"
	}
	return fmt.Sprintf("%s (%s)", p.License, p.Holder)
}

// gitUserName reads git config user.name as the default copyright holder.
func gitUserName() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
