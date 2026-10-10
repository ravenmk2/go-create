// Package wizard interactively fills in generation parameters.
//
// Interactive terminals get the bubbletea TUI (step indicator, live summary
// sidebar, forward/backward navigation); non-TTY environments (pipes,
// redirects) fall back to huh's accessible mode.
package wizard

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

// Run runs the wizard: TTY → bubbletea TUI, non-TTY → huh accessible fallback.
// It returns the selected template; cancellation by the user returns an error.
func Run(templates []*template.Template, p *generate.Params) (*template.Template, error) {
	if len(templates) == 0 {
		return nil, errors.New("no templates available")
	}
	if isTTY() {
		return runTUI(templates, p)
	}
	return runHuh(templates, p)
}

func isTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

// findTemplate resolves the selected template by name.
func findTemplate(templates []*template.Template, name string) (*template.Template, error) {
	for _, t := range templates {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template %q does not exist", name)
}

// gitUserName reads git config user.name as the default copyright holder.
func gitUserName() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
