// Package generate orchestrates project generation: validate parameters → render → write to disk → post-actions (git init / go mod tidy).
package generate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ravenmk2/go-create/internal/template"
)

// Params is the complete parameter set for one generation run.
type Params struct {
	Template string
	Name     string
	Module   string
	Dir      string // target directory, defaults to ./<Name>
	License  string // "" / none / mit / apache-2.0 / gpl-3.0
	Holder   string // copyright holder, required when License is set
	Options  map[string]any
	NoGit    bool
	NoTidy   bool
}

// Result is the generation result.
type Result struct {
	Dir      string
	Files    []string
	Warnings []string
}

var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
var moduleRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*(\.[A-Za-z0-9_-]+)*(/[A-Za-z0-9._-]+)+$`)

// PkgName sanitizes a project name into a valid Go identifier (strips hyphens and other invalid characters).
func PkgName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RepoSlug extracts owner/repo from a github.com module path; returns an empty string for non-GitHub modules.
func RepoSlug(module string) string {
	if !strings.HasPrefix(module, "github.com/") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(module, "github.com/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

// GoVersion returns the current toolchain version (without the go prefix), e.g. 1.26.4.
func GoVersion() string {
	return strings.TrimPrefix(runtime.Version(), "go")
}

// Validate checks the parameter set for completeness (in --yes non-interactive mode any missing value is an error).
func Validate(t *template.Template, p *Params) error {
	var missing []string
	if p.Name == "" {
		missing = append(missing, "project name (positional argument)")
	} else if !nameRE.MatchString(p.Name) {
		return fmt.Errorf("invalid project name %q (must start with an alphanumeric character; may contain letters, digits, ., -, _)", p.Name)
	}
	if p.Name != "" && PkgName(p.Name) == "" {
		return fmt.Errorf("project name %q cannot be converted into a valid Go package name", p.Name)
	}
	if p.Module == "" {
		missing = append(missing, "--module (module path, e.g. github.com/user/project)")
	} else if !moduleRE.MatchString(p.Module) {
		return fmt.Errorf("invalid module path %q", p.Module)
	}
	if !ValidLicense(p.License) {
		return fmt.Errorf("unsupported license %q (choose from %v or none)", p.License, LicenseIDs())
	}
	if p.License != "" && p.License != "none" && p.Holder == "" {
		missing = append(missing, "--holder (license copyright holder)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required parameters:\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}

// Generate runs the generation.
func Generate(t *template.Template, p Params) (*Result, error) {
	if err := Validate(t, &p); err != nil {
		return nil, err
	}
	options, err := t.CoerceOptions(p.Options)
	if err != nil {
		return nil, err
	}

	dir := p.Dir
	if dir == "" {
		dir = p.Name
	}
	if err := checkTargetDir(dir); err != nil {
		return nil, err
	}

	repo := RepoSlug(p.Module)
	var warnings []string
	if repo == "" {
		warnings = append(warnings, "module path is not of the form github.com/<owner>/<repo>; README badge and GHCR image name content is generated in a generic form")
	}
	if v, ok := options["docker_image"]; ok && v == true && repo == "" {
		warnings = append(warnings, "docker_image derives the GHCR image name from the GitHub repository path, which cannot be derived from the current module path; fix .goreleaser.yml manually after generation")
	}
	for _, o := range t.Options {
		if o.Type == template.OptionMultiChoice {
			if list, ok := options[o.Key].([]string); ok && len(list) == 0 {
				warnings = append(warnings, fmt.Sprintf("option %s is empty; related matrix/list content will be empty", o.Key))
			}
		}
	}

	ctx := &template.Context{
		Name:            p.Name,
		Module:          p.Module,
		Repo:            repo,
		PkgName:         PkgName(p.Name),
		GoVersion:       GoVersion(),
		License:         normalizeLicense(p.License),
		CopyrightHolder: p.Holder,
		Year:            time.Now().Year(),
		Options:         options,
	}

	files, renderWarnings, err := t.Render(ctx)
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, renderWarnings...)

	for _, f := range files {
		dest := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dest, f.Data, f.Mode); err != nil {
			return nil, err
		}
	}

	if ctx.License != "" {
		text, err := LicenseText(ctx.License, p.Holder, ctx.Year)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, "LICENSE"), []byte(text), 0o644); err != nil {
			return nil, err
		}
		files = append(files, template.RenderedFile{Path: "LICENSE"})
	}

	if !p.NoGit {
		if err := run(dir, "git", "init"); err != nil {
			warnings = append(warnings, fmt.Sprintf("git init failed (%v); run it manually", err))
		}
	}
	if !p.NoTidy {
		if err := run(dir, "go", "mod", "tidy"); err != nil {
			warnings = append(warnings, fmt.Sprintf("go mod tidy failed (%v); cd into %s and run it manually", err, dir))
		}
	}

	result := &Result{Dir: dir, Warnings: warnings}
	for _, f := range files {
		result.Files = append(result.Files, f.Path)
	}
	return result, nil
}

func normalizeLicense(id string) string {
	if id == "none" {
		return ""
	}
	return id
}

// checkTargetDir requires the target directory to not exist or be empty.
func checkTargetDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("target directory %s already exists and is not empty", dir)
	}
	return nil
}

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, out)
	}
	return nil
}
