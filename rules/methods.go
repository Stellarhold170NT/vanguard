package rules

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// The R2xx family's YAML records (charter §3.0: metadata is data, embedded
// at build time — the binary never loads rule files at runtime).
var (
	//go:embed data/R2xx-01.yaml
	getNoBodyYAML []byte

	//go:embed data/R2xx-02.yaml
	postCreates201YAML []byte

	//go:embed data/R2xx-03.yaml
	patchPartialYAML []byte

	//go:embed data/R2xx-04.yaml
	deleteNoBodyYAML []byte

	//go:embed data/R2xx-05.yaml
	customMethodPostYAML []byte

	//go:embed data/R2xx-06.yaml
	putFullUpdateYAML []byte
)

// The R2xx family (w3-04): methods & verbs rules over the IR's HTTP
// surface (charter §3.2). Every check reads only Method.Verb, Params
// (in=body), Payload, Response.StatusCode and Response.Type — never an
// AST, never a file (the w3-03 boundary; brief "Protocol & bẫy").
//
// FP discipline (brief deliverable 2, W4's priority): the family splits
// into one MEASURED rule and five CONVENTION rules.
//
//   - R2xx-01 (ERROR) is measured: the handler itself declares a GET body;
//     the IR cannot be wrong about it (adapters keep Payload and the
//     in=body Param in lock-step, w2-02 invariant).
//   - R2xx-02/03/05/06 rest on the documented heuristics below (the action
//     lexicon and the payload/response type comparison) and therefore stay
//     at WARN/INFO with "verify manually" wording where the detection is
//     heuristic; when the heuristic cannot decide (no response type to
//     compare, action-shaped or template path) the rule stays silent.
//
// The checks are pure: findings derive only from the node (charter §5.4);
// the engine stamps RuleID and Severity.

// actionVerbs is the documented lexicon behind R2xx-02 (create shape) and
// R2xx-05 (action shape): path tokens that mark an ACTION rather than a
// resource. Membership is EXACT-token after tokenizing (no stemming), so
// plural resource nouns ("books", "runs", "playlists") never match. Words
// that read as both noun and verb in REST paths ("archive", "draft",
// "batch", "take", "compact") are deliberately absent — a lexicon that is
// too clever floods W4's FP budget; extend it only with a fixture-backed
// case (docs/rules/R2xx-05-custom-method-post.md).
var actionVerbs = map[string]bool{
	"activate": true, "assign": true, "approve": true, "calculate": true,
	"cancel": true, "cleanup": true, "clone": true, "compute": true,
	"convert": true, "deactivate": true, "disable": true, "download": true,
	"duplicate": true, "enable": true, "execute": true, "export": true,
	"fetch": true, "find": true, "generate": true, "get": true, "grant": true,
	"import": true, "invoke": true, "list": true, "lock": true, "lookup": true,
	"migrate": true, "process": true, "publish": true, "purge": true,
	"query": true, "refresh": true, "reject": true, "renew": true,
	"reset": true, "restore": true, "retry": true, "revoke": true,
	"run": true, "scan": true, "search": true, "send": true, "submit": true,
	"sync": true, "translate": true, "trigger": true, "unassign": true,
	"uninstall": true, "unlock": true, "unpublish": true, "update": true,
	"upload": true, "validate": true, "verify": true,
}

// verbSelector matches HTTP Methods of one verb — the family-level
// predicate spelled over IR concepts (charter §5.4). RPC operations never
// match: HTTP verb semantics do not apply to them.
type verbSelector struct{ verb ir.Verb }

func (s verbSelector) Matches(node ir.Node) bool {
	m, ok := node.(ir.Method)
	return ok && m.Verb == s.verb && m.Verb.IsHTTP()
}

// httpMethodSelector matches any HTTP-verb Method (R2xx-05 scans every
// endpoint shape for action paths).
type httpMethodSelector struct{}

func (httpMethodSelector) Matches(node ir.Node) bool {
	m, ok := node.(ir.Method)
	return ok && m.Verb.IsHTTP()
}

// bodyParamOf returns the method's in=body parameter, or nil. The stub and
// spring adapters fill Payload and this param together, but a rule must
// not assume it (the stub allows either half alone) — hasBody reads both.
func bodyParamOf(m *ir.Method) *ir.Param {
	for i := range m.Params {
		if m.Params[i].In == ir.ParamInBody {
			return &m.Params[i]
		}
	}
	return nil
}

// bodyTypeOf names the declared request body, preferring the Payload view.
func bodyTypeOf(m *ir.Method) string {
	if m.Payload != nil {
		return m.Payload.Name
	}
	if p := bodyParamOf(m); p != nil {
		return p.Type.Name
	}
	return ""
}

// bodyLocOf anchors a body finding at the body parameter when it exists
// (charter §3.0: a precise child node), falling back to the method itself.
func bodyLocOf(m *ir.Method) ir.Location {
	if p := bodyParamOf(m); p != nil && p.Location != (ir.Location{}) {
		return p.Location
	}
	return m.Location
}

// actionSegments returns the literal (non-template) path segments that
// carry an action-verb token, in path order — the R2xx-05 scan set.
func actionSegments(p string) []string {
	var out []string
	for _, seg := range strings.Split(strings.Trim(p, "/"), "/") {
		if seg == "" || strings.Contains(seg, "{") {
			continue
		}
		if hasActionToken(seg) {
			out = append(out, seg)
		}
	}
	return out
}

// hasActionToken reports whether one path segment tokenizes to an
// action-verb token. Splitting covers kebab-case, snake_case and
// camelCase: each rune boundary lower→upper starts a token, and every
// non-alphanumeric rune is a separator.
func hasActionToken(segment string) bool {
	for _, tok := range tokenizeSegment(segment) {
		if actionVerbs[tok] {
			return true
		}
	}
	return false
}

// tokenizeSegment splits one path segment into lower-case word tokens.
func tokenizeSegment(segment string) []string {
	var toks []string
	cur := strings.Builder{}
	prevLower := false
	for _, r := range segment {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			if cur.Len() > 0 {
				toks = append(toks, cur.String())
				cur.Reset()
			}
			prevLower = false
		case unicode.IsUpper(r) && prevLower:
			if cur.Len() > 0 {
				toks = append(toks, cur.String())
				cur.Reset()
			}
			cur.WriteRune(unicode.ToLower(r))
			prevLower = false
		default:
			cur.WriteRune(unicode.ToLower(r))
			prevLower = unicode.IsLower(r)
		}
	}
	if cur.Len() > 0 {
		toks = append(toks, cur.String())
	}
	return toks
}

// createShapedPath is R2xx-02's create detection (documented heuristic):
// no custom-method suffix anywhere (":verb" marks an AIP-136 action), no
// action-verb token in any literal segment, and a literal final segment —
// a create posts to a collection (AIP-133), never to an item template.
func createShapedPath(p string) bool {
	if strings.Contains(p, ":") {
		return false
	}
	segs := strings.Split(strings.Trim(p, "/"), "/")
	if len(segs) == 0 || segs[len(segs)-1] == "" || strings.Contains(segs[len(segs)-1], "{") {
		return false
	}
	return len(actionSegments(p)) == 0
}

// customMethodSuggestion builds the AIP-136 replacement for an action
// path (R2xx-05, the acceptance-critical suggestion): the action segment
// becomes a ":action" suffix on the preceding id segment, or on the
// collection root. When the action is the whole path the collection name
// is not derivable, so the placeholder stays visible.
func customMethodSuggestion(p string) string {
	segs := strings.Split(strings.Trim(p, "/"), "/")
	idx := -1
	for i, seg := range segs {
		if strings.Contains(seg, "{") {
			continue
		}
		if hasActionToken(seg) {
			idx = i // last action segment wins: /a/search/{id}/renew → :renew
		}
	}
	if idx < 0 {
		return "POST <collection>:{id}:<action>"
	}
	action := segs[idx]
	prefix := strings.Join(segs[:idx], "/")
	if prefix == "" {
		return "POST <collection>:" + action
	}
	return "POST /" + prefix + ":" + action
}

// getNoBodyCheck implements R2xx-01 (ERROR, measured): a GET that declares
// a request body loses that input to common HTTP clients.
func getNoBodyCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Verb != ir.VerbGet || !m.Verb.IsHTTP() {
		return nil
	}
	body := bodyTypeOf(&m)
	if body == "" {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("GET endpoint %s declares a request body (%s) — many HTTP clients silently drop GET bodies.", m.OperationName, body),
		Suggestion: "move the payload to query parameters",
		Location:   bodyLocOf(&m),
	}}
}

// postCreatesCheck implements R2xx-02 (WARN, heuristic): a POST that looks
// like a resource create — single-resource response on an action-free,
// non-template path — must declare 201/202. Void and collection responses
// are too weak a create signal; the rule prefers missing a marginal case
// over flooding (FP priority #1), and the message asks for verification.
func postCreates201Check(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Verb != ir.VerbPost || !m.Verb.IsHTTP() {
		return nil
	}
	if m.Response.StatusCode == 201 || m.Response.StatusCode == 202 {
		return nil
	}
	resp := m.Response.Type
	if resp.Name == "" || m.Response.IsCollection || resp.IsCollection {
		return nil
	}
	if !createShapedPath(m.Path) {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("POST endpoint %s looks like a resource create (single %s response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201.", m.OperationName, resp.Name),
		Suggestion: "@ResponseStatus(HttpStatus.CREATED) (or return ResponseEntity.status(HttpStatus.CREATED))",
		Location:   m.Response.Location,
	}}
}

// patchPartialCheck implements R2xx-03 (WARN, heuristic): PATCH whose
// payload type equals the declared response type carries the full entity;
// a full replacement belongs to PUT. Differing or undeclared response
// types cannot establish "full", so the rule stays silent.
func patchPartialCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Verb != ir.VerbPatch || !m.Verb.IsHTTP() {
		return nil
	}
	resp := m.Response.Type.Name
	if m.Payload == nil || resp == "" || !sameType(m.Payload.Name, resp) {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("PATCH endpoint %s takes the full %s payload it also returns — a full replacement belongs to PUT, partial updates send a smaller DTO.", m.OperationName, m.Payload.Name),
		Suggestion: "replace PATCH with PUT for this full-replacement update",
		Location:   bodyLocOf(&m),
	}}
}

// deleteNoBodyCheck implements R2xx-04 (WARN): DELETE bodies are legal but
// widely unsupported; bulk deletes read as batch paths or query parameters.
func deleteNoBodyCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Verb != ir.VerbDelete || !m.Verb.IsHTTP() {
		return nil
	}
	body := bodyTypeOf(&m)
	if body == "" {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("DELETE endpoint %s declares a request body (%s) — model bulk deletes as query parameters or a batch path instead.", m.OperationName, body),
		Suggestion: "move the body to query parameters (e.g. ?ids=1,2) or a batch path",
		Location:   bodyLocOf(&m),
	}}
}

// customMethodPostCheck implements R2xx-05 (INFO, heuristic): an action
// path served by a non-POST verb reads as a misplaced custom method;
// AIP-136 spells them POST <collection>/{id}:<action>. POST endpoints
// already satisfy the verb half of the convention — reshaping their path
// overlaps the R1xx family, so they stay silent.
func customMethodPostCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || !m.Verb.IsHTTP() || m.Verb == ir.VerbPost {
		return nil
	}
	segs := actionSegments(m.Path)
	if len(segs) == 0 {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("Endpoint %s (%s %s) looks like an action (non-CRUD) — AIP-136 custom methods read POST <collection>/{id}:<action> (verify manually).", m.OperationName, m.Verb, m.Path),
		Suggestion: customMethodSuggestion(m.Path),
		Location:   m.Location,
	}}
}

// putFullUpdateCheck implements R2xx-06 (WARN, heuristic — the symmetric
// counterpart of R2xx-03, w1-03 pain 7c): PUT whose payload type differs
// from the declared response type looks like a partial update; PATCH owns
// those. Matching or undeclared response types stay silent.
func putFullUpdateCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Verb != ir.VerbPut || !m.Verb.IsHTTP() {
		return nil
	}
	resp := m.Response.Type.Name
	if m.Payload == nil || resp == "" || sameType(m.Payload.Name, resp) {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("PUT endpoint %s takes payload %s but returns %s — the mismatch looks like a partial update, which belongs to PATCH.", m.OperationName, m.Payload.Name, resp),
		Suggestion: "replace PUT with PATCH for this partial update",
		Location:   bodyLocOf(&m),
	}}
}

// sameType compares two source type spellings case-insensitively: the
// payload/response pair must name the SAME type for the full-vs-partial
// heuristics to fire (R2xx-03/06).
func sameType(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// methodRulesYAML pairs every R2xx rule's embedded record with its
// selector and check — one data table like the demo set (adding a rule is
// one YAML file and one row, never engine work).
var methodRulesYAML = []struct {
	yaml  []byte
	sel   engine.Selector
	check engine.CheckFunc
}{
	{getNoBodyYAML, verbSelector{ir.VerbGet}, getNoBodyCheck},
	{postCreates201YAML, verbSelector{ir.VerbPost}, postCreates201Check},
	{patchPartialYAML, verbSelector{ir.VerbPatch}, patchPartialCheck},
	{deleteNoBodyYAML, verbSelector{ir.VerbDelete}, deleteNoBodyCheck},
	{customMethodPostYAML, httpMethodSelector{}, customMethodPostCheck},
	{putFullUpdateYAML, verbSelector{ir.VerbPut}, putFullUpdateCheck},
}

// MethodRules compiles the R2xx family. Registration errors (bad
// metadata) surface here as errors — fail tests, never runtime
// (charter §5.4).
func MethodRules() ([]engine.Rule, error) {
	out := make([]engine.Rule, 0, len(methodRulesYAML))
	for _, d := range methodRulesYAML {
		m, err := ParseMetadata(d.yaml)
		if err != nil {
			return nil, fmt.Errorf("methods rule metadata: %w", err)
		}
		rule, err := Compile(m, d.sel, d.check)
		if err != nil {
			return nil, fmt.Errorf("compile methods rule: %w", err)
		}
		out = append(out, rule)
	}
	return out, nil
}
