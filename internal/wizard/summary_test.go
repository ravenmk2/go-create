package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

func TestSidebarLongValuesRemainSingleLine(t *testing.T) {
	tmpl, err := template.Load("web")
	if err != nil {
		t.Fatal(err)
	}
	p := &generate.Params{Template: "web", Name: "my-app", Module: "github.com/a-very-long-organization/my-app", License: "mit", Holder: strings.Repeat("作者", 20), Options: map[string]any{"description": strings.Repeat("很长的项目描述 👩‍💻 ", 15)}}
	m := newModel([]*template.Template{tmpl}, p)
	send(m, tea.WindowSizeMsg{Width: 116, Height: 32})
	view := stripANSI(m.sidebarView())
	if !strings.Contains(view, "…") || !strings.Contains(view, "my-app") {
		t.Fatalf("long values should be abbreviated: %s", view)
	}
	assertFits(t, view, sidebarWidth, m.vp.Height)
	if strings.Count(view, "Desc:") != 1 || strings.Contains(view, "Features") {
		t.Fatal("sidebar should contain only project rows and the current page")
	}
	// Row positions must remain stable when focus changes within this page.
	before := len(strings.Split(view, "\n"))
	send(m, key("enter"))
	if got := lipgloss.Height(m.sidebarView()); got != before {
		t.Fatalf("sidebar height changed on focus movement: %d -> %d", before, got)
	}
}

func TestConfirmStartsAtTopAndScrollsWithPinnedActions(t *testing.T) {
	tmpl, err := template.Load("web")
	if err != nil {
		t.Fatal(err)
	}
	p := &generate.Params{Template: "web", Name: "my-app", Module: "github.com/a-very-long-organization/my-app", License: "none", Options: map[string]any{"description": strings.Repeat("A long description ", 12)}}
	m := newModel([]*template.Template{tmpl}, p)
	m.stepIdx = len(m.steps) - 1
	m.fieldIdx = 0
	m.focus()
	send(m, tea.WindowSizeMsg{Width: 104, Height: 20})
	view := stripANSI(m.View())
	if m.showSidebar() || m.vp.YOffset != 0 || !strings.Contains(view, "Project") || !strings.Contains(view, "Create project") {
		t.Fatalf("confirmation must start at the top with visible actions:\n%s", view)
	}
	send(m, tea.KeyMsg{Type: tea.KeyPgDown})
	scrolled := stripANSI(m.View())
	if m.vp.YOffset == 0 || !strings.Contains(scrolled, "Create project") {
		t.Fatal("summary must scroll independently of the confirmation actions")
	}
	full := stripANSI(m.confirmationView(m.vp.Width))
	if !strings.Contains(full, p.Module) || strings.Contains(full, "…") || !strings.Contains(full, "✓ On") || !strings.Contains(full, "– Off") {
		t.Fatalf("confirmation must preserve full values and readable switches:\n%s", full)
	}
	send(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.confirmed {
		t.Fatal("right arrow must select Cancel while the summary is scrolled")
	}
	assertFits(t, m.View(), 104, 20)
}

func TestConfirmColorsAndPlainTextFallback(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
	tmpl := testTemplates()[1]
	p := &generate.Params{Template: "web", Name: "demo", Module: "example.com/demo", Options: map[string]any{"feat": true}}
	m := newModel([]*template.Template{tmpl}, p)
	if got := m.confirmationView(80); !strings.Contains(got, "\x1b[") {
		t.Fatal("confirmation must use terminal highlighting")
	}
	if got := summary(tmpl, p, m.options); strings.Contains(got, "\x1b[") || strings.Contains(got, "true") {
		t.Fatal("accessible confirmation must have readable values and no ANSI codes")
	}
}
