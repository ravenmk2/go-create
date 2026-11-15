package wizard

import (
	"errors"
	"strconv"

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

	tmpl       *template.Template // selected template
	options    map[string]any     // coerced option values (prefill and summary fallback)
	builtFor   string             // template name the tail steps were built for
	confirmed  bool               // final Create/Cancel choice
	preview    *generate.Plan
	previewErr error
	editMenu   *selectWidget
	editReturn bool

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
	if m.reviewing() {
		m.enterReview()
	}
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
		if m.editMenu != nil {
			switch msg.String() {
			case "ctrl+c":
				m.canceled = true
				return m, tea.Quit
			case "esc", "e", "E":
				m.editMenu = nil
				m.vp.GotoTop()
				return m, nil
			case "enter":
				i, _ := strconv.Atoi(m.editMenu.value())
				m.editMenu = nil
				m.stepIdx, m.fieldIdx, m.editReturn = i, 0, true
				m.preview = nil
				m.vp.GotoTop()
				m.focus()
				return m, nil
			default:
				m.editMenu.update(msg)
				return m, nil
			}
		}
		if m.reviewing() && (msg.String() == "e" || msg.String() == "E") {
			// Values supplied on the command line are skipped initially, but
			// remain editable from the review page.
			head := []*Step{templateSelectStep(m.templates, m.p), commonStep(m.p, true), licenseStep(m.p, true)}
			m.steps = append(head, m.steps[m.headLen:]...)
			m.headLen = len(head)
			m.stepIdx = len(m.steps) - 1
			var labels, values []string
			for i, step := range m.steps[:len(m.steps)-1] {
				labels = append(labels, step.title)
				values = append(values, strconv.Itoa(i))
			}
			if len(labels) > 0 {
				m.editMenu = newSelectWidget(labels, values, "0", nil)
				m.vp.GotoTop()
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			return m, m.advance()
		case "tab":
			// Tab navigates fields but never submits the final confirmation.
			if !m.reviewing() {
				return m, m.advance()
			}
			return m, nil
		case "shift+tab":
			m.back()
			return m, nil
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
	if m.reviewing() && m.confirmed {
		// Refresh once on submit, so conflicts that appeared while reviewing
		// are shown for another explicit confirmation rather than overwritten.
		previous := m.preview
		m.prepareReview()
		if m.previewErr != nil {
			return nil
		}
		if previous != nil && !sameOverwrites(previous, m.preview) {
			m.confirmed = false
			f.widget.(*selectWidget).cursor = 1
			return nil
		}
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
	if m.editReturn && !m.reviewing() {
		next = len(m.steps) - 1
		m.editReturn = false
	}
	for next < len(m.steps) && len(m.steps[next].visibleFields()) == 0 {
		next++
	}
	if next < len(m.steps) {
		m.stepIdx = next
		m.fieldIdx = 0
		m.vp.GotoTop()
		m.focus()
		if m.reviewing() {
			m.enterReview()
		}
		return nil
	}
	m.done = true
	return tea.Quit
}

func (m *model) prepareReview() {
	m.preview, m.previewErr = generate.Preview(m.tmpl, *m.p)
}

func (m *model) enterReview() {
	m.prepareReview()
	// Overwriting starts on Cancel; the user must explicitly choose the action.
	if m.preview != nil && len(m.preview.Overwrites()) > 0 && !m.p.DryRun {
		w := m.steps[len(m.steps)-1].fields[0].widget.(*selectWidget)
		w.cursor, m.confirmed = 1, false
	}
}

func sameOverwrites(a, b *generate.Plan) bool {
	x, y := a.Overwrites(), b.Overwrites()
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
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
	m.p.Options = raw
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
