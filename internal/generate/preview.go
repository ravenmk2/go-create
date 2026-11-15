package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ravenmk2/go-create/internal/template"
)

// PlannedFile describes an output without writing it.
type PlannedFile struct {
	Path      string
	Overwrite bool
}

// Plan is a read-only preview of the same rendering used by Generate.
type Plan struct {
	Dir      string
	Files    []PlannedFile
	Warnings []string
	NonEmpty bool
}

// Preview validates and renders the project, then inspects destination files.
// It never creates directories or runs post-actions.
func Preview(t *template.Template, p Params) (*Plan, error) {
	dir, files, warnings, err := renderProject(t, p)
	if err != nil {
		return nil, err
	}
	plan := &Plan{Dir: dir, Warnings: warnings}
	root, err := os.OpenRoot(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if root != nil {
		defer func() { _ = root.Close() }()
		d, err := root.Open(".")
		if err != nil {
			return nil, err
		}
		entries, readErr := d.ReadDir(-1)
		_ = d.Close()
		if readErr != nil {
			return nil, readErr
		}
		plan.NonEmpty = len(entries) > 0
	}
	for _, f := range files {
		item := PlannedFile{Path: f.Path}
		if root != nil {
			info, err := root.Stat(filepath.FromSlash(f.Path))
			switch {
			case err == nil:
				if info.IsDir() {
					return nil, fmt.Errorf("output %s conflicts with a directory", f.Path)
				}
				item.Overwrite = true
			case !os.IsNotExist(err):
				return nil, fmt.Errorf("inspect output %s: %w", f.Path, err)
			}
		}
		plan.Files = append(plan.Files, item)
	}
	return plan, nil
}

// Overwrites returns the paths whose existing contents would be replaced.
func (p *Plan) Overwrites() []string {
	var paths []string
	for _, f := range p.Files {
		if f.Overwrite {
			paths = append(paths, f.Path)
		}
	}
	return paths
}

// Summary is plain text suitable for both terminal and accessible confirmations.
func (p *Plan) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d files", len(p.Files))
	if paths := p.Overwrites(); len(paths) > 0 {
		fmt.Fprintf(&b, "\nOverwrite %d existing file(s):", len(paths))
		for _, name := range paths {
			fmt.Fprintf(&b, "\n  ! %s", name)
		}
	} else if p.NonEmpty {
		b.WriteString("\nExisting directory; no files will be overwritten.")
	}
	for _, w := range p.Warnings {
		fmt.Fprintf(&b, "\n! %s", w)
	}
	return b.String()
}
