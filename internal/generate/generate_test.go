package generate

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ravenmk2/go-create/internal/template"
)

func TestGeneratePackageNames(t *testing.T) {
	tmpl := loadBuiltin(t, "lib")
	for _, name := range []string{"123app", "type", "func", "my-app", "My.Lib", "my_lib"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "project")
			_, err := Generate(tmpl, Params{Name: name, Module: "github.com/user/demo", Dir: dir, NoGit: true, NoTidy: true})
			if name == "123app" || name == "type" || name == "func" {
				if err == nil || !strings.Contains(err.Error(), "package name") {
					t.Fatalf("invalid package name error = %v", err)
				}
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatalf("invalid input created output directory: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name+".go"), nil, parser.AllErrors)
			if err != nil {
				t.Fatal(err)
			}
			if f.Name.Name != PkgName(name) {
				t.Fatalf("package = %s, want %s", f.Name.Name, PkgName(name))
			}
		})
	}
}

func TestGenerateRejectsEscapingSymlinks(t *testing.T) {
	tmpl := loadBuiltin(t, "lib")
	for _, name := range []string{"README.md", "LICENSE", ".github"} {
		t.Run(name, func(t *testing.T) {
			dir, outside := t.TempDir(), t.TempDir()
			target := filepath.Join(outside, "sentinel")
			if name == ".github" {
				target = outside
			} else if err := os.WriteFile(target, []byte("untouched"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			_, err := Generate(tmpl, Params{Name: "demo", Module: "github.com/user/demo", Dir: dir, License: "mit", Holder: "Tester", NoGit: true, NoTidy: true})
			if err == nil {
				t.Fatal("generation followed an escaping symlink")
			}
			if name == ".github" {
				entries, err := os.ReadDir(outside)
				if err != nil || len(entries) != 0 {
					t.Fatalf("external directory changed: %v, %v", entries, err)
				}
			} else {
				data, err := os.ReadFile(target)
				if err != nil || string(data) != "untouched" {
					t.Fatalf("external file changed: %q, %v", data, err)
				}
			}
		})
	}
}

func loadBuiltin(t *testing.T, name string) *template.Template {
	t.Helper()
	tmpl, err := template.Load(name)
	if err != nil {
		t.Fatalf("load template %s: %v", name, err)
	}
	return tmpl
}

func hasWarning(res *Result, substr string) bool {
	return slices.ContainsFunc(res.Warnings, func(w string) bool {
		return strings.Contains(w, substr)
	})
}

// TestGenerateIntoNonEmptyDir: generation into a non-empty directory succeeds,
// conflicting files are overwritten with a warning, unrelated files survive.
func TestGenerateIntoNonEmptyDir(t *testing.T) {
	tmpl := loadBuiltin(t, "lib")
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("untouched"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Generate(tmpl, Params{
		Name:   "testproj",
		Module: "github.com/ravenmk2/testproj",
		Dir:    dir,
		NoGit:  true,
		NoTidy: true,
	})
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}
	if !hasWarning(res, "overwrote existing file README.md") {
		t.Errorf("expected overwrite warning, got %v", res.Warnings)
	}
	data, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "sentinel" {
		t.Error("README.md was not overwritten")
	}
	data, err = os.ReadFile(filepath.Join(dir, "keep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "untouched" {
		t.Error("unrelated file keep.txt was modified")
	}
}

// TestRetry: fn is retried until success, up to the attempt limit.
func TestRetry(t *testing.T) {
	failNTimes := func(n int) (func() error, *int) {
		calls := new(int)
		return func() error {
			*calls++
			if *calls <= n {
				return fmt.Errorf("failure %d", *calls)
			}
			return nil
		}, calls
	}

	fn, calls := failNTimes(2)
	if err := retry(3, 0, fn); err != nil {
		t.Fatalf("expected success on 3rd attempt, got %v", err)
	}
	if *calls != 3 {
		t.Errorf("expected 3 calls, got %d", *calls)
	}

	fn, calls = failNTimes(10)
	if err := retry(3, 0, fn); err == nil {
		t.Fatal("expected error after exhausting attempts")
	}
	if *calls != 3 {
		t.Errorf("expected 3 calls, got %d", *calls)
	}
}

// TestGenerateSkipsGitInitWithExistingDotGit: an existing .git directory skips
// git init with a warning instead of reinitializing.
func TestGenerateSkipsGitInitWithExistingDotGit(t *testing.T) {
	tmpl := loadBuiltin(t, "lib")
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Generate(tmpl, Params{
		Name:   "testproj",
		Module: "github.com/ravenmk2/testproj",
		Dir:    dir,
		NoTidy: true,
	})
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}
	if !hasWarning(res, "skipping git init") {
		t.Errorf("expected git init skip warning, got %v", res.Warnings)
	}
}
