package rules_test

// The R2xx scan-based end-to-end test (w3-04) lives in the EXTERNAL test
// package: internal/discovery wires the adapters this fixture exercises,
// so an in-package test would cycle. The fixture repo
// (testdata/methods-rules) carries one controller per rule with a positive
// and negative sample each; the test pins the exact finding set the six
// rules produce through the real walk → detect → parse → lint pipeline.

import (
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/discovery"
	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/rules"
)

const methodsFixtureDir = "../testdata/methods-rules"

// scanMethods lints the fixture with ONLY the R2xx family registered, so
// the assertions isolate this task's rules from the demo set and from the
// families other W3 tasks own.
func scanMethods(t *testing.T) map[string][]engineFinding {
	t.Helper()
	res, err := discovery.Scan(methodsFixtureDir, discovery.ScanOptions{})
	if err != nil {
		t.Fatalf("scan %s: %v", methodsFixtureDir, err)
	}
	if res.NoAPI != "" {
		t.Fatalf("no API surface detected in %s: %s", methodsFixtureDir, res.NoAPI)
	}
	methodRules, err := rules.MethodRules()
	if err != nil {
		t.Fatalf("MethodRules: %v", err)
	}
	reg := rules.NewRegistry()
	for _, r := range methodRules {
		if err := reg.Register(r); err != nil {
			t.Fatalf("register %s: %v", r.ID, err)
		}
	}
	rep, err := engine.NewLinter(reg).Run(res.Surface, nil)
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	out := map[string][]engineFinding{}
	for _, f := range rep.Findings {
		out[f.RuleID] = append(out[f.RuleID], engineFinding{
			file: f.Location.File, line: f.Location.Line, suggestion: f.Suggestion,
		})
	}
	return out
}

// engineFinding mirrors the finding fields the assertions read. The scan
// result is sorted by (file, line, col, ruleId), so per-rule slices keep
// source order.
type engineFinding struct {
	file       string
	line       int
	suggestion string
}

// TestMethodsFixtureFindings pins one finding per rule at the exact source
// line, and — the acceptance line of w3-04 — zero findings anywhere else:
// the negative samples in the same fixture must stay silent.
func TestMethodsFixtureFindings(t *testing.T) {
	got := scanMethods(t)
	want := map[string]engineFinding{
		"R2xx-01": {file: "src/main/java/com/example/methods/controller/GetBodyController.java", line: 19},
		"R2xx-02": {file: "src/main/java/com/example/methods/controller/CreateController.java", line: 26},
		"R2xx-03": {file: "src/main/java/com/example/methods/controller/PatchController.java", line: 16},
		"R2xx-04": {file: "src/main/java/com/example/methods/controller/DeleteController.java", line: 16},
		"R2xx-05": {file: "src/main/java/com/example/methods/controller/ActionController.java", line: 16},
		"R2xx-06": {file: "src/main/java/com/example/methods/controller/PutController.java", line: 17},
	}
	for id, w := range want {
		fs := got[id]
		if len(fs) != 1 {
			t.Errorf("%s: %d findings (%v), want exactly 1", id, len(fs), fs)
			continue
		}
		if fs[0].file != w.file || fs[0].line != w.line {
			t.Errorf("%s: finding at %s:%d, want %s:%d", id, fs[0].file, fs[0].line, w.file, w.line)
		}
	}
}

// TestMethodsFixtureSuggestion pins the R2xx-05 acceptance line: the
// suggestion states the AIP-136 replacement shape built from the real path.
func TestMethodsFixtureSuggestion(t *testing.T) {
	got := scanMethods(t)
	fs := got["R2xx-05"]
	if len(fs) != 1 {
		t.Fatalf("R2xx-05: %d findings, want 1", len(fs))
	}
	if want := "POST /api/v1/books/{id}:generate-code"; fs[0].suggestion != want {
		t.Fatalf("suggestion %q, want %q", fs[0].suggestion, want)
	}
}
