package rules

import (
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/vanguard-lint/vanguard/internal/engine"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

// The R1xx family — Resource & naming rules (charter §3.1; brief w3-03).
// One embedded YAML record per rule (charter §3.0: metadata is data) joined
// to its check here, exactly like the demo set: metadata → ParseMetadata →
// Compile. Checks read only the IR — never a Java AST (charter §12: the
// engine and rules know nothing about tree-sitter).
//
// FP discipline (charter §3.1 severity column): heuristics default to
// WARN/INFO; only R1xx-02 (exact CRUD-verb tokens on the path) is ERROR.

//go:embed data/R1xx-01.yaml
var r1xx01YAML []byte

//go:embed data/R1xx-02.yaml
var r1xx02YAML []byte

//go:embed data/R1xx-03.yaml
var r1xx03YAML []byte

//go:embed data/R1xx-04.yaml
var r1xx04YAML []byte

//go:embed data/R1xx-05.yaml
var r1xx05YAML []byte

// resourceRuleYAML pairs every R1xx record with its selector and check —
// one data table; adding a rule is one YAML file and one row (§3.0).
var resourceRuleYAML = []struct {
	yaml  []byte
	sel   engine.Selector
	check engine.CheckFunc
}{
	{r1xx01YAML, httpMethodSelector{}, pluralCollectionCheck},
	{r1xx02YAML, httpMethodSelector{}, noVerbPathCheck},
	{r1xx03YAML, httpMethodSelector{}, resourcePathPatternCheck},
	{r1xx04YAML, httpMethodSelector{}, pathCasingCheck},
	{r1xx05YAML, typeNodeSelector{}, idFieldNamingCheck},
}

// ResourceRules compiles the whole R1xx family through the production path
// (embedded metadata → ParseMetadata → Compile, full §5.4 validation).
// Registration errors (bad metadata) surface here — fail tests, never
// runtime.
func ResourceRules() ([]engine.Rule, error) {
	out := make([]engine.Rule, 0, len(resourceRuleYAML))
	for _, d := range resourceRuleYAML {
		m, err := ParseMetadata(d.yaml)
		if err != nil {
			return nil, fmt.Errorf("R1xx metadata: %w", err)
		}
		rule, err := Compile(m, d.sel, d.check)
		if err != nil {
			return nil, fmt.Errorf("compile R1xx rule: %w", err)
		}
		out = append(out, rule)
	}
	return out, nil
}

// httpMethodSelector matches HTTP Methods only — path rules have no opinion
// on gRPC rpc operations (ir.Verb.IsHTTP excludes rpc).
type httpMethodSelector struct{}

func (httpMethodSelector) Matches(node ir.Node) bool {
	m, ok := node.(ir.Method)
	return ok && m.Verb.IsHTTP()
}

// typeNodeSelector matches Type nodes (DTOs, records, entities).
type typeNodeSelector struct{}

func (typeNodeSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.Type)
	return ok
}

// ---------------------------------------------------------------------------
// Shared path helpers — the IR path is the full merged route (w3-02 owns the
// base-path merge), normalized and starting with "/".

// pathSegments splits an absolute path into raw segments:
// "/books/{id}/reviews" → ["books", "{id}", "reviews"].
func pathSegments(p string) []string {
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// isParamSegment reports whether a segment is a path variable ("{id}",
// "{id:\\d+}", "{*path}").
func isParamSegment(seg string) bool {
	return strings.Contains(seg, "{") && strings.Contains(seg, "}")
}

// paramName extracts the variable name: "{id}" → "id", "{id:\\d+}" → "id"
// (the regex suffix is dropped), "{*path}" → "path" (Spring catch-all).
func paramName(seg string) string {
	i := strings.Index(seg, "{")
	j := strings.LastIndex(seg, "}")
	if i < 0 || j <= i {
		return ""
	}
	name := seg[i+1 : j]
	if k := strings.Index(name, ":"); k >= 0 {
		name = name[:k]
	}
	return strings.TrimPrefix(name, "*")
}

// isWildcardSegment reports Spring wildcard segments ("*", "**") — they
// carry no resource noun, so every R1xx path check skips them.
func isWildcardSegment(seg string) bool {
	return strings.Contains(seg, "*")
}

// pathTokens splits one segment into lower-case word tokens: kebab-case,
// snake_case and camelCase all split ("getUserById" → get user by id,
// "all-by-Name" → all by name). Token boundaries — never substrings — are
// what keep "getter" and "updates" out of the verb set (test-strategy §3.2
// near-miss design).
func pathTokens(seg string) []string {
	var out []string
	for _, word := range strings.FieldsFunc(seg, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ' '
	}) {
		out = append(out, camelCaseTokens(word)...)
	}
	return out
}

// camelCaseTokens splits a single word at case boundaries, lower-cased
// (pagination.go's camelWords is the name-level splitter — this one is the
// within-word tokenizer shared by pathTokens, w3-07 merge).
func camelCaseTokens(word string) []string {
	rs := []rune(word)
	if len(rs) < 2 {
		return []string{strings.ToLower(word)}
	}
	var out []string
	start := 0
	for i := 1; i < len(rs); i++ {
		if unicode.IsUpper(rs[i]) && (unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1])) {
			out = append(out, strings.ToLower(string(rs[start:i])))
			start = i
		}
	}
	out = append(out, strings.ToLower(string(rs[start:])))
	return out
}

// ---------------------------------------------------------------------------
// R1xx-02 support — the CRUD-verb token set (also reused by R1xx-03 to
// recognize action-shaped segments).
//
// The set is deliberately narrow: CRUD verbs and their direct synonyms.
// Action verbs outside CRUD (search, export, generate, checkout, rebuild…)
// are R2xx-05 custom-method-post's subject; flagging them here would double
// every finding and pull convention-shaped FP risk into an ERROR rule.
var crudVerbTokens = map[string]bool{
	"get": true, "post": true, "put": true, "patch": true, "delete": true,
	"del": true, "create": true, "update": true, "edit": true,
	"modify": true, "remove": true, "destroy": true, "fetch": true,
	"save": true, "add": true, "insert": true, "upsert": true,
	"find": true,
}

// crudVerbHint maps a verb token to the HTTP interaction that replaces it
// (brief w3-03: the suggestion names the right HTTP verb).
var crudVerbHint = map[string]string{
	"get":     "GET /<collection>/{id}",
	"fetch":   "GET /<collection>/{id}",
	"find":    "GET /<collection>/{id}",
	"post":    "POST /<collection>",
	"create":  "POST /<collection>",
	"add":     "POST /<collection>",
	"insert":  "POST /<collection>",
	"upsert":  "PUT /<collection>/{id}",
	"put":     "PUT /<collection>/{id}",
	"update":  "PUT /<collection>/{id} (full) or PATCH /<collection>/{id} (partial)",
	"edit":    "PATCH /<collection>/{id}",
	"modify":  "PATCH /<collection>/{id}",
	"patch":   "PATCH /<collection>/{id}",
	"save":    "PUT /<collection>/{id}",
	"delete":  "DELETE /<collection>/{id}",
	"del":     "DELETE /<collection>/{id}",
	"remove":  "DELETE /<collection>/{id}",
	"destroy": "DELETE /<collection>/{id}",
}

func firstVerbToken(seg string) string {
	for _, tok := range pathTokens(seg) {
		if crudVerbTokens[tok] {
			return tok
		}
	}
	return ""
}

// hasVerbToken reports whether a segment contains a CRUD verb token.
func hasVerbToken(seg string) bool {
	return firstVerbToken(seg) != ""
}

// ---------------------------------------------------------------------------
// R1xx-01 support — plural heuristics, word lists instead of a dictionary
// (static binary, charter R4). The lists are the FP shield: uncountable and
// irregular nouns never fire; s-ending singulars do.

// uncountableNouns have no plural form — as collection names they are fine
// (test-strategy §3.3: /equipment is ok).
var uncountableNouns = map[string]bool{
	"equipment": true, "information": true, "metadata": true, "data": true,
	"feedback": true, "software": true, "hardware": true, "firmware": true,
	"staff": true, "traffic": true, "media": true, "news": true,
	"research": true, "advice": true, "evidence": true, "knowledge": true,
	"progress": true, "furniture": true, "luggage": true, "baggage": true,
	"series": true, "species": true, "aircraft": true, "fish": true,
	"sheep": true, "deer": true, "offspring": true,
}

// irregularPlurals are plural without an s-suffix.
var irregularPlurals = map[string]bool{
	"people": true, "children": true, "men": true, "women": true,
	"feet": true, "teeth": true, "geese": true, "mice": true, "lice": true,
	"oxen": true, "leaves": true, "loaves": true, "thieves": true,
	"wives": true, "wolves": true, "shelves": true, "knives": true,
	"lives": true, "halves": true, "calves": true, "elves": true,
	"scarves": true,
}

// sSingularNouns end in s yet are singular — the only s-endings the rule
// flags (an exact-match blocklist keeps the heuristic honest: "status",
// "address", "analysis" are singular).
var sSingularNouns = map[string]bool{
	"status": true, "bus": true, "gas": true, "lens": true, "alias": true,
	"atlas": true, "canvas": true, "chaos": true, "circus": true,
	"virus": true, "campus": true, "corpus": true, "census": true,
	"genus": true, "nexus": true, "bonus": true, "radius": true,
	"surplus": true, "syllabus": true, "analysis": true, "axis": true,
	"basis": true, "crisis": true, "diagnosis": true, "emphasis": true,
	"oasis": true, "bias": true, "access": true, "address": true,
	"process": true, "success": true,
}

// fPluralNouns pluralize with -ves (shelf → shelves).
var fPluralNouns = map[string]bool{
	"shelf": true, "leaf": true, "wolf": true, "wife": true, "knife": true,
	"life": true, "half": true, "calf": true, "elf": true, "loaf": true,
	"thief": true, "scarf": true,
}

// isPluralNoun reports whether a path segment reads as a plural noun
// (documented heuristic: see the lists above; everything else is plural
// exactly when it ends in s).
func isPluralNoun(seg string) bool {
	s := strings.ToLower(seg)
	if uncountableNouns[s] || irregularPlurals[s] {
		return true
	}
	if sSingularNouns[s] {
		return false
	}
	return strings.HasSuffix(s, "s")
}

// pluralizeNoun builds the suggested plural of a singular segment — a
// documented heuristic; a human reviews the suggestion before applying it.
func pluralizeNoun(seg string) string {
	s := strings.ToLower(seg)
	switch {
	case s == "person":
		return "people"
	case s == "man":
		return "men"
	case s == "woman":
		return "women"
	case strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(rune(s[len(s)-2])):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh"):
		return s + "es"
	case fPluralNouns[s]:
		return s[:len(s)-1] + "ves"
	default:
		return s + "s"
	}
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

// aggregateSegments are non-resource sub-paths (custom-method territory —
// R2xx-05's subject, whose actionSegments scan lives in methods.go) and
// aggregate endpoints. They are exempt from the final-segment plural check
// so GET /books/search is not read as a singular collection.
var aggregateSegments = map[string]bool{
	"search": true, "export": true, "count": true, "lookup": true,
	"stats": true, "statistics": true, "health": true, "status": true,
	"info": true, "metrics": true, "exists": true, "actuator": true,
}

// isPlainResourceSegment reports whether a segment can carry a collection
// noun for R1xx-01: not a variable, not a wildcard, no CRUD verb inside
// (verb-shaped segments are R1xx-02/R1xx-03's subject — one aspect per
// rule, no double reporting).
func isPlainResourceSegment(seg string) bool {
	return !isParamSegment(seg) && !isWildcardSegment(seg) && !hasVerbToken(seg)
}

// responseIsCollection reads the two places the adapters mark collections
// (Response.IsCollection for erased wrappers, Type.IsCollection for
// Page<...>/List<...> spellings).
func responseIsCollection(m ir.Method) bool {
	return m.Response.IsCollection || m.Response.Type.IsCollection
}

// pathWithSegment returns the path with segment idx replaced.
func pathWithSegment(p string, idx int, repl string) string {
	segs := pathSegments(p)
	if idx < 0 || idx >= len(segs) {
		return p
	}
	segs[idx] = repl
	return "/" + strings.Join(segs, "/")
}

// pathWithoutSegment returns the path with one segment removed.
func pathWithoutSegment(p string, idx int) string {
	segs := pathSegments(p)
	if idx < 0 || idx >= len(segs) {
		return p
	}
	segs = append(segs[:idx], segs[idx+1:]...)
	if len(segs) == 0 {
		return "/"
	}
	return "/" + strings.Join(segs, "/")
}

// ---------------------------------------------------------------------------
// R1xx-01 plural-collection

func pluralCollectionCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok {
		return nil
	}
	segs := pathSegments(m.Path)
	if len(segs) == 0 {
		return nil
	}
	var out []engine.Finding
	// (i) every resource segment directly followed by a path variable is a
	// collection position: /book/{id} reads as one book.
	for i := 0; i < len(segs)-1; i++ {
		if !isParamSegment(segs[i+1]) || !isPlainResourceSegment(segs[i]) {
			continue
		}
		if isPluralNoun(segs[i]) {
			continue
		}
		out = append(out, pluralCollectionFinding(m, segs, i))
	}
	// (iii) the final segment of a GET whose response is a collection is the
	// list route itself: /training-plan listing plans.
	last := len(segs) - 1
	if m.Verb == ir.VerbGet && responseIsCollection(m) &&
		isPlainResourceSegment(segs[last]) &&
		!aggregateSegments[strings.ToLower(segs[last])] &&
		!isPluralNoun(segs[last]) {
		out = append(out, pluralCollectionFinding(m, segs, last))
	}
	return out
}

func pluralCollectionFinding(m ir.Method, segs []string, i int) engine.Finding {
	return engine.Finding{
		Message:    fmt.Sprintf("Collection path segment %q is singular — collections use plural nouns so one concept keeps one URL (AIP-131, AIP-122).", segs[i]),
		Suggestion: pathWithSegment(m.Path, i, pluralizeNoun(segs[i])),
		Location:   m.Location,
	}
}

// ---------------------------------------------------------------------------
// R1xx-02 no-verb-path

func noVerbPathCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok {
		return nil
	}
	segs := pathSegments(m.Path)
	var out []engine.Finding
	for _, seg := range segs {
		if isParamSegment(seg) || isWildcardSegment(seg) {
			continue
		}
		verb := firstVerbToken(seg)
		if verb == "" {
			continue
		}
		out = append(out, engine.Finding{
			Message:    fmt.Sprintf("Path segment %q states the CRUD verb %q — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135).", seg, verb),
			Suggestion: crudVerbHint[verb],
			Location:   m.Location,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// R1xx-03 resource-path-pattern

func resourcePathPatternCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok {
		return nil
	}
	segs := pathSegments(m.Path)
	var out []engine.Finding
	for i, seg := range segs {
		if !isParamSegment(seg) || isWildcardSegment(seg) {
			continue
		}
		name := paramName(seg)
		switch {
		case i == 0:
			out = append(out, engine.Finding{
				Message:    fmt.Sprintf("Path starts with the variable {%s} — anchor the collection first: /<collection>/{%s} (AIP-127).", name, name),
				Suggestion: fmt.Sprintf("/<collection>/{%s}", name),
				Location:   m.Location,
			})
		case isParamSegment(segs[i-1]):
			out = append(out, engine.Finding{
				Message:    fmt.Sprintf("Variables {%s} and {%s} are consecutive — insert the sub-collection that owns {%s}: /<parent>/{id}/<sub-collection>/{%s} (AIP-127).", paramName(segs[i-1]), name, name, name),
				Suggestion: fmt.Sprintf("/<parent>/{id}/<sub-collection>/{%s}", name),
				Location:   m.Location,
			})
		case hasVerbToken(segs[i-1]):
			out = append(out, engine.Finding{
				Message:    fmt.Sprintf("Variable {%s} sits under the action segment %q — serve the item under its resource: /<collection>/{%s} (AIP-127).", name, segs[i-1], name),
				Suggestion: fmt.Sprintf("/<collection>/{%s}", name),
				Location:   m.Location,
			})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// R1xx-04 path-casing

// literalSegmentPattern pins kebab-case for literal path segments; a
// lower-case dot suffix (export.csv) stays legal.
var literalSegmentPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// lowerCamelPattern pins lowerCamelCase for path variables.
var lowerCamelPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func pathCasingCheck(_ *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok {
		return nil
	}
	segs := pathSegments(m.Path)
	var out []engine.Finding
	for i, seg := range segs {
		switch {
		case isWildcardSegment(seg):
			continue
		case isParamSegment(seg):
			name := paramName(seg)
			if name == "" || lowerCamelPattern.MatchString(name) {
				continue
			}
			out = append(out, engine.Finding{
				Message:    fmt.Sprintf("Path variable {%s} is not lowerCamelCase — keep one variable convention (AIP-122).", name),
				Suggestion: pathWithSegment(m.Path, i, "{"+pathTokenLowerCamel(name)+"}"),
				Location:   m.Location,
			})
		default:
			if literalSegmentPattern.MatchString(seg) {
				continue
			}
			out = append(out, engine.Finding{
				Message:    fmt.Sprintf("Path segment %q is not kebab-case — keep one segment convention (AIP-122).", seg),
				Suggestion: pathWithSegment(m.Path, i, toKebab(seg)),
				Location:   m.Location,
			})
		}
	}
	return out
}

// toKebab lower-cases a segment and joins its words with single dashes.
func toKebab(seg string) string {
	toks := pathTokens(seg)
	if len(toks) == 0 {
		return seg
	}
	return strings.Join(toks, "-")
}

// pathTokenLowerCamel converts a variable name to lowerCamelCase via the
// R1xx path tokens (payload.go's toLowerCamel is the R4xx field-name
// variant — both predate the w3-07 merge, so the names stay separate).
func pathTokenLowerCamel(name string) string {
	toks := pathTokens(name)
	if len(toks) == 0 {
		return name
	}
	out := toks[0]
	for _, t := range toks[1:] {
		if t == "" {
			continue
		}
		r := []rune(t)
		r[0] = unicode.ToUpper(r[0])
		out += string(r)
	}
	return out
}

// ---------------------------------------------------------------------------
// R1xx-05 id-field-naming

// idSuffixConvention returns the id-suffix spelling of a serialized name:
// "id" (exact self id), "Id", "ID" or "_id"; "" when the name does not END
// in an id word. The suffix must be an exact case-sensitive spelling, so
// words like "android" or "valid" never read as id fields.
func idSuffixConvention(s string) string {
	switch {
	case s == "id":
		return "id"
	case strings.HasSuffix(s, "_id"):
		return "_id"
	case strings.HasSuffix(s, "Id") && len(s) > 2:
		return "Id"
	case strings.HasSuffix(s, "ID") && len(s) > 2:
		return "ID"
	}
	return ""
}

func idFieldNamingCheck(ctx *engine.LintContext, node ir.Node) []engine.Finding {
	t, ok := node.(ir.Type)
	if !ok {
		return nil
	}
	var out []engine.Finding

	// A. reference-id suffix mix inside one type — on the SERIALIZED name so
	// a JSON key that spells the id differently counts too (the wire is what
	// clients see). The self id is the field named exactly "id" — its JSON
	// key spelling is R4xx-02 field-casing's subject, not a reference.
	convs := map[string][]string{}
	for _, f := range t.Fields {
		ser := f.JSONName
		if ser == "" {
			ser = f.Name
		}
		if f.Name == "id" || ser == "id" {
			continue
		}
		if c := idSuffixConvention(ser); c != "" {
			convs[c] = append(convs[c], ser)
		}
	}
	if len(convs) > 1 {
		out = append(out, engine.Finding{
			Message:    fmt.Sprintf("Type %s mixes reference-id spellings (%s) — keep one: <resource>Id (AIP-140).", t.Name, describeConventions(convs)),
			Suggestion: "spell every reference id <resource>Id",
			Location:   t.Location,
		})
	}

	// B. self-id style mix across the surface (v0.1 reports mixes, it does
	// not enforce one convention — charter §3.1 R1xx-05).
	out = append(out, selfIDMixFinding(ctx, t)...)
	return out
}

// describeConventions renders the offending spellings deterministically:
// '"ID"-style (bookID), "_id"-style (member_id), "Id"-style (publisherId)'.
func describeConventions(convs map[string][]string) string {
	keys := make([]string, 0, len(convs))
	for c := range convs {
		keys = append(keys, c)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, c := range keys {
		fields := append([]string(nil), convs[c]...)
		sort.Strings(fields)
		parts = append(parts, fmt.Sprintf("%q-style (%s)", c, strings.Join(fields, ", ")))
	}
	return strings.Join(parts, ", ")
}

// dtoSuffixes are the type-name decorations vanguard sees in Spring code;
// selfIDStyle strips them before comparing the id field's resource prefix
// with the type name (orderId inside OrderDto).
var dtoSuffixes = []string{"Dto", "DTO", "Request", "Response", "Entity",
	"Payload", "Record", "Summary", "Digest", "Item", "Info", "Model"}

// selfIDStyle reports how a Type spells its own id: "id" when an exact id
// field exists; "resource-id" when an id-shaped field's resource prefix is
// exactly the type's own name (orderId inside OrderDto); "" when it has no
// opinion. Exact equality after suffix stripping keeps projection types
// (AuthorSummaryDto with editorId) out of the comparison — the loose
// prefix match flagged them as self-style and cried wolf (w3-03 FP round).
func selfIDStyle(t ir.Type) (style, field string) {
	for _, f := range t.Fields {
		if f.Name == "id" {
			return "id", f.Name
		}
	}
	for _, f := range t.Fields {
		c := idSuffixConvention(f.Name)
		if c == "" || c == "id" {
			continue
		}
		prefix := strings.ToLower(strings.TrimSuffix(f.Name, c))
		name := strings.ToLower(trimTypeSuffixes(t.Name))
		if prefix != "" && prefix == name {
			return "resource-id", f.Name
		}
	}
	return "", ""
}

// trimTypeSuffixes repeatedly strips one decoration suffix from a type
// name ("OrderDto" → "order", "MemberRecord" → "member").
func trimTypeSuffixes(name string) string {
	out := name
	for _, suf := range dtoSuffixes {
		if len(out) > len(suf) && strings.HasSuffix(out, suf) {
			out = strings.TrimSuffix(out, suf)
		}
	}
	return out
}

// selfIDMixFinding fires when the surface holds BOTH self-id styles and the
// visited type spells its own id after the resource: the "id" style wins
// the message, the deviating type carries the finding.
func selfIDMixFinding(ctx *engine.LintContext, t ir.Type) []engine.Finding {
	if ctx == nil || ctx.Surface == nil {
		return nil
	}
	styles := map[string]bool{}
	for i := range ctx.Surface.Types {
		s, _ := selfIDStyle(ctx.Surface.Types[i])
		if s != "" {
			styles[s] = true
		}
	}
	if !styles["id"] || !styles["resource-id"] {
		return nil // one style or none — no mix to report (FP gate)
	}
	style, field := selfIDStyle(t)
	if style != "resource-id" {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("Type %s names its own id %q while other types use \"id\" — keep one self-id convention (AIP-140).", t.Name, field),
		Suggestion: "rename the field to \"id\", or align the other types on <resource>Id",
		Location:   t.Location,
	}}
}
