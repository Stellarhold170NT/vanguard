// The external half of the w3-06 test set (package rules_test): it runs
// the R5xx and R6xx rules through the REAL pipeline — discovery walk →
// Spring adapter (java tree-sitter + proto declaration reader) → IR →
// engine — over the testdata/spring-errors fixture, pinning the exact
// finding set per file. The in-package tests (errors_test.go,
// versioning_test.go) pin the per-rule heuristics; this file pins that the
// checks derive everything from the IR the adapter emitted — the ErrorScheme
// is consumed as-is, no rule ever re-opens a source file (w3-06 acceptance).
package rules_test

import (
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/discovery"
	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/rules"
)

const errorsRepo = "../testdata/spring-errors"

// scanErrors runs the full pipeline over the fixture and lints the result
// with the complete w3-06 rule set.
func scanErrors(t *testing.T) []engine.Finding {
	t.Helper()
	res, err := discovery.Scan(errorsRepo, discovery.ScanOptions{})
	if err != nil {
		t.Fatalf("Scan(%s): %v", errorsRepo, err)
	}
	reg := rules.NewRegistry()
	for _, build := range []func() ([]engine.Rule, error){rules.ErrorRules, rules.VersioningRules} {
		rs, err := build()
		if err != nil {
			t.Fatalf("build rules: %v", err)
		}
		for _, r := range rs {
			if err := reg.Register(r); err != nil {
				t.Fatalf("register %s: %v", r.ID, err)
			}
		}
	}
	report, err := engine.NewLinter(reg).Run(res.Surface, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return report.Findings
}

// TestErrorsFixtureFindsExactly pins the acceptance artifact: every w3-06
// rule has its positive sample here, every negative shape stays silent.
// R6xx-01 stays silent without a config enable (defaultDisabled, §3.6).
func TestErrorsFixtureFindsExactly(t *testing.T) {
	res, err := discovery.Scan(errorsRepo, discovery.ScanOptions{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(res.Surface.GrpcServices) != 2 {
		t.Fatalf("proto services = %d, want 2 (LibraryService, AuditService; the commented-out one excluded)", len(res.Surface.GrpcServices))
	}
	byFile := map[string]map[string]int{}
	for _, f := range scanErrors(t) {
		file := byFile[f.Location.File]
		if file == nil {
			file = map[string]int{}
			byFile[f.Location.File] = file
		}
		file[f.RuleID]++
	}
	want := map[string]map[string]int{
		// handleValidation returns Map<String,Object> while 3 handlers
		// share the ErrorResponse envelope (R5xx-01); handleBusiness maps
		// the app-declared BusinessException to 500 (R5xx-02, the ERROR).
		"src/main/java/com/example/shop/advice/GlobalExceptionHandler.java": {"R5xx-01": 1, "R5xx-02": 1},
		// create (explicit 200) and createV2 (implicit 200) on the
		// create-shaped path; the 201, the /cancel action and the GET are
		// the negatives.
		"src/main/java/com/example/shop/controller/OrderController.java": {"R5xx-03": 2},
		// DoMagic is the only off-pattern rpc (GetBook Standard, SyncData
		// VerbNoun, ListEvents Standard; the second service is compliant).
		"src/main/proto/library.proto": {"R6xx-02": 1},
	}
	for file := range byFile {
		if _, known := want[file]; !known {
			t.Fatalf("unexpected findings in %s: %v", file, byFile[file])
		}
	}
	for file, wantRules := range want {
		gotRules := byFile[file]
		if len(gotRules) != len(wantRules) {
			t.Fatalf("%s: rule counts = %v, want %v", file, gotRules, wantRules)
		}
		for id, n := range wantRules {
			if gotRules[id] != n {
				t.Errorf("%s: %s count = %d, want %d", file, id, gotRules[id], n)
			}
		}
	}
}

// TestErrorsFixtureAnchorsOnErrorScheme pins the w3-06 acceptance: the
// R5xx findings anchor on the ErrorScheme handlers w3-02 collected (the
// advice file), and the R6xx-02 finding anchors on the proto service
// declaration — the rule reads the IR, never a source file.
func TestErrorsFixtureAnchorsOnErrorScheme(t *testing.T) {
	findings := scanErrors(t)
	seen := map[string]string{} // ruleID → anchored file
	for _, f := range findings {
		seen[f.RuleID] = f.Location.File
	}
	if seen["R5xx-01"] != "src/main/java/com/example/shop/advice/GlobalExceptionHandler.java" {
		t.Errorf("R5xx-01 anchored at %q, want the advice file", seen["R5xx-01"])
	}
	if seen["R6xx-02"] != "src/main/proto/library.proto" {
		t.Errorf("R6xx-02 anchored at %q, want the proto service declaration", seen["R6xx-02"])
	}
}
