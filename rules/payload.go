package rules

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// The R4xx payload family (w3-05, charter §3.4, AIP-121/140/142): the data
// the API actually puts on the wire. Every check reads only the IR —
// Method.Payload/Response TypeRefs and Types[].Fields — never a source
// file. Heuristic boundaries are documented per rule in docs/rules/ and
// pinned by tests: R4xx-01 fires only on the strong IR signal (an entity
// the scan itself saw), never on a bare name match.

//go:embed data/R4xx-01.yaml
var noEntityInPayloadYAML []byte

//go:embed data/R4xx-02.yaml
var fieldCasingYAML []byte

//go:embed data/R4xx-03.yaml
var timeFieldStandardYAML []byte

// baseTypeName strips generic arguments and package qualification:
// "java.util.List<com.x.Youth>" → "List", "OrderResponse[]" → "OrderResponse".
func baseTypeName(name string) string {
	if i := strings.IndexByte(name, '<'); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSuffix(name, "[]")
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// genericArgs returns the top-level type arguments of a generic spelling:
// "Map<String,Youth>" → ["String", "Youth"], "Page<List<Youth>>" →
// ["List<Youth>"]. Non-generic names yield nil.
func genericArgs(name string) []string {
	start := strings.IndexByte(name, '<')
	end := strings.LastIndexByte(name, '>')
	if start < 0 || end <= start {
		return nil
	}
	var (
		out   []string
		depth int
		last  = start
	)
	for i := start + 1; i < end; i++ {
		switch name[i] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(name[last+1:i]))
				last = i
			}
		}
	}
	out = append(out, strings.TrimSpace(name[last+1:end]))
	return out
}

// entityIndex classifies the scanned Types for R4xx-01. soleEntity holds
// name → package only when exactly ONE scanned type carries the simple
// name and it is an entity — the strong, unambiguous signal. A name shared
// by several scanned types (an entity and a same-name DTO, or several
// entities) is shadowed: the adapter resolves references first-declaration
// wins (documented w3-02 policy), so the rule cannot tell which type a
// ref names and stays silent instead of guessing.
type entityIndex struct {
	soleEntity map[string]string
}

func buildEntityIndex(surface *ir.ApiSurface) entityIndex {
	idx := entityIndex{soleEntity: map[string]string{}}
	counts := map[string]int{}
	entityPkg := map[string]string{}
	for i := range surface.Types {
		t := &surface.Types[i]
		counts[t.Name]++
		if t.IsEntity {
			entityPkg[t.Name] = t.Package
		}
	}
	for name := range counts {
		if counts[name] == 1 {
			if pkg, is := entityPkg[name]; is {
				idx.soleEntity[name] = pkg
			}
		}
	}
	return idx
}

// entityRef reports the entity a top-level TypeRef names, if any. The
// package must agree when the adapter resolved it (the acceptance: a DTO
// with the same simple name in a clearly different package never fires);
// an unresolved package falls back to the single in-scope candidate.
func (idx entityIndex) entityRef(ref ir.TypeRef) (string, bool) {
	name := baseTypeName(ref.Name)
	pkg, ok := idx.soleEntity[name]
	if !ok {
		return "", false
	}
	if ref.Package != "" && ref.Package != pkg {
		return "", false
	}
	return name, true
}

// entityArgNames lists the entity names exposed inside a generic
// spelling's arguments ("Page<Youth>" → [Youth]). Arguments carry no
// package in the IR, so only the unambiguous sole-entity names qualify.
func (idx entityIndex) entityArgNames(name string) []string {
	var out []string
	for _, a := range genericArgs(name) {
		base := baseTypeName(a)
		if base == "" {
			continue
		}
		if _, ok := idx.soleEntity[base]; ok {
			out = append(out, base)
		}
		out = append(out, idx.entityArgNames(a)...)
	}
	return out
}

// noEntityInPayloadCheck is R4xx-01: one finding per method that puts an
// entity into the payload or the response (top level or generic argument).
func noEntityInPayloadCheck(ctx *engine.LintContext, node ir.Node) []engine.Finding {
	m, ok := node.(ir.Method)
	if !ok || ctx.Surface == nil {
		return nil
	}
	idx := buildEntityIndex(ctx.Surface)
	var exposed []string
	if m.Payload != nil {
		if name, ok := idx.entityRef(*m.Payload); ok {
			exposed = append(exposed, name)
		}
		exposed = append(exposed, idx.entityArgNames(m.Payload.Name)...)
	}
	if m.Response.Type.Name != "" {
		if name, ok := idx.entityRef(m.Response.Type); ok {
			exposed = append(exposed, name)
		}
		exposed = append(exposed, idx.entityArgNames(m.Response.Type.Name)...)
	}
	exposed = dedupe(exposed)
	if len(exposed) == 0 {
		return nil
	}
	return []engine.Finding{{
		Message: fmt.Sprintf("%s %s exposes the ORM entity %s — the database schema and lazy relations leak into the wire contract.",
			m.Verb, m.Path, strings.Join(exposed, ", ")),
		Suggestion: "map the entity to a dedicated DTO (e.g. " + exposed[0] + "Response) before it reaches the wire",
		Location:   node.Loc(),
	}}
}

// dedupe removes repeated names, keeping first-seen order.
func dedupe(names []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// ownerType finds the Type declaring a Field (fields are visited without
// their owner in the engine walk — Locations are exact, so a scan by
// position is unambiguous and cheap).
func ownerType(surface *ir.ApiSurface, f ir.Field) *ir.Type {
	for i := range surface.Types {
		t := &surface.Types[i]
		for _, fld := range t.Fields {
			if fld.Location == f.Location {
				return t
			}
		}
	}
	return nil
}

// isLowerCamel reports whether s is a lowerCamelCase identifier: it starts
// with a lower-case letter and carries only letters and digits (no _- or
// space separators, no PascalCase head).
func isLowerCamel(s string) bool {
	if s == "" {
		return true
	}
	for i, r := range s {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.':
			return false
		case unicode.IsDigit(r):
			if i == 0 {
				return false
			}
		case unicode.IsUpper(r):
			if i == 0 {
				return false
			}
		case unicode.IsLower(r):
			// fine
		default:
			return false
		}
	}
	return true
}

// lowerCamelOf re-spells a name as lowerCamelCase: split into words, join
// with every word capitalized except the first ("created_by" →
// "createdBy", "CreatedBy" → "createdBy").
func lowerCamelOf(s string) string {
	words := camelWordsSplit(s)
	if len(words) == 0 {
		return s
	}
	var b strings.Builder
	for i, w := range words {
		if i == 0 {
			b.WriteString(strings.ToLower(w))
			continue
		}
		r := []rune(strings.ToLower(w))
		if len(r) > 0 {
			r[0] = unicode.ToUpper(r[0])
		}
		b.WriteString(string(r))
	}
	return b.String()
}

// fieldCasingCheck is R4xx-02: DTO fields must serialize lowerCamelCase.
// The effective JSON name is checked — Field.JSONName when the adapter
// resolved @JsonProperty/@SerializedName, Field.Name otherwise. Entity
// types are out of scope: their casing is persistence territory, and
// entities must not be on the wire at all (R4xx-01).
func fieldCasingCheck(ctx *engine.LintContext, node ir.Node) []engine.Finding {
	f, ok := node.(ir.Field)
	if !ok || ctx.Surface == nil {
		return nil
	}
	if t := ownerType(ctx.Surface, f); t != nil && t.IsEntity {
		return nil
	}
	effective := f.JSONName
	if effective == "" {
		effective = f.Name
	}
	if isLowerCamel(effective) {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("Field %q is not lowerCamelCase — the payload JSON convention is lowerCamelCase.", effective),
		Suggestion: lowerCamelOf(effective),
		Location:   node.Loc(),
	}}
}

// standardTimeTypes serialize as RFC3339 (charter §3.4 names Instant and
// OffsetDateTime; ZonedDateTime carries the same wire format).
var standardTimeTypes = map[string]bool{
	"Instant": true, "OffsetDateTime": true, "ZonedDateTime": true,
}

// legacyTimeTypes are the spellings the rule flags on time-like fields:
// unstructured strings, the legacy java.util/java.sql types, zone-less
// java.time types (ISO-8601 but no offset → not RFC3339) and raw epoch
// numbers.
var legacyTimeTypes = map[string]bool{
	"String": true, "CharSequence": true,
	"Date": true, "Calendar": true, "Timestamp": true,
	"LocalDateTime": true, "LocalTime": true,
	"long": true, "Long": true, "int": true, "Integer": true,
}

// timeWordSet are the last words that make a field name time-like
// (createdAt, validUntil, startDate…). Word-splitting keeps the substring
// traps silent ("candidate", "format", "validate" are single words).
var timeWordSet = map[string]bool{
	"at": true, "time": true, "timestamp": true, "date": true,
	"when": true, "until": true, "since": true,
	"deadline": true, "expiry": true, "expires": true,
}

// timeFieldStandardCheck is R4xx-03: flag time-like DTO fields whose type
// cannot serialize as RFC3339. LocalDate stays acceptable for date-only
// names (*date); unknown custom types stay silent (the rule under-reports
// rather than guessing).
func timeFieldStandardCheck(ctx *engine.LintContext, node ir.Node) []engine.Finding {
	f, ok := node.(ir.Field)
	if !ok || ctx.Surface == nil {
		return nil
	}
	if t := ownerType(ctx.Surface, f); t != nil && t.IsEntity {
		return nil
	}
	if !timeWordSet[lastWordLower(f.JSONName)] && !timeWordSet[lastWordLower(f.Name)] {
		return nil
	}
	base := baseTypeName(f.Type.Name)
	if standardTimeTypes[base] {
		return nil
	}
	if base == "LocalDate" && lastWordLower(effectiveJSONName(f)) == "date" {
		return nil // date-only field, date-only type
	}
	if !legacyTimeTypes[base] {
		return nil
	}
	return []engine.Finding{{
		Message:    fmt.Sprintf("Field %q holds a time value as %s — clients get no RFC3339 structure to parse.", effectiveJSONName(f), base),
		Suggestion: "use Instant or OffsetDateTime so the wire format is RFC3339",
		Location:   node.Loc(),
	}}
}

// effectiveJSONName is the name the payload actually carries.
func effectiveJSONName(f ir.Field) string {
	if f.JSONName != "" {
		return f.JSONName
	}
	return f.Name
}

// PayloadRules compiles the R4xx family: R4xx-01 no-entity-in-payload
// (ERROR), R4xx-02 field-casing (WARN), R4xx-03 time-field-standard (INFO).
func PayloadRules() ([]engine.Rule, error) {
	return compileFamily("payload", []familyMember{
		{noEntityInPayloadYAML, methodSelector{}, noEntityInPayloadCheck},
		{fieldCasingYAML, fieldSelector{}, fieldCasingCheck},
		{timeFieldStandardYAML, fieldSelector{}, timeFieldStandardCheck},
	})
}

// fieldSelector matches Field nodes — the payload schema atoms.
type fieldSelector struct{}

func (fieldSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.Field)
	return ok
}
