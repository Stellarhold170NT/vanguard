package cli

import (
	"bytes"
	"testing"
)

// TestRunPrintsVersionOnNoArgs keeps the skeleton binary behavior w1-06
// verified: bare invocation prints the version banner and exits 0.
func TestRunPrintsVersionOnNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run(nil, "0.1.0-dev", &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := stdout.String(); got != "vanguard 0.1.0-dev\n" {
		t.Fatalf("stdout = %q, want %q", got, "vanguard 0.1.0-dev\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// TestRunVersionSubcommand covers the one §6.1 subcommand the stub keeps.
func TestRunVersionSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version"}, "9.9.9", &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := stdout.String(); got != "vanguard 9.9.9\n" {
		t.Fatalf("stdout = %q, want %q", got, "vanguard 9.9.9\n")
	}
}

// TestRunUnknownCommandExits2 pins the stub's misuse path to the §6.3
// tool-error channel: message on stderr, exit code 2, nothing on stdout.
// w2-06 revisits the exact code when the cobra tree lands.
func TestRunUnknownCommandExits2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"scan"}, "0.1.0-dev", &stdout, &stderr); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("unknown-command message must go to stderr")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}
