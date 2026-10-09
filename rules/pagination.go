package rules

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// The R3xx pagination family (w3-05, charter §3.3, AIP-158): the shape of
// collection endpoints. Every check reads only the IR the adapters emit —
// Method.Verb/Path/Response/Pagination and the bound Params — never a
// source file. The heuristics are deliberately narrow and documented in
// docs/rules/: W4 attacks exactly these edges.

//go:embed data/R3xx-01.yaml
var listPaginatedYAML []byte

//go:embed data/R3xx-02.yaml
var listEnvelopeYAML []byte

//go:embed data/R3xx-03.yaml
var unboundedPageSizeYAML []byte

// bareCollectionBases are the container types whose JSON serialization is a
// bare array: the adapter's collectionBases minus the Spring envelope
// shapes (Page/Slice/PageImpl carry total/page metadata already). Any other
// collection spelling (a custom PageResponse, a generated wrapper) is
// treated as an envelope — the rule under-reports rather than guessing.
var bareCollectionBases = map[string]bool{
	"List": true, "ArrayList": true, "LinkedList": true,
	"Collection": true, "Iterable": true, "Stream": true,
	"Set": true, "HashSet": true, "LinkedHashSet": true,
	"SortedSet": true, "TreeSet": true,
	"Queue": true, "Deque": true,
}

// camelWords splits a field/parameter name into its words: on separators
// (_ and -) and at lower→upper camelCase transitions. Digits stay attached
// to the preceding word. "created_at" → [created at]; "pageSize" →
// [page size]; "URL" → [URL]; "candidate" → [candidate] (one word — the
// substring traps "format"/"candidate" never read as "at"/"date").
func camelWords(name string) []string {
	var words []string
	start := 0
	rs := []rune(name)
	for i := 1; i < len(rs); i++ {
		boundary := false
		switch {
		case rs[i] == '_' || rs[i] == '-':
			boundary = true
		case unicode.IsUpper(rs[i]) && (unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1])):
			boundary = true
		}
		if boundary {
			words = append(words, string(rs[start:i]))
			if rs[i] == '_' || rs[i] == '-' {
				start = i + 1
			} else {
				start = i
			}
		}
	}
	words = append(words, string(rs[start:]))
	return words
}

// lastWordLower returns the last camelWord of name, lower-cased (""
// for an empty name).
func lastWordLower(name string) string {
	w := camelWords(name)
	if len(w) == 0 {
		return ""
	}
	return strings.ToLower(w[len(w)-1])
}

// isListEndpoint spells the R3xx predicate "this operation serves a
// collection": an HTTP GET whose response is a declared collection and
// whose path is not an item path (the last path segment carries no
// template variable — /orders is a collection, /orders/{id} is an item).
func isListEndpoint(m ir.Method) bool {
	if m.Verb != ir.VerbGet || !m.Response.IsCollection || m.Response.Type.Name == "" {
		return false
	}
	segments := strings.Split(m.Path, "/")
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i] == "" {
			continue
		}
		return !strings.Contains(segments[i], "{")
	}
	return true // the root path itself — still a collection read
}

// unpaginated reports whether the method takes no pagination input at all.
// Anything that is not explicitly pageable/params counts (the IR states
// the shape explicitly; Validate rejects unknown spellings).
func unpaginated(m ir.Method) bool {
	switch m.Pagination.Style {
	case ir.PaginationPageable, ir.PaginationParams:
		return false
	}
	return true
}

// listPaginatedCheck is R3xx-01: flag the unpaginated list endpoints.
func listPaginatedCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || !isListEndpoint(m) || !unpaginated(m) {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("GET %s returns an unpaginated collection — one client request can dump the whole table.", m.Path),
		Suggestion: "add a Pageable parameter or page/size query parameters to bound the result set",
		Location:   node.Loc(),
	}}
}

// listEnvelopeCheck is R3xx-02: flag the bare-array collection responses.
func listEnvelopeCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || !m.Response.IsCollection || m.Response.Type.Name == "" {
		return nil
	}
	if !bareCollectionBases[baseTypeName(m.Response.Type.Name)] &&
		!strings.HasSuffix(m.Response.Type.Name, "[]") {
		return nil
	}
	return []engine.Finding{{
		Message: fmt.Sprintf("%s returns a bare %s — a bare JSON array leaves no room for total/page metadata.",
			m.Path, m.Response.Type.Name),
		Suggestion: "wrap the items in an envelope record carrying items plus paging metadata (total, page)",
		Location:   node.Loc(),
	}}
}

// sizeCarryingNames are the last words that make a query parameter a page
// size (pageSize, maxSize, limit… — but not page or offset themselves).
var sizeCarryingNames = map[string]bool{"size": true, "limit": true}

// sizeCarryingName reports whether the parameter name denotes a page size:
// the whole name, or its last camelCase word, is size/limit.
func sizeCarryingName(name string) bool {
	if name == "" {
		return false
	}
	last := lastWordLower(name)
	return last == "size" || last == "limit"
}

// unboundedPageSizeCheck is R3xx-03: flag size/limit query parameters that
// carry no @Max-style cap. Framework Pageable arguments are out of scope —
// their cap is a resolver concern the IR cannot see (w3-02 limitation).
func unboundedPageSizeCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	p, ok := node.(ir.Param)
	if !ok || p.In != ir.ParamInQuery || !sizeCarryingName(p.Name) {
		return nil
	}
	if strings.Contains(p.Validation, "@Max") {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("Query parameter %q has no upper bound — a single client can request unbounded pages.", p.Name),
		Suggestion: "annotate the parameter with @Max (e.g. @Max(100)) to cap the page size",
		Location:   node.Loc(),
	}}
}

// familyMember pairs one embedded metadata record with its selector and
// check — the same join DemoRules uses, shared by the w3-05 families.
type familyMember struct {
	yaml  []byte
	sel   engine.Selector
	check engine.CheckFunc
}

// compileFamily builds every member through the production path
// (ParseMetadata → Compile, full §5.4 validation); a bad record fails the
// build, never runtime.
func compileFamily(family string, members []familyMember) ([]engine.Rule, error) {
	out := make([]engine.Rule, 0, len(members))
	for _, m := range members {
		meta, err := ParseMetadata(m.yaml)
		if err != nil {
			return nil, fmt.Errorf("%s rule metadata: %w", family, err)
		}
		rule, err := Compile(meta, m.sel, m.check)
		if err != nil {
			return nil, fmt.Errorf("compile %s rule: %w", family, err)
		}
		out = append(out, rule)
	}
	return out, nil
}

// PaginationRules compiles the R3xx family: R3xx-01 list-paginated (WARN),
// R3xx-02 list-envelope (INFO), R3xx-03 unbounded-page-size (INFO).
func PaginationRules() ([]engine.Rule, error) {
	return compileFamily("pagination", []familyMember{
		{listPaginatedYAML, methodSelector{}, listPaginatedCheck},
		{listEnvelopeYAML, methodSelector{}, listEnvelopeCheck},
		{unboundedPageSizeYAML, queryParamSelector{}, unboundedPageSizeCheck},
	})
}

// queryParamSelector matches Param nodes bound to the query string — the
// only binding where page sizing lives.
type queryParamSelector struct{}

func (queryParamSelector) Matches(node ir.Node) bool {
	p, ok := node.(ir.Param)
	return ok && p.In == ir.ParamInQuery
}
