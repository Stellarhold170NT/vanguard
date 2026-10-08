package cli

import (
	"strings"
	"testing"
)

// TestExplainPrintsMetadataAndExamples pins the §6.1 explain contract: the
// doc pointer, summary and the good/bad examples from metadata.
func TestExplainPrintsMetadataAndExamples(t *testing.T) {
	code, stdout, stderr := runCLI(t, "explain", "R6xx-99")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"R6xx-99",
		"demo-trailing-slash",
		"ERROR",
		"Path must not end with a trailing slash",
		"GET /api/v1/youth",
		"docs/rules/R6xx-99-demo-trailing-slash.md",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("explain output missing %q:\n%s", want, out)
		}
	}
}

// TestExplainUnknownRuleExitsTwo: an unknown id is a tool error that points
// at the catalog (§6.3 stderr discipline).
func TestExplainUnknownRuleExitsTwo(t *testing.T) {
	code, stdout, stderr := runCLI(t, "explain", "R6xx-43")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	stderrText := stderr.String()
	if !strings.Contains(stderrText, "unknown rule") || !strings.Contains(stderrText, "--list-rules") {
		t.Fatalf("stderr = %q, want the unknown-rule message with a --list-rules hint", stderrText)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}
