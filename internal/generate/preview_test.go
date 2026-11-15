package generate

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPreviewMatchesGenerationWithoutSideEffects(t *testing.T) {
	tmpl := loadBuiltin(t, "web")
	for _, existing := range []bool{false, true} {
		p := Params{Name: "demo", Module: "github.com/user/demo", Dir: filepath.Join(t.TempDir(), "demo"), License: "mit", Holder: "Tester", Options: map[string]any{"docker_dev": false, "docker_image": false}}
		if existing {
			if err := os.Mkdir(p.Dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"README.md", "LICENSE", "keep.txt"} {
				if err := os.WriteFile(filepath.Join(p.Dir, name), []byte("sentinel"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
		plan, err := Preview(tmpl, p)
		if err != nil {
			t.Fatal(err)
		}
		if plan.NonEmpty != existing {
			t.Fatalf("NonEmpty=%v", plan.NonEmpty)
		}
		if existing {
			if got := plan.Overwrites(); !slices.Equal(got, []string{"README.md", "LICENSE"}) {
				t.Fatalf("overwrites=%v", got)
			}
			entries, err := os.ReadDir(p.Dir)
			if err != nil || len(entries) != 3 {
				t.Fatalf("preview changed files: %v %v", entries, err)
			}
			for _, name := range []string{"README.md", "LICENSE", "keep.txt"} {
				data, err := os.ReadFile(filepath.Join(p.Dir, name))
				if err != nil || string(data) != "sentinel" {
					t.Fatalf("preview wrote %s", name)
				}
			}
		} else if _, err := os.Stat(p.Dir); !os.IsNotExist(err) {
			t.Fatalf("preview created directory: %v", err)
		}
		p.NoGit, p.NoTidy = true, true
		result, err := Generate(tmpl, p)
		if err != nil {
			t.Fatal(err)
		}
		var paths []string
		for _, f := range plan.Files {
			paths = append(paths, f.Path)
		}
		if !slices.Equal(paths, result.Files) {
			t.Fatalf("preview=%v, generated=%v", paths, result.Files)
		}
		if slices.Contains(paths, "Dockerfile") {
			t.Fatal("preview ignored options")
		}
	}
}

func TestPreviewRejectsInvalidInputAndDirectoryConflict(t *testing.T) {
	tmpl := loadBuiltin(t, "lib")
	p := Params{Name: "demo", Module: "github.com/user/demo", Dir: t.TempDir()}
	if err := os.Mkdir(filepath.Join(p.Dir, "README.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(tmpl, p); err == nil {
		t.Fatal("directory conflict accepted")
	}
	p.Name = "bad name"
	if _, err := Preview(tmpl, p); err == nil {
		t.Fatal("invalid name accepted")
	}
}

func TestGenerateDryRunSkipsAllSideEffects(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "project")
	var skipped []Step
	result, err := GenerateWithProgress(loadBuiltin(t, "lib"), Params{Name: "demo", Module: "github.com/user/demo", Dir: dir, DryRun: true}, func(p Progress) {
		if p.Status == Skipped {
			skipped = append(skipped, p.Step)
		}
	})
	if err != nil || result == nil || len(result.Files) == 0 {
		t.Fatalf("dry-run result: %v, %v", result, err)
	}
	if !slices.Equal(skipped, []Step{StepWrite, StepGit, StepTidy}) {
		t.Fatalf("skipped=%v", skipped)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote files: %v", err)
	}
}
