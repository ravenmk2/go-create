package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
)

func TestGenerationLogStates(t *testing.T) {
	for _, state := range []generate.Status{generate.Completed, generate.Warning, generate.Failed, generate.Skipped} {
		t.Run(string(state), func(t *testing.T) {
			var out bytes.Buffer
			r := newGenerationReporter(&out, false)
			r.update(generate.Progress{Step: generate.StepTidy, Status: generate.Running})
			if !strings.Contains(out.String(), "[4/4] Resolve dependencies") {
				t.Fatal("the phase must be visible before it completes")
			}
			event := generate.Progress{Step: generate.StepTidy, Status: state, Message: "dependency result"}
			if state == generate.Warning {
				event.Warnings = []string{"network unavailable"}
			}
			r.update(event)
			var err error
			if state == generate.Failed {
				err = errors.New("generation failed")
			}
			r.finish("my-app", &generate.Result{Dir: "my app", Files: []string{"README.md"}}, err)
			log := out.String()
			if strings.Contains(log, "\x1b[") || strings.Count(log, "[4/4]") != 1 {
				t.Fatalf("redirected logs must be plain and avoid duplicate phase headings: %s", log)
			}
			if state == generate.Failed {
				if strings.Contains(log, "Project created") || !strings.Contains(log, "Generation failed at Resolve dependencies") {
					t.Fatalf("fatal error must never report success: %s", log)
				}
			} else if state == generate.Warning {
				if !strings.Contains(log, "created with 1 warning") || !strings.Contains(log, "network unavailable") || !strings.Contains(log, "  go mod tidy") {
					t.Fatalf("warnings must retain the project and show recovery steps: %s", log)
				}
			} else if state == generate.Skipped && !strings.Contains(log, "1 skipped") {
				t.Fatal("skips must be counted separately")
			}
		})
	}
}

func TestGenerationColorAndNoColor(t *testing.T) {
	var out bytes.Buffer
	r := newGenerationReporter(&out, true)
	r.update(generate.Progress{Step: generate.StepWrite, Status: generate.Completed, Message: "Files written"})
	if !strings.Contains(out.String(), "\x1b[") {
		t.Fatal("TTY logs must have color")
	}
	t.Setenv("NO_COLOR", "1")
	if colorOutput(os.Stdout) {
		t.Fatal("NO_COLOR must disable terminal highlighting")
	}
	if colorOutput(&out) {
		t.Fatal("redirected output must not have color")
	}
}

func TestRunGenerateLogsAndFailure(t *testing.T) {
	tmpl, err := template.Load("lib")
	if err != nil {
		t.Fatal(err)
	}
	for _, fail := range []bool{false, true} {
		dir := filepath.Join(t.TempDir(), "project with spaces")
		if fail {
			if err := os.WriteFile(dir, []byte("blocked"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)
		err := runGenerate(cmd, tmpl, generate.Params{Name: "project", Module: "github.com/owner/project", Dir: dir, NoGit: true, NoTidy: true})
		if fail {
			if err == nil || !cmd.SilenceErrors || strings.Contains(out.String(), "Project created") {
				t.Fatalf("write error must retain its exit status and avoid a success message: %v\n%s", err, out.String())
			}
		} else if err != nil || !strings.Contains(out.String(), "2 completed · 2 skipped") || !strings.Contains(out.String(), "cd '"+dir+"'") {
			t.Fatalf("unexpected success result: %v\n%s", err, out.String())
		}
	}
}
