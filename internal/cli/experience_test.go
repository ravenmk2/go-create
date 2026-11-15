package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDryRunDoesNotWriteOrRunPostActions(t *testing.T) {
	for _, existing := range []bool{false, true} {
		dir := filepath.Join(t.TempDir(), "project")
		if existing {
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("sentinel"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		var out bytes.Buffer
		cmd := newCmd()
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{"lib", "demo", "--module", "github.com/user/demo", "--dir", dir, "--yes", "--dry-run"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Dry run complete") || strings.Contains(out.String(), "Project created") || strings.Contains(out.String(), "git init") {
			t.Fatal(out.String())
		}
		if existing {
			data, err := os.ReadFile(filepath.Join(dir, "README.md"))
			if err != nil || string(data) != "sentinel" || !strings.Contains(out.String(), "overwrite README.md") {
				t.Fatalf("unexpected preview: %s %v", out.String(), err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("dry run created files: %v %v", entries, err)
			}
		} else if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("dry run created directory: %v", err)
		}
	}
}

func TestTemplateDiscovery(t *testing.T) {
	var out bytes.Buffer
	list := listCmd()
	list.SetOut(&out)
	if err := list.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "--set") || !strings.Contains(out.String(), "show <template>") {
		t.Fatal(out.String())
	}
	out.Reset()
	show := showCmd()
	show.SetOut(&out)
	show.SetArgs([]string{"web"})
	if err := show.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"http_framework", "nethttp", "chi", "gin", "echo", "Default:", "Example: --set", "ci_platforms"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show missing %q", want)
		}
	}
}

func TestActivityStopsBeforeFinalOutput(t *testing.T) {
	var out bytes.Buffer
	a := &generationActivity{out: &out, enabled: true}
	for i := 0; i < 3; i++ {
		a.start()
		a.stop()
	}
	a.stop()
	if !strings.Contains(out.String(), "Working") || !strings.HasSuffix(out.String(), "\r\x1b[2K") {
		t.Fatal("activity must render and clear its line")
	}
	out.Reset()
	a.enabled = false
	a.start()
	a.stop()
	if out.Len() != 0 {
		t.Fatal("plain output must not animate")
	}
}
