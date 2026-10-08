package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// boolPtr is a config-override helper (RuleOverride.Disabled is *bool: nil =
// not specified, false = explicitly enable a default-off rule).
func boolPtr(b bool) *bool { return &b }

const fullConfigYAML = `version: 1

frameworks:
  java: [spring-boot-mvc]

include: ["src/main/**"]
exclude:
  - "**/generated/**"
  - "vendor/**"

rules:
  R1xx:
    severity: WARN
  R1xx-02:
    severity: ERROR
    disabled: false
  R6xx-01:
    disabled: false
    options:
      versionPattern: "/v[0-9]+"

suppressions:
  - rule: R2xx
    paths: ["src/legacy/**"]
    reason: "legacy module, refactor W2"
`

// TestLoadFullConfig parses the complete §6.4.1 schema into Config.
func TestLoadFullConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vanguard.yaml")
	if err := os.WriteFile(path, []byte(fullConfigYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if got := cfg.Frameworks["java"]; len(got) != 1 || got[0] != "spring-boot-mvc" {
		t.Errorf("Frameworks[java] = %v, want [spring-boot-mvc]", got)
	}
	if len(cfg.Include) != 1 || cfg.Include[0] != "src/main/**" {
		t.Errorf("Include = %v, want [src/main/**]", cfg.Include)
	}
	if len(cfg.Exclude) != 2 || cfg.Exclude[0] != "**/generated/**" {
		t.Errorf("Exclude = %v, want [**/generated/** vendor/**]", cfg.Exclude)
	}
	if len(cfg.Rules) != 3 {
		t.Fatalf("Rules has %d entries, want 3", len(cfg.Rules))
	}
	ov := cfg.Rules["R1xx-02"]
	if ov.Disabled == nil || *ov.Disabled != false {
		t.Errorf("R1xx-02.Disabled = %v, want false", ov.Disabled)
	}
	if ov.Severity != SeverityError {
		t.Errorf("R1xx-02.Severity = %q, want ERROR", ov.Severity)
	}
	if ov := cfg.Rules["R6xx-01"]; ov.Options["versionPattern"] != "/v[0-9]+" {
		t.Errorf("R6xx-01.Options = %v, want versionPattern /v[0-9]+", ov.Options)
	}
	if len(cfg.Suppressions) != 1 {
		t.Fatalf("Suppressions has %d entries, want 1", len(cfg.Suppressions))
	}
	sup := cfg.Suppressions[0]
	if sup.Rule != "R2xx" || len(sup.Paths) != 1 || sup.Paths[0] != "src/legacy/**" || sup.Reason != "legacy module, refactor W2" {
		t.Errorf("suppression = %+v, want R2xx src/legacy/** with reason", sup)
	}
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
	if cfg.Dir != filepath.Dir(path) {
		t.Errorf("Dir = %q, want %q", cfg.Dir, filepath.Dir(path))
	}
}

// TestLoadDefaults checks the schema defaults for a minimal config: include
// defaults to ["**"] (charter §6.4.1) and every other collection stays empty.
func TestLoadDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vanguard.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Include) != 1 || cfg.Include[0] != "**" {
		t.Errorf("Include = %v, want default [**]", cfg.Include)
	}
	if len(cfg.Exclude) != 0 {
		t.Errorf("Exclude = %v, want empty", cfg.Exclude)
	}
	if len(cfg.Rules) != 0 || len(cfg.Frameworks) != 0 || len(cfg.Suppressions) != 0 {
		t.Errorf("collections not empty: %+v", cfg)
	}
}

// TestLoadCaseInsensitiveSeverity accepts severity spellings in any case and
// normalizes to the canonical constants.
func TestLoadCaseInsensitiveSeverity(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vanguard.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nrules:\n  R1xx-02:\n    severity: warn\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Rules["R1xx-02"].Severity; got != SeverityWarn {
		t.Errorf("severity = %q, want WARN", got)
	}
}

// TestLoadSchemaErrors is the exit-2 catalog (charter §6.3): every schema
// violation fails with a message that cites the config file and the line of
// the offending key.
func TestLoadSchemaErrors(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		line     int
		contains string
	}{
		{"missing-version", "frameworks: {}\n", 1, `missing required key "version"`},
		{"empty-file", "", 1, `missing required key "version"`},
		{"wrong-version", "version: 2\n", 1, "unsupported schema version"},
		{"version-not-int", "version: one\n", 1, `"version"`},
		{"unknown-top-key", "version: 1\nrulez: {}\n", 2, `unknown key "rulez"`},
		{"unknown-rule-key", "version: 1\nrules:\n  R9xx-01: {}\n", 3, "invalid rule key"},
		{"unknown-rule-subkey", "version: 1\nrules:\n  R1xx-02:\n    severi: WARN\n", 4, `unknown key "severi"`},
		{"bad-severity", "version: 1\nrules:\n  R1xx-02:\n    severity: FATAL\n", 4, "invalid severity"},
		{"bad-disabled-type", "version: 1\nrules:\n  R1xx-02:\n    disabled: yes-please\n", 4, "disabled"},
		{"bad-options-type", "version: 1\nrules:\n  R1xx-02:\n    options: 5\n", 4, "options"},
		{"suppression-missing-rule", "version: 1\nsuppressions:\n  - paths: [a]\n", 3, "rule"},
		{"suppression-missing-paths", "version: 1\nsuppressions:\n  - rule: R1xx\n", 3, "paths"},
		{"suppression-bad-glob", "version: 1\nsuppressions:\n  - rule: R1xx\n    paths: [\"src/[ab\"]\n", 4, "paths"},
		{"rules-not-map", "version: 1\nrules: 5\n", 2, "rules"},
		{"duplicate-top-key", "version: 1\nversion: 1\n", 2, "duplicate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".vanguard.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load succeeded, want error containing %q at line %d", tc.contains, tc.line)
			}
			wantPrefix := filepath.Base(path) + ":" + strconv.Itoa(tc.line) + ":"
			if !strings.Contains(err.Error(), wantPrefix) {
				t.Fatalf("error %q does not cite %s", err.Error(), wantPrefix)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("error %q does not mention %q", err.Error(), tc.contains)
			}
		})
	}
}

// TestLoadMissingFile reports an unreadable config as a tool error (exit 2
// class) naming the file.
func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nowhere.yaml")
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "nowhere.yaml") {
		t.Fatalf("Load missing file: err = %v, want error naming the file", err)
	}
}

// TestDiscoverWalksUp pins the golangci-style discovery (charter §6.4.2):
// the nearest .vanguard.yaml at or above the start directory wins; nothing
// found anywhere up the tree is reported as ok=false, not as an error.
func TestDiscoverWalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if path, ok, err := Discover(nested); ok || err != nil || path != "" {
		t.Fatalf("Discover without config: path=%q ok=%v err=%v, want \"\", false, nil", path, ok, err)
	}

	rootCfg := filepath.Join(root, ".vanguard.yaml")
	if err := os.WriteFile(rootCfg, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, ok, err := Discover(nested)
	if err != nil || !ok || path != rootCfg {
		t.Fatalf("Discover nested: path=%q ok=%v err=%v, want %q", path, ok, err, rootCfg)
	}

	midCfg := filepath.Join(root, "a", ".vanguard.yaml")
	if err := os.WriteFile(midCfg, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, ok, err = Discover(nested)
	if err != nil || !ok || path != midCfg {
		t.Fatalf("Discover nearest wins: path=%q ok=%v err=%v, want %q", path, ok, err, midCfg)
	}

	path, ok, err = Discover(root)
	if err != nil || !ok || path != rootCfg {
		t.Fatalf("Discover at config dir: path=%q ok=%v err=%v, want %q", path, ok, err, rootCfg)
	}
}

// TestResolveOverrides covers the override merge (charter §6.4.2 #4): the
// family prefix applies first, the exact id second — so an exact override
// wins in both the disable and the enable direction, and severity/options
// follow the same precedence. Metadata defaults fill what config leaves.
func TestResolveOverrides(t *testing.T) {
	base := Rule{
		ID:       "R1xx-02",
		Slug:     "no-verb-path",
		Category: "resources",
		Severity: SeverityError,
		Selector: stubSelector{match: true},
		Check:    func(*LintContext, ir.Node) []Finding { return nil },
	}
	familySeverity := func() *Config {
		return &Config{Rules: map[string]RuleOverride{"R1xx": {Severity: SeverityWarn}}}
	}

	t.Run("metadata default without config", func(t *testing.T) {
		got := resolveRule(nil, base)
		if got.Disabled || got.Severity != SeverityError || got.Options != nil {
			t.Fatalf("resolveRule(nil) = %+v", got)
		}
	})
	t.Run("family severity override", func(t *testing.T) {
		got := resolveRule(familySeverity(), base)
		if got.Severity != SeverityWarn {
			t.Fatalf("severity = %q, want WARN", got.Severity)
		}
	})
	t.Run("exact severity beats family", func(t *testing.T) {
		cfg := familySeverity()
		cfg.Rules["R1xx-02"] = RuleOverride{Severity: SeverityInfo}
		got := resolveRule(cfg, base)
		if got.Severity != SeverityInfo {
			t.Fatalf("severity = %q, want INFO", got.Severity)
		}
	})
	t.Run("family disable then exact enable", func(t *testing.T) {
		cfg := &Config{Rules: map[string]RuleOverride{
			"R1xx":    {Disabled: boolPtr(true)},
			"R1xx-02": {Disabled: boolPtr(false)},
		}}
		if got := resolveRule(cfg, base); got.Disabled {
			t.Fatal("rule disabled, want enabled (exact enable overrides family disable)")
		}
	})
	t.Run("exact disable beats family enable", func(t *testing.T) {
		cfg := &Config{Rules: map[string]RuleOverride{
			"R1xx":    {Disabled: boolPtr(false)},
			"R1xx-02": {Disabled: boolPtr(true)},
		}}
		if got := resolveRule(cfg, base); !got.Disabled {
			t.Fatal("rule enabled, want disabled (exact disable overrides family enable)")
		}
	})
	t.Run("default disabled rule stays off without config", func(t *testing.T) {
		off := base
		off.DefaultDisabled = true
		if got := resolveRule(nil, off); !got.Disabled {
			t.Fatal("default-disabled rule enabled with nil config")
		}
	})
	t.Run("explicit false enables default disabled rule", func(t *testing.T) {
		off := base
		off.DefaultDisabled = true
		cfg := &Config{Rules: map[string]RuleOverride{"R1xx-02": {Disabled: boolPtr(false)}}}
		if got := resolveRule(cfg, off); got.Disabled {
			t.Fatal("disabled: false in config did not enable the rule")
		}
	})
	t.Run("options merge family then exact", func(t *testing.T) {
		cfg := &Config{Rules: map[string]RuleOverride{
			"R1xx":    {Options: map[string]any{"a": 1, "b": 2}},
			"R1xx-02": {Options: map[string]any{"b": 3}},
		}}
		got := resolveRule(cfg, base)
		if got.Options["a"] != 1 || got.Options["b"] != 3 {
			t.Fatalf("options = %v, want a=1 b=3", got.Options)
		}
	})
	t.Run("unspecified disabled does not enable default disabled rule", func(t *testing.T) {
		off := base
		off.DefaultDisabled = true
		cfg := familySeverity()
		if got := resolveRule(cfg, off); !got.Disabled {
			t.Fatal("override without disabled key enabled a default-disabled rule")
		}
	})
}
