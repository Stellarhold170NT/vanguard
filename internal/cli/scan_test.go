package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cleanStubService is a minimal stub fixture with no rule violations.
const cleanStubService = `{ "service": "Clean",
  "basePath": "/v1/clean",
  "framework": "stub-http",
  "methods": [
    {"name": "getThing", "verb": "GET", "path": "/{id}",
     "params": [{"name": "id", "in": "path", "type": "string"}],
     "response": {"type": "Thing", "statusCode": 200}}
  ],
  "types": [
    {"name": "Thing", "kind": "record", "fields": [
      {"name": "id", "type": "string"}]}
  ]
}
`

// violatingStubService carries exactly one ERROR finding: a method path
// with a trailing slash (demo rule R6xx-99, the w2-04 data fixture).
const violatingStubService = `{ "service": "Violating",
  "basePath": "/v1/violating",
  "framework": "stub-http",
  "methods": [
    {"name": "getThing", "verb": "GET", "path": "/{id}/",
     "params": [{"name": "id", "in": "path", "type": "string"}],
     "response": {"type": "Thing", "statusCode": 200}}
  ],
  "types": [
    {"name": "Thing", "kind": "record", "fields": [
      {"name": "id", "type": "string"}]}
  ]
}
`

// writeStubRepo materializes files (relative path → content) under a fresh
// temp directory and returns the root, so every scan test stays hermetic:
// nothing outside the temp tree is read or written.
func writeStubRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return root
}

// writeConfig drops one config file under root (relative path → content).
func writeConfig(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// runCLI invokes Run with fresh output buffers and returns (exit code,
// stdout, stderr).
func runCLI(t *testing.T, args ...string) (int, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, testBuild(), &stdout, &stderr)
	return code, &stdout, &stderr
}

// TestScanCleanRepoExitsZero pins §6.3 code 0 on a clean scan.
func TestScanCleanRepoExitsZero(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/clean.stub.json": cleanStubService})
	code, stdout, stderr := runCLI(t, "scan", root)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("pretty scan must print the report to stdout")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty without --verbose", stderr.String())
	}
}

// TestScanViolationExitsOne pins §6.3 code 1: one ERROR finding → exit 1,
// and the pretty output names the rule.
func TestScanViolationExitsOne(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, stdout, stderr := runCLI(t, "scan", root)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "R6xx-99") {
		t.Fatalf("stdout = %q, want the R6xx-99 finding", stdout.String())
	}
}

// TestScanSARIFFileKeepsStdoutEmpty pins the w2-06 trap list: findings go
// to the --output file and stdout receives nothing (no banner, no noise).
func TestScanSARIFFileKeepsStdoutPure(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	out := filepath.Join(t.TempDir(), "out.sarif")
	code, stdout, stderr := runCLI(t, "scan", root, "--format", "sarif", "-o", out)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty when --output is used", stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read %s: %v", out, err)
	}
	sarif := string(data)
	if !strings.Contains(sarif, `"version": "2.1.0"`) {
		t.Fatalf("SARIF file missing the 2.1.0 version line:\n%s", sarif)
	}
	if !strings.Contains(sarif, "R6xx-99") {
		t.Fatalf("SARIF file missing the R6xx-99 result:\n%s", sarif)
	}
}

// TestScanJSONStdoutIsPureMachineOutput pins the §6.3 stdout discipline:
// json output starts immediately with the document, stays valid JSON and
// carries no ANSI escape bytes.
func TestScanJSONStdoutIsPureMachineOutput(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, stdout, stderr := runCLI(t, "scan", root, "--format", "json")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr.String())
	}
	out := stdout.String()
	if !json.Valid([]byte(out)) {
		t.Fatalf("stdout is not valid JSON:\n%s", out)
	}
	if trimmed := strings.TrimSpace(out); !strings.HasPrefix(trimmed, "{") {
		t.Fatalf("stdout starts with %q, want the JSON document", trimmed[:1])
	}
	if strings.ContainsRune(out, '\x1b') {
		t.Fatal("JSON stdout carries ANSI escape bytes")
	}
}

// TestScanUnknownFormatExitsTwo pins format validation before any work.
func TestScanUnknownFormatExitsTwo(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/clean.stub.json": cleanStubService})
	code, stdout, stderr := runCLI(t, "scan", root, "--format", "xml")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "xml") {
		t.Fatalf("stderr = %q, want the bad format named", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

// TestScanMissingConfigFileExitsTwo: --config pointing nowhere is a tool
// error with the file named (§6.3).
func TestScanMissingConfigFileExitsTwo(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/clean.stub.json": cleanStubService})
	code, stdout, stderr := runCLI(t, "scan", root, "--config", filepath.Join(t.TempDir(), "missing.yaml"))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "missing.yaml") {
		t.Fatalf("stderr = %q, want the missing config file named", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

// TestScanBrokenConfigSchemaExitsTwo passes the w2-04 contract through the
// CLI: schema errors cite file and line on stderr, findings never print.
func TestScanBrokenConfigSchemaExitsTwo(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/clean.stub.json": cleanStubService})
	cfg := writeConfig(t, t.TempDir(), ".vanguard.yaml", "version: 1\nbogusKey: true\n")
	code, stdout, stderr := runCLI(t, "scan", root, "--config", cfg)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), ".vanguard.yaml:2:") {
		t.Fatalf("stderr = %q, want a file:line schema error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

// TestScanRulesFilterRunsSelectedRule: --rules with the demo rule keeps the
// finding; the same rule disabled via config under the same filter exits 0.
func TestScanRulesFilterRunsSelectedRule(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, _, stderr := runCLI(t, "scan", root, "--rules", "R6xx-99")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr.String())
	}

	cfg := writeConfig(t, t.TempDir(), ".vanguard.yaml", "version: 1\nrules:\n  R6xx-99:\n    disabled: true\n")
	code, stdout, stderr := runCLI(t, "scan", root, "--rules", "R6xx-99", "--config", cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("pretty output expected even with zero findings")
	}
}

// TestScanRulesFilterRejectsUnknownReference fails loudly on a typo'd CI
// config instead of silently scanning nothing (exit 2).
func TestScanRulesFilterRejectsUnknownReference(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/clean.stub.json": cleanStubService})
	code, stdout, stderr := runCLI(t, "scan", root, "--rules", "R9xx-01")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "R9xx-01") {
		t.Fatalf("stderr = %q, want the unknown rule named", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

// TestScanSeverityThresholdIsDisplayOnly pins test-strategy §5.4 #6: the
// --severity flag changes what is displayed, never the exit code.
func TestScanSeverityThresholdIsDisplayOnly(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	cfg := writeConfig(t, t.TempDir(), ".vanguard.yaml", "version: 1\nrules:\n  R6xx-99:\n    severity: WARN\n")

	code, stdout, stderr := runCLI(t, "scan", root, "--config", cfg)
	if code != 0 {
		t.Fatalf("WARN-only findings must exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "R6xx-99") {
		t.Fatalf("default threshold INFO must display the WARN finding: %q", stdout.String())
	}

	code, stdout, _ = runCLI(t, "scan", root, "--config", cfg, "--severity", "ERROR")
	if code != 0 {
		t.Fatalf("--severity must not change the exit code, got %d", code)
	}
	if strings.Contains(stdout.String(), "R6xx-99") {
		t.Fatalf("--severity ERROR must hide the WARN finding: %q", stdout.String())
	}
}

// TestScanVerboseEvidenceOnStderr pins the --verbose contract: evidence
// goes to stderr with the [vanguard] trail while stdout stays pure JSON.
func TestScanVerboseEvidenceOnStderr(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, stdout, stderr := runCLI(t, "scan", root, "--format", "json", "--verbose")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !json.Valid([]byte(stdout.String())) {
		t.Fatalf("stdout is not valid JSON:\n%s", stdout.String())
	}
	for _, want := range []string{"[vanguard] walk:", "[vanguard] selected adapter: stub"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want it to contain %q", stderr.String(), want)
		}
	}
}

// TestScanNoAPIExitsZeroWithNotice pins §6.3: a repo without an API
// surface is a clean exit 0 with a clear message — on the human channel
// for pretty, on stderr for machine formats.
func TestScanNoAPIExitsZeroWithNotice(t *testing.T) {
	empty := t.TempDir()
	code, stdout, stderr := runCLI(t, "scan", empty)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nothing to check") {
		t.Fatalf("pretty stdout = %q, want the no-API notice", stdout.String())
	}

	code, stdout, stderr = runCLI(t, "scan", empty, "--format", "json")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("json stdout = %q, want empty (notice must go to stderr)", stdout.String())
	}
	if !strings.Contains(stderr.String(), "nothing to check") {
		t.Fatalf("json stderr = %q, want the no-API notice", stderr.String())
	}
}

// TestScanMissingRootExitsTwo: a scan root that cannot be walked is a tool
// error (§6.3 code 2), not a finding.
func TestScanMissingRootExitsTwo(t *testing.T) {
	code, stdout, stderr := runCLI(t, "scan", filepath.Join(t.TempDir(), "does-not-exist"))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "does-not-exist") {
		t.Fatalf("stderr = %q, want the missing root named", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

// TestScanDiscoversConfigFromParentDir pins §6.4.2 #1: the nearest
// .vanguard.yaml above the scan target applies.
func TestScanDiscoversConfigFromParentDir(t *testing.T) {
	parent := writeStubRepo(t, map[string]string{"repo/src/violating.stub.json": violatingStubService})
	writeConfig(t, parent, ".vanguard.yaml", "version: 1\nrules:\n  R6xx-99:\n    disabled: true\n")
	code, _, stderr := runCLI(t, "scan", filepath.Join(parent, "repo"))
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
}

// TestScanConfigAndNoConfigMutuallyExclusive guards the flag pair.
func TestScanConfigAndNoConfigMutuallyExclusive(t *testing.T) {
	root := writeStubRepo(t, map[string]string{"src/clean.stub.json": cleanStubService})
	code, _, stderr := runCLI(t, "scan", root, "--config", "x.yaml", "--no-config")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("stderr = %q, want the conflict explanation", stderr.String())
	}
}

// TestScanNoColorForcesPlainOutputOnTTY: --no-color must strip ANSI bytes
// even when the output would be colored (simulated TTY).
func TestScanNoColorForcesPlainOutputOnTTY(t *testing.T) {
	restore := simulateTTY(true)
	defer restore()
	root := writeStubRepo(t, map[string]string{"src/violating.stub.json": violatingStubService})
	code, stdout, _ := runCLI(t, "scan", root, "--no-color")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if strings.ContainsRune(stdout.String(), '\x1b') {
		t.Fatal("pretty stdout carries ANSI escape bytes under --no-color")
	}
}

// TestScanAcceptanceStubRepoSARIF is the literal w2-06 acceptance command:
// `vanguard scan testdata/stub-repo --format sarif -o out.sarif` runs and
// exits with the standard code (the fixture is clean → 0).
func TestScanAcceptanceStubRepoSARIF(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.sarif")
	code, stdout, stderr := runCLI(t, "scan", "../../testdata/stub-repo", "--format", "sarif", "-o", out)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty when --output is used", stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read %s: %v", out, err)
	}
	if !strings.Contains(string(data), `"version": "2.1.0"`) {
		t.Fatalf("SARIF file missing the 2.1.0 version line:\n%s", data)
	}
}
