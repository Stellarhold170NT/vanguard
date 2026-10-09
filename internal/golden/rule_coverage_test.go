package golden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The W3 rule-coverage contract (w3-07): the java-spring showcase must
// produce at least one finding for every registered rule, and its clean
// companion must stay silent on all of them — so a rule that lands without
// a golden fixture (or one that starts firing on convention-following
// code) fails this test, not a demo.
//
// R6xx-99 is the one documented exemption: its trigger (a raw trailing
// slash) is normalized away by the IR path contract — the spring adapter's
// mergePath joins with path.Join, and the stub merge cleans it too (see
// internal/cli/scan_test.go) — so no fixture repo can express it. The rule
// is pinned by rules/demo_test.go on hand-built IR.

// ruleIDsFromCatalog runs `vanguard scan --list-rules` — the public
// registry surface (§6.2) — and returns the sorted rule ids. Reading the
// catalog through the binary (not by importing the rules package) keeps
// this file inside the harness's exec-only contract.
func ruleIDsFromCatalog(t *testing.T, bin, root string) []string {
	t.Helper()
	res, err := runBinary(bin, root, []string{"scan", "--list-rules"})
	if err != nil {
		t.Fatalf("run --list-rules: %v", err)
	}
	if res.Exit != 0 {
		t.Fatalf("--list-rules exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", res.Exit, res.Stdout, res.Stderr)
	}
	var ids []string
	for _, line := range strings.Split(string(res.Stdout), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) >= 4 && strings.HasPrefix(parts[0], "R") {
			ids = append(ids, parts[0])
		}
	}
	if len(ids) == 0 {
		t.Fatalf("--list-rules printed no rule ids\nstdout:\n%s", res.Stdout)
	}
	return ids
}

// findingRuleIDs unmarshals the §6.7 json document and returns the set of
// rule ids that produced at least one finding.
func findingRuleIDs(t *testing.T, stdout []byte) map[string]bool {
	t.Helper()
	var doc struct {
		Findings []struct {
			RuleID string `json:"ruleId"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout, &doc); err != nil {
		t.Fatalf("parse json findings: %v\nstdout:\n%s", err, stdout)
	}
	out := make(map[string]bool, len(doc.Findings))
	for _, f := range doc.Findings {
		out[f.RuleID] = true
	}
	return out
}

func TestJavaSpringGoldenCoversEveryRule(t *testing.T) {
	bin, err := BinaryPath()
	if err != nil {
		t.Fatalf("resolve fixture binary: %v", err)
	}
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	cases, err := LoadCases(filepath.Join(root, "testdata", "golden"))
	if err != nil {
		t.Fatalf("load golden cases: %v", err)
	}
	var showcase, clean *Case
	for i := range cases {
		switch cases[i].ID {
		case "java-spring":
			showcase = &cases[i]
		case "java-spring-clean":
			clean = &cases[i]
		}
	}
	if showcase == nil || clean == nil {
		t.Fatalf("golden: java-spring showcase cases missing (java-spring=%v, java-spring-clean=%v)", showcase != nil, clean != nil)
	}

	res, err := showcase.Run(bin, root, "json")
	if err != nil {
		t.Fatalf("run java-spring: %v", err)
	}
	if res.Exit != 1 {
		t.Fatalf("java-spring exit = %d, want 1 (the showcase must carry ERROR findings)\nstdout:\n%s", res.Exit, res.Stdout)
	}
	fired := findingRuleIDs(t, res.Stdout)

	// R6xx-99 is the documented adapter-unreachable exemption (see the
	// package comment above).
	exempt := map[string]bool{"R6xx-99": true}

	catalog := ruleIDsFromCatalog(t, bin, root)
	var missing []string
	for _, id := range catalog {
		if !fired[id] && !exempt[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("java-spring showcase: %d registered rule(s) have no violating fixture: %s — add a fixture endpoint or document an exemption in internal/golden (w3-07 contract)", len(missing), strings.Join(missing, ", "))
	}

	res, err = clean.Run(bin, root, "json")
	if err != nil {
		t.Fatalf("run java-spring-clean: %v", err)
	}
	if res.Exit != 0 {
		t.Fatalf("java-spring-clean exit = %d, want 0 (the clean rewrite must stay silent)\nstdout:\n%s", res.Exit, res.Stdout)
	}
	if len(findingRuleIDs(t, res.Stdout)) != 0 {
		t.Fatalf("java-spring-clean produced findings — the clean rewrite must satisfy every rule:\n%s", res.Stdout)
	}
}

// TestJavaSpringSnapshotsExist pins the w3-07 format triple: both java
// showcase cases snapshot pretty, json AND sarif (charter §6.2), so the
// README's "3 format" claim is checkable.
func TestJavaSpringSnapshotsFormats(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	for _, id := range []string{"java-spring", "java-spring-clean"} {
		for _, format := range []string{"pretty", "json", "sarif"} {
			path := filepath.Join(root, "testdata", "golden", id, "snapshot."+format)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("%s: %s missing (run make golden-update): %v", id, filepath.Base(path), err)
			}
		}
	}
}
