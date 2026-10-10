package generate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ravenmk2/go-create/internal/template"
)

// TestTemplateMatrix iterates over the tests combinations declared by each builtin template:
// actually generate → assert no warnings and no residual delimiters → go mod tidy + go build to verify the output compiles.
// go build needs network access to fetch dependencies and is skipped in -short mode.
func TestTemplateMatrix(t *testing.T) {
	templates, err := template.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range templates {
		if tmpl.Source != "builtin" {
			continue
		}
		if len(tmpl.Tests) == 0 {
			t.Errorf("template %s does not declare a tests self-check matrix", tmpl.Name)
			continue
		}
		for _, tc := range tmpl.Tests {
			t.Run(tmpl.Name+"/"+testCaseName(tc), func(t *testing.T) {
				t.Parallel()
				dir := filepath.Join(t.TempDir(), "testproj")
				res, err := Generate(tmpl, Params{
					Name:    "testproj",
					Module:  "github.com/ravenmk2/testproj",
					Dir:     dir,
					License: "mit",
					Holder:  "Tester",
					Options: tc.Set,
					NoGit:   true,
					NoTidy:  true,
				})
				if err != nil {
					t.Fatalf("generation failed: %v", err)
				}
				for _, w := range res.Warnings {
					t.Errorf("generation warning: %s", w)
				}
				assertNoResidue(t, dir)
				assertOptionGatedFiles(t, tmpl.Name, dir, tc)

				if !testing.Short() {
					// module fetching can hit transient proxy failures; retry like Generate does
					if err := retry(3, 2*time.Second, func() error {
						return runCmdE(dir, nil, "go", "mod", "tidy")
					}); err != nil {
						t.Fatal(err)
					}
					// embed scenario: create the web/dist placeholder file before compiling per convention (same as the CI step)
					if _, err := os.Stat(filepath.Join(dir, "web", "embed.go")); err == nil {
						distDir := filepath.Join(dir, "web", "dist")
						if err := os.MkdirAll(distDir, 0o755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<!-- placeholder -->\n"), 0o644); err != nil {
							t.Fatal(err)
						}
					}
					runCmd(t, dir, []string{"CGO_ENABLED=0"}, "go", "build", "./...")
				}
			})
		}
	}
}

func testCaseName(tc template.TestCase) string {
	if tc.Name != "" {
		return tc.Name
	}
	return "default"
}

// assertNoResidue ensures the output contains no residual <% delimiters or unrendered {{ }} path variables.
func assertNoResidue(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if strings.Contains(rel, "{{") {
			t.Errorf("path contains an unrendered variable: %s", rel)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "<%") {
			t.Errorf("%s contains residual delimiter <%%", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// assertOptionGatedFiles spot-checks option-gated files of known templates.
func assertOptionGatedFiles(t *testing.T, tmplName, dir string, tc template.TestCase) {
	t.Helper()
	has := func(rel string) bool {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		return err == nil
	}
	switch tmplName {
	case "web":
		if v, _ := tc.Set["docker_image"].(bool); v != has("Dockerfile") {
			t.Errorf("docker_image=%v does not match Dockerfile presence", v)
		}
		if v, _ := tc.Set["embed_frontend"].(bool); v != has("web/embed.go") {
			t.Errorf("embed_frontend=%v does not match web/embed.go presence", v)
		}
		if v, _ := tc.Set["sqlite"].(bool); v != has("internal/store/store.go") {
			t.Errorf("sqlite=%v does not match internal/store/store.go presence", v)
		}
	case "cli":
		wantRelease := true
		if v, ok := tc.Set["goreleaser"].(bool); ok {
			wantRelease = v
		}
		if wantRelease != has(".goreleaser.yml") {
			t.Errorf("goreleaser=%v does not match .goreleaser.yml presence", wantRelease)
		}
	case "lib":
		if has("cmd") {
			t.Error("library should not generate a cmd/ directory")
		}
	}
}

func runCmd(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	if err := runCmdE(dir, env, name, args...); err != nil {
		t.Fatal(err)
	}
}

func runCmdE(dir string, env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return nil
}
