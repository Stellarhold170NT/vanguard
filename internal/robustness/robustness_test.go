package robustness

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vanguard-lint/vanguard/internal/golden"
)

// Hostile-input test windows. Each is generous for a clean machine but far
// below the pre-fix behaviour of the cases that found real defects (the
// glob-bomb scan hung unbounded before the matcher fix).
const (
	// quickCase bounds normal-shape scans (traversal, symlinks, unicode,
	// parse bombs).
	quickCase = 60 * time.Second
	// hostileConfigCase bounds the glob-bomb scan; pre-fix it never
	// finished, post-fix it is well under a second.
	hostileConfigCase = 60 * time.Second
)

// violatingController fires R1xx-01 (singular collection path, WARN,
// default-enabled) under the default config on every fixture it lands in.
const violatingController = `package com.example.hostile;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** R1xx-01 bait: a singular collection path on a GET list endpoint. */
@RestController
@RequestMapping("/user")
public class ViolatingController {

    @GetMapping
    public List<String> list() {
        return null;
    }
}
`

// minimalPom pins the Spring adapter marker (discovery: pom.xml → java).
const minimalPom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>hostile-fixture</artifactId>
  <version>0.0.1</version>
</project>
`

// runVanguard execs the real binary (the golden harness contract: exactly
// the way CI invokes it) with an explicit timeout. A timeout is a test
// failure — for hostile inputs "finished" and "bounded" are the same
// requirement.
func runVanguard(t *testing.T, workDir string, timeout time.Duration, args ...string) (int, string, string) {
	t.Helper()
	bin, err := golden.BinaryPath()
	if err != nil {
		t.Fatalf("build fixture binary: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("vanguard %v timed out after %s — hostile input must finish in bounded time", args, timeout)
	}
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("run vanguard %v: %v\nstderr:\n%s", args, err, stderr.String())
		}
	}
	assertNoPanic(t, stderr.String())
	return exitCodeOf(t, err), stdout.String(), stderr.String()
}

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	t.Fatalf("unexpected run error: %v", err)
	return -1
}

// assertNoPanic fails on Go runtime crash markers: a panic exits 2 — the
// same code as a tool error — so the exit code alone cannot distinguish
// "clean tool error" from "crashed".
func assertNoPanic(t *testing.T, stderr string) {
	t.Helper()
	for _, marker := range []string{"panic: ", "runtime error: ", "goroutine ", "fatal error: "} {
		if strings.Contains(stderr, marker) {
			t.Fatalf("vanguard crashed (stderr contains %q):\n%s", marker, stderr)
		}
	}
}

// assertNormalExit pins the §6.3 tool contract: the only legal exits are
// 0 (clean/WARN-INFO), 1 (ERROR findings) and 2 (tool error).
func assertNormalExit(t *testing.T, exit int) {
	t.Helper()
	if exit != 0 && exit != 1 && exit != 2 {
		t.Fatalf("exit code %d is outside the §6.3 contract (0/1/2)", exit)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// springTree lays out a minimal scan root: pom.xml + one violating
// controller under src/. Extra files are added by the callers.
func springTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "pom.xml"), minimalPom)
	write(t, filepath.Join(root, "src", "com", "example", "ViolatingController.java"), violatingController)
	return root
}

// ---------------------------------------------------------------------------
// 1. Path traversal
// ---------------------------------------------------------------------------

// TestTraversalConfigGlobsCannotEscapeRoot feeds the config the `../`
// shapes a hostile repo would try: include/exclude/suppression globs that
// point above the config directory, next to a violating file placed
// OUTSIDE the scan root. The scanner must behave as if the outside file
// did not exist: findings carry only root-relative paths, nothing outside
// is read, no absolute path leaks into any stream.
func TestTraversalConfigGlobsCannotEscapeRoot(t *testing.T) {
	root := springTree(t)
	parent := filepath.Dir(root)
	outside := filepath.Join(parent, "outside")
	write(t, filepath.Join(outside, "SecretController.java"), violatingController)

	write(t, filepath.Join(root, ".vanguard.yaml"), `version: 1
include: ["../**", "**"]
exclude: ["../outside/**", "../../**"]
suppressions:
  - rule: R1xx-01
    paths: ["../outside/**", "../**"]
    reason: "hostile probe: suppress everything the scanner should never see"
`)

	exit, stdout, stderr := runVanguard(t, root, quickCase, "scan", ".", "--format", "json")
	assertNormalExit(t, exit)

	// The hostile globs are schema-valid, so the config must LOAD (rejecting
	// a config is legal exit-2 behaviour, but this fixture pins the stronger
	// outcome: the scan proceeds and still reports the inside violation).
	if n := countFindings(t, stdout); n == 0 {
		t.Fatalf("hostile-glob config suppressed all scanning — expected the inside violation to be reported:\n%s", stdout)
	}

	for _, stream := range []string{stdout, stderr} {
		if strings.Contains(stream, outside) {
			t.Fatalf("scan output references a path outside the scan root (%s):\n%s", outside, stream)
		}
		if strings.Contains(stream, "SecretController") {
			t.Fatalf("scan output names a file outside the scan root:\n%s", stream)
		}
	}
	if strings.Contains(stdout, "../") {
		t.Fatalf("finding paths must be scan-root-relative, got ../ references:\n%s", stdout)
	}
}

// TestScanArgumentWithDotDotSegmentsIsAResolvedRoot pins the operator-side
// boundary: a scan target containing `..` is the operator choosing another
// root — the tool resolves it (EvalSymlinks) and everything stays relative
// to the RESOLVED root. It must scan cleanly, not crash, and never report
// paths above the resolved root.
func TestScanArgumentWithDotDotSegmentsIsAResolvedRoot(t *testing.T) {
	root := springTree(t)
	nested := filepath.Join(root, "a", "b")
	write(t, filepath.Join(nested, "pom.xml"), minimalPom)
	write(t, filepath.Join(nested, "src", "com", "example", "ViolatingController.java"), violatingController)

	exit, stdout, stderr := runVanguard(t, nested, quickCase, "scan", "../..", "--format", "json")
	assertNormalExit(t, exit)

	if n := countFindings(t, stdout); n == 0 {
		t.Fatalf("a `..`-bearing scan argument must resolve and still scan (expected findings):\n%s", stdout)
	}

	if strings.Contains(stdout, "../../") || strings.Contains(stderr, "../../") {
		t.Fatalf("paths must be relative to the resolved root, got ../../ references\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

// ---------------------------------------------------------------------------
// 2. Symlinks pointing outside the scan root
// ---------------------------------------------------------------------------

// TestSymlinksOutsideRootAreNeverFollowed plants a file symlink and a
// directory symlink, both pointing OUTSIDE the scan root (to a violating
// file the scanner must never read), plus an in-root loop symlink. The
// scan must produce exactly the baseline findings (the same tree without
// the links), record the skips, and stay bounded.
func TestSymlinksOutsideRootAreNeverFollowed(t *testing.T) {
	root := springTree(t)
	parent := filepath.Dir(root)
	outside := filepath.Join(parent, "outside")
	write(t, filepath.Join(outside, "SecretController.java"), violatingController)
	write(t, filepath.Join(outside, "nested", "NestedController.java"), violatingController)
	if err := os.Symlink(filepath.Join(outside, "SecretController.java"), filepath.Join(root, "link-secret.java")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "nested"), filepath.Join(root, "docs-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "loop"), filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}

	// Baseline: the identical tree without any symlink.
	baseline := springTree(t)
	exitB, stdoutB, _ := runVanguard(t, baseline, quickCase, "scan", ".", "--format", "json")
	assertNormalExit(t, exitB)

	exit, stdout, stderr := runVanguard(t, root, quickCase, "scan", ".", "--format", "json", "--verbose")
	assertNormalExit(t, exit)

	if strings.Contains(stdout+stderr, outside) || strings.Contains(stdout, "SecretController") || strings.Contains(stdout, "NestedController") {
		t.Fatalf("scan followed a symlink outside the root\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "symlink or special file (not followed)") {
		t.Fatalf("walker did not record the symlink skips in verbose mode:\n%s", stderr)
	}
	if got, want := countFindings(t, stdout), countFindings(t, stdoutB); got != want {
		t.Fatalf("symlinked tree produced %d findings, baseline %d", got, want)
	}
}

// ---------------------------------------------------------------------------
// 3. Zip-bomb-ish nesting (depth bomb)
// ---------------------------------------------------------------------------

// TestDeepDirectoryChainIsBounded builds a 60-level directory chain with a
// violating controller on every level — the zip-bomb-ish "many nested"
// shape. The published depth cap (walker DefaultMaxDepth = 32) must stop
// the descent: files through depth 32 are scanned, everything deeper is
// skipped with a recorded reason, and the scan finishes in bounded time.
func TestDeepDirectoryChainIsBounded(t *testing.T) {
	root := springTree(t) // root pom.xml + depth-1 violating controller
	dir := root
	for i := 1; i <= 60; i++ {
		dir = filepath.Join(dir, "level")
		write(t, filepath.Join(dir, "ViolatingController.java"), violatingController)
	}

	exit, stdout, stderr := runVanguard(t, root, quickCase, "scan", ".", "--format", "json", "--verbose")
	assertNormalExit(t, exit)

	if !strings.Contains(stderr, "exceeds max depth") {
		t.Fatalf("deep chain was not stopped by the published depth cap:\n%s", stderr)
	}
	// Findings-bearing files at depth ≤ 32 (directly in root + level 1..31)
	// are scanned; level 32's directory itself is skipped, so its file is
	// unreachable.
	const maxDepth = 32
	if n := countDistinctFiles(t, stdout); n != maxDepth {
		t.Fatalf("scanned %d files, want %d (the depth cap)", n, maxDepth)
	}
}

// ---------------------------------------------------------------------------
// 4. Parse bomb (deep AST)
// ---------------------------------------------------------------------------

// TestParseBombDeepNestingDoesNotCrash feeds a file with pathological
// nesting depth (block soup + nested generics, both far beyond any
// hand-written source) under the published size cap. tree-sitter must
// degrade to diagnostics — never panic, never hang, never OOM.
func TestParseBombDeepNestingDoesNotCrash(t *testing.T) {
	root := springTree(t)

	var b strings.Builder
	b.WriteString("package com.example.bomb;\n\nclass Bomb {\n  void x() {\n")
	for i := 0; i < 5000; i++ {
		b.WriteString("{\n")
	}
	for i := 0; i < 5000; i++ {
		b.WriteString("}\n")
	}
	b.WriteString("  }\n}\n")
	write(t, filepath.Join(root, "src", "com", "example", "BlockBomb.java"), b.String())

	var g strings.Builder
	g.WriteString("package com.example.bomb;\n\nimport java.util.List;\n\nclass GenericBomb {\n")
	g.WriteString("  List<")
	for i := 0; i < 2000; i++ {
		g.WriteString("List<")
	}
	g.WriteString("String")
	for i := 0; i < 2000; i++ {
		g.WriteString(">")
	}
	g.WriteString(" field;\n}\n")
	write(t, filepath.Join(root, "src", "com", "example", "GenericBomb.java"), g.String())

	exit, _, _ := runVanguard(t, root, quickCase, "scan", ".", "--format", "json")
	assertNormalExit(t, exit)
}

// ---------------------------------------------------------------------------
// 5. Unicode / special-character filenames
// ---------------------------------------------------------------------------

// TestUnicodeAndSpecialFilenamesRoundTrip scans a tree whose filenames
// cover the hostile-alphabet space (combining marks, emoji, spaces,
// punctuation, control-ish characters that stay POSIX-legal) and requires
// the names to survive the pipeline intact in both machine formats —
// including a newline, the JSON-escaping worst case. A leading-dot file is
// deliberately present to pin the documented hidden-file skip.
func TestUnicodeAndSpecialFilenamesRoundTrip(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "pom.xml"), minimalPom)
	names := []string{
		"TàiLiệuController.java",
		"🎮GameController.java",
		"name with spaces.java",
		"semi;colon&amp.java",
		"back\\slash.java",
		"new\nline.java",
		"quote\".java",
		"dollar$().java",
		"UPPER-lower_123.java",
	}
	src := filepath.Join(root, "src")
	for _, name := range names {
		write(t, filepath.Join(src, name), violatingController)
	}
	write(t, filepath.Join(src, ".hidden-controller.java"), violatingController)

	exit, stdout, _ := runVanguard(t, root, quickCase, "scan", ".", "--format", "json")
	assertNormalExit(t, exit)

	// Name checks go through the PARSED report: a raw-substring check
	// would demand the JSON-escaped byte shape (\\ for \, \n for newline)
	// and pin escaping details, not round-tripping.
	rep := parseReport(t, stdout)
	scanned := map[string]bool{}
	for _, f := range rep.Findings {
		scanned[f.Location.File] = true
	}
	if !scanned[filepath.ToSlash(filepath.Join("src", "TàiLiệuController.java"))] {
		t.Fatalf("UTF-8 filename did not survive the JSON output:\n%s", stdout)
	}
	// Every non-hidden name must be scanned and named with intact UTF-8;
	// the hidden file is skipped by design (walker: hidden files are
	// config, not sources).
	for _, name := range names {
		if !scanned[filepath.ToSlash(filepath.Join("src", name))] {
			t.Fatalf("scanned output lost filename %q (scanned set %v)", name, keys(scanned))
		}
	}
	for _, f := range rep.Findings {
		if strings.Contains(f.Location.File, ".hidden-controller") {
			t.Fatalf("hidden file was scanned (documented skip violated)")
		}
	}

	// The SARIF view of the same tree must be equally intact: parse it and
	// require every name as a URI with its UTF-8 intact.
	exitS, stdoutS, _ := runVanguard(t, root, quickCase, "scan", ".", "--format", "sarif")
	assertNormalExit(t, exitS)
	uris := sarifURIs(t, stdoutS)
	for _, name := range names {
		uri := filepath.ToSlash(filepath.Join("src", name))
		if !uris[uri] {
			t.Fatalf("SARIF output lost filename %q (URIs: %d entries)", name, len(uris))
		}
	}
}

// sarifURIs extracts every artifactLocation.uri from a SARIF document.
func sarifURIs(t *testing.T, stdout string) map[string]bool {
	t.Helper()
	var doc struct {
		Runs []struct {
			Results []struct {
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("SARIF output is not valid JSON: %v", err)
	}
	out := map[string]bool{}
	for _, run := range doc.Runs {
		for _, res := range run.Results {
			for _, loc := range res.Locations {
				out[loc.PhysicalLocation.ArtifactLocation.URI] = true
			}
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// ---------------------------------------------------------------------------
// 6. Hostile config: glob bomb
// ---------------------------------------------------------------------------

// TestHostileConfigGlobCannotHang ships the worst `.vanguard.yaml` the
// brief names: a suppression glob built from dozens of `**` segments that
// never matches. The suppression path evaluates every glob per finding, so
// a hostile config must not turn the scan into an unbounded fork-walk.
// The scan must finish within the bound, keep its findings (the glob does
// not match), and exit within the §6.3 contract.
//
// Measured pre-fix: 8 `**` segments took 9.2 s, 10 blew past 15 s, and the
// 64-segment fixture here never returned (exponential in the number of
// wildcards — see internal/engine/glob.go).
func TestHostileConfigGlobCannotHang(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "pom.xml"), minimalPom)
	dir := root
	for i := 1; i <= 31; i++ { // deepest file lands at depth 32, the cap
		dir = filepath.Join(dir, "d")
	}
	write(t, filepath.Join(dir, "ViolatingController.java"), violatingController)

	globBomb := strings.Repeat("**/", 64) + "nope.java"
	write(t, filepath.Join(root, ".vanguard.yaml"), "version: 1\nsuppressions:\n  - rule: R1xx-01\n    paths: [\""+globBomb+"\"]\n    reason: \"hostile glob probe\"\n")

	exit, stdout, _ := runVanguard(t, root, hostileConfigCase, "scan", ".", "--format", "json")
	assertNormalExit(t, exit)

	if !strings.Contains(stdout, "ViolatingController.java") {
		t.Fatalf("finding for the deep controller vanished (hostile glob must not swallow results):\n%s", stdout)
	}
	if n := suppressedCount(t, stdout); n != 0 {
		t.Fatalf("suppressed %d findings, want 0 (the bomb glob matches nothing)", n)
	}
}

// ---------------------------------------------------------------------------
// finding-count helpers (§6.7 JSON report parsing)
// ---------------------------------------------------------------------------

type scanReport struct {
	Summary struct {
		Findings struct {
			Error   int `json:"error"`
			Warning int `json:"warning"`
			Info    int `json:"info"`
		} `json:"findings"`
		Suppressed int `json:"suppressed"`
	} `json:"summary"`
	Findings []struct {
		Location struct {
			File string `json:"file"`
		} `json:"location"`
	} `json:"findings"`
}

func parseReport(t *testing.T, stdout string) scanReport {
	t.Helper()
	var rep scanReport
	if err := json.Unmarshal([]byte(stdout), &rep); err != nil {
		t.Fatalf("scan stdout is not the §6.7 JSON report: %v\noutput:\n%s", err, stdout)
	}
	return rep
}

// countFindings counts every finding (any severity) in a JSON report.
func countFindings(t *testing.T, stdout string) int {
	t.Helper()
	rep := parseReport(t, stdout)
	s := rep.Summary.Findings
	return s.Error + s.Warning + s.Info
}

// countDistinctFiles counts the distinct scanned files that carried a
// finding.
func countDistinctFiles(t *testing.T, stdout string) int {
	t.Helper()
	rep := parseReport(t, stdout)
	seen := map[string]bool{}
	for _, f := range rep.Findings {
		seen[f.Location.File] = true
	}
	return len(seen)
}

func suppressedCount(t *testing.T, stdout string) int {
	t.Helper()
	return parseReport(t, stdout).Summary.Suppressed
}
