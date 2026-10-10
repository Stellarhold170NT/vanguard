package rules

import (
	"testing"

	"github.com/vanguard-lint/vanguard/internal/engine"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

// The R2xx family (w3-04): unit tests pin every check's contract over
// hand-built IR nodes — the cheap tier that makes the heuristics explicit
// before the scan-based fixture test (methods_scan_test.go) proves the
// end-to-end wiring through the spring adapter.

// method builds an ir.Method with an optional payload and response in one
// call: empty payload means no request body, empty respType means void.
// A payload always comes with the matching in=body Param (the IR invariant
// adapters keep — w2-02), except where a test says stub-style explicitly.
func method(verb ir.Verb, path, payload, respType string, status int, collection bool) ir.Method {
	out := ir.Method{
		OperationName: "op",
		Verb:          verb,
		Path:          path,
		Response:      ir.Response{StatusCode: status, IsCollection: collection},
		Pagination:    ir.Pagination{Style: ir.PaginationNone},
	}
	if respType != "" {
		out.Response.Type = ir.TypeRef{Name: respType, IsCollection: collection}
	}
	if payload != "" {
		out.Payload = &ir.TypeRef{Name: payload}
		out.Params = append(out.Params, ir.Param{
			Name: "request", In: ir.ParamInBody, Type: ir.TypeRef{Name: payload},
			Location: ir.Location{File: "C.java", Line: 7, Column: 30},
		})
	}
	return out
}

// runCheck applies one rule's check to a method.
func runCheck(t *testing.T, r engine.Rule, m ir.Method) []engine.Finding {
	t.Helper()
	return r.Check(nil, m)
}

func findingAt(fs []engine.Finding, i int) engine.Finding { return fs[i] }

// TestMethodRules pins the family contract: six rules, charter §3.2 ids,
// slugs, severities and the "methods" category.
func TestMethodRules(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	want := []struct {
		id, slug string
		severity engine.Severity
	}{
		{"R2xx-01", "get-no-body", engine.SeverityError},
		{"R2xx-02", "post-creates-201", engine.SeverityWarn},
		{"R2xx-03", "patch-partial", engine.SeverityWarn},
		{"R2xx-04", "delete-no-body", engine.SeverityWarn},
		{"R2xx-05", "custom-method-post", engine.SeverityInfo},
		{"R2xx-06", "put-full-update", engine.SeverityWarn},
	}
	if len(rules) != len(want) {
		t.Fatalf("MethodRules returned %d rules, want %d", len(rules), len(want))
	}
	for i, w := range want {
		r := rules[i]
		if r.ID != w.id || r.Slug != w.slug || r.Severity != w.severity {
			t.Errorf("rule %d = %s %s %s, want %s %s %s", i, r.ID, r.Slug, r.Severity, w.id, w.slug, w.severity)
		}
		if r.Category != "methods" {
			t.Errorf("%s: category %q, want methods", r.ID, r.Category)
		}
		if r.DocPath == "" || r.Summary == "" || r.ExampleGood == "" || r.ExampleBad == "" {
			t.Errorf("%s: metadata incomplete (docPath/summary/examples)", r.ID)
		}
	}
}

// TestMethodRulesRegisterable proves the family passes the §5.4 registry
// validation catalog as a unit (unique ids and slugs).
func TestMethodRulesRegistry(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	reg := NewRegistry()
	for _, r := range rules {
		if err := reg.Register(r); err != nil {
			t.Fatalf("register %s: %v", r.ID, err)
		}
	}
	if got := len(reg.All()); got != 6 {
		t.Fatalf("registry holds %d rules, want 6", got)
	}
}

// TestGetNoBody pins R2xx-01: any GET with a declared request body — via
// Payload or a stub-style in=body param — is one ERROR finding anchored at
// the body param; everything else is silent.
func TestGetNoBody(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	r := rules[0]
	if r.ID != "R2xx-01" {
		t.Fatalf("rules[0] = %s, want R2xx-01", r.ID)
	}
	if got := runCheck(t, r, method(ir.VerbGet, "/books/auto-complete", "BookQuery", "List<BookDto>", 200, true)); len(got) != 1 {
		t.Fatalf("GET with payload: %d findings, want 1", len(got))
	} else if f := got[0]; f.Location.File != "C.java" {
		t.Fatalf("finding anchored at %v, want the body param", f.Location)
	}
	if got := runCheck(t, r, method(ir.VerbGet, "/books", "", "BookDto", 200, false)); len(got) != 0 {
		t.Fatalf("GET without body: %d findings, want 0", len(got))
	}
	// A stub-style body param without Payload is still a declared body.
	stub := method(ir.VerbGet, "/books", "", "BookDto", 200, false)
	stub.Params = []ir.Param{{Name: "q", In: ir.ParamInBody, Type: ir.TypeRef{Name: "BookQuery"}}}
	if got := runCheck(t, r, stub); len(got) != 1 {
		t.Fatalf("GET with in=body param only: %d findings, want 1", len(got))
	}
	// The selector narrows to GET: POST/DELETE bodies are other rules' jobs.
	if r.Selector.Matches(ir.Method{Verb: ir.VerbPost, Path: "/books"}) {
		t.Fatal("selector matched a POST method")
	}
	if !r.Selector.Matches(ir.Method{Verb: ir.VerbGet, Path: "/books"}) {
		t.Fatal("selector rejected a GET method")
	}
	if r.Selector.Matches(ir.Method{Verb: ir.VerbRPC, Path: "Query"}) {
		t.Fatal("selector matched an rpc method")
	}
	if r.Selector.Matches(ir.Service{}) {
		t.Fatal("selector matched a non-Method node")
	}
}

// TestPostCreates201 pins R2xx-02's create heuristic: single-resource
// response on an action-free collection path without 201/202 fires; void,
// collection and action-shaped posts stay silent.
func TestPostCreates201(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	r := rules[1]
	if r.ID != "R2xx-02" {
		t.Fatalf("rules[1] = %s, want R2xx-02", r.ID)
	}
	cases := []struct {
		name string
		m    ir.Method
		fire bool
	}{
		{"implicit 200 create", method(ir.VerbPost, "/books", "CreateBookRequest", "BookDto", 0, false), true},
		{"declared 200", method(ir.VerbPost, "/books", "CreateBookRequest", "BookDto", 200, false), true},
		{"declared 201", method(ir.VerbPost, "/books", "CreateBookRequest", "BookDto", 201, false), false},
		{"declared 202 accepted", method(ir.VerbPost, "/books", "CreateBookRequest", "BookDto", 202, false), false},
		{"void response", method(ir.VerbPost, "/books", "CreateBookRequest", "", 0, false), false},
		{"collection response", method(ir.VerbPost, "/books", "Query", "List<BookDto>", 200, true), false},
		{"action path search", method(ir.VerbPost, "/books/search", "Query", "BookDto", 200, false), false},
		{"action path export", method(ir.VerbPost, "/books/export", "Query", "BookDto", 200, false), false},
		{"item path template", method(ir.VerbPost, "/books/{id}", "Query", "BookDto", 200, false), false},
		{"custom method suffix", method(ir.VerbPost, "/books/{id}:archive", "Query", "BookDto", 200, false), false},
	}
	for _, tc := range cases {
		if got := runCheck(t, r, tc.m); (len(got) > 0) != tc.fire {
			t.Errorf("%s: %d findings, want fire=%v", tc.name, len(got), tc.fire)
		}
	}
}

// TestPatchPartial pins R2xx-03: PATCH whose payload type equals the
// declared response type is a full replacement; differing types stay silent.
func TestPatchPartial(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	r := rules[2]
	if r.ID != "R2xx-03" {
		t.Fatalf("rules[2] = %s, want R2xx-03", r.ID)
	}
	if got := runCheck(t, r, method(ir.VerbPatch, "/books/{id}", "BookDto", "BookDto", 200, false)); len(got) != 1 {
		t.Fatalf("PATCH with full payload: %d findings, want 1", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbPatch, "/books/{id}", "BookStatusUpdate", "BookDto", 200, false)); len(got) != 0 {
		t.Fatalf("PATCH with partial payload: %d findings, want 0", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbPatch, "/books/{id}", "BookStatusUpdate", "", 204, false)); len(got) != 0 {
		t.Fatalf("PATCH without declared response: %d findings, want 0", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbPatch, "/books/{id}", "", "BookDto", 200, false)); len(got) != 0 {
		t.Fatalf("PATCH without payload: %d findings, want 0", len(got))
	}
}

// TestDeleteNoBody pins R2xx-04: DELETE with a declared body — Payload or a
// stub-style in=body param — fires once; plain deletes stay silent.
func TestDeleteNoBody(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	r := rules[3]
	if r.ID != "R2xx-04" {
		t.Fatalf("rules[3] = %s, want R2xx-04", r.ID)
	}
	if got := runCheck(t, r, method(ir.VerbDelete, "/books", "List<String>", "", 204, false)); len(got) != 1 {
		t.Fatalf("DELETE with payload: %d findings, want 1", len(got))
	}
	stub := method(ir.VerbDelete, "/books", "", "", 204, false)
	stub.Params = []ir.Param{{Name: "ids", In: ir.ParamInBody, Type: ir.TypeRef{Name: "List<String>"}}}
	if got := runCheck(t, r, stub); len(got) != 1 {
		t.Fatalf("DELETE with in=body param only: %d findings, want 1", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbDelete, "/books/{id}", "", "", 204, false)); len(got) != 0 {
		t.Fatalf("DELETE without body: %d findings, want 0", len(got))
	}
}

// TestCustomMethodPost pins R2xx-05: non-POST verbs on action paths fire
// once, with the AIP-136 replacement text; POST endpoints and resource
// paths stay silent.
func TestCustomMethodPost(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	r := rules[4]
	if r.ID != "R2xx-05" {
		t.Fatalf("rules[4] = %s, want R2xx-05", r.ID)
	}
	if got := runCheck(t, r, method(ir.VerbGet, "/api/v1/books/{id}/generate-code", "", "CodeDto", 200, false)); len(got) != 1 {
		t.Fatalf("GET action endpoint: %d findings, want 1", len(got))
	} else if want := "POST /api/v1/books/{id}:generate-code"; got[0].Suggestion != want {
		t.Fatalf("suggestion %q, want %q", got[0].Suggestion, want)
	}
	if got := runCheck(t, r, method(ir.VerbPost, "/books/{id}:archive", "", "BookDto", 200, false)); len(got) != 0 {
		t.Fatalf("POST custom method: %d findings, want 0", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbGet, "/books/{id}", "", "BookDto", 200, false)); len(got) != 0 {
		t.Fatalf("resource path: %d findings, want 0", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbPost, "/reports/export", "", "Report", 200, false)); len(got) != 0 {
		t.Fatalf("POST action endpoint: %d findings, want 0", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbGet, "/export", "", "Report", 200, false)); len(got) != 1 {
		t.Fatalf("root action path: %d findings, want 1", len(got))
	} else if want := "POST <collection>:export"; got[0].Suggestion != want {
		t.Fatalf("root suggestion %q, want %q", got[0].Suggestion, want)
	}
}

// TestPutFullUpdate pins R2xx-06: PUT whose payload type differs from the
// declared response type fires; matching types and void responses stay silent.
func TestPutFullUpdate(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	r := rules[5]
	if r.ID != "R2xx-06" {
		t.Fatalf("rules[5] = %s, want R2xx-06", r.ID)
	}
	if got := runCheck(t, r, method(ir.VerbPut, "/books/{id}", "BookUpdateRequest", "BookDto", 200, false)); len(got) != 1 {
		t.Fatalf("PUT with partial payload: %d findings, want 1", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbPut, "/books/{id}", "BookDto", "BookDto", 200, false)); len(got) != 0 {
		t.Fatalf("PUT with full payload: %d findings, want 0", len(got))
	}
	if got := runCheck(t, r, method(ir.VerbPut, "/books/{id}", "BookUpdateRequest", "", 204, false)); len(got) != 0 {
		t.Fatalf("PUT without declared response: %d findings, want 0", len(got))
	}
}

// TestActionTokens pins the shared action-verb tokenizer: camelCase and
// kebab-case segment spellings resolve to the same tokens, templates never
// match, and plural nouns are exact-token safe.
func TestActionTokens(t *testing.T) {
	cases := []struct {
		segment string
		want    bool
	}{
		{"generate-code", true},
		{"generateCode", true},
		{"export", true},
		{"search", true},
		{"auto-complete", false}, // documented miss: "auto"/"complete" are not action verbs
		{"books", false},
		{"book", false},
		{"playlists", false},
		{"runs", false},
		{"scans", false},
		{"{id}", false},
	}
	for _, tc := range cases {
		if got := hasActionToken(tc.segment); got != tc.want {
			t.Errorf("hasActionToken(%q) = %v, want %v", tc.segment, got, tc.want)
		}
	}
	if segs := actionSegmentsOf("/books/{id}/generate-code"); len(segs) != 1 || segs[0] != "generate-code" {
		t.Fatalf("actionSegmentsOf = %v, want [generate-code]", segs)
	}
}

// TestBasePathStripping pins the method-level-path discipline: an action
// word in the service's BASE path must not taint every endpoint under it —
// the verb rules judge the method-level shape (with a context Service),
// and the create detection treats a POST on the base itself as the
// collection root.
func TestBasePathStripping(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	ctx := &engine.LintContext{Service: &ir.Service{Name: "Search", BasePath: "/v1/search"}}
	// GET /v1/search/archive under base /v1/search: the method path "/archive"
	// carries no action token — the base-path "search" must not fire R2xx-05.
	m := method(ir.VerbGet, "/v1/search/archive", "", "ArchiveResult", 200, false)
	if got := runCheckCtx(t, rules[4], ctx, m); len(got) != 0 {
		t.Fatalf("base-path action word: %d findings, want 0", len(got))
	}
	// The method-level shape still fires.
	m = method(ir.VerbGet, "/v1/search/archive/export", "", "Report", 200, false)
	if got := runCheckCtx(t, rules[4], ctx, m); len(got) != 1 {
		t.Fatalf("method-level action: %d findings, want 1", len(got))
	} else if want := "POST /v1/search/archive:export"; got[0].Suggestion != want {
		t.Fatalf("suggestion %q, want %q", got[0].Suggestion, want)
	}
	// R2xx-02: a POST on the base path itself is the collection root (a
	// create-shaped path), not an empty segment.
	create := method(ir.VerbPost, "/v1/orders", "CreateOrderRequest", "Order", 200, false)
	if got := runCheckCtx(t, rules[1], ctx, create); len(got) != 1 {
		t.Fatalf("POST on base path: %d findings, want 1 (create-shaped)", len(got))
	}
}

// runCheckCtx applies one rule's check with an explicit context.
func runCheckCtx(t *testing.T, r engine.Rule, ctx *engine.LintContext, m ir.Method) []engine.Finding {
	t.Helper()
	return r.Check(ctx, m)
}

// TestSuggestionsAreReplacementText pins the §3.0 suggestion discipline on
// the R2xx set: every suggestion is concrete text, never advice prose, and
// the R2xx-05 message names the AIP-136 pattern.
func TestSuggestionsAreReplacementText(t *testing.T) {
	rules, err := MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	f := runCheck(t, rules[0], method(ir.VerbGet, "/books", "BookQuery", "BookDto", 200, false))
	if len(f) != 1 || f[0].Suggestion == "" {
		t.Fatalf("R2xx-01 suggestion missing: %v", f)
	}
	if msg := f[0].Message; msg == "" || msg[len(msg)-1] != '.' {
		t.Fatalf("R2xx-01 message not a full sentence: %q", msg)
	}
	f = runCheck(t, rules[4], method(ir.VerbGet, "/books/{id}/restore", "", "BookDto", 200, false))
	if len(f) != 1 {
		t.Fatalf("R2xx-05: %d findings, want 1", len(f))
	}
	if want := "POST /books/{id}:restore"; f[0].Suggestion != want {
		t.Fatalf("R2xx-05 suggestion %q, want %q", f[0].Suggestion, want)
	}
}
