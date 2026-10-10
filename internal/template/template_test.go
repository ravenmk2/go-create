package template

import (
	"strings"
	"testing"
	"testing/fstest"
)

func newTestFS(yaml string, files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte(yaml)},
	}
	for name, content := range files {
		fsys["files/"+name] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

const baseYAML = `
name: test
description: test template
options:
  - {key: feat, type: boolean, prompt: Feature, default: false}
  - {key: fw, type: choice, prompt: Framework, choices: [{value: a, label: A}, {value: b, label: B}], default: a}
  - {key: platforms, type: multichoice, prompt: Platforms, choices: [{value: linux, label: Linux}, {value: windows, label: Windows}], default: [linux]}
files:
  - src: main.go.tpl
    dest: cmd/{{ .Name }}/main.go
    render: true
  - src: feat.go.tpl
    render: true
    when: feat && fw == "a"
  - src: static.txt
`

func baseFS() fstest.MapFS {
	return newTestFS(baseYAML, map[string]string{
		"main.go.tpl": "package main // <% .Module %> <% lower .Name %>",
		"feat.go.tpl": "package feat\n// <%if has .Options.platforms \"linux\"%>linux<%end%>",
		"static.txt":  "plain\n",
		"orphan.txt":  "orphan\n",
	})
}

func testContext() *Context {
	return &Context{
		Name:    "my-app",
		Module:  "github.com/user/my-app",
		Repo:    "user/my-app",
		PkgName: "myapp",
		Options: map[string]any{
			"feat":      true,
			"fw":        "a",
			"platforms": []string{"linux", "windows"},
		},
	}
}

func TestParseOK(t *testing.T) {
	tmpl, err := parse("test", baseFS(), "builtin")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Left != "<%" || tmpl.Right != "%>" {
		t.Errorf("default delimiters should be ecb, got %q %q", tmpl.Left, tmpl.Right)
	}
	// dest defaults to src without the .tpl suffix
	if tmpl.Files[1].Dest != "feat.go" {
		t.Errorf("wrong default dest derivation: %q", tmpl.Files[1].Dest)
	}
	// orphan.txt is not listed
	if len(tmpl.Unlisted) != 1 || tmpl.Unlisted[0] != "orphan.txt" {
		t.Errorf("wrong Unlisted: %v", tmpl.Unlisted)
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"unknown field":                     "name: x\nunknown: 1\n",
		"bad key":                           "name: x\noptions: [{key: 'Bad', type: boolean, prompt: p}]\nfiles: []\n",
		"unknown type":                      "name: x\noptions: [{key: a, type: magic, prompt: p}]\nfiles: []\n",
		"select without choices":            "name: x\noptions: [{key: a, type: choice, prompt: p}]\nfiles: []\n",
		"src does not exist":                "name: x\nfiles: [{src: nope.go}]\n",
		"when references undeclared option": "name: x\nfiles: [{src: static.txt, when: nope}]\n",
		"when is not boolean":               "name: x\noptions: [{key: a, type: string, prompt: p}]\nfiles: [{src: static.txt, when: a}]\n",
		"unknown delims preset":             "name: x\ndelims: nope\nfiles: []\n",
		"unsupported schema version":        "name: x\nschema: 99\nfiles: []\n",
	}
	for name, y := range cases {
		fsys := newTestFS(y, map[string]string{"static.txt": "x"})
		if _, err := parse("test", fsys, "builtin"); err == nil {
			t.Errorf("%s: expected an error, but it passed", name)
		}
	}
}

func TestSchemaDefaultsToCurrentVersion(t *testing.T) {
	tmpl, err := parse("x", newTestFS("name: x\nfiles: []\n", nil), "builtin")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Schema != SchemaVersion {
		t.Errorf("absent schema should default to %d, got %d", SchemaVersion, tmpl.Schema)
	}
}

func TestDelimsCustom(t *testing.T) {
	fsys := newTestFS("name: x\ndelims: [\"<<\", \">>\"]\nfiles: []\n", nil)
	tmpl, err := parse("x", fsys, "builtin")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Left != "<<" || tmpl.Right != ">>" {
		t.Errorf("wrong custom delimiters: %q %q", tmpl.Left, tmpl.Right)
	}
}

func TestRender(t *testing.T) {
	tmpl, err := parse("test", baseFS(), "builtin")
	if err != nil {
		t.Fatal(err)
	}
	files, warnings, err := tmpl.Render(testContext())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 { // main.go + feat.go + static.txt (orphan is not in the manifest)
		t.Fatalf("wrong file count: %d", len(files))
	}
	if files[0].Path != "cmd/my-app/main.go" {
		t.Errorf("dest variable not substituted: %q", files[0].Path)
	}
	if !strings.Contains(string(files[0].Data), "github.com/user/my-app") ||
		!strings.Contains(string(files[0].Data), "my-app") {
		t.Errorf("wrong content rendering: %s", files[0].Data)
	}
	if !strings.Contains(string(files[1].Data), "linux") {
		t.Errorf("has function did not take effect: %s", files[1].Data)
	}
	// orphan.txt should produce an unlisted warning
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "orphan.txt") {
			found = true
		}
	}
	if !found {
		t.Errorf("missing unlisted-file warning: %v", warnings)
	}
}

func TestRenderWhenFilters(t *testing.T) {
	tmpl, err := parse("test", baseFS(), "builtin")
	if err != nil {
		t.Fatal(err)
	}
	ctx := testContext()
	ctx.Options["feat"] = false
	files, _, err := tmpl.Render(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.Contains(f.Path, "feat.go") {
			t.Errorf("feat.go should not be generated when feat is false")
		}
	}
}

func TestRenderStaticWithDelimsWarns(t *testing.T) {
	fsys := newTestFS("name: x\nfiles:\n  - src: static.txt\n", map[string]string{
		"static.txt": "contains variable <% .Name %> but is not marked render",
	})
	tmpl, err := parse("x", fsys, "builtin")
	if err != nil {
		t.Fatal(err)
	}
	_, warnings, err := tmpl.Render(testContext())
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "render: true") {
		t.Errorf("should warn about static file containing delimiters: %v", warnings)
	}
}

func TestRenderDestUnsafe(t *testing.T) {
	fsys := newTestFS("name: x\nfiles:\n  - src: evil.txt\n    dest: ../evil.txt\n", map[string]string{
		"evil.txt": "x",
	})
	tmpl, err := parse("x", fsys, "builtin")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := tmpl.Render(testContext()); err == nil {
		t.Error("dest containing .. should error")
	}
}

func TestCoerceOptions(t *testing.T) {
	tmpl, err := parse("test", baseFS(), "builtin")
	if err != nil {
		t.Fatal(err)
	}

	// happy path: boolean string, choice, multichoice yaml list
	out, err := tmpl.CoerceOptions(map[string]any{
		"feat":      "true",
		"fw":        "b",
		"platforms": []any{"linux", "windows"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["feat"] != true || out["fw"] != "b" {
		t.Errorf("type coercion error: %v", out)
	}

	// undeclared key
	if _, err := tmpl.CoerceOptions(map[string]any{"nope": "x"}); err == nil {
		t.Error("undeclared key should error")
	}
	// select value out of range
	if _, err := tmpl.CoerceOptions(map[string]any{"fw": "c"}); err == nil {
		t.Error("invalid select value should error")
	}
	// multichoice value out of range
	if _, err := tmpl.CoerceOptions(map[string]any{"platforms": []string{"plan9"}}); err == nil {
		t.Error("invalid multichoice value should error")
	}
	// invalid boolean value
	if _, err := tmpl.CoerceOptions(map[string]any{"feat": "maybe"}); err == nil {
		t.Error("invalid boolean value should error")
	}
}
