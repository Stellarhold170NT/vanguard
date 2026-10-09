package cli

import (
	"bytes"
	"strings"
	"testing"
)

// testBuild is the build identity tests run with; it must be visible in
// version output so the ldflags wiring is observable.
func testBuild() BuildInfo {
	return BuildInfo{Version: "0.2.0-test", Commit: "abc1234", Date: "2026-10-08T00:00:00Z"}
}

// TestRunVersionSubcommandPrintsBuildInfo pins the §6.1 version contract:
// one machine-friendly line with version, commit and build date.
func TestRunVersionSubcommandPrintsBuildInfo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, testBuild(), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := "vanguard 0.2.0-test (commit abc1234, built 2026-10-08T00:00:00Z)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// TestRunVersionFallsBackToDevWithoutBuildInfo pins the w2-06 brief rule:
// a build without git metadata reports dev/unknown and never fails.
func TestRunVersionFallsBackToDevWithoutBuildInfo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, BuildInfo{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := "vanguard dev (commit unknown, built unknown)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestRunNoArgsShowsHelp replaces the stub's version-on-no-args behaviour:
// with the full §6.1 tree, bare `vanguard` points people at the commands.
func TestRunNoArgsShowsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, testBuild(), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout = %q, want usage help", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// TestRunUnknownCommandExits2 pins the §6.3 misuse path: message on stderr,
// exit 2, stdout stays empty.
func TestRunUnknownCommandExits2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"deploy"}, testBuild(), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want an unknown-command message", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

// TestRunUnknownFlagExits2WithHint pins the w1-02 UX lesson: a flag error
// says what failed and where the usage lives.
func TestRunUnknownFlagExits2WithHint(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--frobnicate"}, testBuild(), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	stderrText := stderr.String()
	if !strings.Contains(stderrText, "--frobnicate") || !strings.Contains(stderrText, "vanguard scan --help") {
		t.Fatalf("stderr = %q, want the flag error plus a 'vanguard scan --help' hint", stderrText)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

// TestScanHelpDocumentsEveryFlag pins the acceptance criterion "help rõ
// ràng": scan's --help names every §6.2 flag with usage examples present.
func TestScanHelpDocumentsEveryFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--help"}, testBuild(), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	help := stdout.String()
	for _, flag := range []string{"--format", "--output", "--config", "--severity", "--no-color", "--verbose", "--rules", "--list-rules"} {
		if !strings.Contains(help, flag) {
			t.Errorf("--help output missing flag %s", flag)
		}
	}
	if !strings.Contains(help, "Example:") && !strings.Contains(help, "Examples:") {
		t.Error("--help output carries no usage examples")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// TestSubcommandHelpAvailable walks the remaining §6.1 subcommands: each
// must answer --help on stdout with exit 0.
func TestSubcommandHelpAvailable(t *testing.T) {
	for _, cmd := range []string{"check", "explain", "init", "version"} {
		t.Run(cmd, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run([]string{cmd, "--help"}, testBuild(), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if !strings.Contains(stdout.String(), "Usage:") {
				t.Fatalf("stdout = %q, want usage help", stdout.String())
			}
		})
	}
}
