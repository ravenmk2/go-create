package template

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strings"
	texttemplate "text/template"
)

// Context is the render context: common parameters + template options.
type Context struct {
	Name            string // project name / binary name
	Module          string // go.mod module path
	Repo            string // owner/repo portion of a github.com module, empty otherwise
	PkgName         string // Name sanitized into a valid Go identifier
	GoVersion       string // e.g. 1.26.4
	License         string // mit / apache-2.0 / gpl-3.0 / "" (none)
	CopyrightHolder string
	Year            int
	Options         map[string]any
}

// RenderedFile is one output file pending write to disk.
type RenderedFile struct {
	Path string
	Data []byte
	Mode fs.FileMode
}

var templateFuncs = texttemplate.FuncMap{
	"lower": strings.ToLower,
	"has": func(list any, v string) bool {
		switch l := list.(type) {
		case []string:
			for _, s := range l {
				if s == v {
					return true
				}
			}
		case []any:
			for _, e := range l {
				if e == v {
					return true
				}
			}
		}
		return false
	},
}

// Render renders the whole output file tree per the manifest, returning the files and
// warnings (unlisted files, static files that appear to contain unrendered variables).
func (t *Template) Render(ctx *Context) ([]RenderedFile, []string, error) {
	var out []RenderedFile
	var warnings []string
	for _, u := range t.Unlisted {
		warnings = append(warnings, fmt.Sprintf("files/%s exists but is not listed in the template.yaml manifest; skipped", u))
	}

	for i := range t.Files {
		f := &t.Files[i]
		ok, err := f.applies(ctx.Options)
		if err != nil {
			return nil, nil, fmt.Errorf("file %s: when evaluation failed: %w", f.Src, err)
		}
		if !ok {
			continue
		}

		dest, err := renderDest(f.Dest, ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("file %s: dest rendering failed: %w", f.Src, err)
		}
		if err := validateDest(dest); err != nil {
			return nil, nil, fmt.Errorf("file %s: %w", f.Src, err)
		}

		data, err := fs.ReadFile(t.root, path.Join("files", f.Src))
		if err != nil {
			return nil, nil, fmt.Errorf("file %s: %w", f.Src, err)
		}

		if f.Render {
			data, err = renderContent(f.Src, data, f.left, f.right, ctx)
			if err != nil {
				return nil, nil, err
			}
		} else if bytes.Contains(data, []byte(t.Left)) && bytes.Contains(data, []byte(t.Right)) {
			warnings = append(warnings, fmt.Sprintf("%s is copied statically but its content contains delimiters %s…%s; mark render: true in the manifest if rendering is expected", f.Src, t.Left, t.Right))
		}

		mode := fs.FileMode(0o644)
		if strings.HasSuffix(dest, ".sh") {
			mode = 0o755
		}
		out = append(out, RenderedFile{Path: dest, Data: data, Mode: mode})
	}
	return out, warnings, nil
}

// applies evaluates the file-level when condition; it is always true without a condition.
func (f *FileEntry) applies(options map[string]any) (bool, error) {
	if f.prog == nil {
		return true, nil
	}
	out, err := runExpr(f.prog, options)
	if err != nil {
		return false, err
	}
	b, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("expression result is not a boolean: %v", out)
	}
	return b, nil
}

// renderDest renders the destination path: fixed {{ }} delimiters.
func renderDest(dest string, ctx *Context) (string, error) {
	if !strings.Contains(dest, "{{") {
		return dest, nil
	}
	tpl, err := texttemplate.New("dest").Funcs(templateFuncs).Option("missingkey=error").Parse(dest)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// validateDest validates the rendered destination path: relative, no .., non-empty.
func validateDest(dest string) error {
	if dest == "" {
		return fmt.Errorf("dest is empty after rendering")
	}
	if path.IsAbs(dest) || strings.HasPrefix(dest, "..") || strings.Contains(dest, "/../") {
		return fmt.Errorf("dest %q must be a relative path inside the project", dest)
	}
	return nil
}

// renderContent renders file content with the entry's delimiters.
func renderContent(src string, data []byte, left, right string, ctx *Context) ([]byte, error) {
	tpl, err := texttemplate.New(src).Delims(left, right).Funcs(templateFuncs).Option("missingkey=error").Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("file %s: template parse failed: %w", src, err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("file %s: template render failed: %w", src, err)
	}
	return buf.Bytes(), nil
}
