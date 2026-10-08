package rules

import (
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// anySelector matches everything — selector behavior is engine territory;
// these tests only pin metadata → engine.Rule binding and validation.
type anySelector struct{}

func (anySelector) Matches(ir.Node) bool { return true }

func nopCheck(*engine.LintContext, ir.Node) []engine.Finding { return nil }

func validMetadata() Metadata {
	return Metadata{
		ID:          "R1xx-02",
		Slug:        "no-verb-path",
		Category:    "resources",
		Severity:    "ERROR",
		Summary:     "Path must not contain a CRUD verb.",
		DocPath:     "docs/rules/R1xx-02-no-verb-path.md",
		ExampleGood: `@DeleteMapping("/drafts/{id}")`,
		ExampleBad:  `@DeleteMapping("/delete")`,
	}
}

// TestCompileFromMetadata pins the data→engine.Rule join: metadata fields
// map 1:1 onto the engine rule struct, severity converts to the canonical
// constant, and selector/check attach unchanged.
func TestCompileFromMetadata(t *testing.T) {
	m := validMetadata()
	m.DefaultDisabled = true
	m.Options = []string{"verbWhitelist"}
	rule, err := Compile(m, anySelector{}, nopCheck)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if rule.ID != "R1xx-02" || rule.Slug != "no-verb-path" || rule.Category != "resources" {
		t.Fatalf("identity = %+v", rule)
	}
	if rule.Severity != engine.SeverityError {
		t.Fatalf("Severity = %q, want ERROR", rule.Severity)
	}
	if !rule.DefaultDisabled || len(rule.Options) != 1 || rule.Options[0] != "verbWhitelist" {
		t.Fatalf("flags = %+v options = %v", rule, rule.Options)
	}
	if rule.Selector == nil || rule.Check == nil {
		t.Fatal("selector/check not attached")
	}
	if !rule.Selector.Matches(ir.Method{}) {
		t.Fatal("selector lost its behavior")
	}
}

// TestCompileValidationErrors pins the registry validation catalog
// (charter §5.4): id format + family band, kebab slug, severity, summary,
// doc path, examples — a rule that fails any of these never registers.
func TestCompileValidationErrors(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*Metadata)
		contains string
	}{
		{"bad-id-format", func(m *Metadata) { m.ID = "R9xx-01" }, "id"},
		{"bad-id-short", func(m *Metadata) { m.ID = "R1xx-1" }, "id"},
		{"lowercase-id", func(m *Metadata) { m.ID = "r1xx-01" }, "id"},
		{"empty-slug", func(m *Metadata) { m.Slug = "" }, "slug"},
		{"non-kebab-slug", func(m *Metadata) { m.Slug = "NoVerbPath" }, "slug"},
		{"slug-underscore", func(m *Metadata) { m.Slug = "no_verb" }, "slug"},
		{"empty-category", func(m *Metadata) { m.Category = "" }, "category"},
		{"bad-severity", func(m *Metadata) { m.Severity = "FATAL" }, "severity"},
		{"empty-severity", func(m *Metadata) { m.Severity = "" }, "severity"},
		{"empty-summary", func(m *Metadata) { m.Summary = "" }, "summary"},
		{"empty-doc-path", func(m *Metadata) { m.DocPath = "" }, "doc"},
		{"missing-good-example", func(m *Metadata) { m.ExampleGood = "" }, "example"},
		{"missing-bad-example", func(m *Metadata) { m.ExampleBad = "" }, "example"},
		{"duplicate-option", func(m *Metadata) { m.Options = []string{"a", "a"} }, "option"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMetadata()
			tc.m(&m)
			_, err := Compile(m, anySelector{}, nopCheck)
			if err == nil {
				t.Fatalf("Compile accepted invalid metadata, want error about %q", tc.contains)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("error %q does not mention %q", err.Error(), tc.contains)
			}
		})
	}
	// Nil selector/check are engine.Rule-level validation too.
	m := validMetadata()
	if _, err := Compile(m, nil, nopCheck); err == nil {
		t.Fatal("Compile accepted nil selector")
	}
	if _, err := Compile(m, anySelector{}, nil); err == nil {
		t.Fatal("Compile accepted nil check")
	}
}

// TestRegistryRegisterAndLookup: registration validates, lookup resolves by
// id, All() iterates sorted by id (deterministic, B-6).
func TestRegistryRegisterAndLookup(t *testing.T) {
	reg := NewRegistry()
	r1, err := Compile(validMetadata(), anySelector{}, nopCheck)
	if err != nil {
		t.Fatal(err)
	}
	m2 := validMetadata()
	m2.ID = "R1xx-01"
	m2.Slug = "plural-collection"
	r2, err := Compile(m2, anySelector{}, nopCheck)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(r1); err != nil {
		t.Fatalf("Register r1: %v", err)
	}
	if err := reg.Register(r2); err != nil {
		t.Fatalf("Register r2: %v", err)
	}
	if err := reg.Register(r1); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate id: err = %v, want duplicate error", err)
	}
	m3 := validMetadata()
	m3.ID = "R1xx-03"
	m3.Slug = "no-verb-path" // slug already taken by r1
	r3, _ := Compile(m3, anySelector{}, nopCheck)
	if err := reg.Register(r3); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate slug: err = %v, want duplicate error", err)
	}

	got, ok := reg.ByID("R1xx-01")
	if !ok || got.Slug != "plural-collection" {
		t.Fatalf("ByID(R1xx-01) = %+v ok=%v", got, ok)
	}
	if _, ok := reg.ByID("R2xx-01"); ok {
		t.Fatal("ByID found an unregistered rule")
	}
	all := reg.All()
	if len(all) != 2 || all[0].ID != "R1xx-01" || all[1].ID != "R1xx-02" {
		t.Fatalf("All() = [%s, %s], want sorted by id", all[0].ID, all[1].ID)
	}
	// Compile-level check: the registry refuses rules that never validated.
	if err := reg.Register(engine.Rule{ID: "garbage"}); err == nil {
		t.Fatal("registry accepted a rule with a garbage id")
	}
}
