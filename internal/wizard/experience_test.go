package wizard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

func reachReview(t *testing.T, m *model) {
	t.Helper()
	for i := 0; i < 100 && !m.reviewing() && m.err == nil; i++ {
		send(m, key("enter"))
	}
	if !m.reviewing() || m.err != nil || m.previewErr != nil {
		t.Fatalf("review failed: %v / %v", m.err, m.previewErr)
	}
}

func TestTabValidatesFieldsAndNeverSubmitsReview(t *testing.T) {
	p := &generate.Params{Template: "cli", License: "none", Dir: t.TempDir()}
	m := newModel(testTemplates(), p)
	send(m, key("bad name"), key("tab"))
	if m.fieldIdx != 0 || m.curField().err == "" {
		t.Fatal("Tab accepted invalid name")
	}
	m.curField().widget.(*textWidget).setValue("demo")
	send(m, key("tab"))
	if m.fieldIdx != 1 {
		t.Fatal("Tab did not move to module")
	}
	send(m, key("invalid path/repo"), key("tab"))
	if m.fieldIdx != 1 || m.curField().err == "" {
		t.Fatal("invalid module accepted")
	}
	m.curField().widget.(*textWidget).setValue("github.com/user/demo")
	send(m, key("tab"))
	if !m.reviewing() {
		t.Fatal("valid fields did not advance")
	}
	send(m, key("tab"))
	if m.done {
		t.Fatal("Tab submitted confirmation")
	}
}

func TestReviewEditsPresetProjectAndReturnsDirectly(t *testing.T) {
	tmpl, err := template.Load("lib")
	if err != nil {
		t.Fatal(err)
	}
	p := &generate.Params{Template: "lib", Name: "demo", Module: "github.com/user/demo", License: "none", Dir: t.TempDir()}
	m := newModel([]*template.Template{tmpl}, p)
	reachReview(t, m)
	send(m, key("e"), key("down"), key("enter")) // Project, including CLI preset values.
	if m.curStep().title != "Project" {
		t.Fatal("edit menu did not open Project")
	}
	m.curField().widget.(*textWidget).setValue("renamed")
	send(m, key("tab"), key("tab"))
	if !m.reviewing() || p.Name != "renamed" || m.previewErr != nil {
		t.Fatal("editing did not return to review")
	}
	var found bool
	for _, f := range m.preview.Files {
		if f.Path == "renamed.go" {
			found = true
		}
	}
	if !found {
		t.Fatal("review preview did not refresh after edit")
	}
}

func TestReviewIntegratesOverwriteConfirmation(t *testing.T) {
	tmpl, err := template.Load("lib")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &generate.Params{Template: "lib", Name: "demo", Module: "github.com/user/demo", License: "none", Dir: dir}
	m := newModel([]*template.Template{tmpl}, p)
	reachReview(t, m)
	if m.confirmed || !strings.Contains(stripANSI(m.confirmationView(80)), "README.md") || !strings.Contains(stripANSI(m.confirmActions(80)), "Overwrite") {
		t.Fatal("overwrite must be explicit and list conflicts")
	}
	send(m, tea.KeyMsg{Type: tea.KeyLeft}, key("enter"))
	if !m.done || !m.confirmed {
		t.Fatal("explicit overwrite confirmation failed")
	}
	data, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil || string(data) != "sentinel" {
		t.Fatal("wizard wrote files before generation")
	}
}

func TestNewConflictRequiresAnotherConfirmation(t *testing.T) {
	tmpl, err := template.Load("lib")
	if err != nil {
		t.Fatal(err)
	}
	p := &generate.Params{Template: "lib", Name: "demo", Module: "github.com/user/demo", License: "none", Dir: t.TempDir()}
	m := newModel([]*template.Template{tmpl}, p)
	reachReview(t, m)
	if err := os.WriteFile(filepath.Join(p.Dir, "README.md"), []byte("new file"), 0o644); err != nil {
		t.Fatal(err)
	}
	send(m, key("enter"))
	if m.done || m.confirmed || len(m.preview.Overwrites()) != 1 {
		t.Fatal("new conflict was not brought back for confirmation")
	}
}

func TestReviewAndEditLayoutsFit(t *testing.T) {
	tmpl, err := template.Load("web")
	if err != nil {
		t.Fatal(err)
	}
	m := newModel([]*template.Template{tmpl}, &generate.Params{Template: "web", Name: "demo", Module: "github.com/user/demo", License: "none", Dir: t.TempDir(), DryRun: true})
	reachReview(t, m)
	for _, size := range []tea.WindowSizeMsg{{Width: 116, Height: 32}, {Width: 80, Height: 24}, {Width: 40, Height: 14}, {Width: 20, Height: 8}} {
		send(m, size)
		assertFits(t, m.View(), size.Width, size.Height)
		send(m, key("e"))
		assertFits(t, m.View(), size.Width, size.Height)
		send(m, key("esc"))
	}
}

func TestBooleanDensityAndSingleHighlight(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	f := optionField(template.Option{Key: "enabled", Type: template.OptionBoolean, Prompt: "Enable feature"}, &generate.Params{Options: map[string]any{}}, true)
	if got := f.view(true, 60); lipgloss.Height(got) != 1 {
		t.Fatalf("short boolean should occupy one line: %q", got)
	}
	assertFits(t, f.view(true, 16), 16, 8)
	m := newModel(testTemplates(), &generate.Params{})
	send(m, tea.WindowSizeMsg{Width: 116, Height: 32})
	if strings.Contains(m.sidebarView(), "48;2;") {
		t.Fatal("sidebar competes with control background highlight")
	}
}
