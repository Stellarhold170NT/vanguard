package rules

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// demoRuleYAML is the demo rule's metadata record — the acceptance fixture
// of w2-04 proving the data path end to end: the rule's identity lives in
// YAML, not Go. The w3-03+ families follow exactly this shape.
//
//go:embed data/R6xx-99.yaml
var demoRuleYAML []byte

// demoSelector matches only Method nodes: the framework-level predicate
// "this rule talks about one API operation", spelled over IR concepts.
type demoSelector struct{}

func (demoSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.Method)
	return ok
}

// demoCheck flags a Method whose full path ends with a trailing slash.
// Pure: findings derive only from the node; the engine stamps RuleID and
// Severity (§5.4).
func demoCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok {
		return nil
	}
	if !strings.HasSuffix(m.Path, "/") {
		return nil
	}
	return []engine.Finding{{
		Message:    "Path ends with a trailing slash — normalize to one canonical form.",
		Suggestion: strings.TrimSuffix(m.Path, "/"),
		Location:   node.Loc(),
	}}
}

// DemoRule builds the demo rule through the production path: embedded
// metadata → ParseMetadata → Compile (full §5.4 validation). It is
// registered like any w3-03+ rule; nothing about it is special-cased in
// the engine.
func DemoRule() (engine.Rule, error) {
	m, err := ParseMetadata(demoRuleYAML)
	if err != nil {
		return engine.Rule{}, fmt.Errorf("demo rule metadata: %w", err)
	}
	return Compile(m, demoSelector{}, demoCheck)
}
