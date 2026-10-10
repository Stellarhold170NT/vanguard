package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanguard-lint/vanguard/internal/engine"
)

// TestInitWritesCommentedTemplate pins the §6.1 init contract: a .vanguard.yaml
// template with comments, written into the given directory.
func TestInitWritesCommentedTemplate(t *testing.T) {
	dir := t.TempDir()
	path, err := writeInitConfig(dir, "0.2.0-test")
	if err != nil {
		t.Fatalf("writeInitConfig: %v", err)
	}
	if path != filepath.Join(dir, ".vanguard.yaml") {
		t.Fatalf("path = %q, want the config inside dir", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := string(data)
	for _, want := range []string{"version: 1", "# ", "include:", "exclude:", "rules:", "suppressions:"} {
		if !strings.Contains(out, want) {
			t.Errorf("template missing %q:\n%s", want, out)
		}
	}
}

// TestInitTemplateIsAValidConfig closes the loop with the w2-04 parser:
// what init writes must load as a valid v0.1 config.
func TestInitTemplateIsAValidConfig(t *testing.T) {
	path, err := writeInitConfig(t.TempDir(), "0.2.0-test")
	if err != nil {
		t.Fatalf("writeInitConfig: %v", err)
	}
	cfg, err := engine.Load(path)
	if err != nil {
		t.Fatalf("engine.Load(init template): %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("cfg.Version = %d, want 1", cfg.Version)
	}
}

// TestInitRefusesOverwrite: an existing config is never silently replaced.
func TestInitRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if _, err := writeInitConfig(dir, "0.2.0-test"); err != nil {
		t.Fatalf("first writeInitConfig: %v", err)
	}
	_, err := writeInitConfig(dir, "0.2.0-test")
	if err == nil {
		t.Fatal("second writeInitConfig must refuse to overwrite")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want an already-exists message", err)
	}
}
