package wizard

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

func testTemplates() []*template.Template {
	return []*template.Template{
		{Name: "cli", Description: "CLI application"},
		{Name: "web", Description: "Web server", Options: []template.Option{
			{Key: "feat", Type: template.OptionBoolean, Prompt: "Feature", Default: true},
			{Key: "name_opt", Type: template.OptionString, Prompt: "Name option", Pattern: `^[a-z]+$`},
			{Key: "fw", Type: template.OptionChoice, Prompt: "Framework", Default: "a",
				Choices: []template.Choice{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}},
			{Key: "platforms", Type: template.OptionMultiChoice, Prompt: "Platforms", Default: []string{"linux"},
				Choices: []template.Choice{{Value: "linux", Label: "Linux"}, {Value: "win", Label: "Windows"}}},
		}},
	}
}

func opt(key, group string) template.Option {
	return template.Option{Key: key, Type: template.OptionBoolean, Prompt: key, Group: group}
}

func TestPaginateOptionsGrouped(t *testing.T) {
	pages := paginateOptions([]template.Option{
		opt("a", "Basic"), opt("b", "CI"), opt("c", "Basic"), opt("d", "CI"),
	})
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if pages[0].title != "Basic" || pages[1].title != "CI" {
		t.Fatalf("page titles = %q, %q; want Basic, CI (first-appearance order)", pages[0].title, pages[1].title)
	}
	if got := pages[0].options; len(got) != 2 || got[0].Key != "a" || got[1].Key != "c" {
		t.Fatalf("Basic page options = %v, want [a c]", got)
	}
}

func TestPaginateOptionsUngroupedChunking(t *testing.T) {
	var options []template.Option
	for _, k := range []string{"o1", "o2", "o3", "o4", "o5", "o6", "o7"} {
		options = append(options, opt(k, ""))
	}
	pages := paginateOptions(options)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if len(pages[0].options) != maxUngroupedPerPage || len(pages[1].options) != 2 {
		t.Fatalf("page sizes = %d, %d; want 5, 2", len(pages[0].options), len(pages[1].options))
	}
	for _, pg := range pages {
		if pg.title != ungroupedTitle || pg.grouped {
			t.Errorf("ungrouped page = %q (grouped=%v), want %q", pg.title, pg.grouped, ungroupedTitle)
		}
	}
}

func TestPaginateOptionsMixed(t *testing.T) {
	pages := paginateOptions([]template.Option{
		opt("u1", ""), opt("g1", "Group"), opt("u2", ""), opt("u3", ""),
	})
	if len(pages) != 3 {
		t.Fatalf("got %d pages, want 3", len(pages))
	}
	if pages[0].title != ungroupedTitle || pages[1].title != "Group" || pages[2].title != ungroupedTitle {
		t.Fatalf("titles = %q, %q, %q; want %q, Group, %q",
			pages[0].title, pages[1].title, pages[2].title, ungroupedTitle, ungroupedTitle)
	}
	if len(pages[2].options) != 2 {
		t.Fatalf("last ungrouped page has %d options, want 2", len(pages[2].options))
	}
}

func TestPaginateOptionsGroupNamedLikeUngrouped(t *testing.T) {
	// A declared group called "Options" must not merge with ungrouped pages.
	pages := paginateOptions([]template.Option{opt("u", ""), opt("g", "Options")})
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if pages[0].grouped || !pages[1].grouped {
		t.Fatalf("grouped flags = %v, %v; want false, true", pages[0].grouped, pages[1].grouped)
	}
}

func stepTitles(steps []*Step) []string {
	titles := make([]string, 0, len(steps))
	for _, s := range steps {
		titles = append(titles, s.title)
	}
	return titles
}

func TestBuildStepsAllMissing(t *testing.T) {
	steps := buildSteps(testTemplates(), &generate.Params{})
	want := []string{"Template", "Project", "License"}
	if !slices.Equal(stepTitles(steps), want) {
		t.Fatalf("steps = %v, want %v", stepTitles(steps), want)
	}
}

func TestBuildStepsSkipsProvidedValues(t *testing.T) {
	cases := []struct {
		name   string
		params generate.Params
		want   []string
	}{
		{"template preset", generate.Params{Template: "web"}, []string{"Project", "License"}},
		{"name preset", generate.Params{Name: "demo"}, []string{"Template", "Project", "License"}},
		{"name+module preset", generate.Params{Name: "demo", Module: "example.com/demo"}, []string{"Template", "License"}},
		{"license none preset", generate.Params{License: "none"}, []string{"Template", "Project"}},
		{"license+holder preset", generate.Params{License: "mit", Holder: "Raven"}, []string{"Template", "Project"}},
		{"license preset without holder", generate.Params{License: "mit"}, []string{"Template", "Project", "License"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := stepTitles(buildSteps(testTemplates(), &c.params)); !slices.Equal(got, c.want) {
				t.Fatalf("steps = %v, want %v", got, c.want)
			}
		})
	}
}

func TestBuildStepsPartialCommon(t *testing.T) {
	// Only the module field remains when the name is preset.
	steps := buildSteps(testTemplates(), &generate.Params{Name: "demo"})
	if len(steps[1].fields) != 1 || steps[1].fields[0].title != "Module path" {
		t.Fatalf("common step fields = %v, want only the module input", steps[1].fields)
	}
}

func TestLicenseStepHolderVisibility(t *testing.T) {
	p := &generate.Params{}
	s := licenseStep(p)
	if len(s.fields) != 2 {
		t.Fatalf("license step has %d fields, want 2 (license + holder)", len(s.fields))
	}
	holder := s.fields[1]
	if holder.isVisible() {
		t.Fatal("holder must be hidden before a license is chosen")
	}
	p.License = "none"
	if holder.isVisible() {
		t.Fatal("holder must stay hidden for license none")
	}
	p.License = "mit"
	if !holder.isVisible() {
		t.Fatal("holder must appear once a real license is chosen")
	}
}

func TestHolderStepPrefillsGitDefault(t *testing.T) {
	p := &generate.Params{License: "mit"}
	s := licenseStep(p)
	if len(s.fields) != 1 || s.fields[0].kind != kindInput {
		t.Fatalf("holder step fields = %v, want one input", s.fields)
	}
	// gitUserName may be empty outside a git environment; the field must still
	// reject blank values.
	s.fields[0].widget.(*textWidget).setValue("  ")
	if got := s.fields[0].validate(); got == "" {
		t.Fatal("blank holder must fail validation")
	}
}

func TestOptionFieldCommitAndValidate(t *testing.T) {
	tmpl := testTemplates()[1]
	p := &generate.Params{Options: map[string]any{}}
	current, err := tmpl.CoerceOptions(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Boolean: default true commits to p.Options.
	bf := optionField(tmpl.Options[0], p, current["feat"])
	bf.commit()
	if p.Options["feat"] != true {
		t.Fatalf("feat = %v, want true", p.Options["feat"])
	}

	// String with pattern: invalid value fails, empty value passes.
	sf := optionField(tmpl.Options[1], p, current["name_opt"])
	sf.widget.(*textWidget).setValue("ABC")
	if got := sf.validate(); got == "" {
		t.Fatal("uppercase value must fail the pattern")
	}
	sf.widget.(*textWidget).setValue("")
	if got := sf.validate(); got != "" {
		t.Fatalf("empty value must skip pattern validation, got %q", got)
	}
	sf.widget.(*textWidget).setValue("abc")
	if got := sf.validate(); got != "" {
		t.Fatalf("lowercase value must pass, got %q", got)
	}

	// Choice: cursor follows the coerced value and commits the selection.
	cf := optionField(tmpl.Options[2], p, current["fw"])
	cf.commit()
	if p.Options["fw"] != "a" {
		t.Fatalf("fw = %v, want a", p.Options["fw"])
	}

	// MultiChoice: default selection prefilled, space toggles.
	mf := optionField(tmpl.Options[3], p, current["platforms"])
	mf.commit()
	if got := p.Options["platforms"]; !slices.Equal(got.([]string), []string{"linux"}) {
		t.Fatalf("platforms = %v, want [linux]", got)
	}
	mw := mf.widget.(*multiWidget)
	mw.update(tea.KeyMsg{Type: tea.KeyDown})
	mw.update(tea.KeyMsg{Type: tea.KeySpace})
	mf.commit()
	if got := p.Options["platforms"]; !slices.Equal(got.([]string), []string{"linux", "win"}) {
		t.Fatalf("platforms = %v, want [linux win]", got)
	}
}

func TestOptionStepsPrefillFromSetValues(t *testing.T) {
	tmpl := testTemplates()[1]
	p := &generate.Params{Options: map[string]any{"fw": "b"}}
	current, err := tmpl.CoerceOptions(p.Options)
	if err != nil {
		t.Fatal(err)
	}
	steps := optionSteps(tmpl, p, current)
	if len(steps) != 1 || len(steps[0].fields) != 4 {
		t.Fatalf("option steps = %d pages / %d fields, want 1/4", len(steps), len(steps[0].fields))
	}
	var choice *Field
	for _, f := range steps[0].fields {
		if f.title == "Framework" {
			choice = f
		}
	}
	if choice == nil {
		t.Fatal("Framework field not found")
	}
	if got := choice.widget.(*selectWidget).value(); got != "b" {
		t.Fatalf("Framework prefilled with %q, want b (--set value)", got)
	}
}

func TestConfirmStepCreateCancel(t *testing.T) {
	tmpl := testTemplates()[1]
	p := &generate.Params{Options: map[string]any{}}
	current, err := tmpl.CoerceOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	var confirmed bool
	s := confirmStep(tmpl, p, current, &confirmed)
	if !confirmed {
		t.Fatal("confirm step must default to Create")
	}
	f := s.fields[0]
	f.commit()
	if !confirmed {
		t.Fatal("committing with the cursor on Create must keep confirmed=true")
	}
	// Moving the cursor to Cancel and committing turns the result into cancel.
	f.widget.update(tea.KeyMsg{Type: tea.KeyDown})
	f.commit()
	if confirmed {
		t.Fatal("committing Cancel must set confirmed=false")
	}
}

func TestNewModelBuildsTail(t *testing.T) {
	p := &generate.Params{}
	m := newModel(testTemplates(), p)
	if m.err != nil {
		t.Fatal(m.err)
	}
	// Template select defaults to the first entry, so the tail is built for it.
	if m.tmpl == nil || m.tmpl.Name != "cli" || p.Template != "cli" {
		t.Fatalf("tmpl = %v, p.Template = %q; want cli/cli", m.tmpl, p.Template)
	}
	// cli declares no options: tail is just the confirmation step.
	want := []string{"Template", "Project", "License", "Confirm"}
	if !slices.Equal(stepTitles(m.steps), want) {
		t.Fatalf("steps = %v, want %v", stepTitles(m.steps), want)
	}
}

func TestSyncTailRebuildsOnTemplateChange(t *testing.T) {
	p := &generate.Params{}
	m := newModel(testTemplates(), p)
	if m.err != nil {
		t.Fatal(m.err)
	}
	p.Template = "web"
	m.syncTail()
	if m.err != nil {
		t.Fatal(m.err)
	}
	if m.tmpl.Name != "web" || m.builtFor != "web" {
		t.Fatalf("tail built for %q, want web", m.builtFor)
	}
	// web declares 4 ungrouped options: one option page + confirmation.
	want := []string{"Template", "Project", "License", "Options", "Confirm"}
	if !slices.Equal(stepTitles(m.steps), want) {
		t.Fatalf("steps = %v, want %v", stepTitles(m.steps), want)
	}
}

func TestAdvanceValidatesBeforeProceeding(t *testing.T) {
	p := &generate.Params{}
	m := newModel(testTemplates(), p)
	m.stepIdx, m.fieldIdx = 1, 0 // Project step, name input
	if cmd := m.advance(); cmd != nil {
		t.Fatal("advance must not finish on an invalid field")
	}
	if m.stepIdx != 1 || m.fieldIdx != 0 {
		t.Fatalf("position moved to %d/%d; invalid name must not advance", m.stepIdx, m.fieldIdx)
	}
	if m.curField().err == "" {
		t.Fatal("validation error must be recorded on the field")
	}
}
