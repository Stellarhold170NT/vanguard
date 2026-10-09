// Command perfbench drives the w4-03 performance + determinism bench
// (test strategy §5) against a sandbox build of vanguard.
//
// One invocation runs the whole §5.2/§5.3/§5.4 protocol:
//
//   - generate the deterministic bench repo (internal/benchgen) unless
//     -skip-gen says it is already there;
//   - `-runs` full scans (wall clock + ru_maxrss of the child), p50/p95/p99
//     nearest-rank over the raw numbers against the §5.3 target;
//   - the §5.4 determinism matrix — {corpus, bench repo} × {pretty, json,
//     sarif}, two runs each, sha256 run-1 vs run-2, 6/6 pairs identical;
//   - cold-start timings for `vanguard version`;
//   - everything above lands in -out as JSON for reports/w4-03.md to quote.
//
// Every child process runs under a hard timeout (-timeout, process amendment
// 2) and no measurement is normalized on this side of the pipe: a failing
// determinism pair is reported as a failure, never patched away.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Stellarhold170NT/vanguard/internal/benchgen"
)

const (
	// targetSeconds is the §5.3 gate: nearest-rank p95 wall-clock for a full
	// scan of the bench repo. Raw run times are always recorded next to it —
	// a miss is evidence for a threshold debate, never quietly adjusted.
	targetSeconds = 60.0

	// minRuns is the §5.3 protocol floor (">= 3 runs"; 5 recommended).
	minRuns = 3
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var (
		bin        = flag.String("bin", "bin/vanguard", "vanguard binary to benchmark")
		moduleRoot = flag.String("module-root", ".", "vanguard module root (must contain testdata/golden)")
		benchDir   = flag.String("bench-dir", "", "bench repo directory (default: <module-root>/bench/bench-repo)")
		corpusDir  = flag.String("corpus", "", "corpus source for the §5.4 pairs (default: <module-root>/testdata/golden/java-spring)")
		files      = flag.Int("files", benchgen.DefaultFiles, "bench repo .java file count")
		seed       = flag.Int64("seed", benchgen.DefaultSeed, "bench generation seed")
		runs       = flag.Int("runs", 5, "full-scan runs of the bench repo")
		skipGen    = flag.Bool("skip-gen", false, "reuse -bench-dir instead of regenerating it")
		out        = flag.String("out", "testdata/perf-results.json", "results JSON to write")
		timeLimit  = flag.Duration("timeout", 300*time.Second, "hard timeout for one child process")
		image      = flag.String("image", "", "measurement environment image, recorded verbatim in the JSON (§5.2)")
		cpu        = flag.String("cpu", "", "measurement environment CPU allocation, e.g. 2 (§5.2)")
		mem        = flag.String("mem", "", "measurement environment memory, e.g. 4Gi (§5.2)")
	)
	flag.Parse()
	if *runs < minRuns {
		fmt.Fprintf(os.Stderr, "perfbench: -runs %d below protocol minimum %d\n", *runs, minRuns)
		return 2
	}
	if err := mainE(cfg{
		bin: *bin, moduleRoot: *moduleRoot, benchDir: *benchDir, corpusDir: *corpusDir,
		files: *files, seed: *seed, runs: *runs, skipGen: *skipGen, out: *out,
		timeout: *timeLimit, image: *image, cpu: *cpu, mem: *mem,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "perfbench: %v\n", err)
		return 2
	}
	return 0
}

type cfg struct {
	bin, moduleRoot, benchDir, corpusDir string
	files                                int
	seed                                 int64
	runs                                 int
	skipGen                              bool
	out                                  string
	timeout                              time.Duration
	image, cpu, mem                      string
}

type result struct {
	SchemaVersion  int         `json:"schemaVersion"`
	Environment    environment `json:"environment"`
	BenchRepo      benchRepo   `json:"benchRepo"`
	Runs           []scanRun   `json:"runs"`
	Summary        Summary     `json:"summary"`
	Determinism    []pair      `json:"determinism"`
	IdenticalPairs int         `json:"identicalPairs"`
	TotalPairs     int         `json:"totalPairs"`
	ColdStart      []coldStart `json:"coldStart"`
}

type environment struct {
	Image string `json:"image"`
	CPU   string `json:"cpu"`
	Mem   string `json:"mem"`
}

func mainE(c cfg) error {
	root, err := filepath.Abs(c.moduleRoot)
	if err != nil {
		return fmt.Errorf("resolve module root: %w", err)
	}
	if c.benchDir == "" {
		c.benchDir = filepath.Join(root, "bench", "bench-repo")
	}
	if c.corpusDir == "" {
		c.corpusDir = filepath.Join(root, "testdata", "golden", "java-spring")
	}

	// Phase 1 — determinism pair count and bench repo materialization.
	var bench benchRepo
	if c.skipGen {
		java, err := countJava(c.benchDir)
		if err != nil {
			return fmt.Errorf("-skip-gen set: %w", err)
		}
		bench = benchRepo{JavaFiles: java, Dir: c.benchDir}
		bench.Digest = "not-recomputed (-skip-gen)"
	} else {
		res, err := benchgen.Generate(benchgen.Plan{Seed: c.seed, Files: c.files, Mix: benchgen.DefaultMix}, c.benchDir, true)
		if err != nil {
			return fmt.Errorf("generate bench repo: %w", err)
		}
		bench = benchRepo{
			JavaFiles: res.JavaFiles,
			LOC:       res.Lines,
			Bytes:     res.Bytes,
			Digest:    res.Digest,
			Dir:       c.benchDir,
		}
	}

	if bench.JavaFiles < benchgen.DefaultFiles {
		fmt.Fprintf(os.Stderr, "perfbench: WARNING: bench repo has %d .java files, below the §5.1 target of %d\n",
			bench.JavaFiles, benchgen.DefaultFiles)
	}

	res := result{
		SchemaVersion: 1,
		Environment:   environment{Image: c.image, CPU: c.cpu, Mem: c.mem},
		BenchRepo:     bench, // the material's own bookkeeping must reach the JSON, not only the console
	}

	// Phase 2 — cold starts.
	for i := 0; i < 3; i++ {
		start := time.Now()
		exit, err := runChild(c.bin, root, c.timeout, "version")
		if err != nil {
			return fmt.Errorf("cold start %d: %w", i+1, err)
		}
		res.ColdStart = append(res.ColdStart, coldStart{
			Index: i + 1, Seconds: time.Since(start).Seconds(), Exit: exit,
		})
	}

	// Phase 3 — the raw scan runs.
	outFile, err := os.CreateTemp("", "vanguard-perfbench-out-*")
	if err != nil {
		return fmt.Errorf("temp output file: %w", err)
	}
	defer os.Remove(outFile.Name())
	outFile.Close()

	for i := 0; i < c.runs; i++ {
		rec := scanRun{}
		if err := scopedScanRun(&rec, c.bin, root, c.benchDir, outFile.Name(), "json", c.timeout); err != nil {
			return fmt.Errorf("scan run %d: %w", i+1, err)
		}
		rec.Index = i + 1
		if rec.Exit != 0 && rec.Exit != 1 {
			return fmt.Errorf("scan run %d: unexpected exit code %d", i+1, rec.Exit)
		}
		if bench.JavaFiles > 0 {
			rec.FilesPerSec = float64(bench.JavaFiles) / rec.WallSeconds
		}
		res.Runs = append(res.Runs, rec)
	}

	// Phase 4 — determinism matrix: {benchRepo, corpus} × {pretty, json, sarif}.
	for _, source := range []struct{ name, dir string }{{"corpus", c.corpusDir}, {"bench", c.benchDir}} {
		for _, format := range formats {
			p, err := determinismPair(c.bin, root, source.dir, format, c.timeout)
			if err != nil {
				return fmt.Errorf("determinism pair %s/%s: %w", source.name, format, err)
			}
			p.Source = source.name
			p.Format = format
			res.Determinism = append(res.Determinism, p)
			if p.Identical {
				res.IdenticalPairs++
			}
			res.TotalPairs++
		}
	}

	times := make([]time.Duration, len(res.Runs))
	for i, r := range res.Runs {
		times[i] = time.Duration(r.WallSeconds * float64(time.Second))
	}
	res.Summary = Summarize(times, bench.JavaFiles, targetSeconds)
	res.Summary.MeasurementNote = fmt.Sprintf(
		"%d full scans on the raw wall clock; nearest-rank percentiles; with n=%d the p95 is the slowest run (conservative)",
		c.runs, c.runs)

	f, err := os.Create(c.out)
	if err != nil {
		return fmt.Errorf("write %s: %w", c.out, err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&res); err != nil {
		f.Close()
		return fmt.Errorf("encode results: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", c.out, err)
	}

	printReport(&res, bench)
	if res.IdenticalPairs != res.TotalPairs {
		return fmt.Errorf("determinism: only %d/%d pairs byte-identical", res.IdenticalPairs, res.TotalPairs)
	}
	if !res.Summary.P95UnderTarget {
		return fmt.Errorf("p95 %.2fs exceeds the %.0fs target", res.Summary.P95Seconds, targetSeconds)
	}
	return nil
}

// benchRepo carries the generated material's own bookkeeping. The digest is
// the generator's byte-identity proof quoted by reports/w4-03.md (§5.1).
type benchRepo struct {
	Dir       string `json:"dir"`
	JavaFiles int    `json:"javaFiles"`
	LOC       int    `json:"loc"`
	Bytes     int64  `json:"bytes"`
	Digest    string `json:"digest"`
}

type coldStart struct {
	Index   int     `json:"index"`
	Seconds float64 `json:"seconds"`
	Exit    int     `json:"exit"`
}

// pair is one cell of the §5.4 matrix: source × format, hashed twice.
type pair struct {
	Source    string `json:"source"`
	Format    string `json:"format"`
	Sha256    string `json:"sha256"`
	Sha256Run string `json:"sha256Run2"`
	Bytes     int64  `json:"bytes"`
	Identical bool   `json:"identical"`
}

type scanRun struct {
	Index       int     `json:"index"`
	WallSeconds float64 `json:"wallSeconds"`
	FilesPerSec float64 `json:"filesPerSec"`
	MaxRssKiB   int64   `json:"maxRssKiB"`
	Exit        int     `json:"exit"`
}

var formats = []string{"pretty", "json", "sarif"}
