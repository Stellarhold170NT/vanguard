package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func slotAt(id, rule, file string, line int) slot {
	return slot{ID: id, Rule: rule, File: file, Line: line}
}

func findingAt(rule, file string, line int) finding {
	f := finding{RuleID: rule}
	f.Location.File = file
	f.Location.Line = line
	f.Message = "msg " + rule
	return f
}

// ---------------------------------------------------------------------------
// Alignment

func TestAlignMatchesSlotsWithinTolerance(t *testing.T) {
	exp := expectations{Sample: "s", Slots: []slot{
		slotAt("as-001", "R1xx-02", "src/A.java", 27),
		slotAt("as-002", "R2xx-03", "src/B.java", 41),
	}}
	rep := scanReport{Findings: []finding{
		findingAt("R1xx-02", "src/A.java", 25), // within +-4 -> matched
		findingAt("R2xx-03", "src/B.java", 46), // 5 lines away -> NOT matched
	}}
	rows, fnRows := align(rep, exp, []string{"src/A.java", "src/B.java"})
	// two finding rows: the matched A.java span + the unexpected B.java
	// firing (5 lines off the slot — becomes an unexpected row, the gate
	// labels it), plus one fn candidate for the unmatched slot
	if len(rows) != 2 {
		t.Fatalf("want 2 finding rows, got %+v", rows)
	}
	if rows[0].ID != "F-001" || rows[0].File != "src/A.java" {
		t.Fatalf("first row should be the matched A.java span, got %+v", rows[0])
	}
	if len(rows[0].FromSlots) != 1 || rows[0].FromSlots[0] != "as-001" {
		t.Fatalf("row should carry slot as-001, got %v", rows[0].FromSlots)
	}
	if len(rows[1].FromSlots) != 0 {
		t.Fatalf("B.java firing must be an unexpected row, got %v", rows[1].FromSlots)
	}
	if len(fnRows) != 1 || fnRows[0].ID != "as-002" {
		t.Fatalf("unmatched slot must become fn candidate as-002, got %v", fnRows)
	}
	if fnRows[0].Kind != "fn-candidate" || fnRows[0].Rules[0] != "R2xx-03" {
		t.Fatalf("fn candidate shape wrong: %+v", fnRows[0])
	}
}

func TestAlignMergesSameSpanRulesIntoOneRow(t *testing.T) {
	exp := expectations{Sample: "s", Slots: []slot{
		slotAt("as-001", "R1xx-02", "src/A.java", 23),
		slotAt("as-002", "R1xx-03", "src/A.java", 23),
		slotAt("as-003", "R2xx-05", "src/A.java", 23),
	}}
	rep := scanReport{Findings: []finding{
		findingAt("R1xx-02", "src/A.java", 23),
		findingAt("R1xx-03", "src/A.java", 23),
		findingAt("R2xx-05", "src/A.java", 23),
	}}
	rows, fnRows := align(rep, exp, []string{"src/A.java"})
	if len(fnRows) != 0 {
		t.Fatalf("all slots should match, got fn rows %v", fnRows)
	}
	if len(rows) != 1 {
		t.Fatalf("same-span findings must merge into one row, got %d", len(rows))
	}
	if got := strings.Join(rows[0].Rules, ","); got != "R1xx-02,R1xx-03,R2xx-05" {
		t.Fatalf("row must carry all three rules, got %q", got)
	}
	if len(rows[0].FromSlots) != 3 {
		t.Fatalf("row must reference all three slots, got %v", rows[0].FromSlots)
	}
}

func TestAlignRuleMismatchBecomesFnCandidate(t *testing.T) {
	exp := expectations{Sample: "s", Slots: []slot{
		slotAt("as-001", "R5xx-01", "src/A.java", 44),
	}}
	rep := scanReport{Findings: []finding{
		findingAt("R1xx-02", "src/A.java", 44), // same span, wrong rule
	}}
	rows, fnRows := align(rep, exp, []string{"src/A.java"})
	if len(rows) != 1 || len(rows[0].FromSlots) != 0 {
		t.Fatalf("finding must not adopt the mismatched slot: %+v", rows)
	}
	if len(fnRows) != 1 {
		t.Fatalf("slot must survive as fn candidate")
	}
}

// ---------------------------------------------------------------------------
// Verdict folding (the precision/recall math)

func inventory() (*results, []row, []row) {
	res := &results{Schema: "t", Labeled: true}
	res.Aggregate.Precision, res.Aggregate.Recall = "n/a", "n/a"
	res.PerRule = []ruleMetrics{
		{Rule: "R1xx-02", Claimed: 3, Observed: 3, Precision: "n/a", Recall: "n/a"},
		{Rule: "R2xx-01", Claimed: 1, Observed: 2, Precision: "n/a", Recall: "n/a"},
		{Rule: "R3xx-01", Claimed: 1, Observed: 1, Precision: "n/a", Recall: "n/a"},
	}
	rows := []row{
		{ID: "F-001", Kind: "finding", Rules: []string{"R1xx-02"}},
		{ID: "F-002", Kind: "finding", Rules: []string{"R1xx-02"}},
		{ID: "F-003", Kind: "finding", Rules: []string{"R2xx-01"}},
		{ID: "F-004", Kind: "finding", Rules: []string{"R2xx-01"}},
		{ID: "F-005", Kind: "finding", Rules: []string{"R3xx-01"}},
	}
	fnRows := []row{
		{ID: "as-003", Kind: "fn-candidate", Rules: []string{"R1xx-02"}},
	}
	return res, rows, fnRows
}

func TestApplyVerdictsMath(t *testing.T) {
	res, rows, fnRows := inventory()
	labeled := map[string]labeledRow{
		"F-001": {GateVerdict: "TP"},
		"F-002": {GateVerdict: "FP", ReasonCode: "heuristic-context"},
		"F-003": {GateVerdict: "TP"},
		"F-004": {GateVerdict: "TP"},
		"F-005": {GateVerdict: "FP", ReasonCode: "app-convention"},
		"as-003": {GateVerdict: "FN", ReasonCode: "predicate-gap"},
	}
	applyVerdicts(res, rows, fnRows, labeled)

	byRule := map[string]ruleMetrics{}
	for _, m := range res.PerRule {
		byRule[m.Rule] = m
	}
	if m := byRule["R1xx-02"]; m.TP != 1 || m.FP != 1 || m.FN != 1 || m.Precision != "50.0%" || m.Recall != "50.0%" {
		t.Fatalf("R1xx-02 math wrong: %+v", m)
	}
	if m := byRule["R2xx-01"]; m.TP != 2 || m.FP != 0 || m.Precision != "100.0%" {
		t.Fatalf("R2xx-01 math wrong: %+v", m)
	}
	if m := byRule["R3xx-01"]; m.TP != 0 || m.FP != 1 || m.Precision != "0.0%" || m.Recall != "n/a" {
		t.Fatalf("R3xx-01 math wrong (recall must be n/a, 0/0): %+v", m)
	}
	if res.Aggregate.TP != 3 || res.Aggregate.FP != 2 || res.Aggregate.FN != 1 {
		t.Fatalf("aggregate counts wrong: %+v", res.Aggregate)
	}
	if res.Aggregate.Precision != "60.0%" || res.Aggregate.Recall != "75.0%" {
		t.Fatalf("aggregate ratios wrong: %+v", res.Aggregate)
	}
	if got := res.Aggregate.FPShare; got < 39.9 || got > 40.1 {
		t.Fatalf("fp share wrong: %v", got)
	}
	if len(res.Unresolved) != 0 {
		t.Fatalf("no rows should be unresolved, got %v", res.Unresolved)
	}
}

func TestApplyVerdictsUnresolvedAndDisputed(t *testing.T) {
	res, rows, fnRows := inventory()
	labeled := map[string]labeledRow{
		"F-001":  {GateVerdict: "TP"},
		"F-002":  {GateVerdict: "UNCLEAR", Note: "need context"},
		"F-003":  {GateVerdict: "DISPUTED", Note: "charter vs aip.dev"},
		"as-003": {GateVerdict: "CLAIM-REJECTED", Note: "claim misread the rule"},
		// F-004, F-005 unlabeled entirely
	}
	applyVerdicts(res, rows, fnRows, labeled)
	want := []string{"F-002", "F-003", "F-004", "F-005"}
	if strings.Join(res.Unresolved, ",") != strings.Join(want, ",") {
		t.Fatalf("unresolved = %v, want %v", res.Unresolved, want)
	}
	if len(res.Disputes) != 1 || res.Disputes[0] != "F-003" {
		t.Fatalf("disputes = %v, want [F-003]", res.Disputes)
	}
	if res.Aggregate.TP != 1 || res.Aggregate.FP != 0 {
		t.Fatalf("only resolved rows must count: %+v", res.Aggregate)
	}
}

func TestValidateLabelsRejectsWrongKindVerdict(t *testing.T) {
	rows := []row{{ID: "F-001", Kind: "finding", Rules: []string{"R1xx-02"}}}
	fnRows := []row{{ID: "as-001", Kind: "fn-candidate", Rules: []string{"R1xx-02"}}}
	if err := validateLabels(map[string]labeledRow{"F-001": {GateVerdict: "FN"}}, rows, fnRows); err == nil {
		t.Fatal("FN on a finding row must be rejected")
	}
	if err := validateLabels(map[string]labeledRow{"as-001": {GateVerdict: "TP"}}, rows, fnRows); err == nil {
		t.Fatal("TP on a fn-candidate row must be rejected")
	}
	if err := validateLabels(map[string]labeledRow{"F-001": {GateVerdict: "FP"}, "as-001": {GateVerdict: "FN"}}, rows, fnRows); err != nil {
		t.Fatalf("valid labels must pass: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CSV round trip

func TestCSVRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "labels.csv")
	rows := []row{
		{ID: "F-001", Kind: "finding", Rules: []string{"R1xx-02", "R1xx-03"}, File: "src/A.java", Line: 23,
			Message: "msg | with pipe", FromSlots: []string{"as-001"}, AipRef: "https://google.aip.dev/131"},
	}
	fnRows := []row{{ID: "as-009", Kind: "fn-candidate", Rules: []string{"R5xx-01"}, File: "src/B.java", Line: 44, Message: "claim"}}
	if err := writeCSVFile(path, rows, fnRows); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"R1xx-02, R1xx-03"`) {
		t.Fatal("csv writer must quote the comma-carrying rules field")
	}

	// the labeler fills the verdict columns in place: the three empty
	// verdict fields (gate_verdict, reason_code, note) become three values
	edited := strings.Replace(string(raw), ",,,https://google.aip.dev/131", "TP,heuristic-context,lexicon too wide,https://google.aip.dev/131", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readLabeledCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["F-001"].GateVerdict != "TP" || got["F-001"].ReasonCode != "heuristic-context" {
		t.Fatalf("labels lost in round trip: %+v", got["F-001"])
	}
	if _, ok := got["as-009"]; ok && got["as-009"].GateVerdict != "" {
		t.Fatalf("unlabeled row must carry empty verdict: %+v", got["as-009"])
	}

	// unknown ids are a stale sheet, not a measurement: the id check
	// lives in validateLabels (rows are fixed by align)
	if err := validateLabels(map[string]labeledRow{"ZZZ-001": {GateVerdict: "TP"}}, rows, fnRows); err == nil {
		t.Fatal("unknown row id must be rejected as stale")
	}
}

// ---------------------------------------------------------------------------
// Inventory + normalization

func TestCountEndpointsAndRpc(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "web"), 0o755)
	java := "package x;\n@RestController\nclass C {\n @GetMapping(\"/a\") void a() {}\n @PostMapping void b() {}\n}\n"
	_ = os.WriteFile(filepath.Join(dir, "web", "A.java"), []byte(java), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "web", "B.java"), []byte(java), 0o644)
	proto := "syntax = \"proto3\";\nservice S {\n  rpc GetThing(R) returns (T);\n}\n// rpc Commented(R) returns (T);\n"
	_ = os.WriteFile(filepath.Join(dir, "s.proto"), []byte(proto), 0o644)
	http, rpc := countEndpoints(dir)
	if http != 4 {
		t.Fatalf("http endpoints = %d, want 4 (2 per file)", http)
	}
	if rpc != 1 {
		t.Fatalf("rpc = %d, want 1 (commented rpc must not count)", rpc)
	}
}

func TestNormalizeFileSuffixMatch(t *testing.T) {
	files := []string{"src/main/java/x/A.java", "proto/s.proto"}
	if got := normalizeFile("/abs/repo/testdata/audit-sample/src/main/java/x/A.java", files); got != "src/main/java/x/A.java" {
		t.Fatalf("absolute path must map by suffix, got %q", got)
	}
	if got := normalizeFile("./src/main/java/x/A.java", files); got != "src/main/java/x/A.java" {
		t.Fatalf("./ prefix must be stripped, got %q", got)
	}
	if got := normalizeFile("proto/s.proto", files); got != "proto/s.proto" {
		t.Fatalf("exact match must win, got %q", got)
	}
	if got := normalizeFile("elsewhere/unknown.java", files); got != "elsewhere/unknown.java" {
		t.Fatalf("unknown path must pass through, got %q", got)
	}
}
