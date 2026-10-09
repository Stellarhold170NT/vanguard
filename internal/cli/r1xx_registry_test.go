package cli

import (
	"strings"
	"testing"
)

// w3-03 wires the R1xx family into the production registry: --list-rules
// and explain must see the five resource & naming rules with their full
// metadata (charter §3.1).
func TestDefaultRegistryIncludesR1xx(t *testing.T) {
	reg, err := registrySource()
	if err != nil {
		t.Fatalf("registrySource: %v", err)
	}
	want := map[string]string{
		"R1xx-01": "plural-collection",
		"R1xx-02": "no-verb-path",
		"R1xx-03": "resource-path-pattern",
		"R1xx-04": "path-casing",
		"R1xx-05": "id-field-naming",
	}
	for id, slug := range want {
		rule, ok := reg.ByID(id)
		if !ok {
			t.Fatalf("registry is missing %s — the R1xx family must register (w3-03)", id)
		}
		if rule.Slug != slug {
			t.Errorf("%s slug = %q, want %q", id, rule.Slug, slug)
		}
		if rule.Category != "resources" {
			t.Errorf("%s category = %q, want resources", id, rule.Category)
		}
		if rule.DocPath == "" || !strings.HasSuffix(rule.DocPath, ".md") {
			t.Errorf("%s docPath = %q, want a docs/rules page", id, rule.DocPath)
		}
	}
	if got := len(reg.All()); got < 10 {
		t.Errorf("registry holds %d rules, want >= 10 (5 demo + 5 R1xx)", got)
	}
}

// The --rules family selector keeps working for the new family: "R1xx"
// keeps exactly the five resource & naming rules.
func TestRuleSelectorKeepsR1xxFamily(t *testing.T) {
	base, err := registrySource()
	if err != nil {
		t.Fatalf("registrySource: %v", err)
	}
	keep, err := parseRuleSelector("R1xx", base)
	if err != nil {
		t.Fatalf("parseRuleSelector(R1xx): %v", err)
	}
	if len(keep) != 5 {
		t.Fatalf("R1xx selector kept %d rules, want 5", len(keep))
	}
	for id := range keep {
		if !strings.HasPrefix(id, "R1xx-") {
			t.Errorf("selector kept %s outside the family", id)
		}
	}
}
