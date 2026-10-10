package perf

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/vanguard-lint/vanguard/internal/benchgen"
)

// corpusDir is the golden corpus the §5.4 matrix runs against: the w3-07
// java-spring showcase (the adversarial corpus from w4-01 does not exist at
// w4-03 time — the brief's dependency is w3-07, and this note is honest
// about which corpus the matrix pins).
func corpusDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(ModuleRoot(t), "testdata", "golden", "java-spring")
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("corpus %s missing: %v", dir, err)
	}
	return dir
}

func sha256of(t *testing.T, b []byte) string {
	t.Helper()
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TestScanOutputIsByteIdenticalAcrossRuns is §5.4 pair rows 1–3 (corpus ×
// {pretty, json, sarif}): two consecutive runs must produce byte-identical
// output — sha256 run-1 = run-2, no normalization (§5.4: determinism is
// never patched in the harness; a differing byte is an engine/render bug).
func TestScanOutputIsByteIdenticalAcrossRuns(t *testing.T) {
	root := ModuleRoot(t)
	corpusDir(t)                            // fail fast when the corpus is missing
	target := "testdata/golden/java-spring" // relative on purpose: paths in the output stay relative

	for _, format := range []string{"pretty", "json", "sarif"} {
		format := format
		t.Run(format, func(t *testing.T) {
			first := Run(t, root, "scan", target, "--format", format, "--no-color")
			second := Run(t, root, "scan", target, "--format", format, "--no-color")

			if first.Exit != second.Exit {
				t.Fatalf("exit codes differ: %d vs %d\nstderr1: %s\nstderr2: %s",
					first.Exit, second.Exit, first.Stderr, second.Stderr)
			}
			if sha256of(t, first.Stdout) != sha256of(t, second.Stdout) {
				t.Fatalf("output not byte-identical across runs (%s, exit %d):\n--- run 1 ---\n%s\n--- run 2 ---\n%s",
					format, first.Exit, first.Stdout, second.Stdout)
			}
		})
	}
}

// TestScanBenchRepoIsByteIdenticalAcrossRuns is §5.4 pair rows 4–6 (bench
// repo × {pretty, json, sarif}) on the tree the generator produces. The
// full-scale 10k-file matrix belongs to the perfbench protocol (§5.2, the
// numbers reports/w4-03.md quotes); this in-suite guard runs the same six
// comparisons on a small generated tree so the determinism contract stays
// enforced by `go test ./...` on every commit, not only during the bench.
func TestScanBenchRepoIsByteIdenticalAcrossRuns(t *testing.T) {
	root := ModuleRoot(t)
	dir := t.TempDir()
	plan := benchgen.Plan{Seed: 42, Files: 120, Mix: benchgen.DefaultMix}
	res, err := benchgen.Generate(plan, dir, true)
	if err != nil {
		t.Fatalf("generate bench repo: %v", err)
	}
	if res.JavaFiles != plan.Files {
		t.Fatalf("generated %d java files, want %d", res.JavaFiles, plan.Files)
	}

	for _, format := range []string{"pretty", "json", "sarif"} {
		format := format
		t.Run(format, func(t *testing.T) {
			first := Run(t, root, "scan", dir, "--format", format, "--no-color")
			second := Run(t, root, "scan", dir, "--format", format, "--no-color")

			if first.Exit != second.Exit {
				t.Fatalf("exit codes differ: %d vs %d\nstderr1: %s\nstderr2: %s",
					first.Exit, second.Exit, first.Stderr, second.Stderr)
			}
			if first.Exit != 0 && first.Exit != 1 {
				t.Fatalf("bench-repo scan exit %d (stderr: %s)", first.Exit, first.Stderr)
			}
			if sha256of(t, first.Stdout) != sha256of(t, second.Stdout) {
				t.Fatalf("bench-repo output not byte-identical across runs (%s):\n--- run 1 ---\n%s\n--- run 2 ---\n%s",
					format, first.Stdout, second.Stdout)
			}
		})
	}
}
