// Package template loads, validates, and renders project templates.
//
// A template is a directory: template.yaml (the descriptor) + files/ (all source files).
// The files manifest in the descriptor defines each entry: source file → whether to render → content delimiters → destination path → generation condition.
package template

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gopkg.in/yaml.v3"
)

// OptionType describes an option type (Jenkins-style naming) and determines both its form widget and data type:
// boolean→bool (confirm), string→string (text input), choice→string (single choice), multichoice→[]string (multi choice).
type OptionType string

const (
	OptionBoolean     OptionType = "boolean"
	OptionString      OptionType = "string"
	OptionChoice      OptionType = "choice"
	OptionMultiChoice OptionType = "multichoice"
)

// Choice is one selectable value of a choice / multichoice option.
type Choice struct {
	Value string `yaml:"value"`
	Label string `yaml:"label"`
}

// Option is an option declared in the template descriptor.
type Option struct {
	Key     string     `yaml:"key"`
	Type    OptionType `yaml:"type"`
	Prompt  string     `yaml:"prompt"`
	Help    string     `yaml:"help"`
	Default any        `yaml:"default"`
	Choices []Choice   `yaml:"choices"`
	Pattern string     `yaml:"pattern"` // regex validation for the input
}

// FileEntry is one mapping in the output file manifest.
type FileEntry struct {
	Src    string `yaml:"src"`    // source file path relative to files/
	Dest   string `yaml:"dest"`   // output relative path, a fixed {{ }} variable template; defaults to src (without the .tpl suffix)
	Render bool   `yaml:"render"` // whether to pass through the template engine
	Delims string `yaml:"delims"` // delimiter preset name (ecb/go/brackets); defaults to the template-level setting
	When   string `yaml:"when"`   // expr-lang conditional expression

	left, right string
	prog        *vm.Program
}

// TestCase is one option combination of the self-test matrix.
type TestCase struct {
	Name string         `yaml:"name"`
	Set  map[string]any `yaml:"set"`
}

// SchemaVersion is the template.yaml schema version this build supports.
const SchemaVersion = 1

// Template is a fully loaded template.
type Template struct {
	Name        string
	Description string
	Schema      int
	Options     []Option
	Files       []FileEntry
	Tests       []TestCase
	Source      string // builtin | user
	Left, Right string // default delimiters for content rendering
	Unlisted    []string

	root fs.FS // template directory (containing template.yaml and files/)
}

// delimValue is a delimiter pair.
type delimValue [2]string

var delimPresets = map[string]delimValue{
	"ecb":      {"<%", "%>"},
	"go":       {"{{", "}}"},
	"brackets": {"[[", "]]"},
}

// UnmarshalYAML supports a preset name (ecb) or a custom two-element list (["<<", ">>"]).
func (d *delimValue) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var preset string
		if err := value.Decode(&preset); err != nil {
			return err
		}
		v, ok := delimPresets[preset]
		if !ok {
			return fmt.Errorf("unknown delimiter preset %q (choose ecb / go / brackets, or a custom [left, right])", preset)
		}
		*d = v
		return nil
	}
	var pair []string
	if err := value.Decode(&pair); err != nil {
		return fmt.Errorf("delims must be a preset name or a [left, right] list: %w", err)
	}
	if len(pair) != 2 || pair[0] == "" || pair[1] == "" {
		return fmt.Errorf("custom delims list must have exactly two non-empty elements, got %v", pair)
	}
	*d = delimValue{pair[0], pair[1]}
	return nil
}

type spec struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Schema      int         `yaml:"schema"`
	Delims      delimValue  `yaml:"delims"`
	Options     []Option    `yaml:"options"`
	Files       []FileEntry `yaml:"files"`
	Tests       []TestCase  `yaml:"tests"`
}

var optionKeyRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// parse loads and validates a template from a template directory.
func parse(dirName string, root fs.FS, source string) (*Template, error) {
	data, err := fs.ReadFile(root, "template.yaml")
	if err != nil {
		return nil, fmt.Errorf("read template.yaml: %w", err)
	}
	var s spec
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse template.yaml: %w", err)
	}

	t := &Template{
		Name:        s.Name,
		Description: s.Description,
		Schema:      s.Schema,
		Options:     s.Options,
		Files:       s.Files,
		Tests:       s.Tests,
		Source:      source,
		Left:        s.Delims[0],
		Right:       s.Delims[1],
		root:        root,
	}
	if t.Name == "" {
		t.Name = dirName
	}
	if t.Schema == 0 {
		t.Schema = SchemaVersion
	}
	if t.Schema != SchemaVersion {
		return nil, fmt.Errorf("unsupported template schema version %d (supported: %d)", t.Schema, SchemaVersion)
	}
	if t.Left == "" {
		t.Left, t.Right = delimPresets["ecb"][0], delimPresets["ecb"][1]
	}

	if err := t.validateOptions(); err != nil {
		return nil, err
	}
	if err := t.prepareFiles(); err != nil {
		return nil, err
	}
	t.collectUnlisted()
	return t, nil
}

func (t *Template) validateOptions() error {
	for i, o := range t.Options {
		if !optionKeyRE.MatchString(o.Key) {
			return fmt.Errorf("option #%d: invalid key %q (must match %s)", i, o.Key, optionKeyRE)
		}
		switch o.Type {
		case OptionBoolean, OptionString, OptionChoice, OptionMultiChoice:
		default:
			return fmt.Errorf("option %s: unknown type %q (choose boolean/string/choice/multichoice)", o.Key, o.Type)
		}
		if o.Prompt == "" {
			return fmt.Errorf("option %s: missing prompt", o.Key)
		}
		if o.Type == OptionChoice || o.Type == OptionMultiChoice {
			if len(o.Choices) == 0 {
				return fmt.Errorf("option %s: type %s requires choices", o.Key, o.Type)
			}
		}
		if o.Type == OptionString && o.Pattern != "" {
			if _, err := regexp.Compile(o.Pattern); err != nil {
				return fmt.Errorf("option %s: pattern is not a valid regex: %w", o.Key, err)
			}
		}
		if o.Default != nil {
			if _, err := t.coerceValue(o, o.Default); err != nil {
				return fmt.Errorf("option %s: invalid default: %w", o.Key, err)
			}
		}
	}
	return nil
}

// prepareFiles validates the file manifest, resolves delimiters, and compiles when expressions.
func (t *Template) prepareFiles() error {
	env := t.exprEnv()
	for i := range t.Files {
		f := &t.Files[i]
		if f.Src == "" {
			return fmt.Errorf("file #%d: missing src", i)
		}
		if strings.Contains(f.Src, "..") || path.IsAbs(f.Src) {
			return fmt.Errorf("file %s: src must be a relative path inside files/", f.Src)
		}
		if _, err := fs.Stat(t.root, path.Join("files", f.Src)); err != nil {
			return fmt.Errorf("file %s: src does not exist under files/", f.Src)
		}
		if f.Dest == "" {
			f.Dest = strings.TrimSuffix(f.Src, ".tpl")
		}
		f.left, f.right = t.Left, t.Right
		if f.Delims != "" {
			pair, ok := delimPresets[f.Delims]
			if !ok {
				return fmt.Errorf("file %s: unknown delimiter preset %q", f.Src, f.Delims)
			}
			f.left, f.right = pair[0], pair[1]
		}
		if f.When != "" {
			prog, err := expr.Compile(f.When, expr.Env(env), expr.AsBool())
			if err != nil {
				return fmt.Errorf("file %s: failed to compile when expression: %w", f.Src, err)
			}
			f.prog = prog
		}
	}
	return nil
}

// collectUnlisted finds files that exist under files/ but are not listed in the manifest.
func (t *Template) collectUnlisted() {
	listed := make(map[string]bool, len(t.Files))
	for _, f := range t.Files {
		listed[f.Src] = true
	}
	_ = fs.WalkDir(t.root, "files", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		src := strings.TrimPrefix(p, "files/")
		if !listed[src] {
			t.Unlisted = append(t.Unlisted, src)
		}
		return nil
	})
	slices.Sort(t.Unlisted)
}

// Option looks up an option definition by key.
func (t *Template) Option(key string) *Option {
	for i := range t.Options {
		if t.Options[i].Key == key {
			return &t.Options[i]
		}
	}
	return nil
}

// exprEnv builds the compile environment for when expressions (option key → default value type).
func (t *Template) exprEnv() map[string]any {
	env := make(map[string]any, len(t.Options))
	for _, o := range t.Options {
		v, _ := t.coerceValue(o, o.Default)
		env[o.Key] = v
	}
	return env
}

// normalizedDefault returns an option's default value (the zero value of its type when undeclared).
func (t *Template) normalizedDefault(o Option) any {
	v, err := t.coerceValue(o, o.Default)
	if err != nil {
		v = nil
	}
	if v != nil {
		return v
	}
	switch o.Type {
	case OptionBoolean:
		return false
	case OptionMultiChoice:
		return []string{}
	default:
		return ""
	}
}

// coerceValue coerces external input (--set strings, yaml values from tests, wizard values) into the option's data type.
func (t *Template) coerceValue(o Option, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch o.Type {
	case OptionBoolean:
		switch b := v.(type) {
		case bool:
			return b, nil
		case string:
			switch strings.ToLower(b) {
			case "true", "yes", "1":
				return true, nil
			case "false", "no", "0":
				return false, nil
			}
		}
		return nil, fmt.Errorf("value %v is not a boolean", v)
	case OptionString:
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("value %v is not a string", v)
		}
		return s, nil
	case OptionChoice:
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("value %v is not a string", v)
		}
		if !slices.ContainsFunc(o.Choices, func(c Choice) bool { return c.Value == s }) {
			return nil, fmt.Errorf("value %q is not among the choices", s)
		}
		return s, nil
	case OptionMultiChoice:
		var items []string
		switch list := v.(type) {
		case string: // --set key={a,b} is already parsed as a list; a bare string is treated as a single value
			items = []string{list}
		case []string:
			items = list
		case []any: // template.yaml tests section
			for _, e := range list {
				s, ok := e.(string)
				if !ok {
					return nil, fmt.Errorf("list element %v is not a string", e)
				}
				items = append(items, s)
			}
		default:
			return nil, fmt.Errorf("value %v is not a list", v)
		}
		for _, s := range items {
			if !slices.ContainsFunc(o.Choices, func(c Choice) bool { return c.Value == s }) {
				return nil, fmt.Errorf("value %q is not among the choices", s)
			}
		}
		if items == nil {
			items = []string{}
		}
		return items, nil
	}
	return nil, fmt.Errorf("unknown option type %q", o.Type)
}

// CoerceOptions validates and merges externally set option values; unset options take their defaults.
// Keys in raw must be declared options, and values must match the option type.
func (t *Template) CoerceOptions(raw map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(t.Options))
	for _, o := range t.Options {
		out[o.Key] = t.normalizedDefault(o)
	}
	for k, v := range raw {
		o := t.Option(k)
		if o == nil {
			return nil, fmt.Errorf("template %s does not declare option %q", t.Name, k)
		}
		cv, err := t.coerceValue(*o, v)
		if err != nil {
			return nil, fmt.Errorf("option %s: %w", k, err)
		}
		out[k] = cv
	}
	return out, nil
}
