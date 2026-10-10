package engine

import (
	"strings"
	"testing"

	"github.com/vanguard-lint/vanguard/internal/ir"
)

// commentAt builds a one-line comment record at the given line.
func commentAt(file string, line int, text string) Comment {
	return Comment{Location: ir.Location{File: file, Line: line, Column: 5}, Text: text}
}

// findingAt builds a finding the way the engine stamps it before asking the
// suppression layer (RuleID/Severity set, Location = the node).
func findingAt(ruleID, file string, line int) Finding {
	return Finding{
		RuleID:   ruleID,
		Severity: SeverityError,
		Message:  "stub finding",
		Location: ir.Location{File: file, Line: line, Column: 10},
	}
}

const supFile = "src/main/java/YouthResource.java"

// TestInlineSameLineAndAbove pins §6.5: the ignore comment covers findings
// on its own line and on the line immediately below; a comment two lines
// above (or below) does not.
func TestInlineSameLineAndAbove(t *testing.T) {
	idx := CommentIndex{supFile: []Comment{
		commentAt(supFile, 12, "// vanguard:ignore R6xx-99 legacy route"),
	}}
	sup, diags := NewInlineSuppression(idx)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if ok, reason := sup.Suppressed(findingAt("R6xx-99", supFile, 12)); !ok || reason != "legacy route" {
		t.Fatalf("same line: suppressed=%v reason=%q, want true/legacy route", ok, reason)
	}
	if ok, reason := sup.Suppressed(findingAt("R6xx-99", supFile, 13)); !ok || reason != "legacy route" {
		t.Fatalf("line above comment: suppressed=%v reason=%q, want true/legacy route", ok, reason)
	}
	if ok, _ := sup.Suppressed(findingAt("R6xx-99", supFile, 14)); ok {
		t.Fatal("finding two lines below comment suppressed, want not suppressed")
	}
	if ok, _ := sup.Suppressed(findingAt("R6xx-99", supFile, 11)); ok {
		t.Fatal("finding above comment suppressed, want not suppressed")
	}
}

// TestInlineEmptyReason covers a directive without a reason.
func TestInlineEmptyReason(t *testing.T) {
	idx := CommentIndex{supFile: []Comment{
		commentAt(supFile, 12, "// vanguard:ignore R6xx-99"),
	}}
	sup, diags := NewInlineSuppression(idx)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if ok, reason := sup.Suppressed(findingAt("R6xx-99", supFile, 12)); !ok || reason != "" {
		t.Fatalf("suppressed=%v reason=%q, want true/\"\"", ok, reason)
	}
}

// TestInlineFamilyPrefixAndExactID pins id matching: a family prefix
// directive covers every rule of the family, an exact id only that rule
// (§6.5: "Match id đầy đủ hoặc prefix họ").
func TestInlineFamilyPrefixAndExactID(t *testing.T) {
	idx := CommentIndex{supFile: []Comment{
		commentAt(supFile, 10, "// vanguard:ignore R6xx family-wide"),
		commentAt(supFile, 20, "// vanguard:ignore R1xx-02 exact"),
	}}
	sup, diags := NewInlineSuppression(idx)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if ok, _ := sup.Suppressed(findingAt("R6xx-99", supFile, 10)); !ok {
		t.Fatal("family prefix did not cover R6xx-99")
	}
	if ok, _ := sup.Suppressed(findingAt("R1xx-02", supFile, 20)); !ok {
		t.Fatal("exact id directive did not cover R1xx-02")
	}
	if ok, _ := sup.Suppressed(findingAt("R1xx-02", supFile, 10)); ok {
		t.Fatal("R6xx directive suppressed an R1xx-02 finding")
	}
	if ok, _ := sup.Suppressed(findingAt("R6xx-01", supFile, 20)); ok {
		t.Fatal("R1xx-02 directive suppressed an R6xx-01 finding")
	}
}

// TestInlineBlock pins the begin/end block form: every line between begin
// and end (inclusive) is covered for the directive's rule scope.
func TestInlineBlock(t *testing.T) {
	idx := CommentIndex{supFile: []Comment{
		commentAt(supFile, 10, "// vanguard:ignore-begin R2xx bulk legacy"),
		commentAt(supFile, 20, "// vanguard:ignore-end R2xx"),
	}}
	sup, diags := NewInlineSuppression(idx)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	for _, line := range []int{10, 15, 20} {
		if ok, reason := sup.Suppressed(findingAt("R2xx-01", supFile, line)); !ok || reason != "bulk legacy" {
			t.Fatalf("line %d: suppressed=%v reason=%q, want true/bulk legacy", line, ok, reason)
		}
	}
	for _, line := range []int{9, 21} {
		if ok, _ := sup.Suppressed(findingAt("R2xx-01", supFile, line)); ok {
			t.Fatalf("line %d suppressed outside block", line)
		}
	}
	// Another family inside the block is not covered.
	if ok, _ := sup.Suppressed(findingAt("R1xx-02", supFile, 15)); ok {
		t.Fatal("R2xx block suppressed an R1xx-02 finding")
	}
}

// TestInlineBlockUnterminatedAndStrayEnd verifies the diagnostic pair that
// keeps blocks from silently suppressing to end-of-file: an ignore-begin
// without an ignore-end, and an ignore-end without an ignore-begin, both
// produce a diagnostic citing the file and line.
func TestInlineBlockUnterminatedAndStrayEnd(t *testing.T) {
	idx := CommentIndex{supFile: []Comment{
		commentAt(supFile, 10, "// vanguard:ignore-begin R2xx"),
		commentAt(supFile, 14, "// vanguard:ignore-end R1xx"),
	}}
	sup, diags := NewInlineSuppression(idx)
	if len(diags) != 2 {
		t.Fatalf("diagnostics = %+v, want 2 (unterminated begin + stray end)", diags)
	}
	joined := diags[0].Message + diags[1].Message
	if !strings.Contains(joined, "R2xx") || !strings.Contains(joined, "R1xx") {
		t.Fatalf("diagnostics do not name both rule refs: %+v", diags)
	}
	for _, d := range diags {
		if d.Location.File != supFile {
			t.Fatalf("diagnostic file = %q, want %q", d.Location.File, supFile)
		}
		if d.Location.Line != 10 && d.Location.Line != 14 {
			t.Fatalf("diagnostic line = %d, want 10 or 14", d.Location.Line)
		}
	}
	// The unterminated block still suppresses to end of file (lenient), so
	// the scan continues to be useful while the typo stays visible.
	if ok, _ := sup.Suppressed(findingAt("R2xx-01", supFile, 99)); !ok {
		t.Fatal("unterminated block did not cover lines below")
	}
	if ok, _ := sup.Suppressed(findingAt("R1xx-02", supFile, 14)); ok {
		t.Fatal("stray end covered findings for its rule")
	}
}

// TestInlineMalformedDirective: a directive whose rule ref is not a valid
// id/prefix must NOT suppress anything (fail-open: the finding still fires)
// and must surface a diagnostic with the comment location.
func TestInlineMalformedDirective(t *testing.T) {
	idx := CommentIndex{supFile: []Comment{
		commentAt(supFile, 12, "// vanguard:ignore oops typo"),
		commentAt(supFile, 15, "// vanguard:ignoreR6xx-99 no space"),
	}}
	sup, diags := NewInlineSuppression(idx)
	if len(diags) != 2 {
		t.Fatalf("diagnostics = %+v, want 2 malformed-directive diagnostics", diags)
	}
	if ok, _ := sup.Suppressed(findingAt("R6xx-99", supFile, 12)); ok {
		t.Fatal("malformed directive suppressed a finding")
	}
}

// TestInlinePerFileIsolation: directives in one file do not cover findings
// in another file, even at the same line.
func TestInlinePerFileIsolation(t *testing.T) {
	other := "src/main/java/OtherResource.java"
	idx := CommentIndex{supFile: []Comment{
		commentAt(supFile, 12, "// vanguard:ignore R6xx-99"),
	}}
	sup, diags := NewInlineSuppression(idx)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if ok, _ := sup.Suppressed(findingAt("R6xx-99", other, 12)); ok {
		t.Fatal("directive from another file suppressed a finding")
	}
}

// TestPathSuppressions covers the config-side path suppression (§6.4.1):
// rule id/prefix match AND file glob match, with the reason carried for the
// verbose note.
func TestPathSuppressions(t *testing.T) {
	ps, err := NewPathSuppressions([]PathSuppression{
		{Rule: "R1xx", Paths: []string{"src/legacy/**"}, Reason: "legacy module"},
		{Rule: "R2xx-01", Paths: []string{"**/generated/**"}},
	})
	if err != nil {
		t.Fatalf("NewPathSuppressions: %v", err)
	}
	if ok, reason := ps.Suppressed(findingAt("R1xx-02", "src/legacy/Old.java", 5)); !ok || reason != "legacy module" {
		t.Fatalf("legacy: suppressed=%v reason=%q, want true/legacy module", ok, reason)
	}
	if ok, _ := ps.Suppressed(findingAt("R1xx-02", "src/main/New.java", 5)); ok {
		t.Fatal("finding outside legacy glob suppressed")
	}
	if ok, _ := ps.Suppressed(findingAt("R2xx-02", "src/generated/Gen.java", 5)); ok {
		t.Fatal("exact-id entry suppressed a different rule of the family")
	}
	if ok, _ := ps.Suppressed(findingAt("R2xx-01", "src/generated/Gen.java", 5)); !ok {
		t.Fatal("exact-id entry did not cover its own rule in a generated path")
	}
	if _, err := NewPathSuppressions([]PathSuppression{{Rule: "nope", Paths: []string{"**"}}}); err == nil {
		t.Fatal("invalid rule ref accepted")
	}
	if _, err := NewPathSuppressions([]PathSuppression{{Rule: "R1xx", Paths: []string{"src/[ab"}}}); err == nil {
		t.Fatal("malformed glob accepted")
	}
}
