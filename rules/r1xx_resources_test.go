package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vanguard-lint/vanguard/internal/engine"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

// Charter §3.1 pins the R1xx family: five rules, fixed ids/slugs/severities.
// The metadata records under rules/data must carry exactly this identity.
func TestR1xxMetadataMatchesCharter(t *testing.T) {
	want := []struct {
		id, slug, category, severity string
	}{
		{"R1xx-01", "plural-collection", "resources", "WARN"},
		{"R1xx-02", "no-verb-path", "resources", "ERROR"},
		{"R1xx-03", "resource-path-pattern", "resources", "WARN"},
		{"R1xx-04", "path-casing", "resources", "WARN"},
		{"R1xx-05", "id-field-naming", "resources", "INFO"},
	}
	rulesSet, err := ResourceRules()
	if err != nil {
		t.Fatalf("ResourceRules: %v", err)
	}
	if len(rulesSet) != len(want) {
		t.Fatalf("got %d R1xx rules, want %d", len(rulesSet), len(want))
	}
	for i, w := range want {
		r := rulesSet[i]
		if r.ID != w.id || r.Slug != w.slug || r.Category != w.category || string(r.Severity) != w.severity {
			t.Errorf("rule[%d] = {%s %s %s %s}, want {%s %s %s %s}",
				i, r.ID, r.Slug, r.Category, r.Severity, w.id, w.slug, w.category, w.severity)
		}
		if r.DocPath != "docs/rules/"+r.ID+"-"+r.Slug+".md" {
			t.Errorf("rule %s: docPath = %q, want docs/rules/<id>-<slug>.md", r.ID, r.DocPath)
		}
		if r.Summary == "" || r.ExampleGood == "" || r.ExampleBad == "" {
			t.Errorf("rule %s: summary/examples must be non-empty (--list-rules renders them)", r.ID)
		}
	}
}

// Registering the whole family into a fresh registry must succeed — unique
// ids and slugs, valid metadata (§5.4: registration errors fail tests).
func TestR1xxRegistersCleanly(t *testing.T) {
	reg := NewRegistry()
	rulesSet, err := ResourceRules()
	if err != nil {
		t.Fatalf("ResourceRules: %v", err)
	}
	for _, r := range rulesSet {
		if err := reg.Register(r); err != nil {
			t.Fatalf("register %s: %v", r.ID, err)
		}
	}
	if got := len(reg.All()); got != len(rulesSet) {
		t.Fatalf("registry holds %d rules, want %d", got, len(rulesSet))
	}
}

// methodOf builds a minimal Method node for the path checks.
func methodOf(verb ir.Verb, path string) ir.Method {
	return ir.Method{
		OperationName: "probe",
		Verb:          verb,
		Path:          path,
		Response:      ir.Response{Type: ir.TypeRef{Name: "String"}},
		Location:      ir.Location{File: "probe.java", Line: 1, Column: 1},
	}
}

func collectionMethodOf(verb ir.Verb, path string) ir.Method {
	m := methodOf(verb, path)
	m.Response.IsCollection = true
	m.Response.Type = ir.TypeRef{Name: "List<BookDto>", IsCollection: true}
	return m
}

func runRule(t *testing.T, r engine.Rule, node ir.Node) []engine.Finding {
	t.Helper()
	if !r.Selector.Matches(node) {
		return nil
	}
	return r.Check(&engine.LintContext{}, node)
}

func requireFindings(t *testing.T, r engine.Rule, node ir.Node, want int) []engine.Finding {
	t.Helper()
	got := runRule(t, r, node)
	if len(got) != want {
		t.Fatalf("%s on %s %q: got %d finding(s) (%v), want %d",
			r.ID, irMethodVerb(node), irMethodPath(node), len(got), got, want)
	}
	for _, f := range got {
		if f.Message == "" {
			t.Fatalf("%s: empty message", r.ID)
		}
		if f.Location.File == "" {
			t.Fatalf("%s: finding must carry the node location", r.ID)
		}
	}
	return got
}

func irMethodVerb(node ir.Node) string {
	if m, ok := node.(ir.Method); ok {
		return string(m.Verb)
	}
	return "?"
}

func irMethodPath(node ir.Node) string {
	if m, ok := node.(ir.Method); ok {
		return m.Path
	}
	return "?"
}

func ruleByID(t *testing.T, id string) engine.Rule {
	t.Helper()
	rulesSet, err := ResourceRules()
	if err != nil {
		t.Fatalf("ResourceRules: %v", err)
	}
	for _, r := range rulesSet {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("rule %s not found", id)
	return engine.Rule{}
}

func TestR1xx01PluralCollection(t *testing.T) {
	r := ruleByID(t, "R1xx-01")

	// Collection segment before {id} must be plural; suggestion is the fixed path.
	got := requireFindings(t, r, methodOf(ir.VerbGet, "/book/{id}"), 1)
	if got[0].Suggestion != "/books/{id}" {
		t.Errorf("suggestion = %q, want the pluralized path", got[0].Suggestion)
	}
	requireFindings(t, r, methodOf(ir.VerbGet, "/books/{id}"), 0)

	// Sub-collection between two variables.
	requireFindings(t, r, methodOf(ir.VerbGet, "/books/{bookId}/review/{reviewId}"), 1)
	requireFindings(t, r, methodOf(ir.VerbGet, "/books/{bookId}/reviews/{reviewId}"), 0)

	// Uncountable nouns never fire.
	requireFindings(t, r, methodOf(ir.VerbGet, "/equipment/{id}"), 0)

	// Final segment of a list endpoint (GET + collection response).
	requireFindings(t, r, collectionMethodOf(ir.VerbGet, "/training-plan"), 1)
	requireFindings(t, r, collectionMethodOf(ir.VerbGet, "/training-plans"), 0)

	// Action/aggregate sub-paths are R2xx-05's subject, not plural nouns.
	requireFindings(t, r, collectionMethodOf(ir.VerbGet, "/books/search"), 0)

	// Non-collection responses and non-GET verbs stay silent.
	requireFindings(t, r, methodOf(ir.VerbGet, "/training-plan"), 0)
	requireFindings(t, r, methodOf(ir.VerbPost, "/training-plan"), 0)

	// gRPC operations carry no HTTP path semantics.
	rpc := methodOf(ir.VerbRPC, "/training-plan")
	if r.Selector.Matches(rpc) {
		t.Errorf("selector must not match rpc methods")
	}
}

func TestR1xx02NoVerbPath(t *testing.T) {
	r := ruleByID(t, "R1xx-02")

	// The three charter/brief shapes: /delete, /update/draft, /getUserById.
	requireFindings(t, r, methodOf(ir.VerbDelete, "/delete/{id}"), 1)
	requireFindings(t, r, methodOf(ir.VerbPost, "/update/draft"), 1)
	requireFindings(t, r, methodOf(ir.VerbGet, "/getUserById"), 1)

	// Near-miss nouns: exact-token matching only.
	requireFindings(t, r, methodOf(ir.VerbGet, "/budget-getter/{id}"), 0)
	requireFindings(t, r, methodOf(ir.VerbPost, "/updates"), 0)
	requireFindings(t, r, methodOf(ir.VerbPost, "/statistics/rebuild"), 0)

	// Clean paths and path variables stay silent.
	requireFindings(t, r, methodOf(ir.VerbGet, "/health"), 0)
	requireFindings(t, r, methodOf(ir.VerbGet, "/books/{id}"), 0)

	// The suggestion of the redundant /delete case proposes the path without it.
	got := requireFindings(t, r, methodOf(ir.VerbDelete, "/delete/{id}"), 1)
	if !strings.Contains(got[0].Suggestion, "/") {
		t.Errorf("suggestion %q should propose a concrete route", got[0].Suggestion)
	}
}

func TestR1xx03ResourcePathPattern(t *testing.T) {
	r := ruleByID(t, "R1xx-03")

	// Flat action route instead of /collection/{id}.
	requireFindings(t, r, methodOf(ir.VerbGet, "/find-by-id/{id}"), 1)
	// Consecutive path variables — the sub-collection is missing.
	requireFindings(t, r, methodOf(ir.VerbGet, "/orders/{orderId}/{itemId}"), 1)
	// Path opening with a variable — no collection anchors it.
	requireFindings(t, r, methodOf(ir.VerbGet, "/{memberId}/loans"), 1)

	// Legal shapes.
	requireFindings(t, r, methodOf(ir.VerbGet, "/authors/{authorId}/books/{bookId}"), 0)
	requireFindings(t, r, methodOf(ir.VerbGet, "/libraries/{libraryId}"), 0)
	requireFindings(t, r, methodOf(ir.VerbGet, "/books"), 0)
	requireFindings(t, r, methodOf(ir.VerbGet, "/health"), 0)
}

func TestR1xx04PathCasing(t *testing.T) {
	r := ruleByID(t, "R1xx-04")

	// Literal segments must be kebab-case; the suggestion is the fixed path.
	got := requireFindings(t, r, methodOf(ir.VerbGet, "/search/all-by-Name"), 1)
	if got[0].Suggestion != "/search/all-by-name" {
		t.Errorf("suggestion = %q, want /search/all-by-name", got[0].Suggestion)
	}
	got = requireFindings(t, r, methodOf(ir.VerbGet, "/loanOrders"), 1)
	if got[0].Suggestion != "/loan-orders" {
		t.Errorf("suggestion = %q, want /loan-orders", got[0].Suggestion)
	}
	requireFindings(t, r, methodOf(ir.VerbGet, "/order_items"), 1)

	// Path variables must be lowerCamelCase.
	got = requireFindings(t, r, methodOf(ir.VerbGet, "/trainings/{YouthId}"), 1)
	if !strings.Contains(got[0].Suggestion, "{youthId}") {
		t.Errorf("suggestion = %q, want the camel-cased variable", got[0].Suggestion)
	}
	requireFindings(t, r, methodOf(ir.VerbGet, "/orders/{user_id}"), 1)

	// Legal casing.
	requireFindings(t, r, methodOf(ir.VerbGet, "/audio-books/{bookId}"), 0)
	requireFindings(t, r, methodOf(ir.VerbGet, "/api-keys/{keyId}"), 0)
	requireFindings(t, r, methodOf(ir.VerbGet, "/v1/reports"), 0)
}

// typeOf builds a Type node; fields get their serialized name via JSONName.
func typeOf(name string, fields ...ir.Field) ir.Type {
	return ir.Type{
		Name:     name,
		Kind:     ir.KindRecord,
		Fields:   fields,
		Location: ir.Location{File: "probe.java", Line: 1, Column: 1},
	}
}

func fieldOf(name, jsonName string) ir.Field {
	return ir.Field{
		Name:     name,
		JSONName: jsonName,
		Type:     ir.TypeRef{Name: "Long"},
		Location: ir.Location{File: "probe.java", Line: 2, Column: 3},
	}
}

func TestR1xx05IdFieldNaming(t *testing.T) {
	r := ruleByID(t, "R1xx-05")

	// Mixed reference-id suffixes inside one DTO.
	mixed := typeOf("StockItemDto",
		fieldOf("bookID", ""), fieldOf("member_id", ""), fieldOf("publisherId", ""))
	got := requireFindings(t, r, mixed, 1)
	if !strings.Contains(got[0].Message, "bookID") {
		t.Errorf("message should name the offending fields, got %q", got[0].Message)
	}

	// JSON keys participate: user_id vs orderId is a mix on the wire.
	requireFindings(t, r, typeOf("FulfillmentDto",
		fieldOf("userId", "user_id"), fieldOf("orderId", "orderId")), 1)

	// The charter convention itself stays silent: id for self, <resource>Id for references.
	requireFindings(t, r, typeOf("LoanDto",
		fieldOf("id", ""), fieldOf("bookId", ""), fieldOf("memberId", "")), 0)
	// One consistent reference convention.
	requireFindings(t, r, typeOf("AuthorSummaryDto",
		fieldOf("authorId", ""), fieldOf("publisherId", "")), 0)
	// A single reference id has nothing to be inconsistent with.
	requireFindings(t, r, typeOf("ViewDto", fieldOf("viewerId", "")), 0)
	// Words that merely end in "id" are not id fields.
	requireFindings(t, r, typeOf("Device", fieldOf("android", "")), 0)

	// Surface-level self-id mix: OrderDto's own id is orderId while MemberDto uses id.
	surface := &ir.ApiSurface{Types: []ir.Type{
		typeOf("OrderDto", fieldOf("orderId", ""), fieldOf("tracking", "")),
		typeOf("MemberDto", fieldOf("id", ""), fieldOf("name", "")),
	}}
	ctx := &engine.LintContext{Surface: surface}
	orderFindings := r.Check(ctx, surface.Types[0])
	if len(orderFindings) != 1 {
		t.Fatalf("self-id mix not reported for OrderDto: %v", orderFindings)
	}
	memberFindings := r.Check(ctx, surface.Types[1])
	if len(memberFindings) != 0 {
		t.Errorf("MemberDto follows the id convention — must stay silent, got %v", memberFindings)
	}

	// A lone resource-id style with no counterexample is not a mix (v0.1 reports mixes only).
	lone := &ir.ApiSurface{Types: []ir.Type{
		typeOf("OrderDto", fieldOf("orderId", ""), fieldOf("tracking", "")),
	}}
	if got := r.Check(&engine.LintContext{Surface: lone}, lone.Types[0]); len(got) != 0 {
		t.Errorf("single self-id style must stay silent, got %v", got)
	}
}

// The engine must stamp what the check leaves out: a found violation keeps
// Message/Suggestion/Location from the check and gets RuleID/Severity from
// the engine (§5.4 — covered here at the seam the engine relies on).
func TestR1xxCheckOutputsAreEngineShaped(t *testing.T) {
	r := ruleByID(t, "R1xx-02")
	got := runRule(t, r, methodOf(ir.VerbDelete, "/delete/{id}"))
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.RuleID != "" || f.Severity != "" {
		t.Errorf("checks must not stamp RuleID/Severity themselves (§5.4): got %s/%s", f.RuleID, f.Severity)
	}
	if f.Location != (ir.Location{File: "probe.java", Line: 1, Column: 1}) {
		t.Errorf("finding must anchor at the method location, got %+v", f.Location)
	}
	if reflect.DeepEqual(f.Message, "") {
		t.Errorf("message required")
	}
}
