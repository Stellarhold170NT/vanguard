package rules

import (
	"strings"
	"testing"

	"github.com/vanguard-lint/vanguard/internal/engine"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

// The R6xx family tests (w3-06): versioning & multi-protocol rules over
// hand-built IR. R6xx-01 is the charter's default-OFF policy rule — the
// enable path and the versionPattern option are pinned here. R6xx-02 reads
// the declaration-level GrpcServices the adapter emits from .proto files.

// boolPtr comes from demo_test.go (package rules).

// TestR6xxRulesFromData pins the data-driven identity of the family,
// including R6xx-01's DefaultDisabled (charter §3.6).
func TestR6xxRulesFromData(t *testing.T) {
	rs, err := VersioningRules()
	if err != nil {
		t.Fatalf("VersioningRules: %v", err)
	}
	if len(rs) != 2 {
		t.Fatalf("rules = %+v, want 2 members", rs)
	}
	byID := map[string]engine.Rule{}
	for _, r := range rs {
		byID[r.ID] = r
		if r.Category != "versioning" {
			t.Errorf("%s category = %q, want versioning", r.ID, r.Category)
		}
		if r.Slug == "" || r.DocPath == "" || r.Summary == "" || r.ExampleGood == "" || r.ExampleBad == "" {
			t.Errorf("%s metadata incomplete: %+v", r.ID, r)
		}
	}
	vp, ok := byID["R6xx-01"]
	if !ok {
		t.Fatalf("R6xx-01 missing: %+v", rs)
	}
	if vp.Severity != engine.SeverityWarn || !vp.DefaultDisabled {
		t.Errorf("R6xx-01 = severity %s defaultDisabled %v, want WARN + true (policy rule, charter §3.6)", vp.Severity, vp.DefaultDisabled)
	}
	gm, ok := byID["R6xx-02"]
	if !ok {
		t.Fatalf("R6xx-02 missing: %+v", rs)
	}
	if gm.Severity != engine.SeverityInfo || gm.DefaultDisabled {
		t.Errorf("R6xx-02 = severity %s defaultDisabled %v, want INFO + false", gm.Severity, gm.DefaultDisabled)
	}
}

// runVersioningRules lints one service (R6xx-01) or one proto service
// (R6xx-02) through the family under an optional config.
func runVersioningRules(t *testing.T, surface *ir.ApiSurface, cfg *engine.Config) []engine.Finding {
	t.Helper()
	return runVersioningReport(t, surface, cfg).Findings
}

// runVersioningReport lints and hands back the whole report — the
// panicking-rule diagnostics (an invalid option value) are observable
// there, not as a raw panic: the engine isolates a panicking rule into a
// diagnostic and continues (§5.4).
func runVersioningReport(t *testing.T, surface *ir.ApiSurface, cfg *engine.Config) *engine.Report {
	t.Helper()
	rs, err := VersioningRules()
	if err != nil {
		t.Fatalf("VersioningRules: %v", err)
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
	return report
}

// enableR6xx01 turns the default-off policy rule on (severity/option
// overrides layer on top by the caller).
func enableR6xx01(options map[string]any) *engine.Config {
	return &engine.Config{Rules: map[string]engine.RuleOverride{
		"R6xx-01": {Disabled: boolPtr(false), Options: options},
	}}
}

func serviceWithBase(base string) *ir.ApiSurface {
	return &ir.ApiSurface{
		Source: ir.Source{Lang: "java", Framework: "spring-boot"},
		Services: []ir.Service{{
			Name:     "OrderController",
			BasePath: base,
			Location: ir.Location{File: "src/OrderController.java", Line: 12, Column: 7},
			Methods: []ir.Method{{
				OperationName: "list", Verb: ir.VerbGet, Path: base + "/orders",
				Location: ir.Location{File: "src/OrderController.java", Line: 20, Column: 9},
			}},
		}},
	}
}

// TestR6xx01VersionedPath — fires when the base path carries no version
// segment under the effective pattern; empty bases and matched patterns
// stay silent; the pattern is configurable; a malformed pattern is loud.
func TestR6xx01VersionedPath(t *testing.T) {
	t.Run("default pattern", func(t *testing.T) {
		cases := []struct {
			base string
			want bool
		}{
			{"/api/v1/orders", false},
			{"/v2/orders", false},
			{"/api/orders", true},
			{"/orders", true},
			{"", false}, // no base path — nothing to version at service level
			{"/api/v10/orders", false},
			{"/api/rev1/orders", true}, // the pattern is segment-anchored: /vN, not vN substrings
		}
		for _, tc := range cases {
			got := errFindingsByRule(runVersioningRules(t, serviceWithBase(tc.base), enableR6xx01(nil)), "R6xx-01")
			if tc.want && len(got) != 1 {
				t.Errorf("base %q: findings = %d, want 1: %+v", tc.base, len(got), got)
			}
			if !tc.want && len(got) != 0 {
				t.Errorf("base %q: findings = %d, want 0: %+v", tc.base, len(got), got)
			}
		}
	})
	t.Run("default-disabled until enabled", func(t *testing.T) {
		if got := errFindingsByRule(runVersioningRules(t, serviceWithBase("/api/orders"), nil), "R6xx-01"); len(got) != 0 {
			t.Fatalf("R6xx-01 fired without a config enable: %+v", got)
		}
		cfg := &engine.Config{Rules: map[string]engine.RuleOverride{"R6xx-01": {Disabled: boolPtr(false)}}}
		if got := errFindingsByRule(runVersioningRules(t, serviceWithBase("/api/orders"), cfg), "R6xx-01"); len(got) != 1 {
			t.Fatalf("R6xx-01 did not fire after disabled:false: %+v", got)
		}
	})
	t.Run("custom pattern", func(t *testing.T) {
		cfg := enableR6xx01(map[string]any{"versionPattern": `^/api/v\d+`})
		if got := errFindingsByRule(runVersioningRules(t, serviceWithBase("/api/v1/orders"), cfg), "R6xx-01"); len(got) != 0 {
			t.Errorf("custom pattern should match /api/v1: %+v", got)
		}
		if got := errFindingsByRule(runVersioningRules(t, serviceWithBase("/orders/v1"), cfg), "R6xx-01"); len(got) != 1 {
			t.Errorf("custom pattern should reject /orders/v1 (anchor is ^): %+v", got)
		}
	})
	t.Run("malformed pattern is loud", func(t *testing.T) {
		cfg := enableR6xx01(map[string]any{"versionPattern": "/v[0-9"})
		report := runVersioningReport(t, serviceWithBase("/api/orders"), cfg)
		// The engine isolates the panicking rule into a diagnostic naming
		// the rule and the broken pattern — loud, non-fatal (§5.4).
		if len(report.Diagnostics) != 1 || !strings.Contains(report.Diagnostics[0].Message, "invalid versionPattern") {
			t.Fatalf("diagnostics = %+v, want the invalid-pattern panic surfaced", report.Diagnostics)
		}
		if len(report.Findings) != 0 {
			t.Fatalf("the broken-pattern rule must produce no findings: %+v", report.Findings)
		}
	})
}

func protoSurface(name string, rpcs ...string) *ir.ApiSurface {
	return &ir.ApiSurface{
		Source: ir.Source{Lang: "java", Framework: "spring-boot"},
		GrpcServices: []ir.GrpcService{{
			Name:     name,
			Rpcs:     rpcs,
			Location: ir.Location{File: "src/main/proto/library.proto", Line: 9, Column: 1},
		}},
	}
}

// TestR6xx02GrpcStandardMethods — Standard Method prefixes and VerbNoun
// custom methods pass; noun-first, unknown-verb and bare-word rpcs are
// flagged; the allow config exempts a name with a reason.
func TestR6xx02GrpcStandardMethods(t *testing.T) {
	t.Run("compliant names stay silent", func(t *testing.T) {
		surface := protoSurface("LibraryService",
			"GetBook", "ListBooks", "CreateBook", "UpdateBook", "DeleteBook",
			"BatchGetBooks", "BatchCreateBooks", "SearchBooks", "SyncData", "ArchiveBook",
		)
		if got := errFindingsByRule(runVersioningRules(t, surface, nil), "R6xx-02"); len(got) != 0 {
			t.Fatalf("compliant rpcs flagged: %+v", got)
		}
	})
	t.Run("off-pattern names are flagged", func(t *testing.T) {
		for _, rpc := range []string{
			"DoMagic",   // unknown verb — neither Standard nor VerbNoun
			"BooksSync", // noun-first — the verb does not lead the name
			"Handle",    // single word, no resource noun
			"Foo",       // single word
		} {
			surface := protoSurface("LibraryService", rpc)
			got := errFindingsByRule(runVersioningRules(t, surface, nil), "R6xx-02")
			if len(got) != 1 {
				t.Errorf("rpc %q: findings = %d, want 1: %+v", rpc, len(got), got)
			}
		}
	})
	t.Run("allow option exempts a name", func(t *testing.T) {
		cfg := &engine.Config{Rules: map[string]engine.RuleOverride{
			"R6xx-02": {Options: map[string]any{"allow": []any{"DoMagic"}}},
		}}
		surface := protoSurface("LibraryService", "DoMagic")
		if got := errFindingsByRule(runVersioningRules(t, surface, cfg), "R6xx-02"); len(got) != 0 {
			t.Fatalf("allowed rpc flagged: %+v", got)
		}
	})
	t.Run("message names the rpc and the service", func(t *testing.T) {
		surface := protoSurface("LibraryService", "DoStuff")
		got := errFindingsByRule(runVersioningRules(t, surface, nil), "R6xx-02")
		if len(got) != 1 || !strings.Contains(got[0].Message, "LibraryService") || !strings.Contains(got[0].Message, "DoStuff") {
			t.Fatalf("message should name service and rpc: %+v", got)
		}
	})
}
