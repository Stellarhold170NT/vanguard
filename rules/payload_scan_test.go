// The external half of the w3-05 test set (package rules_test): it runs the
// six w3-05 rules through the REAL pipeline — discovery walk → Spring
// adapter → IR → engine — over the testdata/spring-payload fixture, pinning
// the exact finding set per file. The in-package tests (pagination_test.go,
// payload_test.go) pin the per-rule heuristics; this file pins that the
// checks derive everything from the IR the adapter emitted (the w3-05
// acceptance: no rule ever re-opens a .java file).
package rules_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/discovery"
	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/rules"
)

const payloadRepo = "../testdata/spring-payload"

// scanPayload runs the full pipeline over the fixture and lints the result
// with the complete w3-05 rule set.
func scanPayload(t *testing.T) []engine.Finding {
	t.Helper()
	res, err := discovery.Scan(payloadRepo, discovery.ScanOptions{})
	if err != nil {
		t.Fatalf("Scan(%s): %v", payloadRepo, err)
	}
	reg := rules.NewRegistry()
	for _, build := range []func() ([]engine.Rule, error){rules.PaginationRules, rules.PayloadRules} {
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

// findingsByFile groups findings by the file they anchor on, keyed rule →
// count (the shapes are asserted, not raw line numbers — the adapter owns
// the exact positions and pins them in its own tests).
func findingsByFile(findings []engine.Finding) map[string]map[string]int {
	out := map[string]map[string]int{}
	for _, f := range findings {
		file := map[string]int{}
		if out[f.Location.File] != nil {
			file = out[f.Location.File]
		}
		file[f.RuleID]++
		out[f.Location.File] = file
	}
	return out
}

// TestPayloadFixtureFindsExactly pins the acceptance artifact: every w3-05
// rule has its positive sample here, every negative shape stays silent.
func TestPayloadFixtureFindsExactly(t *testing.T) {
	byFile := findingsByFile(scanPayload(t))
	want := map[string]map[string]int{
		// recent(limit @Max(200)): bare List envelope gap only — the cap
		// silences R3xx-03, the params style silences R3xx-01.
		"src/main/java/com/example/payload/audit/AuditController.java": {"R3xx-02": 1},
		// current() returns the @Entity Customer directly — the family's
		// only ERROR.
		"src/main/java/com/example/payload/customer/CustomerController.java": {"R4xx-01": 1},
		// created_by (snake_case) + updatedAt (java.util.Date); startDate as
		// LocalDate and recordedAt as Instant are the time-field negatives.
		"src/main/java/com/example/payload/dto/AuditDto.java": {"R4xx-02": 1, "R4xx-03": 1},
		// listOrders: unpaginated + bare List (both R3xx-01 and R3xx-02);
		// page(Pageable) + Page<> silent; search: bare List (R3xx-02) and an
		// uncapped size param (R3xx-03) — the params style silences R3xx-01.
		"src/main/java/com/example/payload/order/OrderController.java": {"R3xx-01": 1, "R3xx-02": 2, "R3xx-03": 1},
		// The Youth DTO matches a typical entity NAME but no entity is in
		// scope — R4xx-01 must not fire on a name alone.
		"src/main/java/com/example/payload/web/YouthController.java": {},
		// The entity's snake_case field is out of R4xx-02's DTO scope.
		"src/main/java/com/example/payload/domain/Customer.java": {},
	}
	if len(byFile) != len(want) {
		t.Fatalf("files with findings = %v, want exactly the %d fixture files", byFile, len(want))
	}
	for file, wantRules := range want {
		gotRules, ok := byFile[file]
		if !ok {
			t.Fatalf("no findings for %s: %v", file, byFile)
		}
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

// TestPayloadFixtureMessagesAreActionable keeps the §3.0 contract visible
// end to end: reason first, concrete fix in the suggestion.
func TestPayloadFixtureMessagesAreActionable(t *testing.T) {
	findings := scanPayload(t)
	for _, f := range findings {
		if !strings.HasSuffix(strings.TrimSpace(f.Message), ".") {
			t.Errorf("%s message lacks sentence form: %q", f.RuleID, f.Message)
		}
	}
	byID := map[string]string{}
	for _, f := range findings {
		byID[f.RuleID] = f.Message
	}
	if !strings.Contains(byID["R4xx-01"], "Customer") {
		t.Errorf("R4xx-01 should name the exposed entity: %q", byID["R4xx-01"])
	}
	if !strings.Contains(byID["R3xx-03"], "size") {
		t.Errorf("R3xx-03 should name the offending parameter: %q", byID["R3xx-03"])
	}
}

// TestRuleSourceReadsIROnly pins the architecture boundary (w3-05
// acceptance): the check implementations import nothing beyond the engine
// and IR packages — no adapter, no tree-sitter, no file I/O.
func TestRuleSourceReadsIROnly(t *testing.T) {
	for _, src := range []string{"pagination.go", "payload.go", "pagination_test.go", "payload_test.go"} {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		for _, banned := range []string{
			"github.com/Stellarhold170NT/vanguard/adapters/",
			"github.com/smacker/go-tree-sitter",
			"os.ReadFile", "ioutil.",
		} {
			if strings.Contains(string(data), banned) {
				t.Errorf("%s must not reference %s — rules read the IR only", src, banned)
			}
		}
	}
}
