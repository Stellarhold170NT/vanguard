package engine

import (
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// stubNode is the smallest possible IR node for engine-side tests; w2-02
// replaces it with the real inventory nodes.
type stubNode struct{}

func (stubNode) Loc() ir.Location { return ir.Location{File: "X.java", Line: 1, Column: 1} }

// stubSelector proves the Selector contract is implementable as declared.
type stubSelector struct{ match bool }

func (s stubSelector) Matches(ir.Node) bool { return s.match }

// stubRegistry proves the Registry contract is implementable as declared;
// w3-03 provides the production registry backed by rule metadata.
type stubRegistry struct{ rules []Rule }

func (r *stubRegistry) Register(rule Rule) error {
	r.rules = append(r.rules, rule)
	return nil
}

func (r *stubRegistry) ByID(id string) (Rule, bool) {
	for _, rule := range r.rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return Rule{}, false
}

func (r *stubRegistry) All() []Rule { return r.rules }

// TestEngineContractsCompile pins the engine surface that w2-04 implements
// and the rules package (w3-03+) codes against: selector → check → registry.
func TestEngineContractsCompile(t *testing.T) {
	var selector Selector = stubSelector{match: true}
	var check CheckFunc = func(ctx *LintContext, node ir.Node) []Finding {
		return []Finding{{
			Message:  "stub finding",
			Location: node.Loc(),
		}}
	}
	if !selector.Matches(stubNode{}) {
		t.Fatal("stub selector should match")
	}
	findings := check(&LintContext{}, stubNode{})
	if len(findings) != 1 || findings[0].Message != "stub finding" {
		t.Fatalf("unexpected findings: %+v", findings)
	}

	var registry Registry = &stubRegistry{}
	rule := Rule{
		ID:       "R2xx-01",
		Slug:     "get-no-body",
		Category: "methods",
		Severity: SeverityError,
		Selector: selector,
		Check:    check,
	}
	if err := registry.Register(rule); err != nil {
		t.Fatalf("register rule: %v", err)
	}
	got, ok := registry.ByID("R2xx-01")
	if !ok || got.Slug != "get-no-body" {
		t.Fatalf("ByID lookup failed: got %+v, ok %v", got, ok)
	}
	if all := registry.All(); len(all) != 1 {
		t.Fatalf("All() returned %d rules, want 1", len(all))
	}
}

// TestSeverityConstantsAreDistinct guards the §3.0 taxonomy values.
func TestSeverityConstantsAreDistinct(t *testing.T) {
	severities := []Severity{SeverityError, SeverityWarn, SeverityInfo}
	seen := make(map[Severity]bool, len(severities))
	for _, s := range severities {
		if s == "" {
			t.Fatal("severity constant must not be empty")
		}
		if seen[s] {
			t.Fatalf("duplicate severity %q", s)
		}
		seen[s] = true
	}
}
