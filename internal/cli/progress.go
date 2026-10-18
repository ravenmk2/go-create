package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"

	"github.com/ravenmk2/go-create/internal/generate"
)

type generationReporter struct {
	out      io.Writer
	accent   lipgloss.Style
	success  lipgloss.Style
	warning  lipgloss.Style
	failure  lipgloss.Style
	seen     map[generate.Step]bool
	states   map[generate.Step]generate.Status
	failed   generate.Step
	warnings int
}

func colorOutput(out io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := out.(*os.File)
	return ok && (isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()))
}

func newGenerationReporter(out io.Writer, color bool) *generationReporter {
	r := lipgloss.NewRenderer(out)
	profile := termenv.Ascii
	if color {
		profile = termenv.TrueColor
	}
	r.SetColorProfile(profile)
	style := func(light, dark string) lipgloss.Style {
		return r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: light, Dark: dark})
	}
	return &generationReporter{
		out: out, seen: map[generate.Step]bool{}, states: map[generate.Step]generate.Status{},
		accent: style("#367478", "#91C6C9"), success: style("#216643", "#8ED3AE"),
		warning: style("#80591B", "#EDC57D"), failure: style("#AD3838", "#F09B99"),
	}
}

func (r *generationReporter) update(event generate.Progress) {
	first := !r.seen[event.Step]
	if first {
		_, _ = fmt.Fprintf(r.out, "%s %s\n", r.accent.Render(fmt.Sprintf("[%d/%d]", event.Step, generate.StepCount)), event.Step)
		r.seen[event.Step] = true
	}
	r.states[event.Step] = event.Status
	mark, style := "◐", r.accent
	switch event.Status {
	case generate.Running:
		// The step heading is printed before work starts; retries add detail.
		if first {
			return
		}
	case generate.Completed:
		mark, style = "✓", r.success
	case generate.Skipped:
		mark = "–"
	case generate.Warning:
		mark, style = "!", r.warning
	case generate.Failed:
		mark, style = "✗", r.failure
		r.failed = event.Step
	}
	_, _ = fmt.Fprintln(r.out, style.Render("  "+mark+" "+event.Message))
	for _, warning := range event.Warnings {
		_, _ = fmt.Fprintln(r.out, r.warning.Render("    "+strings.ReplaceAll(warning, "\n", "\n    ")))
	}
	r.warnings += len(event.Warnings)
}

func (r *generationReporter) finish(name string, result *generate.Result, err error) {
	if err != nil {
		_, _ = fmt.Fprintf(r.out, "\n%s\n", r.failure.Render("✗ Generation failed at "+r.failed.String()))
		if r.failed == generate.StepWrite {
			_, _ = fmt.Fprintln(r.out, "Some files may have been written. Check the error above before retrying.")
		}
		return
	}
	status := r.success.Render("✓ Project created")
	if r.warnings > 0 {
		status = r.warning.Render(fmt.Sprintf("! Project created with %d warning(s)", r.warnings))
	}
	completed, skipped, warned := 0, 0, 0
	for _, state := range r.states {
		switch state {
		case generate.Completed:
			completed++
		case generate.Skipped:
			skipped++
		case generate.Warning:
			warned++
		}
	}
	counts := fmt.Sprintf("%d completed · %d skipped", completed, skipped)
	if warned > 0 {
		counts += fmt.Sprintf(" · %d with warnings", warned)
	}
	_, _ = fmt.Fprintf(r.out, "\n%s\n%s · %d files · %s\nDirectory: %s\n",
		status, name, len(result.Files), counts, r.accent.Render(result.Dir))
	_, _ = fmt.Fprintf(r.out, "\nNext steps:\n  cd %s\n", shellDirectory(result.Dir))
	if r.states[generate.StepGit] == generate.Warning {
		_, _ = fmt.Fprintln(r.out, r.accent.Render("  git init"))
	}
	if state := r.states[generate.StepTidy]; state == generate.Warning || state == generate.Skipped {
		_, _ = fmt.Fprintln(r.out, r.accent.Render("  go mod tidy"))
	}
}

// Single quotes keep directory names containing spaces or shell expansions
// literal in both PowerShell and POSIX shells.
func shellDirectory(dir string) string {
	if !strings.ContainsAny(dir, " \t\r\n'$`\";&|()<>*?[]{}!\\") {
		return dir
	}
	if os.PathSeparator == '\\' {
		return "'" + strings.ReplaceAll(dir, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(dir, "'", "'\"'\"'") + "'"
}
