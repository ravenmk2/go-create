package generate

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestProgressTracksActualWork(t *testing.T) {
	tmpl := loadBuiltin(t, "lib")
	dir := filepath.Join(t.TempDir(), "project")
	var events []Progress
	result, err := GenerateWithProgress(tmpl, Params{
		Name: "project", Module: "github.com/owner/project", Dir: dir, NoGit: true, NoTidy: true,
	}, func(event Progress) {
		events = append(events, event)
		if event.Step == StepWrite && event.Status == Running {
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatal("Write must start before files are created")
			}
		}
		if event.Step == StepWrite && event.Status == Completed {
			if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
				t.Fatalf("Write must complete after files exist: %v", err)
			}
		}
	})
	if err != nil || result == nil {
		t.Fatalf("generation failed: %v", err)
	}
	wantSteps := []Step{StepRender, StepRender, StepWrite, StepWrite, StepGit, StepTidy}
	wantStates := []Status{Running, Completed, Running, Completed, Skipped, Skipped}
	if len(events) != len(wantSteps) {
		t.Fatalf("unexpected progress: %+v", events)
	}
	for i, event := range events {
		if event.Step != wantSteps[i] || event.Status != wantStates[i] {
			t.Fatalf("event %d = %+v", i, event)
		}
	}
}

func TestProgressStopsAtFatalError(t *testing.T) {
	for _, phase := range []Step{StepRender, StepWrite} {
		t.Run(phase.String(), func(t *testing.T) {
			p := Params{Name: "project", Module: "github.com/owner/project", Dir: filepath.Join(t.TempDir(), "blocked")}
			if phase == StepRender {
				p.Module = ""
			} else if err := os.WriteFile(p.Dir, []byte("keep"), 0o644); err != nil {
				t.Fatal(err)
			}
			var events []Progress
			result, err := GenerateWithProgress(loadBuiltin(t, "lib"), p, func(e Progress) { events = append(events, e) })
			if err == nil || result != nil {
				t.Fatal("expected fatal generation error")
			}
			last := events[len(events)-1]
			if last.Step != phase || last.Status != Failed || last.Message == "" {
				t.Fatalf("missing failure event: %+v", events)
			}
			for _, event := range events {
				if event.Step > phase {
					t.Fatal("a later phase ran after failure")
				}
			}
		})
	}
}

func TestProgressPostActionWarningsAndRetries(t *testing.T) {
	// Missing tools force portable, immediate failures without network access.
	t.Setenv("PATH", t.TempDir())
	var events []Progress
	result, err := GenerateWithProgress(loadBuiltin(t, "lib"), Params{
		Name: "project", Module: "github.com/owner/project", Dir: t.TempDir(),
	}, func(e Progress) { events = append(events, e) })
	if err != nil || result == nil {
		t.Fatalf("post-action warnings must preserve the generated project: %v", err)
	}
	for _, phase := range []Step{StepGit, StepTidy} {
		if !slices.ContainsFunc(events, func(e Progress) bool {
			return e.Step == phase && e.Status == Warning && len(e.Warnings) > 0
		}) {
			t.Fatalf("missing warning for %s: %+v", phase, events)
		}
	}
	if !slices.ContainsFunc(events, func(e Progress) bool { return strings.Contains(e.Message, "attempt 3/3") }) {
		t.Fatal("retry progress must be visible")
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "README.md")); err != nil {
		t.Fatal(err)
	}
}
