package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"

	"github.com/ravenmk2/go-create/internal/generate"
)

// confirmNonEmptyDir asks for explicit confirmation when the target directory
// already exists and is not empty. Interactive flows only — --yes skips it.
func confirmNonEmptyDir(p *generate.Params) error {
	dir := p.Dir
	if dir == "" {
		dir = p.Name
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	shown := min(len(entries), 5)
	names := make([]string, 0, shown)
	for _, e := range entries[:shown] {
		names = append(names, e.Name())
	}
	more := ""
	if len(entries) > shown {
		more = fmt.Sprintf(" … and %d more", len(entries)-shown)
	}

	var ok bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(fmt.Sprintf("Target directory %s is not empty (%s%s).\nConflicting files will be overwritten. Continue?", dir, strings.Join(names, ", "), more)).
			Affirmative("Continue").
			Negative("Cancel").
			Value(&ok),
	))
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		form = form.WithAccessible(true)
	}
	if err := form.Run(); err != nil {
		return err
	}
	if !ok {
		return errors.New("canceled")
	}
	return nil
}
