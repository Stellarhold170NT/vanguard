package rules

import (
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// The R5xx family tests (w3-06): error & status-semantics rules over
// hand-built IR surfaces. The heuristics the W4 phase will attack are
// pinned here: the envelope-majority inference (R5xx-01), the
// app-ownership + status pair that decides a business 500 (R5xx-02), and
// the create-shape guards of R5xx-03.

// errHandler builds one ErrorHandler at a stable location.
func errHandler(exc, resp string, status int) ir.ErrorHandler {
	return ir.ErrorHandler{
		ExceptionType: exc,
		ResponseType:  resp,
		StatusCode:    status,
		Location:      ir.Location{File: "src/GlobalExceptionHandler.java", Line: 20, Column: 9},
	}
}

// errSurface assembles a surface whose ErrorScheme holds handlers and whose
// Types carry envelope fields, addressed by simple name.
func errSurface(handlers []ir.ErrorHandler, types ...ir.Type) *ir.ApiSurface {
	s := &ir.ApiSurface{
		Source: ir.Source{Lang: "java", Framework: "spring-boot"},
	}
	s.ErrorScheme.Handlers = handlers
	s.Types = types
	return s
}

func pojo(name string, fields ...string) ir.Type {
	t := ir.Type{Name: name, Kind: ir.KindPOJO, Package: "com.example.app.dto",
		Location: ir.Location{File: "src/" + name + ".java", Line: 5, Column: 7}}
	for _, f := range fields {
		t.Fields = append(t.Fields, ir.Field{
			Name:     f,
			Type:     ir.TypeRef{Name: "String"},
			Location: ir.Location{File: "src/" + name + ".java", Line: 8, Column: 12},
		})
	}
	return t
}

// runErrorRules lints the surface with the R5xx family (optionally under a
// config that pins rule options).
func runErrorRules(t *testing.T, surface *ir.ApiSurface, cfg *engine.Config) []engine.Finding {
	t.Helper()
	rs, err := ErrorRules()
	if err != nil {
		t.Fatalf("ErrorRules: %v", err)
	}
	reg := NewRegistry()
	for _, r := range rs {
		if err := reg.Register(r); err != nil {
			t.Fatalf("register %s: %v", r.ID, err)
		}
	}
	report, err := engine.NewLinter(reg).Run(surface, cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return report.Findings
}

// TestR5xxRulesFromData pins the data-driven identity of the family.
func TestR5xxRulesFromData(t *testing.T) {
	rs, err := ErrorRules()
	if err != nil {
		t.Fatalf("ErrorRules: %v", err)
	}
	want := map[string]engine.Severity{"R5xx-01": engine.SeverityWarn, "R5xx-02": engine.SeverityError, "R5xx-03": engine.SeverityWarn}
	if len(rs) != len(want) {
		t.Fatalf("rules = %+v, want %d members", rs, len(want))
	}
	for _, r := range rs {
		if r.Severity != want[r.ID] {
			t.Errorf("%s severity = %s, want %s", r.ID, r.Severity, want[r.ID])
		}
		if r.Category != "errors" {
			t.Errorf("%s category = %q, want errors", r.ID, r.Category)
		}
		if r.Slug == "" || r.DocPath == "" || r.Summary == "" || r.ExampleGood == "" || r.ExampleBad == "" {
			t.Errorf("%s metadata incomplete: %+v", r.ID, r)
		}
	}
}

// TestR5xx01MajorityEnvelope — auto mode: the envelope most handlers
// return IS the app's standard; a deviating handler splits the contract.
func TestR5xx01MajorityEnvelope(t *testing.T) {
	surface := errSurface(
		[]ir.ErrorHandler{
			errHandler("BookNotFoundException", "ErrorResponse", 404),
			errHandler("IllegalArgumentException", "ErrorResponse", 0),
			errHandler("ValidationFailed", "Map<String,Object>", 400),
		},
		pojo("ErrorResponse", "code", "message", "details"),
	)
	findings := findByRule(runErrorRules(t, surface, nil), "R5xx-01")
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1 (the Map handler only): %+v", len(findings), findings)
	}
	if !strings.Contains(findings[0].Message, "Map<String,Object>") || !strings.Contains(findings[0].Message, "ErrorResponse") {
		t.Errorf("message should name both the deviation and the standard: %q", findings[0].Message)
	}
}

// TestR5xx01SilentWithoutConsensus — with no resolvable majority (all
// distinct, tie, or a majority the scan cannot resolve) the rule stays
// silent: an unprovable standard must never produce a finding.
func TestR5xx01SilentWithoutConsensus(t *testing.T) {
	for name, handlers := range map[string][]ir.ErrorHandler{
		"all distinct": {
			errHandler("A", "ErrorResponse", 404),
			errHandler("B", "ProblemDetail", 400),
		},
		"tie": {
			errHandler("A", "ErrorResponse", 404),
			errHandler("B", "ProblemDetail", 400),
			errHandler("C", "Map<String,Object>", 400),
		},
		"unresolvable majority": {
			errHandler("A", "Map<String,Object>", 400),
			errHandler("B", "Map<String,Object>", 400),
			errHandler("C", "ErrorResponse", 404),
		},
		"single handler": {
			errHandler("A", "Map<String,Object>", 400),
		},
	} {
		surface := errSurface(handlers, pojo("ErrorResponse", "code", "message"))
		if got := findByRule(runErrorRules(t, surface, nil), "R5xx-01"); len(got) != 0 {
			t.Errorf("%s: findings = %d, want 0: %+v", name, len(got), got)
		}
	}
}

// TestR5xx01PinnedScheme — config pins the expected shape: the handler's
// response type is field-checked against {code,message} (or the RFC-7807
// set); unresolvable and void handlers stay silent (under-report).
func TestR5xx01PinnedScheme(t *testing.T) {
	enabled := func(scheme string) *engine.Config {
		return &engine.Config{Rules: map[string]engine.RuleOverride{
			"R5xx-01": {Options: map[string]any{"scheme": scheme}},
		}}
	}
	t.Run("code-message-details", func(t *testing.T) {
		surface := errSurface(
			[]ir.ErrorHandler{
				errHandler("A", "ErrorResponse", 404),
				errHandler("B", "BareError", 400),
				errHandler("C", "UnknownType", 400),
				errHandler("D", "", 400),
			},
			pojo("ErrorResponse", "code", "message", "details"),
			pojo("BareError", "reason"),
		)
		findings := findByRule(runErrorRules(t, surface, enabled("code-message-details")), "R5xx-01")
		if len(findings) != 1 {
			t.Fatalf("findings = %d, want 1 (BareError only): %+v", len(findings), findings)
		}
		if !strings.Contains(findings[0].Message, "BareError") {
			t.Errorf("message should name the offending type: %q", findings[0].Message)
		}
	})
	t.Run("problem-json", func(t *testing.T) {
		surface := errSurface(
			[]ir.ErrorHandler{
				errHandler("A", "Problem", 400),
				errHandler("B", "BareError", 400),
			},
			pojo("Problem", "title", "status", "detail"),
			pojo("BareError", "reason"),
		)
		findings := findByRule(runErrorRules(t, surface, enabled("problem-json")), "R5xx-01")
		if len(findings) != 1 || !strings.Contains(findings[0].Message, "BareError") {
			t.Fatalf("findings = %+v, want 1 on BareError", findings)
		}
	})
	t.Run("unknown scheme value", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("unknown scheme option must be loud (panic → engine diagnostic), got silence")
			}
		}()
		surface := errSurface([]ir.ErrorHandler{errHandler("A", "ErrorResponse", 404)}, pojo("ErrorResponse", "code"))
		_ = runErrorRules(t, surface, enabled("yaml-ish"))
	})
}

// TestR5xx02No500ForBusiness — the ERROR heuristic: app-owned exceptions
// (the adapter resolved their package) may not map to 500; external and
// non-Exception-suffixed names stay silent; the exception's own
// @ResponseStatus counts only when the handler declares none.
func TestR5xx02No500ForBusiness(t *testing.T) {
	cases := []struct {
		name    string
		handler ir.ErrorHandler
		want    bool
	}{
		{"business exception at 500", func() ir.ErrorHandler {
			h := errHandler("InsufficientBalanceException", "ErrorResponse", 500)
			h.ExceptionPackage = "com.example.shop.service"
			return h
		}(), true},
		{"same exception at 404", func() ir.ErrorHandler {
			h := errHandler("NotFoundException", "ErrorResponse", 404)
			h.ExceptionPackage = "com.example.shop.exception"
			return h
		}(), false},
		{"external fallback handler at 500", errHandler("Exception", "ErrorResponse", 500), false},
		{"app class without Exception suffix at 500", func() ir.ErrorHandler {
			h := errHandler("DomainRuleViolation", "ErrorResponse", 500)
			h.ExceptionPackage = "com.example.shop.service"
			return h
		}(), false},
		{"handler status wins over exception status", func() ir.ErrorHandler {
			h := errHandler("InsufficientBalanceException", "ErrorResponse", 404)
			h.ExceptionPackage = "com.example.shop.service"
			h.ExceptionStatus = 500
			return h
		}(), false},
		{"exception-declared 500 when handler undeclared", func() ir.ErrorHandler {
			h := errHandler("InsufficientBalanceException", "ErrorResponse", 0)
			h.ExceptionPackage = "com.example.shop.service"
			h.ExceptionStatus = 500
			return h
		}(), true},
	}
	for _, tc := range cases {
		surface := errSurface([]ir.ErrorHandler{tc.handler}, pojo("ErrorResponse", "code", "message"))
		got := findByRule(runErrorRules(t, surface, nil), "R5xx-02")
		if tc.want && len(got) != 1 {
			t.Errorf("%s: findings = %d, want 1: %+v", tc.name, len(got), got)
		}
		if !tc.want && len(got) != 0 {
			t.Errorf("%s: findings = %d, want 0: %+v", tc.name, len(got), got)
		}
	}
}

// TestR5xx03StatusSemantics — POST on a create-shaped path with a declared
// single body must answer 201/202; explicit 200 and the implicit
// (undeclared) 200 both fire; every other shape stays silent.
func TestR5xx03StatusSemantics(t *testing.T) {
	method := func(verb ir.Verb, path string, status int, resp string, collection bool) ir.Method {
		m := ir.Method{OperationName: "op", Verb: verb, Path: path, Location: ir.Location{File: "src/OrderController.java", Line: 30, Column: 10}}
		m.Response = ir.Response{StatusCode: status, IsCollection: collection, Location: m.Location}
		if resp != "" {
			m.Response.Type = ir.TypeRef{Name: resp}
		}
		return m
	}
	cases := []struct {
		name   string
		method ir.Method
		want   bool
	}{
		{"explicit 200 create", method(ir.VerbPost, "/api/v1/orders", 200, "OrderDto", false), true},
		{"implicit 200 create", method(ir.VerbPost, "/api/v1/orders", 0, "OrderDto", false), true},
		{"declared 201 create", method(ir.VerbPost, "/api/v1/orders", 201, "OrderDto", false), false},
		{"declared 202 create", method(ir.VerbPost, "/api/v1/orders", 202, "OrderDto", false), false},
		{"action word segment", method(ir.VerbPost, "/api/v1/orders/cancel", 200, "OrderDto", false), false},
		{"custom-method colon", method(ir.VerbPost, "/api/v1/orders/{id}:archive", 200, "OrderDto", false), false},
		{"item path template", method(ir.VerbPost, "/api/v1/orders/{id}", 200, "OrderDto", false), false},
		{"collection response", method(ir.VerbPost, "/api/v1/orders/search", 200, "List<OrderDto>", true), false},
		{"void handler", method(ir.VerbPost, "/api/v1/orders", 200, "", false), false},
		{"no-content status", method(ir.VerbPost, "/api/v1/orders", 204, "OrderDto", false), false},
		{"GET never fires", method(ir.VerbGet, "/api/v1/orders", 200, "OrderDto", false), false},
	}
	for _, tc := range cases {
		surface := errSurface(nil)
		surface.Services = []ir.Service{{Name: "OrderController", Methods: []ir.Method{tc.method}}}
		got := findByRule(runErrorRules(t, surface, nil), "R5xx-03")
		if tc.want && len(got) != 1 {
			t.Errorf("%s: findings = %d, want 1: %+v", tc.name, len(got), got)
		}
		if !tc.want && len(got) != 0 {
			t.Errorf("%s: findings = %d, want 0: %+v", tc.name, len(got), got)
		}
	}
}
