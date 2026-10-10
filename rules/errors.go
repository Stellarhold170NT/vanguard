package rules

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/vanguard-lint/vanguard/internal/engine"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

// The R5xx errors family (w3-06, charter §3.5, AIP-193 + AIP-133): one
// error contract per app, business exceptions never answer 500, and a
// creating POST answers 201/202. Every check reads only the IR the
// adapters emit — the ErrorScheme handlers w3-02 collects (plus the
// w3-06 ExceptionPackage/ExceptionStatus resolution), Method.Response,
// and the surface Types — never a source file. The heuristics are
// deliberately FP-first and documented in docs/rules/.

//go:embed data/R5xx-01.yaml
var unifiedErrorShapeYAML []byte

//go:embed data/R5xx-02.yaml
var no500ForBusinessYAML []byte

//go:embed data/R5xx-03.yaml
var statusSemanticsYAML []byte

// errorHandlerSelector matches ErrorScheme handlers — the nodes the R5xx
// family reasons about (walkSurface visits them outside services).
type errorHandlerSelector struct{}

func (errorHandlerSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.ErrorHandler)
	return ok
}

// errorSchemeNames are the option values R5xx-01 accepts for its scheme
// option. "auto" (the default) infers the app's envelope from the
// handlers themselves; the other two pin the expected shape and
// field-check every resolvable response type.
const r5xx01Auto = "auto"

var r5xx01Schemes = map[string]bool{
	r5xx01Auto:             true,
	"code-message-details": true,
	"problem-json":         true,
}

// problemJSONFields is the RFC-7807 member set; a problem+json body must
// carry at least two of them (type/title/status/detail/instance) so a
// near-miss DTO with a single lucky name stays silent.
var problemJSONFields = []string{"type", "title", "status", "detail", "instance"}

// unifiedErrorShapeCheck is R5xx-01: flag the handler whose answer deviates
// from the app's error envelope. In auto mode the envelope is the response
// type most handlers share (≥ 2, strictly dominant, resolvable as an IR
// Type — an unprovable standard stays silent); under a pinned scheme the
// handler's response type is field-checked against the required shape.
func unifiedErrorShapeCheck(ctx *engine.LintContext, node ir.Node) []engine.Finding {
	h, ok := node.(ir.ErrorHandler)
	if !ok || h.ResponseType == "" {
		return nil
	}
	scheme := "auto"
	if v := optionString(ctx, "scheme"); v != "" {
		scheme = v
	}
	if !r5xx01Schemes[scheme] {
		panic(fmt.Sprintf("rule R5xx-01: invalid scheme option %q (want auto, code-message-details or problem-json)", scheme))
	}
	if scheme != r5xx01Auto {
		return pinMiss(ctx, h, scheme)
	}

	std, n, total := majorityEnvelope(ctx)
	if std == "" {
		return nil // no resolvable consensus — an unprovable standard stays silent
	}
	if h.ResponseType == std {
		return nil
	}
	return []engine.Finding{{
		Message: fmt.Sprintf("@ExceptionHandler for %s answers %s while %d of %d handlers answer with %s — clients would parse two error contracts.",
			h.ExceptionType, h.ResponseType, n, total, std),
		Suggestion: "answer with the shared " + std + " envelope from every @ExceptionHandler",
		Location:   h.Location,
	}}
}

// majorityEnvelope infers the app's standard error envelope from the
// ErrorScheme: the response type that (a) at least two handlers share,
// (b) strictly dominates every other spelling, and (c) resolves to a Type
// in the surface (an envelope the scan can see fields of). "" when no
// such consensus exists.
func majorityEnvelope(ctx *engine.LintContext) (standard string, n, total int) {
	counts := map[string]int{}
	total = 0
	for _, h := range ctx.Surface.ErrorScheme.Handlers {
		if h.ResponseType == "" {
			continue // void handlers carry no envelope contract
		}
		counts[h.ResponseType]++
		total++
	}
	best, bestN, tie := "", 0, false
	for spelling, c := range counts {
		switch {
		case c > bestN:
			best, bestN, tie = spelling, c, false
		case c == bestN:
			tie = true
		}
	}
	if bestN < 2 || tie {
		return "", 0, total
	}
	if resolveType(ctx, best) == nil {
		return "", 0, total
	}
	return best, bestN, total
}

// resolveType looks up a response type spelling's base name in the
// surface's Types (simple-name addressing, the adapter's own convention).
func resolveType(ctx *engine.LintContext, spelling string) *ir.Type {
	base := spelling
	if i := strings.IndexByte(base, '<'); i > 0 {
		base = base[:i]
	}
	for i := range ctx.Surface.Types {
		if ctx.Surface.Types[i].Name == base {
			return &ctx.Surface.Types[i]
		}
	}
	return nil
}

// jsonFieldName is the effective serialized name of a field (the R4xx-02
// convention): an explicit JSONName wins, otherwise the field name.
func jsonFieldName(f ir.Field) string {
	if f.JSONName != "" {
		return f.JSONName
	}
	return f.Name
}

// typeHasFields reports whether the type declares all the named fields.
func typeHasFields(t *ir.Type, want ...string) bool {
	have := map[string]bool{}
	for _, f := range t.Fields {
		have[jsonFieldName(f)] = true
	}
	for _, w := range want {
		if !have[w] {
			return false
		}
	}
	return true
}

// pinMiss applies a pinned scheme: the handler's response type must carry
// the scheme's required fields when the scan can resolve the type.
// Void and unresolvable handlers stay silent (under-report, never guess).
func pinMiss(ctx *engine.LintContext, h ir.ErrorHandler, scheme string) []engine.Finding {
	t := resolveType(ctx, h.ResponseType)
	if t == nil {
		return nil
	}
	var want string
	switch scheme {
	case "code-message-details":
		if typeHasFields(t, "code", "message") {
			return nil
		}
		want = "code, message (details optional)"
	case "problem-json":
		have := 0
		for _, f := range t.Fields {
			for _, p := range problemJSONFields {
				if jsonFieldName(f) == p {
					have++
				}
			}
		}
		if have >= 2 {
			return nil
		}
		want = "at least two of: type, title, status, detail, instance"
	default:
		return nil
	}
	return []engine.Finding{{
		Message: fmt.Sprintf("@ExceptionHandler for %s answers %s, which lacks the unified error scheme fields (%s).",
			h.ExceptionType, h.ResponseType, want),
		Suggestion: "answer with the unified error envelope (" + want + ")",
		Location:   h.Location,
	}}
}

// optionString reads one declared string option from the merged rule
// options; "" when absent. A non-string value is a loud configuration
// error — the engine turns the panic into a per-rule diagnostic (§5.4)
// instead of silently ignoring the option.
func optionString(ctx *engine.LintContext, key string) string {
	if ctx == nil || ctx.Options == nil {
		return ""
	}
	v, ok := ctx.Options[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		panic(fmt.Sprintf("rule option %q must be a string, got %T", key, v))
	}
	return s
}

// optionStrings reads one declared list-of-names option; nil when absent.
// Accepts []string and the []any spelling YAML produces, with string
// elements only.
func optionStrings(ctx *engine.LintContext, key string) []string {
	if ctx == nil || ctx.Options == nil {
		return nil
	}
	v, ok := ctx.Options[key]
	if !ok || v == nil {
		return nil
	}
	switch list := v.(type) {
	case []string:
		return list
	case []any:
		out := make([]string, 0, len(list))
		for _, e := range list {
			s, ok := e.(string)
			if !ok {
				panic(fmt.Sprintf("rule option %q must be a list of strings, got element %T", key, e))
			}
			out = append(out, s)
		}
		return out
	default:
		panic(fmt.Sprintf("rule option %q must be a list of strings, got %T", key, v))
	}
}

// no500ForBusinessCheck is R5xx-02: a business exception mapped to 500.
// The 500 must be certain — declared by the handler (or the advice class)
// or, when they declare nothing, by the exception's own @ResponseStatus —
// and the exception must be app-owned (the adapter resolved its package
// from the repo) with the Exception suffix. External fallback handlers and
// non-suffixed names stay silent (FP-first).
func no500ForBusinessCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	h, ok := node.(ir.ErrorHandler)
	if !ok {
		return nil
	}
	effective := h.StatusCode
	if effective == 0 {
		effective = h.ExceptionStatus
	}
	if effective != 500 || h.ExceptionPackage == "" || !strings.HasSuffix(h.ExceptionType, "Exception") {
		return nil
	}
	return []engine.Finding{{
		Message: fmt.Sprintf("@ExceptionHandler maps the business exception %s to HTTP 500 — a business rule violation is a 4xx outcome, not a server failure.",
			h.ExceptionType),
		Suggestion: "map " + h.ExceptionType + " to a 4xx status naming the outcome (e.g. 409 Conflict or 422 Unprocessable Entity)",
		Location:   h.Location,
	}}
}

// r5xxActionWords are the path segments that make a POST an action rather
// than a creation (exact-token, lower-cased, no stemming — the w3-04
// lexicon discipline). A POST whose final literal segment is in this set
// is not create-shaped; misses under-report by design.
var r5xxActionWords = map[string]bool{
	"activate": true, "approve": true, "archive": true, "assign": true,
	"calculate": true, "cancel": true, "clone": true, "close": true,
	"complete": true, "convert": true, "deactivate": true, "disable": true,
	"duplicate": true, "enable": true, "execute": true, "export": true,
	"find": true, "generate": true, "grant": true, "import": true,
	"invoke": true, "invite": true, "lock": true, "login": true,
	"logout": true, "merge": true, "notify": true, "process": true,
	"publish": true, "purge": true, "query": true, "recalculate": true,
	"refresh": true, "register": true, "reject": true, "renew": true,
	"reset": true, "restore": true, "retry": true, "revoke": true,
	"reopen": true, "run": true, "search": true, "send": true,
	"submit": true, "sync": true, "transfer": true, "trigger": true,
	"unassign": true, "unlock": true, "unpublish": true, "upload": true,
	"validate": true, "verify": true,
}

// lastPathSegment returns the final non-empty segment of an absolute IR
// path ("" when the path is only slashes).
func lastPathSegment(path string) string {
	segments := strings.Split(path, "/")
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i] != "" {
			return segments[i]
		}
	}
	return ""
}

// r5xxCreateShapedPath reports whether a POST path reads as a creation: the
// final segment is a literal (no template variable, no AIP-136 custom
// method colon) and carries no action word. Deliberately the R5xx-03
// variant — it judges only the final segment against the family's own
// action-word set; methods.go's createShapedPath (R2xx-02) scans the whole
// method-level path. The two stay separate after the w3-07 merge so each
// family keeps the exact heuristic its tests pin.
func r5xxCreateShapedPath(path string) bool {
	last := lastPathSegment(path)
	if last == "" || strings.Contains(last, "{") || strings.Contains(last, ":") {
		return false
	}
	return !r5xxActionWords[strings.ToLower(last)]
}

// statusSemanticsCheck is R5xx-03: a POST that creates a resource must
// answer 201/202. An undeclared status counts as the implicit 200 the IR
// documents ("0 means not declared — consumers infer it from the verb"),
// so both the explicit and the conventional 200 fire; any other declared
// status (204, 3xx, 4xx…) is ambiguous on a create path and stays silent.
func statusSemanticsCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || m.Verb != ir.VerbPost || m.Response.IsCollection || m.Response.Type.Name == "" {
		return nil
	}
	if m.Response.StatusCode != 0 && m.Response.StatusCode != 200 {
		return nil
	}
	if !r5xxCreateShapedPath(m.Path) {
		return nil
	}
	var message string
	if m.Response.StatusCode == 200 {
		message = fmt.Sprintf("POST %s answers 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success.", m.Path)
	} else {
		message = fmt.Sprintf("POST %s relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success.", m.Path)
	}
	return []engine.Finding{{
		Message:    message,
		Suggestion: "declare @ResponseStatus(HttpStatus.CREATED) — or 202 Accepted for asynchronous creation",
		Location:   m.Location,
	}}
}

// ErrorRules compiles the R5xx family: R5xx-01 unified-error-shape (WARN),
// R5xx-02 no-500-for-business (ERROR), R5xx-03 status-semantics (WARN).
// The join is the production path (ParseMetadata → Compile, full §5.4
// validation); a bad record fails the build, never runtime.
func ErrorRules() ([]engine.Rule, error) {
	records := []struct {
		yaml  []byte
		sel   engine.Selector
		check engine.CheckFunc
	}{
		{unifiedErrorShapeYAML, errorHandlerSelector{}, unifiedErrorShapeCheck},
		{no500ForBusinessYAML, errorHandlerSelector{}, no500ForBusinessCheck},
		{statusSemanticsYAML, methodSelector{}, statusSemanticsCheck},
	}
	out := make([]engine.Rule, 0, len(records))
	for _, rec := range records {
		meta, err := ParseMetadata(rec.yaml)
		if err != nil {
			return nil, fmt.Errorf("errors rule metadata: %w", err)
		}
		rule, err := Compile(meta, rec.sel, rec.check)
		if err != nil {
			return nil, fmt.Errorf("compile errors rule: %w", err)
		}
		out = append(out, rule)
	}
	return out, nil
}
