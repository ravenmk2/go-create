package wizard

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// widget is a single interactive control inside a Field. The model forwards
// key messages to the focused field's widget; widgets push value changes
// through the onChange callback wired at construction, so the sidebar summary
// stays live.
type widget interface {
	update(msg tea.KeyMsg)
	view(focused bool, width int) string
	setFocus(focused bool)
}

// textWidget wraps bubbles/textinput for free-form string entry.
type textWidget struct {
	input    textinput.Model
	onChange func(string)
}

func newTextWidget(value, placeholder string, onChange func(string)) *textWidget {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Placeholder = placeholder
	ti.Prompt = ""
	return &textWidget{input: ti, onChange: onChange}
}

func (w *textWidget) update(msg tea.KeyMsg) {
	var cmd tea.Cmd
	w.input, cmd = w.input.Update(msg)
	_ = cmd // blink commands are ignored; the static cursor is enough
	if w.onChange != nil {
		w.onChange(w.input.Value())
	}
}

func (w *textWidget) view(_ bool, width int) string {
	if width > 4 {
		w.input.Width = width - 4
	}
	return w.input.View()
}

func (w *textWidget) setFocus(focused bool) {
	if focused {
		w.input.Focus()
	} else {
		w.input.Blur()
	}
}

func (w *textWidget) value() string { return w.input.Value() }

// setValue replaces the content programmatically (e.g. injecting a default).
func (w *textWidget) setValue(v string) { w.input.SetValue(v) }

// selectWidget is a single-choice cursor list (up/down move, enter confirms
// at the field level).
type selectWidget struct {
	labels   []string
	values   []string
	cursor   int
	onChange func(string)
}

func newSelectWidget(labels, values []string, current string, onChange func(string)) *selectWidget {
	w := &selectWidget{labels: labels, values: values, onChange: onChange}
	for i, v := range values {
		if v == current {
			w.cursor = i
			break
		}
	}
	return w
}

func (w *selectWidget) update(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k":
		if w.cursor > 0 {
			w.cursor--
			w.changed()
		}
	case "down", "j":
		if w.cursor < len(w.values)-1 {
			w.cursor++
			w.changed()
		}
	}
}

func (w *selectWidget) changed() {
	if w.onChange != nil {
		w.onChange(w.value())
	}
}

func (w *selectWidget) view(focused bool, width int) string {
	if width < 1 {
		width = 1
	}
	wrap := lipgloss.NewStyle().Width(width)
	var b strings.Builder
	for i, label := range w.labels {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := "  " + label
		if i == w.cursor {
			line = "▸ " + label
			if focused {
				line = accentStyle.Render(line)
			}
		}
		b.WriteString(wrap.Render(line))
	}
	return b.String()
}

func (w *selectWidget) setFocus(bool) {}

func (w *selectWidget) value() string { return w.values[w.cursor] }

// multiWidget is a multi-choice cursor list: up/down move, space toggles the
// item under the cursor, enter confirms at the field level.
type multiWidget struct {
	labels   []string
	values   []string
	cursor   int
	selected []bool
	onChange func([]string)
}

func newMultiWidget(labels, values []string, current []string, onChange func([]string)) *multiWidget {
	w := &multiWidget{labels: labels, values: values, selected: make([]bool, len(values)), onChange: onChange}
	for i, v := range values {
		w.selected[i] = slices.Contains(current, v)
	}
	return w
}

func (w *multiWidget) update(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k":
		if w.cursor > 0 {
			w.cursor--
		}
	case "down", "j":
		if w.cursor < len(w.values)-1 {
			w.cursor++
		}
	case " ":
		w.selected[w.cursor] = !w.selected[w.cursor]
		w.changed()
	}
}

func (w *multiWidget) changed() {
	if w.onChange != nil {
		w.onChange(w.value())
	}
}

func (w *multiWidget) view(focused bool, width int) string {
	if width < 1 {
		width = 1
	}
	wrap := lipgloss.NewStyle().Width(width)
	var b strings.Builder
	for i, label := range w.labels {
		if i > 0 {
			b.WriteByte('\n')
		}
		box := "[ ]"
		if w.selected[i] {
			box = "[x]"
		}
		line := "  " + fmt.Sprintf("%s %s", box, label)
		if i == w.cursor {
			line = "▸ " + fmt.Sprintf("%s %s", box, label)
			if focused {
				line = accentStyle.Render(line)
			}
		}
		b.WriteString(wrap.Render(line))
	}
	return b.String()
}

func (w *multiWidget) setFocus(bool) {}

func (w *multiWidget) value() []string {
	out := make([]string, 0, len(w.values))
	for i, v := range w.values {
		if w.selected[i] {
			out = append(out, v)
		}
	}
	return out
}
