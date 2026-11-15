package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewYesValidatesStringPattern(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	tmplDir := filepath.Join(home, ".config", "go-create", "templates", "pattern-test")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "name: pattern-test\noptions:\n  - {key: ident, type: string, prompt: Identifier, pattern: '^[a-z]+$'}\n"
	if err := os.WriteFile(filepath.Join(tmplDir, "template.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "project")
	cmd := newCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"pattern-test", "demo", "--module", "github.com/user/demo", "--dir", dir, "--yes", "--no-git", "--no-tidy", "--set", "ident=123-invalid"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "ident") || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("pattern error = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("invalid options created output: %v", err)
	}
}
