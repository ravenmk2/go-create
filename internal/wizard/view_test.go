package wizard

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes escape sequences so plain-text assertions are possible.
func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// wordyTemplates returns a template whose prompts, help texts, and choice
// labels are deliberately longer than any reasonable main area.
func wordyTemplates() []*template.Template {
	return []*template.Template{{
		Name:        "wordy",
		Description: "Wordy template " + strings.Repeat("with a very long description ", 6),
		Options: []template.Option{
			{
				Key:    "verbose",
				Type:   template.OptionBoolean,
				Prompt: "Enable the extremely verbose and thoroughly documented mode of operation",
				Help:   strings.Repeat("detailed explanation of what this option controls ", 8),
			},
			{
				Key:    "pick",
				Type:   template.OptionChoice,
				Prompt: "Pick one",
				Choices: []template.Choice{
					{Value: "a", Label: "A choice with an extraordinarily long label that must wrap nicely within the main area"},
					{Value: "b", Label: "B"},
				},
			},
		},
	}}
}

// assertFits verifies every rendered line fits the terminal width and the
// whole view fits the terminal height.
func assertFits(t *testing.T, v string, width, height int) {
	t.Helper()
	lines := strings.Split(v, "\n")
	if len(lines) > height {
		t.Fatalf("view is %d lines tall, terminal height is %d", len(lines), height)
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > width {
			t.Fatalf("line %d is %d cells wide (max %d): %q", i, w, width, stripANSI(line))
		}
	}
}

// TestViewWrapsWithinWidth renders steps with long titles, help texts, and
// choice labels at framed and compact sizes; nothing may overflow.
func TestViewWrapsWithinWidth(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{
		{Width: 80, Height: 24}, // framed
		{Width: 72, Height: 20}, // framed, minimum size
		{Width: 40, Height: 14}, // compact
	} {
		// Everything preset: the flow starts directly on the options page.
		m := newModel(wordyTemplates(), &generate.Params{
			Template: "wordy", Name: "demo", Module: "example.com/demo", License: "none",
		})
		if m.err != nil {
			t.Fatal(m.err)
		}
		send(m, size)
		assertFits(t, m.View(), size.Width, size.Height)
		// The long choice labels of the second field must wrap too.
		send(m, key("enter"))
		assertFits(t, m.View(), size.Width, size.Height)
	}
}

// TestViewWrapsLongValidationError checks that a long validation error wraps
// instead of overflowing the main area.
func TestViewWrapsLongValidationError(t *testing.T) {
	p := &generate.Params{}
	m := newModel(wordyTemplates(), p)
	send(m, tea.WindowSizeMsg{Width: 50, Height: 16})
	f := m.curField()
	f.err = strings.Repeat("validation failed because the value is unacceptable ", 3)
	assertFits(t, m.View(), 50, 16)
}

// TestWrappedContentKeepsFocusVisible checks that the focus-tracking scroll
// still works when wrapped fields make the content taller than the viewport.
func TestWrappedContentKeepsFocusVisible(t *testing.T) {
	m := newModel(wordyTemplates(), &generate.Params{
		Template: "wordy", Name: "demo", Module: "example.com/demo", License: "none",
	})
	send(m, tea.WindowSizeMsg{Width: 40, Height: 8})
	send(m, key("enter")) // advance to the second (last) field
	content, ranges := m.stepContent()
	if lipgloss.Height(content) <= m.vp.Height {
		t.Fatal("wordy content should exceed the tiny viewport")
	}
	m.View() // rendering triggers refresh(), which scrolls to the focus
	last := ranges[len(ranges)-1]
	if bottom := last.y + last.height; bottom > m.vp.YOffset+m.vp.Height {
		t.Fatalf("focused field bottom %d is below the viewport (%d+%d)", bottom, m.vp.YOffset, m.vp.Height)
	}
}

// TestSidebarHierarchy checks the sidebar's visual hierarchy: the rows owned
// by the current step carry the accent marker, and empty values render their
// dimmed placeholder.
func TestSidebarHierarchy(t *testing.T) {
	p := &generate.Params{}
	m := newModel(testTemplates(), p)
	send(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	v := stripANSI(m.sidebarView())
	if !strings.Contains(v, "▸ Template:") {
		t.Fatal("Template row must be marked active on the template step")
	}
	if strings.Contains(v, "▸ Name:") {
		t.Fatal("Name row must not be active on the template step")
	}
	if !strings.Contains(v, "Name: <required>") || !strings.Contains(v, "Module: <required>") {
		t.Fatal("empty required values must show the <required> placeholder")
	}

	send(m, key("enter")) // → Project step (name + module)
	v = stripANSI(m.sidebarView())
	if strings.Contains(v, "▸ Template:") {
		t.Fatal("Template row must not stay active after leaving its step")
	}
	if !strings.Contains(v, "▸ Name:") || !strings.Contains(v, "▸ Module:") {
		t.Fatal("Name and Module rows must be active on the project step")
	}

	// The committed template stays visible as a plain (inactive) value.
	if !strings.Contains(v, "Template: cli") {
		t.Fatal("committed template value must remain in the sidebar")
	}
}

// TestSidebarHighlightsOptionRows checks that option rows are marked active on
// their own option page.
func TestSidebarHighlightsOptionRows(t *testing.T) {
	p := &generate.Params{Template: "web", Name: "demo", Module: "example.com/demo", License: "none"}
	m := newModel(testTemplates(), p)
	send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// The flow starts on web's single options page.
	if got := m.curStep().title; got != "Options" {
		t.Fatalf("current step = %q, want Options", got)
	}
	v := stripANSI(m.sidebarView())
	if !strings.Contains(v, "▸ Feature:") {
		t.Fatal("option rows must be marked active on their option page")
	}
	if strings.Contains(v, "▸ Name:") {
		t.Fatal("common rows must not be active on an option page")
	}
}
