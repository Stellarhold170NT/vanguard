package cli

import (
	"strings"
	"testing"
)

// TestCheckSuppressesPrettyOnNonTTY pins the §6.1 check contract: on a
// non-TTY stdout with the default pretty format, findings are suppressed —
// one summary line on stderr carries the verdict drivers.
func TestCheckSuppressesPrettyOnNonTTY(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, stdout, stderr := runCLI(t, "check", root)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty (non-TTY pretty must be suppressed)", stdout.String())
	}
	for _, want := range []string{"vanguard check", "1 ERROR"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want it to contain %q", stderr.String(), want)
		}
	}
}

// TestCheckExplicitJSONStillRenders pins the explicit-format exception:
// --format json on a non-TTY renders the full machine report to stdout.
func TestCheckExplicitJSONStillRenders(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, stdout, _ := runCLI(t, "check", root, "--format", "json")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "R6xx-99") {
		t.Fatalf("stdout = %q, want the machine-readable findings", stdout.String())
	}
}

// TestCheckCleanRepoExitsZero: the summary line must also carry the clean
// verdict so a green CI log still explains itself.
func TestCheckCleanRepoExitsZero(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/clean.stub.json": cleanStubService})
	code, stdout, stderr := runCLI(t, "check", root)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "0 ERROR") {
		t.Fatalf("stderr = %q, want the 0-ERROR summary", stderr.String())
	}
}

// TestCheckOnTTYRendersPretty pins the other half of §6.1: an interactive
// check still shows the full pretty report (simulated TTY).
func TestCheckOnTTYRendersPretty(t *testing.T) {
	restore := simulateTTY(true)
	defer restore()
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, stdout, _ := runCLI(t, "check", root)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "R6xx-99") {
		t.Fatalf("stdout = %q, want the pretty findings on a TTY", stdout.String())
	}
}
