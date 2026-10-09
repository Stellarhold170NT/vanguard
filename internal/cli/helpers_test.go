package cli

import (
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
	"github.com/Stellarhold170NT/vanguard/rules"
)

// simulateTTY overrides the process TTY probe for the duration of a test so
// both §6.1 check branches (suppressed vs rendered pretty) stay hermetic:
// go test stdout is a file, so production detection would always be false.
func simulateTTY(on bool) func() {
	prev := ttyDetector
	ttyDetector = func() bool { return on }
	return func() { ttyDetector = prev }
}

// withTestRegistry swaps in the builtin registry plus the fixture rule
// R6xx-42 (testRule) for the duration of one test. The builtin set has no
// stub-reachable violation — R6xx-99 flags raw trailing slashes, which the
// stub path merge cleans per the IR contract (discovery/stub.go
// mergeStubPath) — so the §6.3 exit-1 plumbing is pinned through this
// registered fixture rule; the w3-01+ adapters make R6xx-99 reachable for
// real end-to-end.
func withTestRegistry(t *testing.T) {
	t.Helper()
	prev := registrySource
	reg, err := defaultRegistry()
	if err != nil {
		t.Fatalf("defaultRegistry: %v", err)
	}
	rule, err := testRule()
	if err != nil {
		t.Fatalf("testRule: %v", err)
	}
	if err := reg.Register(rule); err != nil {
		t.Fatalf("register %s: %v", rule.ID, err)
	}
	registrySource = func() (*rules.Registry, error) { return reg, nil }
	t.Cleanup(func() { registrySource = prev })
}

// fixtureSelector matches every Method node.
type fixtureSelector struct{}

func (fixtureSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.Method)
	return ok
}

// testRule is the fixture rule: ERROR on every Method whose operation name
// starts with "bad" (the violating fixtures name their method badGetThing,
// the clean ones never do). R6xx-42 keeps the id pattern (family 6) while
// staying distinct from the real R6xx-99.
func testRule() (engine.Rule, error) {
	m := rules.Metadata{
		ID:          "R6xx-42",
		Slug:        "demo-bad-method",
		Category:    "demo",
		Severity:    "ERROR",
		Summary:     "Fixture rule: stub methods whose name starts with 'bad' are flagged.",
		DocPath:     "docs/rules/R6xx-42-demo-bad-method.md",
		ExampleGood: "GET /api/v1/youth",
		ExampleBad:  "GET /api/v1/youth/",
	}
	return rules.Compile(m, fixtureSelector{}, func(_ *engine.LintContext, node ir.Node) []engine.Finding {
		mth, ok := node.(ir.Method)
		if !ok || !strings.HasPrefix(mth.OperationName, "bad") {
			return nil
		}
		return []engine.Finding{{
			Message:    "Fixture: method name starts with 'bad'.",
			Suggestion: "rename the method",
			Location:   node.Loc(),
		}}
	})
}
