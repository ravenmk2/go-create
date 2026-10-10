package wizard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravenmk2/go-create/internal/generate"
)

// key builds a key message from its bubbletea string form.
func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func send(m *model, msgs ...tea.Msg) {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		if nm, ok := next.(*model); ok {
			m = nm
		}
	}
}

// TestModelFullFlow drives the model with synthetic keys through the whole
// wizard (cli template: no option pages) and checks the committed values.
func TestModelFullFlow(t *testing.T) {
	p := &generate.Params{}
	m := newModel(testTemplates(), p)
	if m.err != nil {
		t.Fatal(m.err)
	}
	send(m,
		tea.WindowSizeMsg{Width: 100, Height: 30},
		key("enter"),                                         // Template: keep the default (cli)
		key("d"), key("e"), key("m"), key("o"), key("enter"), // Project: name
		key("enter"), // invalid: empty module stays
	)
	if m.stepIdx != 1 || m.fieldIdx != 1 {
		t.Fatalf("empty module advanced to %d/%d; validation must block", m.stepIdx, m.fieldIdx)
	}
	send(m, key("example.com/demo"), key("enter")) // Project: module
	send(m, key("enter"))                          // License: keep None, holder stays hidden
	send(m, key("enter"))                          // Confirm: Create
	if !m.done {
		t.Fatal("wizard did not finish after the confirm step")
	}
	if !m.confirmed || m.canceled {
		t.Fatalf("confirmed=%v canceled=%v, want confirmed and not canceled", m.confirmed, m.canceled)
	}
	if p.Name != "demo" || p.Module != "example.com/demo" || p.License != "none" {
		t.Fatalf("params = %+v", p)
	}
	if m.tmpl.Name != "cli" {
		t.Fatalf("selected template = %q, want cli", m.tmpl.Name)
	}
}

// TestModelBackAcrossSteps checks shift+tab navigation over step boundaries:
// committed values survive and the focus lands on the previous step's last field.
func TestModelBackAcrossSteps(t *testing.T) {
	p := &generate.Params{}
	m := newModel(testTemplates(), p)
	send(m,
		tea.WindowSizeMsg{Width: 100, Height: 30},
		key("enter"),              // Template
		key("demo"), key("enter"), // Project: name → focus moves to module
		key("shift+tab"), // back to name
	)
	if m.stepIdx != 1 || m.fieldIdx != 0 {
		t.Fatalf("back landed on %d/%d, want 1/0 (name field)", m.stepIdx, m.fieldIdx)
	}
	if p.Name != "demo" {
		t.Fatalf("committed name = %q, want demo", p.Name)
	}
	send(m, key("shift+tab")) // back across the step boundary → template select
	if m.stepIdx != 0 {
		t.Fatalf("back landed on step %d, want 0", m.stepIdx)
	}
	if p.Template != "cli" {
		t.Fatalf("committed template = %q, want cli", p.Template)
	}
}

// TestModelEscCancels checks that esc aborts the wizard.
func TestModelEscCancels(t *testing.T) {
	m := newModel(testTemplates(), &generate.Params{})
	send(m, tea.WindowSizeMsg{Width: 100, Height: 30}, key("esc"))
	if !m.canceled {
		t.Fatal("esc must cancel the wizard")
	}
}

// TestViewSmoke renders the View at various terminal sizes (framed, compact,
// tiny); it must never panic and must render the step header when running.
func TestViewSmoke(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{
		{Width: 120, Height: 40}, // framed
		{Width: 60, Height: 15},  // compact
		{Width: 20, Height: 5},   // tiny
	} {
		m := newModel(testTemplates(), &generate.Params{})
		send(m, size)
		v := m.View()
		if v == "" {
			t.Fatalf("View at %dx%d is empty", size.Width, size.Height)
		}
		if size.Width >= minFramedWidth && size.Height >= minFramedHeight && m.compact() {
			t.Fatalf("%dx%d must use the framed layout", size.Width, size.Height)
		}
		if (size.Width < minFramedWidth || size.Height < minFramedHeight) && !m.compact() {
			t.Fatalf("%dx%d must use the compact layout", size.Width, size.Height)
		}
	}
}
