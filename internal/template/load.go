package template

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	builtin "github.com/ravenmk2/go-create/templates"
)

// UserTemplateDir returns the user custom template directory, unified across platforms as ~/.config/go-create/templates.
func UserTemplateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "go-create", "templates"), nil
}

// LoadAll loads all templates: builtin (embed) + user directory; a user template with the same
// name overrides the builtin one. Returns them sorted by name.
func LoadAll() ([]*Template, error) {
	byName := map[string]*Template{}

	entries, err := builtin.FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read builtin templates: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub, err := fs.Sub(builtin.FS, e.Name())
		if err != nil {
			return nil, err
		}
		t, err := parse(e.Name(), sub, "builtin")
		if err != nil {
			return nil, fmt.Errorf("builtin template %s: %w", e.Name(), err)
		}
		byName[e.Name()] = t
	}

	userDir, err := UserTemplateDir()
	if err == nil {
		if dirEntries, err := os.ReadDir(userDir); err == nil {
			for _, e := range dirEntries {
				if !e.IsDir() {
					continue
				}
				dir := filepath.Join(userDir, e.Name())
				if _, err := os.Stat(filepath.Join(dir, "template.yaml")); err != nil {
					continue
				}
				t, err := parse(e.Name(), os.DirFS(dir), "user")
				if err != nil {
					return nil, fmt.Errorf("user template %s: %w", e.Name(), err)
				}
				byName[e.Name()] = t
			}
		}
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	slices.Sort(names)
	out := make([]*Template, 0, len(names))
	for _, name := range names {
		out = append(out, byName[name])
	}
	return out, nil
}

// Load loads a single template by name.
func Load(name string) (*Template, error) {
	all, err := LoadAll()
	if err != nil {
		return nil, err
	}
	for _, t := range all {
		if t.Name == name {
			return t, nil
		}
	}
	names := make([]string, 0, len(all))
	for _, t := range all {
		names = append(names, t.Name)
	}
	return nil, fmt.Errorf("template %q does not exist (available: %v)", name, names)
}
