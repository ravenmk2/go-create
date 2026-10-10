package wizard

import (
	"errors"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

// model is the bubbletea model driving the wizard form: an ordered list of
// steps, a focused field within the current step, and a scrollable main area.
type model struct {
	templates []*template.Template
	p         *generate.Params

	headLen  int     // number of template-independent prefix steps
	steps    []*Step // head steps + template-dependent tail (rebuilt on template change)
	stepIdx  int
	fieldIdx int // index into the current step's visible fields

	tmpl      *template.Template // selected template
	options   map[string]any     // coerced option values (prefill and summary fallback)
	builtFor  string             // template name the tail steps were built for
	confirmed bool               // final Create/Cancel choice

	width  int
	height int
	vp     viewport.Model

	done     bool
	canceled bool
	err      error
}

// runTUI runs the interactive bubbletea wizard and returns the selected
// template; cancellation returns a "canceled" error like the huh path.
func runTUI(templates []*template.Template, p *generate.Params) (*template.Template, error) {
	m := newModel(templates, p)
	if m.err != nil {
		return nil, m.err
	}
	// The alternate screen keeps the scrollback untouched, so the terminal is
	// clean after the wizard exits however it ends.
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return nil, err
	}
	switch {
	case m.err != nil:
		return nil, m.err
	case m.canceled, !m.confirmed:
		return nil, errors.New("canceled")
	default:
		return m.tmpl, nil
	}
}

func newModel(templates []*template.Template, p *generate.Params) *model {
	if p.Options == nil {
		p.Options = map[string]any{}
	}
	m := &model{templates: templates, p: p}
	head := buildSteps(templates, p)
	m.headLen = len(head)
	m.steps = head
	if p.Template == "" {
		// The template select cursor starts on the first entry; mirror that so
		// the sidebar and tail steps reflect the default selection.
		p.Template = templates[0].Name
	}
	m.syncTail()
	m.focus()
	return m
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			return m, m.advance()
		case "shift+tab":
			m.back()
			return m, nil
		case "up", "down":
			// Text inputs ignore arrows; use them as field navigation instead.
			if f := m.curField(); f != nil && f.kind == kindInput {
				if msg.String() == "up" {
					m.back()
					return m, nil
				}
				return m, m.advance()
			}
		case "pgup", "pgdown":
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
		if f := m.curField(); f != nil {
			f.err = "" // editing clears the previous validation error
			f.widget.update(msg)
		}
		return m, nil
	default:
		// Mouse wheel and friends scroll the main viewport.
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
}

// advance validates and commits the focused field, then moves to the next
// visible field or step; past the last step the wizard finishes.
func (m *model) advance() tea.Cmd {
	f := m.curField()
	if f == nil {
		return nil
	}
	if f.validate != nil {
		if errText := f.validate(); errText != "" {
			f.err = errText
			return nil
		}
	}
	f.err = ""
	if f.commit != nil {
		f.commit()
	}
	// The template may have changed; rebuild the dependent tail if so.
	m.syncTail()
	if m.err != nil {
		return tea.Quit
	}
	if m.fieldIdx < len(m.curFields())-1 {
		m.fieldIdx++
		m.focus()
		return nil
	}
	next := m.stepIdx + 1
	for next < len(m.steps) && len(m.steps[next].visibleFields()) == 0 {
		next++
	}
	if next < len(m.steps) {
		m.stepIdx = next
		m.fieldIdx = 0
		m.vp.GotoTop()
		m.focus()
		return nil
	}
	m.done = true
	return tea.Quit
}

// back moves to the previous visible field or step; committed values stay.
func (m *model) back() {
	if f := m.curField(); f != nil {
		f.err = ""
	}
	if m.fieldIdx > 0 {
		m.fieldIdx--
	} else {
		prev := m.stepIdx - 1
		for prev >= 0 && len(m.steps[prev].visibleFields()) == 0 {
			prev--
		}
		if prev < 0 {
			return
		}
		m.stepIdx = prev
		m.fieldIdx = len(m.curFields()) - 1
		m.vp.GotoTop()
	}
	m.focus()
}

// syncTail (re)builds the template-dependent tail (option pages + confirm
// step) when the selected template changed.
func (m *model) syncTail() {
	if m.p.Template == "" || m.builtFor == m.p.Template {
		return
	}
	tmpl, err := findTemplate(m.templates, m.p.Template)
	if err != nil {
		m.err = err
		return
	}
	// --set values that the new template does not declare are dropped.
	raw := make(map[string]any, len(m.p.Options))
	for k, v := range m.p.Options {
		if tmpl.Option(k) != nil {
			raw[k] = v
		}
	}
	current, err := tmpl.CoerceOptions(raw)
	if err != nil {
		m.err = err
		return
	}
	m.tmpl = tmpl
	m.options = current
	tail := append(optionSteps(tmpl, m.p, current), confirmStep(tmpl, m.p, current, &m.confirmed))
	m.steps = append(m.steps[:m.headLen], tail...)
	m.builtFor = m.p.Template
}

func (m *model) curStep() *Step {
	if m.stepIdx >= len(m.steps) {
		return nil
	}
	return m.steps[m.stepIdx]
}

func (m *model) curFields() []*Field {
	if s := m.curStep(); s != nil {
		return s.visibleFields()
	}
	return nil
}

func (m *model) curField() *Field {
	fields := m.curFields()
	if m.fieldIdx >= len(fields) {
		return nil
	}
	return fields[m.fieldIdx]
}

// focus syncs widget focus states (text input cursors) to the model position.
func (m *model) focus() {
	focused := m.curField()
	for _, s := range m.steps {
		for _, f := range s.fields {
			f.widget.setFocus(f == focused)
		}
	}
}

// activeSummaryKeys returns the sidebar row keys owned by the current step's
// visible fields; the sidebar highlights these rows.
func (m *model) activeSummaryKeys() map[string]bool {
	keys := map[string]bool{}
	for _, f := range m.curFields() {
		if f.summaryKey != "" {
			keys[f.summaryKey] = true
		}
	}
	return keys
}
