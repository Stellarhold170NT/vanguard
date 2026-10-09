package mutation

import (
	"regexp"
	"strconv"
	"strings"
)

// Mutator is one deterministic source transformation plus the rule it traps.
// Apply returns the mutated source and whether it applied at all: a mutator
// that finds none of its trigger patterns reports (src, false) and the
// harness logs the case as not-applicable instead of manufacturing a miss.
type Mutator struct {
	Name        string // strategy §4.1 name — the brief w4-02 names are kept verbatim
	Kind        string // "must" (§4.1 MUST set) or "optional"
	Description string // the §4.1 operation, one line
	Pairings    []Pairing
	Apply       func(src []byte) ([]byte, bool)
}

// Pairing wires one pool of ok-cases to the rules the mutated code must trip.
// PoolRule names the manifest rule whose ok-cases supply the material ("*"
// lets the mutator feed on any ok case — verb-swap needs GET endpoints, which
// no single rule's ok set provides). Targets are the rules counted as a hit;
// a case tallies toward every rule in Targets (test-strategy §4.1 lists
// shared mutators as "R2xx-02 (R5xx-03)" — the parenthetical is a second
// target, not an also_expect).
type Pairing struct {
	PoolRule string
	Targets  []string
}

// ---------------------------------------------------------------------------
// Lexicons mirrored from the rule packages (rules/resources.go,
// rules/methods.go, rules/versioning.go — unexported there, so the harness
// keeps its own copy). The copies exist so a mutator can tell "this
// mutation is well-formed for the rule's predicate" from "this mutation is
// a no-op"; drift between the copies surfaces as honest harness misses, and
// the harness run itself is the tripwire.

// crudVerbs are the CRUD-verb path tokens R1xx-02 flags; a mutant path
// segment carrying one is an R1xx-02 violation by the rule's own definition.
var crudVerbs = map[string]bool{
	"get": true, "post": true, "put": true, "patch": true, "delete": true,
	"del": true, "create": true, "update": true, "edit": true,
	"modify": true, "remove": true, "destroy": true, "fetch": true,
	"save": true, "add": true, "insert": true, "upsert": true,
	"find": true,
}

// neverPlural are the s-ending singulars (rules/resources.go sSingularNouns).
var neverPlural = map[string]bool{
	"status": true, "bus": true, "gas": true, "lens": true, "alias": true,
	"atlas": true, "canvas": true, "chaos": true, "circus": true,
	"virus": true, "campus": true, "corpus": true, "census": true,
	"genus": true, "nexus": true, "bonus": true, "radius": true,
	"surplus": true, "syllabus": true, "analysis": true, "axis": true,
	"basis": true, "crisis": true, "diagnosis": true, "emphasis": true,
	"oasis": true, "bias": true, "access": true, "address": true,
	"process": true, "success": true,
}

// alwaysPlural are the uncountable/irregular nouns the plural check exempts
// (rules/resources.go uncountableNouns ∪ irregularPlurals) — singularizing
// them could not produce a violation, so the mutator skips them.
var alwaysPlural = map[string]bool{
	"equipment": true, "information": true, "metadata": true, "data": true,
	"feedback": true, "software": true, "hardware": true, "firmware": true,
	"staff": true, "traffic": true, "media": true, "news": true,
	"research": true, "advice": true, "evidence": true, "knowledge": true,
	"progress": true, "furniture": true, "luggage": true, "baggage": true,
	"series": true, "species": true, "aircraft": true, "fish": true,
	"sheep": true, "deer": true, "offspring": true,
	"people": true, "children": true, "men": true, "women": true,
	"feet": true, "teeth": true, "geese": true, "mice": true, "lice": true,
	"oxen": true, "leaves": true, "loaves": true, "thieves": true,
	"wives": true, "wolves": true, "shelves": true, "knives": true,
	"lives": true, "halves": true, "calves": true, "elves": true,
	"scarves": true,
}

// actionVerbs is rules/methods.go's R2xx lexicon — the action-shape tokens
// behind R2xx-05's predicate.
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

// grpcStandardVerbs are rules/versioning.go's AIP Standard Method prefixes,
// longest first; grpcActionVerbs its custom-method verb set. rename-rpc uses
// them to strip a leading verb and Fetch- it into a deliberate non-standard
// name ("Fetch" is in NEITHER set — the mutant cannot read as standard).
var grpcStandardVerbs = []string{
	"BatchCreate", "BatchDelete", "BatchGet", "BatchUpdate",
	"Get", "List", "Create", "Update", "Delete", "Search",
}

var grpcActionVerbs = map[string]bool{
	"activate": true, "approve": true, "archive": true, "assign": true,
	"cancel": true, "clone": true, "complete": true, "convert": true,
	"deactivate": true, "disable": true, "download": true, "enable": true,
	"execute": true, "export": true, "generate": true, "grant": true,
	"import": true, "invoke": true, "invite": true, "lock": true,
	"login": true, "logout": true, "merge": true, "migrate": true,
	"move": true, "notify": true, "preview": true, "process": true,
	"publish": true, "purge": true, "recalculate": true, "refresh": true,
	"register": true, "reject": true, "renew": true, "reset": true,
	"restore": true, "retry": true, "revoke": true, "send": true,
	"split": true, "submit": true, "sync": true, "transfer": true,
	"trigger": true, "undelete": true, "unlock": true, "unpublish": true,
	"upload": true, "validate": true, "verify": true,
}

// ---------------------------------------------------------------------------
// Source-shape helpers. The corpus fixtures are small, conventionally
// formatted Java/proto (every case ≤ 40 lines, §3.1 discipline); these
// helpers lean on that shape and refuse (report not-applicable) anything
// they cannot transform cleanly. No regexp cleverness that could silently
// corrupt syntax: every edit is a literal, anchored replacement.

// lineSlice splits src into lines, keeping them newline-free.
func lineSlice(src []byte) []string {
	return strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
}

// active reports whether a line carries live code (not a comment). The
// corpus's one commented-out mapping (_global/ok-commented-out-mapping) must
// never be mutated: a mutation of dead text would fake a miss.
func active(line string) bool {
	return !strings.HasPrefix(strings.TrimSpace(line), "//")
}

// joinLines is the inverse of lineSlice for edited sources.
func joinLines(ls []string) []byte {
	return []byte(strings.Join(ls, "\n"))
}

// firstActiveLine returns the index of the first active line containing sub.
func firstActiveLine(ls []string, sub string) (int, bool) {
	for i, l := range ls {
		if active(l) && strings.Contains(l, sub) {
			return i, true
		}
	}
	return -1, false
}

// mappingPath extracts the quoted path of an annotation line; ok=false when
// the line carries no string literal (bare `@GetMapping` under a class-level
// base).
func mappingPath(line string) (string, bool) {
	i := strings.Index(line, "\"")
	if i < 0 {
		return "", false
	}
	rest := line[i+1:]
	j := strings.Index(rest, "\"")
	if j < 0 {
		return "", false
	}
	return rest[:j], true
}

// classBase returns the class-level @RequestMapping path declared before
// mapIdx ("" when the controller has none).
func classBase(ls []string, mapIdx int) string {
	for i := mapIdx - 1; i >= 0; i-- {
		l := ls[i]
		if !strings.Contains(l, "@RequestMapping(") {
			continue
		}
		if p, ok := mappingPath(l); ok {
			return p
		}
	}
	return ""
}

// sigClose finds the first line at or after start whose content closes a
// handler parameter list with `) {`.
func sigClose(ls []string, start int) (int, bool) {
	for i := start; i < len(ls) && i <= start+6; i++ {
		if strings.Contains(ls[i], ") {") {
			return i, true
		}
	}
	return -1, false
}

// returnTypeOf reads a handler signature's return type: the token between
// `public` and the method name ("public Page<BookDto> search(" →
// "Page<BookDto>").
func returnTypeOf(sig string) string {
	i := strings.Index(sig, "(")
	if i < 0 {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(sig[:i]))
	if len(fields) < 2 {
		return ""
	}
	return fields[len(fields)-2]
}

// collectionMarkers are the response spellings the adapters treat as
// collections (rules/pagination.go collectionBases + the Spring Page/Slice
// envelope shapes). A verb-swap of a collection-returning GET could not
// produce a create-shaped POST, so those files are not applicable.
var collectionMarkers = []string{
	"Page<", "List<", "Set<", "Collection<", "Iterable<", "Stream<",
	"Slice<", "Deque<", "Queue<", "ArrayList<", "LinkedList<", "HashSet<",
	"LinkedHashSet<", "TreeSet<", "Flux<", "[]",
}

func collectionSpelling(t string) bool {
	for _, m := range collectionMarkers {
		if strings.Contains(t, m) {
			return true
		}
	}
	return false
}

// camelWords splits a lowerCamel identifier into its words ("createdAt" →
// [created, at], "userURL" → [user, URL]) — the shape camelWordsSplit in
// rules/pagination.go uses.
func camelWords(name string) []string {
	var words []string
	start := 0
	rs := []rune(name)
	for i := 1; i < len(rs); i++ {
		if rs[i] >= 'A' && rs[i] <= 'Z' {
			words = append(words, string(rs[start:i]))
			start = i
		}
	}
	words = append(words, string(rs[start:]))
	return words
}

// camelWordCount is len(camelWords(name)).
func camelWordCount(name string) int { return len(camelWords(name)) }

// snakeOf re-spells a camel identifier snake_case ("createdAt" →
// "created_at", "userURL" → "user_url").
func snakeOf(name string) string {
	parts := camelWords(name)
	for i, w := range parts {
		parts[i] = strings.ToLower(w)
	}
	return strings.Join(parts, "_")
}

// pathTokens splits a path segment into lower-case word tokens on
// separators and camelCase boundaries (rules/resources.go pathTokens).
func pathTokens(seg string) []string {
	var out []string
	for _, word := range strings.FieldsFunc(seg, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ' '
	}) {
		out = append(out, camelWords(word)...)
	}
	for i, t := range out {
		out[i] = strings.ToLower(t)
	}
	return out
}

// hasVerbToken reports whether a segment carries a CRUD-verb token.
func hasVerbToken(seg string) bool {
	for _, t := range pathTokens(seg) {
		if crudVerbs[t] {
			return true
		}
	}
	return false
}

// segments splits an absolute path into non-empty segments.
func segments(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// MUST mutators (test-strategy §4.1 rows 1–17).

// mutSingularizePath singularizes one plural collection segment:
// `/books` → `/book` (strategy §4.1 row 2, brief w4-02 name kept). The
// segment must be plural by the plain s-suffix reading — uncountable and
// irregular nouns are skipped because singularizing them cannot produce the
// violation.
var mutSingularizePath = &Mutator{
	Name: "singularize-path",
	Kind: "must",
	Description: "singularize one plural collection segment (/books → /book); " +
		"uncountable/irregular nouns are skipped (no violation possible)",
	Pairings: []Pairing{{PoolRule: "R1xx-01", Targets: []string{"R1xx-01"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "Mapping(\"") || !active(l) {
				continue
			}
			p, ok := mappingPath(l)
			if !ok {
				continue
			}
			segs := segments(p)
			for s, seg := range segs {
				if strings.Contains(seg, "{") || strings.Contains(seg, "*") {
					continue
				}
				low := strings.ToLower(seg)
				if !strings.HasSuffix(low, "s") || alwaysPlural[low] || neverPlural[low] {
					continue
				}
				segs[s] = seg[:len(seg)-1]
				newPath := "/" + strings.Join(segs, "/")
				ls[i] = strings.Replace(l, p, newPath, 1)
				return joinLines(ls), true
			}
		}
		return src, false
	},
}

// mutInjectVerbPath inserts a CRUD-verb segment before the first path
// variable (`/books/{id}` → `/books/delete-item/{id}`) — the strategy §4.1
// row 1 example, tripping R1xx-02 on the verb token and R1xx-03 on the
// variable that now sits under an action segment.
var mutInjectVerbPath = &Mutator{
	Name: "inject-verb-path",
	Kind: "must",
	Description: "insert the CRUD-verb segment delete-item before a path variable " +
		"(/books/{id} → /books/delete-item/{id})",
	Pairings: []Pairing{{PoolRule: "R1xx-02", Targets: []string{"R1xx-02", "R1xx-03"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		// Preferred shape: a path variable to put the verb segment in front of.
		for i, l := range ls {
			if !strings.Contains(l, "Mapping(\"") || !active(l) {
				continue
			}
			p, ok := mappingPath(l)
			if !ok || !strings.Contains(p, "{") {
				continue
			}
			if hasVerbToken(p) {
				continue // already verb-shaped — not ok material for this mutator
			}
			ls[i] = strings.Replace(l, "{", "delete-item/{", 1)
			return joinLines(ls), true
		}
		// Fallback: append the verb segment to a verb-free path (/updates →
		// /updates/delete-item).
		for i, l := range ls {
			if !strings.Contains(l, "Mapping(\"") || !active(l) {
				continue
			}
			p, ok := mappingPath(l)
			if !ok || p == "" {
				continue
			}
			if hasVerbToken(p) {
				continue
			}
			newPath := strings.TrimSuffix(p, "/") + "/delete-item"
			ls[i] = strings.Replace(l, p, newPath, 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutFlattenResourcePath replaces the literal segment in front of the last
// path variable with `find-by-id` (strategy §4.1 row 3's literal), so the
// variable ends up under an action segment — an R1xx-03 violation.
var mutFlattenResourcePath = &Mutator{
	Name: "flatten-resource-path",
	Kind: "must",
	Description: "replace the literal segment before the last path variable with " +
		"the action literal find-by-id (/libraries/{id} → /find-by-id/{id})",
	Pairings: []Pairing{{PoolRule: "R1xx-03", Targets: []string{"R1xx-03"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "Mapping(\"") || !active(l) {
				continue
			}
			p, ok := mappingPath(l)
			if !ok {
				continue
			}
			segs := segments(p)
			last := -1
			for s := len(segs) - 1; s >= 0; s-- {
				if strings.Contains(segs[s], "{") {
					last = s
					break
				}
			}
			if last <= 0 { // no variable, or it opens the path (already a violation)
				continue
			}
			prev := segs[last-1]
			if strings.Contains(prev, "{") || hasVerbToken(prev) {
				continue
			}
			segs[last-1] = "find-by-id"
			newPath := "/" + strings.Join(segs, "/")
			ls[i] = strings.Replace(l, p, newPath, 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// kebabSegment pins kebab-case literals (rules/resources.go
// literalSegmentPattern).
var kebabSegment = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// mutMisCasePath capitalizes the head of the first kebab-case literal
// segment (`audio-books` → `Audio-books`), breaking the casing convention
// R1xx-04 pins.
var mutMisCasePath = &Mutator{
	Name:        "mis-case-path",
	Kind:        "must",
	Description: "capitalize the first kebab-case literal segment (audio-books → Audio-books)",
	Pairings:    []Pairing{{PoolRule: "R1xx-04", Targets: []string{"R1xx-04"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "Mapping(\"") || !active(l) {
				continue
			}
			p, ok := mappingPath(l)
			if !ok {
				continue
			}
			segs := segments(p)
			for s, seg := range segs {
				if strings.Contains(seg, "{") || strings.Contains(seg, "*") {
					continue
				}
				if !kebabSegment.MatchString(seg) {
					continue
				}
				mutated := strings.ToUpper(seg[:1]) + seg[1:]
				segs[s] = mutated
				newPath := "/" + strings.Join(segs, "/")
				ls[i] = strings.Replace(l, p, newPath, 1)
				return joinLines(ls), true
			}
		}
		return src, false
	},
}

// insertBodyParam adds `, @RequestBody String body` to the first active
// handler of the given mapping annotation. Shared by body-into-get and
// body-into-delete; String needs no import and binds as a request body.
func insertBodyParam(src []byte, mapping string) ([]byte, bool) {
	ls := lineSlice(src)
	mapIdx, ok := firstActiveLine(ls, mapping)
	if !ok {
		return src, false
	}
	closeIdx, ok := sigClose(ls, mapIdx)
	if !ok {
		return src, false
	}
	for i := mapIdx; i <= closeIdx; i++ {
		if strings.Contains(ls[i], "@RequestBody") {
			return src, false // already has a body — not ok material
		}
	}
	ls[closeIdx] = strings.Replace(ls[closeIdx], ") {", ", @RequestBody String body) {", 1)
	return joinLines(ls), true
}

// mutBodyIntoGet gives a GET a request body (strategy §4.1 row 5) — the
// measured R2xx-01 violation.
var mutBodyIntoGet = &Mutator{
	Name:        "body-into-get",
	Kind:        "must",
	Description: "add a @RequestBody String parameter to the GET handler",
	Pairings:    []Pairing{{PoolRule: "R2xx-01", Targets: []string{"R2xx-01"}}},
	Apply:       func(src []byte) ([]byte, bool) { return insertBodyParam(src, "@GetMapping") },
}

// mutBodyIntoDelete gives a DELETE a request body (strategy §4.1 row 8) —
// the R2xx-04 violation.
var mutBodyIntoDelete = &Mutator{
	Name:        "body-into-delete",
	Kind:        "must",
	Description: "add a @RequestBody String parameter to the DELETE handler",
	Pairings:    []Pairing{{PoolRule: "R2xx-04", Targets: []string{"R2xx-04"}}},
	Apply:       func(src []byte) ([]byte, bool) { return insertBodyParam(src, "@DeleteMapping") },
}

// mutVerbSwap flips the first applicable GET read into a POST create without
// 201 (strategy §4.1 row 6, brief w4-02 name kept). Applicability is the
// well-formedness of the mutant: the swapped endpoint must READ as a create
// for the target rules — a literal final path segment and a non-collection
// return. A POST carrying a collection or an item template is not an
// R2xx-02/R5xx-03 violation, so mutating such a file would fabricate a miss.
var mutVerbSwap = &Mutator{
	Name: "verb-swap",
	Kind: "must",
	Description: "@GetMapping → @PostMapping on a read that becomes a create-shaped " +
		"POST without 201 (literal final segment, single response)",
	Pairings: []Pairing{{PoolRule: "*", Targets: []string{"R2xx-02", "R5xx-03"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		mapIdx, ok := firstActiveLine(ls, "@GetMapping")
		if !ok || strings.Contains(ls[mapIdx], "@GetMapping(") &&
			func() bool { p, ok := mappingPath(ls[mapIdx]); return ok && p == "" }() {
			return src, false
		}
		closeIdx, ok := sigClose(ls, mapIdx)
		if !ok {
			return src, false
		}
		if returnTypeOf(ls[closeIdx]) == "" || collectionSpelling(returnTypeOf(ls[closeIdx])) {
			return src, false
		}
		p := classBase(ls, mapIdx)
		if mp, ok := mappingPath(ls[mapIdx]); ok {
			p = mp
		}
		segs := segments(p)
		if len(segs) == 0 {
			return src, false
		}
		final := segs[len(segs)-1]
		if strings.Contains(final, "{") || strings.Contains(final, ":") || hasVerbToken(final) {
			return src, false
		}
		for _, t := range pathTokens(final) {
			if actionVerbs[t] {
				return src, false // action path — R2xx-05 territory, not a create
			}
		}
		ls[mapIdx] = strings.Replace(ls[mapIdx], "@GetMapping", "@PostMapping", 1)
		return joinLines(ls), true
	},
}

// mutRemoveStatusMapping drops the handler's @ResponseStatus (strategy §4.1
// row 7, brief w4-02 name kept): a create-shaped POST answering the implicit
// 200. Applicability requires the POST path to be create-shaped by the R5xx
// family's own definition (literal final segment, no AIP-136 colon, no
// action word) — elsewhere the mutation is a no-op by design.
var mutRemoveStatusMapping = &Mutator{
	Name:        "remove-status-mapping",
	Kind:        "must",
	Description: "remove @ResponseStatus from a create-shaped POST (declared 201/202 → implicit 200)",
	Pairings: []Pairing{
		{PoolRule: "R2xx-02", Targets: []string{"R2xx-02", "R5xx-03"}},
		{PoolRule: "R5xx-03", Targets: []string{"R5xx-03", "R2xx-02"}},
	},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		postIdx, ok := firstActiveLine(ls, "@PostMapping")
		if !ok {
			return src, false
		}
		p := classBase(ls, postIdx)
		if mp, ok := mappingPath(ls[postIdx]); ok {
			p = mp
		}
		segs := segments(p)
		if len(segs) == 0 {
			return src, false
		}
		final := segs[len(segs)-1]
		if strings.Contains(final, "{") || strings.Contains(final, ":") || hasVerbToken(final) {
			return src, false
		}
		for _, t := range pathTokens(final) {
			if actionVerbs[t] {
				return src, false
			}
		}
		statusIdx, ok := firstActiveLine(ls, "@ResponseStatus(")
		if !ok {
			return src, false
		}
		return joinLines(append(ls[:statusIdx], ls[statusIdx+1:]...)), true
	},
}

// mutSwapPutPatch flips @PatchMapping ↔ @PutMapping (strategy §4.1 row 9).
// The two pairings mirror each other: mutating a PUT whose payload equals
// its response turns it into the full-replacement PATCH R2xx-03 flags, and
// mutating a PATCH whose payload differs from its response turns it into the
// partial PUT R2xx-06 flags. Handlers without a declared response type are
// skipped — both rules stay silent on void responses by design.
var mutSwapPutPatch = &Mutator{
	Name: "swap-put-patch",
	Kind: "must",
	Description: "@PatchMapping ↔ @PutMapping on handlers with a declared response " +
		"(PUT-full → PATCH-full, PATCH-partial → PUT-partial)",
	Pairings: []Pairing{
		{PoolRule: "R2xx-06", Targets: []string{"R2xx-03"}},
		{PoolRule: "R2xx-03", Targets: []string{"R2xx-06"}},
	},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		patchIdx, patchOK := firstActiveLine(ls, "@PatchMapping")
		putIdx, putOK := firstActiveLine(ls, "@PutMapping")
		switch {
		case patchOK:
			closeIdx, ok := sigClose(ls, patchIdx)
			if !ok || returnTypeOf(ls[closeIdx]) == "" || returnTypeOf(ls[closeIdx]) == "void" {
				return src, false
			}
			ls[patchIdx] = strings.Replace(ls[patchIdx], "@PatchMapping", "@PutMapping", 1)
			return joinLines(ls), true
		case putOK:
			closeIdx, ok := sigClose(ls, putIdx)
			if !ok || returnTypeOf(ls[closeIdx]) == "" || returnTypeOf(ls[closeIdx]) == "void" {
				return src, false
			}
			ls[putIdx] = strings.Replace(ls[putIdx], "@PutMapping", "@PatchMapping", 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutDropPaginationParam removes the pagination input (Pageable or
// page/size/limit query parameters) from a list endpoint (strategy §4.1
// row 10, brief w4-02 name kept), leaving an unpaginated collection — the
// R3xx-01 violation.
var mutDropPaginationParam = &Mutator{
	Name: "drop-pagination-param",
	Kind: "must",
	Description: "remove the Pageable argument or the page/size/limit query parameters " +
		"from a list endpoint",
	Pairings: []Pairing{{PoolRule: "R3xx-01", Targets: []string{"R3xx-01"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		mapIdx, ok := firstActiveLine(ls, "@GetMapping")
		if !ok {
			return src, false
		}
		closeIdx, ok := sigClose(ls, mapIdx)
		if !ok {
			return src, false
		}
		for i := mapIdx; i <= closeIdx; i++ {
			line := ls[i]
			if !strings.Contains(line, "(") || !strings.Contains(line, ") {") {
				continue
			}
			open := strings.Index(line, "(")
			head, params := line[:open+1], line[open+1:strings.Index(line, ") {")]
			parts := strings.Split(params, ",")
			kept := make([]string, 0, len(parts))
			dropped := false
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if dropPaginationParam(p) {
					dropped = true
					continue
				}
				kept = append(kept, p)
			}
			if !dropped {
				continue
			}
			ls[i] = head + strings.Join(kept, ", ") + ") {"
			return joinLines(ls), true
		}
		return src, false
	},
}

// dropPaginationParam reports whether one parameter carries pagination
// input: a Pageable argument or a query parameter whose name is a
// page/size/limit/offset word (rules/pagination.go's sizeCarryingName plus
// page/offset).
func dropPaginationParam(param string) bool {
	if strings.Contains(param, "Pageable") {
		return true
	}
	name := param
	if i := strings.LastIndex(name, " "); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimPrefix(name, `"`) // @RequestParam("limit") — the name is inside
	switch strings.ToLower(name) {
	case "page", "size", "limit", "offset":
		return true
	}
	return false
}

// mutSplitPaginationStyle switches the SECOND list endpoint's Page<> wrapper
// to List<> (strategy §4.1 row 11) — one file, two pagination styles, the
// inconsistency R3xx-04 exists to flag. R3xx-04 is chartered but not in the
// v0.1 registry (w4-01 §7.1): the pairing is a pre-declared expected miss.
var mutSplitPaginationStyle = &Mutator{
	Name:        "split-pagination-style",
	Kind:        "must",
	Description: "switch the second list endpoint from Page<> to List<> (style split)",
	Pairings:    []Pairing{{PoolRule: "R3xx-04", Targets: []string{"R3xx-04"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		seen := 0
		for i, l := range ls {
			if !strings.Contains(l, "@GetMapping") || !active(l) {
				continue
			}
			closeIdx, ok := sigClose(ls, i)
			if !ok {
				continue
			}
			t := returnTypeOf(ls[closeIdx])
			if !strings.HasPrefix(t, "Page<") {
				continue
			}
			seen++
			if seen < 2 {
				continue
			}
			ls[closeIdx] = strings.Replace(ls[closeIdx], "Page<", "List<", 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutEntityIntoResponse rewires the handler to return the file's entity
// (strategy §4.1 row 12, brief w4-02 name kept) — the strong R4xx-01 signal
// (an entity the scan itself saw, sole in the surface). A file whose entity
// name is shadowed by a same-named DTO is skipped: the rule stays silent
// there by documented policy, so a mutation could only fake a miss.
var mutEntityIntoResponse = &Mutator{
	Name:        "entity-into-response",
	Kind:        "must",
	Description: "return the @Entity type instead of the DTO from the first handler",
	Pairings:    []Pairing{{PoolRule: "R4xx-01", Targets: []string{"R4xx-01"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		entity, ok := soleEntityName(ls)
		if !ok {
			return src, false
		}
		mapIdx, ok := firstActiveLine(ls, "@GetMapping")
		if !ok {
			return src, false
		}
		closeIdx, ok := sigClose(ls, mapIdx)
		if !ok {
			return src, false
		}
		sig := ls[closeIdx]
		t := returnTypeOf(sig)
		if t == "" || t == entity {
			return src, false
		}
		ls[closeIdx] = strings.Replace(sig, " "+t+" ", " "+entity+" ", 1)
		return joinLines(ls), true
	},
}

// soleEntityName returns the simple name of the file's @Entity class when
// exactly one such declaration exists (no same-named record/interface
// shadowing it).
func soleEntityName(ls []string) (string, bool) {
	for i, l := range ls {
		if !strings.Contains(l, "@Entity") || !active(l) {
			continue
		}
		for j := i + 1; j <= i+2 && j < len(ls); j++ {
			name, ok := declaredTypeName(ls[j])
			if !ok {
				continue
			}
			pattern := regexp.MustCompile(`\b(class|record|interface)\s+` + regexp.QuoteMeta(name) + `\b`)
			count := 0
			for _, other := range ls {
				if pattern.MatchString(other) {
					count++
				}
			}
			if count == 1 {
				return name, true
			}
			return "", false // shadowed or duplicated — the rule's blind spot by policy
		}
	}
	return "", false
}

// declaredTypeName reads the simple name from a type declaration line.
func declaredTypeName(line string) (string, bool) {
	m := regexp.MustCompile(`\b(class|record|interface)\s+([A-Za-z_][A-Za-z0-9_]*)`).FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[2], true
}

// mutSnakeCaseField re-spells one multi-word camel DTO field as snake_case
// (strategy §4.1 row 13, brief w4-02 name kept). An explicit @JsonProperty
// key is what reaches the wire, so when present it is the name mutated.
var mutSnakeCaseField = &Mutator{
	Name:        "snake-case-field",
	Kind:        "must",
	Description: "re-spell a multi-word camel DTO field (or its @JsonProperty key) as snake_case",
	Pairings:    []Pairing{{PoolRule: "R4xx-02", Targets: []string{"R4xx-02"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !active(l) {
				continue
			}
			if m := regexp.MustCompile(`@JsonProperty\("([A-Za-z0-9]+)"\)`).FindStringSubmatch(l); m != nil {
				if camelWordCount(m[1]) >= 2 {
					ls[i] = strings.Replace(l, m[1], snakeOf(m[1]), 1)
					return joinLines(ls), true
				}
			}
		}
		for i, l := range ls {
			if !strings.Contains(l, "record ") || !active(l) {
				continue
			}
			open := strings.Index(l, "(")
			closeParen := strings.LastIndex(l, ")")
			if open < 0 || closeParen <= open {
				continue
			}
			for _, f := range strings.Split(l[open+1:closeParen], ",") {
				f = strings.TrimSpace(f)
				if f == "" {
					continue
				}
				fields := strings.Fields(f)
				name := fields[len(fields)-1]
				if strings.Contains(name, "\"") || camelWordCount(name) < 2 {
					continue
				}
				ls[i] = strings.Replace(l, name, snakeOf(name), 1)
				return joinLines(ls), true
			}
		}
		return src, false
	},
}

// mutErrorMapHandler switches the first handler of a dominant error envelope
// to a raw Map (strategy §4.1 row 14) — the deviation-from-consensus R5xx-01
// flags. Files without a dominant envelope (majorityEnvelope's own
// precondition) are skipped: mutating them cannot produce the violation.
var mutErrorMapHandler = &Mutator{
	Name: "error-map-handler",
	Kind: "must",
	Description: "answer the first @ExceptionHandler with Map<String, Object> while " +
		"the majority keeps the envelope",
	Pairings: []Pairing{{PoolRule: "R5xx-01", Targets: []string{"R5xx-01"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		handlers := handlerReturns(ls)
		majority, n := dominantSpelling(handlers)
		if majority == "" || n < 2 {
			return src, false
		}
		for i, l := range ls {
			if !strings.Contains(l, "@ExceptionHandler(") || !active(l) {
				continue
			}
			closeIdx, ok := sigClose(ls, i)
			if !ok || returnTypeOf(ls[closeIdx]) != majority {
				continue
			}
			ls[closeIdx] = strings.Replace(ls[closeIdx], " "+majority+" ", " Map<String, Object> ", 1)
			for j := closeIdx + 1; j < len(ls); j++ {
				if strings.Contains(ls[j], "return new "+majority+"(") {
					ls[j] = regexp.MustCompile(`return new `+regexp.QuoteMeta(majority)+`\(`).
						ReplaceAllString(ls[j], `return Map.of("status", `)
					return joinLines(ls), true
				}
			}
			return src, false
		}
		return src, false
	},
}

// handlerReturns lists the return-type spellings of every active
// @ExceptionHandler method, in file order.
func handlerReturns(ls []string) []string {
	var out []string
	for i, l := range ls {
		if !strings.Contains(l, "@ExceptionHandler(") || !active(l) {
			continue
		}
		closeIdx, ok := sigClose(ls, i)
		if !ok {
			continue
		}
		if t := returnTypeOf(ls[closeIdx]); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// dominantSpelling returns the most frequent spelling and its count (tie →
// "", mirroring the strict-dominance rule of rules/errors.go
// majorityEnvelope).
func dominantSpelling(spellings []string) (string, int) {
	counts := map[string]int{}
	order := []string{}
	for _, s := range spellings {
		if counts[s] == 0 {
			order = append(order, s)
		}
		counts[s]++
	}
	best, bestN, tie := "", 0, false
	for _, s := range order {
		switch {
		case counts[s] > bestN:
			best, bestN, tie = s, counts[s], false
		case counts[s] == bestN:
			tie = true
		}
	}
	if tie {
		return "", 0
	}
	return best, bestN
}

// mutBusinessException500 remaps an app-owned business exception's handler
// to 500 (strategy §4.1 row 15) — the R5xx-02 violation. Handlers already at
// 500 and exceptions without the Exception suffix are skipped (no-op or
// out-of-scope by the rule's own FP gates).
var mutBusinessException500 = &Mutator{
	Name:        "business-exception-500",
	Kind:        "must",
	Description: "remap an app-owned *Exception handler from 4xx to 500",
	Pairings:    []Pairing{{PoolRule: "R5xx-02", Targets: []string{"R5xx-02"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "@ExceptionHandler(") || !active(l) {
				continue
			}
			m := regexp.MustCompile(`@ExceptionHandler\(([A-Za-z0-9_.]+)\.class\)`).FindStringSubmatch(l)
			if m == nil || !strings.HasSuffix(m[1], "Exception") {
				continue
			}
			for j := i + 1; j <= i+3 && j < len(ls); j++ {
				if !strings.Contains(ls[j], "@ResponseStatus(") {
					continue
				}
				if strings.Contains(ls[j], "INTERNAL_SERVER_ERROR") {
					break // already 500 — nothing to mutate
				}
				ls[j] = regexp.MustCompile(`HttpStatus\.[A-Z_]+`).
					ReplaceAllString(ls[j], "HttpStatus.INTERNAL_SERVER_ERROR")
				return joinLines(ls), true
			}
		}
		return src, false
	},
}

// mutAddSecondAdvice appends a second @ControllerAdvice class (strategy
// §4.1 row 16) — the multiple-advice shape R5xx-04 exists to flag. R5xx-04
// is chartered but unregistered in v0.1 (w4-01 §7.1): pre-declared miss.
var mutAddSecondAdvice = &Mutator{
	Name:        "add-second-advice",
	Kind:        "must",
	Description: "append a second @ControllerAdvice answering a raw Map envelope",
	Pairings:    []Pairing{{PoolRule: "R5xx-04", Targets: []string{"R5xx-04"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		if _, ok := firstActiveLine(ls, "@ControllerAdvice"); !ok {
			return src, false
		}
		extra := []string{
			"",
			"@ControllerAdvice",
			"class SecondAdvice {",
			"",
			"    @ExceptionHandler(NullPointerException.class)",
			"    @ResponseStatus(HttpStatus.BAD_REQUEST)",
			"    Map<String, Object> teapot(NullPointerException ex) {",
			"        return Map.of(\"status\", 400, \"detail\", ex.getMessage());",
			"    }",
			"}",
			"",
		}
		return joinLines(append(ls, extra...)), true
	},
}

// mutUnversionedPath strips the /vN segment from the class-level base path
// (strategy §4.1 row 17, brief w4-02 name kept) — the R6xx-01 violation.
// R6xx-01 is registered but default-off (charter A-3): its pairing runs on
// the corpus's enable-r6xx-01 config pass.
var mutUnversionedPath = &Mutator{
	Name:        "unversioned-path",
	Kind:        "must",
	Description: "remove the /vN segment from the class-level @RequestMapping base path",
	Pairings:    []Pairing{{PoolRule: "R6xx-01", Targets: []string{"R6xx-01"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "@RequestMapping(\"") || !active(l) {
				continue
			}
			p, ok := mappingPath(l)
			if !ok {
				continue
			}
			m := regexp.MustCompile(`/v[0-9]+`).FindString(p)
			if m == "" {
				continue
			}
			ls[i] = strings.Replace(l, m, "", 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// ---------------------------------------------------------------------------
// OPTIONAL mutators (test-strategy §4.1, the 8 INFO-rule rows).

// mutMixIdField renames one reference id to the other convention
// (publisherId → publisherID) so one type mixes spellings — the R1xx-05
// violation. Needs two reference-id fields to start from (a mix requires
// them).
var mutMixIdField = &Mutator{
	Name:        "mix-id-field",
	Kind:        "optional",
	Description: "rename one <resource>Id reference field to the <resource>ID spelling",
	Pairings:    []Pairing{{PoolRule: "R1xx-05", Targets: []string{"R1xx-05"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "record ") || !active(l) {
				continue
			}
			open := strings.Index(l, "(")
			closeParen := strings.LastIndex(l, ")")
			if open < 0 || closeParen <= open {
				continue
			}
			var idFields []string
			for _, f := range strings.Split(l[open+1:closeParen], ",") {
				f = strings.TrimSpace(f)
				if f == "" {
					continue
				}
				fields := strings.Fields(f)
				name := fields[len(fields)-1]
				if strings.HasSuffix(name, "Id") && len(name) > 2 && name != "Id" {
					idFields = append(idFields, name)
				}
			}
			if len(idFields) < 2 {
				continue
			}
			victim := idFields[len(idFields)-1]
			ls[i] = strings.Replace(l, victim, victim[:len(victim)-2]+"ID", 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutActionGetSwap flips an action POST to GET (strategy §4.1 OPTIONAL) —
// an action path served by a read verb, R2xx-05's subject. The corpus's
// action fixtures on custom-method colon paths (AIP-136) are skipped: their
// action lives inside the template, where the rule deliberately stays
// silent.
var mutActionGetSwap = &Mutator{
	Name:        "action-get-swap",
	Kind:        "optional",
	Description: "@PostMapping → @GetMapping on a literal action path (/books/search)",
	Pairings:    []Pairing{{PoolRule: "R2xx-05", Targets: []string{"R2xx-05"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "@PostMapping") || !active(l) {
				continue
			}
			p := classBase(ls, i)
			if mp, ok := mappingPath(l); ok {
				p = mp
			}
			if strings.Contains(p, "{") || strings.Contains(p, ":") {
				continue
			}
			action := false
			for _, seg := range segments(p) {
				for _, t := range pathTokens(seg) {
					if actionVerbs[t] {
						action = true
					}
				}
			}
			if !action {
				continue
			}
			ls[i] = strings.Replace(l, "@PostMapping", "@GetMapping", 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutUnwrapEnvelope swaps a wrapped collection response for a bare List
// (strategy §4.1 OPTIONAL) — the R3xx-02 violation.
var mutUnwrapEnvelope = &Mutator{
	Name:        "unwrap-envelope",
	Kind:        "optional",
	Description: "replace the list endpoint's envelope (Page<>/PageResponse<>) with a bare List<>",
	Pairings:    []Pairing{{PoolRule: "R3xx-02", Targets: []string{"R3xx-02"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		mapIdx, ok := firstActiveLine(ls, "@GetMapping")
		if !ok {
			return src, false
		}
		closeIdx, ok := sigClose(ls, mapIdx)
		if !ok {
			return src, false
		}
		sig := ls[closeIdx]
		t := returnTypeOf(sig)
		if t == "" || !strings.Contains(t, "<") || collectionSpelling(t) && strings.HasPrefix(t, "List<") {
			return src, false
		}
		base := t[:strings.Index(t, "<")]
		if base == "List" {
			return src, false // already bare
		}
		ls[closeIdx] = strings.Replace(sig, " "+base+"<", " List<", 1)
		return joinLines(ls), true
	},
}

// mutRemoveSizeCap strips the @Max cap from a size/limit parameter (strategy
// §4.1 OPTIONAL) — the unbounded page R3xx-03 flags.
var mutRemoveSizeCap = &Mutator{
	Name:        "remove-size-cap",
	Kind:        "optional",
	Description: "remove the @Max annotation from a size/limit query parameter",
	Pairings:    []Pairing{{PoolRule: "R3xx-03", Targets: []string{"R3xx-03"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "@Max(") || !active(l) {
				continue
			}
			mutated := regexp.MustCompile(`@Max\([^)]*\)\s*`).ReplaceAllString(l, "")
			if mutated == l {
				continue
			}
			ls[i] = mutated
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutTimeFieldToString retypes a standard time field to String (strategy
// §4.1 OPTIONAL) — the non-RFC3339 wire format R4xx-03 flags.
var mutTimeFieldToString = &Mutator{
	Name:        "time-field-to-string",
	Kind:        "optional",
	Description: "retype a DTO time field (Instant/OffsetDateTime/ZonedDateTime/LocalDate) to String",
	Pairings:    []Pairing{{PoolRule: "R4xx-03", Targets: []string{"R4xx-03"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "record ") || !active(l) {
				continue
			}
			for _, t := range []string{"Instant", "OffsetDateTime", "ZonedDateTime", "LocalDate"} {
				needle := " " + t + " "
				if strings.Contains(l, needle) {
					ls[i] = strings.Replace(l, needle, " String ", 1)
					return joinLines(ls), true
				}
			}
		}
		return src, false
	},
}

// mutDtoSuffixMix strips the decoration suffix from one of two payload types
// (strategy §4.1 OPTIONAL) — the mixed-convention shape R4xx-04 exists to
// flag. Needs two suffixed types (a mix requires them). R4xx-04 is
// chartered but unregistered in v0.1: pre-declared miss.
var mutDtoSuffixMix = &Mutator{
	Name:        "dto-suffix-mix",
	Kind:        "optional",
	Description: "strip the Dto/Request/Response/Record suffix from one of two payload types",
	Pairings:    []Pairing{{PoolRule: "R4xx-04", Targets: []string{"R4xx-04"}}},
	Apply: func(src []byte) ([]byte, bool) {
		suffixes := []string{"Dto", "DTO", "Request", "Response", "Record", "Payload", "Summary", "Digest", "Model"}
		ls := lineSlice(src)
		suffixed := 0
		first := ""
		for _, l := range ls {
			name, ok := declaredTypeName(l)
			if !ok || !active(l) {
				continue
			}
			for _, suf := range suffixes {
				if strings.HasSuffix(name, suf) && len(name) > len(suf) {
					suffixed++
					if first == "" {
						first = name
					}
					break
				}
			}
		}
		if suffixed < 2 || first == "" {
			return src, false
		}
		changed := false
		for i, l := range ls {
			if strings.Contains(l, first) {
				ls[i] = strings.ReplaceAll(l, first, strings.TrimSuffix(first, dtoSuffixOf(first, suffixes)))
				changed = true
			}
		}
		if !changed {
			return src, false
		}
		return joinLines(ls), true
	},
}

func dtoSuffixOf(name string, suffixes []string) string {
	for _, suf := range suffixes {
		if strings.HasSuffix(name, suf) {
			return suf
		}
	}
	return ""
}

// mutRenameRpcNonstandard renames the first rpc to a deliberate
// non-standard, non-VerbNoun name (strategy §4.1 OPTIONAL) — the R6xx-02
// violation. "Fetch" is deliberately absent from both the Standard Method
// and the custom-method verb sets.
var mutRenameRpcNonstandard = &Mutator{
	Name:        "rename-rpc-nonstandard",
	Kind:        "optional",
	Description: "rename the first rpc to a Fetch-prefixed non-standard name",
	Pairings:    []Pairing{{PoolRule: "R6xx-02", Targets: []string{"R6xx-02"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			trimmed := strings.TrimSpace(l)
			if !strings.HasPrefix(trimmed, "rpc ") || !active(l) {
				continue
			}
			name := strings.TrimPrefix(trimmed, "rpc ")
			end := strings.IndexAny(name, "( ")
			if end <= 0 {
				continue
			}
			name = name[:end]
			rest := ""
			for _, verb := range grpcStandardVerbs {
				if strings.HasPrefix(name, verb) {
					rest = name[len(verb):]
					break
				}
			}
			if rest == "" {
				words := camelWords(name)
				if len(words) > 1 && grpcActionVerbs[strings.ToLower(words[0])] {
					rest = name[len(words[0]):]
				}
			}
			if rest == "" || rest == name {
				rest = name // bare noun: Fetch- prefix still breaks the convention
			}
			ls[i] = strings.Replace(l, "rpc "+name+"(", "rpc Fetch"+rest+"(", 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutBumpOnePathVersion bumps one endpoint's /vN while siblings keep theirs
// (strategy §4.1 OPTIONAL) — the version mix R6xx-03 exists to flag.
// R6xx-03 is chartered but unregistered in v0.1: pre-declared miss.
var mutBumpOnePathVersion = &Mutator{
	Name:        "bump-one-path-version",
	Kind:        "optional",
	Description: "bump one mapping path's /vN to /v(N+1) while the rest keep theirs",
	Pairings:    []Pairing{{PoolRule: "R6xx-03", Targets: []string{"R6xx-03"}}},
	Apply: func(src []byte) ([]byte, bool) {
		ls := lineSlice(src)
		for i, l := range ls {
			if !strings.Contains(l, "Mapping(\"") || !active(l) {
				continue
			}
			p, ok := mappingPath(l)
			if !ok {
				continue
			}
			m := regexp.MustCompile(`/v([0-9]+)/`).FindStringSubmatch(p)
			if m == nil {
				continue
			}
			n, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			ls[i] = strings.Replace(l, m[0], "/v"+strconv.Itoa(n+1)+"/", 1)
			return joinLines(ls), true
		}
		return src, false
	},
}

// mutators is the declared catalog, strategy §4.1 order: the 17 MUST rows
// first, then the 8 OPTIONAL rows. Order is fixed — no registry iteration,
// no map ordering — so two runs build the same case list.
var mutators = []*Mutator{
	mutInjectVerbPath,       // 1
	mutSingularizePath,      // 2
	mutFlattenResourcePath,  // 3
	mutMisCasePath,          // 4
	mutBodyIntoGet,          // 5
	mutVerbSwap,             // 6
	mutRemoveStatusMapping,  // 7
	mutBodyIntoDelete,       // 8
	mutSwapPutPatch,         // 9
	mutDropPaginationParam,  // 10
	mutSplitPaginationStyle, // 11
	mutEntityIntoResponse,   // 12
	mutSnakeCaseField,       // 13
	mutErrorMapHandler,      // 14
	mutBusinessException500, // 15
	mutAddSecondAdvice,      // 16
	mutUnversionedPath,      // 17
	mutMixIdField,           // O1
	mutActionGetSwap,        // O2
	mutUnwrapEnvelope,       // O3
	mutRemoveSizeCap,        // O4
	mutTimeFieldToString,    // O5
	mutDtoSuffixMix,         // O6
	mutRenameRpcNonstandard, // O7
	mutBumpOnePathVersion,   // O8
}

// unregisteredRules are the chartered rules missing from the v0.1 registry
// (w4-01 report §7.1). A pairing whose targets all live here cannot catch
// anything by construction: its cases are declared expected-miss BEFORE any
// run and reported as honored when they miss — the corpus-check strict vs
// declared discipline, carried over to mutation.
var unregisteredRules = map[string]bool{
	"R3xx-04": true,
	"R4xx-04": true,
	"R5xx-04": true,
	"R6xx-03": true,
}

// expectedMissReason is the pre-declared reason recorded for honored
// expected-miss cases.
const expectedMissReason = "target rule not registered in the v0.1 rule registry (w4-01 report §7.1) — cannot fire by construction"
