package rules

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// The W2 demo rule set: stub-reachable rules registered in the production
// binary next to R6xx-99 so the golden fixture harness (w2-07) and every
// end-to-end test can drive real ERROR/WARN/INFO findings through the
// pipeline. Each rule is data-first — its identity lives in a YAML record
// under rules/data (charter §3.0) — and follows the R6xx-99 pattern
// exactly: embedded metadata → ParseMetadata → Compile.
//
// The checks read only the stub-expressible IR surface (method name,
// response type, payload, verb); the w3-01+ adapters make richer rules
// reachable, but these four stay valid forever as the engine's own
// fixtures.

//go:embed data/R6xx-91.yaml
var demoBadYAML []byte

//go:embed data/R6xx-92.yaml
var demoNoResponseYAML []byte

//go:embed data/R6xx-93.yaml
var demoGetWithBodyYAML []byte

//go:embed data/R6xx-94.yaml
var demoLegacyPathYAML []byte

// demoBadPrefix is the method-name marker R6xx-91 flags. Fixture method
// names start with it; production-styled names never do, so the rule
// cannot collide with the w2-06 test fixtures (which use the "bad"
// prefix) or with real adapter output.
const demoBadPrefix = "demoBad"

// methodSelector matches every Method node — the framework-level predicate
// "this rule talks about one API operation", spelled over IR concepts
// (charter §5.4).
type methodSelector struct{}

func (methodSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.Method)
	return ok
}

// demoBadCheck flags Methods whose operation name carries the demoBad
// marker. Pure: findings derive only from the node; the engine stamps
// RuleID and Severity (§5.4).
func demoBadCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || !strings.HasPrefix(m.OperationName, demoBadPrefix) {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("Method name %q carries the demoBad fixture marker — name the operation after what it does.", m.OperationName),
		Suggestion: renameWithoutMarker(m.OperationName),
		Location:   node.Loc(),
	}}
}

// renameWithoutMarker strips the demoBad prefix and lower-cases the first
// remaining rune, so the suggestion is replacement text (§3.0), not advice.
func renameWithoutMarker(name string) string {
	rest := strings.TrimPrefix(name, demoBadPrefix)
	if rest == "" {
		return name
	}
	r := []rune(rest)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// demoNoResponseCheck flags Methods without a response type (WARN demo).
func demoNoResponseCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Response.Type.Name != "" {
		return nil
	}
	return []engine.Finding{{
		Message:  fmt.Sprintf("Method %s declares no response type — clients cannot deserialize an undeclared result.", m.OperationName),
		Location: node.Loc(),
	}}
}

// demoGetWithBodyCheck flags GET Methods carrying a request body (INFO demo).
func demoGetWithBodyCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Verb != ir.VerbGet || m.Payload == nil {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("GET endpoint %s declares a request body — many HTTP clients silently drop GET bodies.", m.OperationName),
		Suggestion: "move the payload to query parameters",
		Location:   node.Loc(),
	}}
}

// demoLegacyPathCheck flags Methods served from a /legacy/ path (INFO demo
// and the w2-07 data-driven acceptance artifact: this rule shipped with one
// YAML record and one table row — no other file changed).
func demoLegacyPathCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || !strings.Contains(m.Path, "/legacy/") {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("Method %s is served from a legacy path (%s) — migrate the route to its versioned form.", m.OperationName, m.Path),
		Suggestion: "move the route under its versioned prefix",
		Location:   node.Loc(),
	}}
}

// demoRulesYAML pairs every demo rule's embedded record with its check so
// the set stays one data table: adding a demo rule for a new fixture need
// means one YAML file and one row here — never engine or harness work
// (the w2-07 data-driven acceptance).
var demoRulesYAML = []struct {
	yaml  []byte
	sel   engine.Selector
	check engine.CheckFunc
}{
	{demoRuleYAML, demoSelector{}, demoCheck},
	{demoBadYAML, methodSelector{}, demoBadCheck},
	{demoNoResponseYAML, methodSelector{}, demoNoResponseCheck},
	{demoGetWithBodyYAML, methodSelector{}, demoGetWithBodyCheck},
	{demoLegacyPathYAML, methodSelector{}, demoLegacyPathCheck},
}

// DemoRules compiles the whole demo set. Registration errors (bad
// metadata) surface here as errors — fail tests, never runtime
// (charter §5.4). R6xx-99's builder stays in demo.go (DemoRule, the
// w2-04 acceptance fixture); this table is the full set the CLI registers.
func DemoRules() ([]engine.Rule, error) {
	out := make([]engine.Rule, 0, len(demoRulesYAML))
	for _, d := range demoRulesYAML {
		m, err := ParseMetadata(d.yaml)
		if err != nil {
			return nil, fmt.Errorf("demo rule metadata: %w", err)
		}
		rule, err := Compile(m, d.sel, d.check)
		if err != nil {
			return nil, fmt.Errorf("compile demo rule: %w", err)
		}
		out = append(out, rule)
	}
	return out, nil
}
