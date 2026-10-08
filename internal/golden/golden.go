// Package golden is the W2 golden fixture harness (w2-07; test strategy
// §2.1, tier 1): it runs the REAL vanguard binary via exec — exactly the
// way CI invokes it — against fixture directories under testdata/golden/,
// snapshots the output of every format, and compares the normalized bytes
// against the snapshot files committed next to each fixture.
//
// Contract:
//
//   - A case is one directory under testdata/golden/ containing a
//     case.yaml manifest plus any fixture files. The manifest pins the
//     scan target, extra CLI args and the expected exit code; snapshot
//     files (snapshot.pretty / snapshot.json / snapshot.sarif, and
//     snapshot.stderr for stderr runs) live in the same directory so a
//     review sees fixture and expectation side by side.
//   - Output normalization (normalize.go) removes only what CANNOT be
//     deterministic — the wall-clock duration and the absolute paths of
//     the machine the test runs on — before comparing. Everything else is
//     pinned byte-for-byte (charter B-6, w4-03 determinism).
//   - Missing snapshots fail the run with the regeneration command, so an
//     intentional behavior change is always a reviewed diff (the
//     `make golden-update` contract).
//
// The harness is data-driven on purpose: adding a case for W3/W4 means
// adding a directory — never editing this package.
package golden

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// defaultFormats is the format triple every case snapshots unless its
// manifest narrows it (charter §6.2: pretty, json, sarif).
var defaultFormats = []string{"pretty", "json", "sarif"}

// runTimeout bounds ONE binary invocation; a hung scan fails the case
// instead of hanging the suite (process-amendment 2: no unbounded waits).
const runTimeout = 60 * time.Second

// buildTimeout bounds the one-time `go build` of the fixture binary.
const buildTimeout = 5 * time.Minute

// Case is one fixture directory: a manifest plus its files and snapshots.
type Case struct {
	ID   string // directory name — also the test name
	Dir  string // absolute path of the case directory
	Note string // human note from the manifest (documentation only)

	Target   string   // scan target relative to Dir ("." by default)
	Args     []string // extra CLI args; "{case}"/"{module}" expand to relative paths
	Exit     int      // expected exit code (§6.3)
	Formats  []string // output formats to snapshot
	Snapshot string   // which stream(es) to snapshot: stdout | stderr | both
}

// manifest mirrors case.yaml. Every field is optional except the implicit
// directory name; unknown keys fail loudly (a typo'd manifest must not
// silently run with defaults).
type manifest struct {
	Target   string   `yaml:"target"`
	Args     []string `yaml:"args"`
	Exit     *int     `yaml:"exit"`
	Formats  []string `yaml:"formats"`
	Snapshot string   `yaml:"snapshot"`
	Note     string   `yaml:"note"`
	Extra    []byte   `yaml:"-"`
}

// LoadCases reads every case directory under dir (sorted by name for
// deterministic test order). A directory without case.yaml is skipped so
// scratch material can coexist; a malformed manifest is an error.
func LoadCases(dir string) ([]Case, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("golden: read %s: %w", dir, err)
	}
	var cases []Case
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		caseDir := filepath.Join(dir, e.Name())
		manifestPath := filepath.Join(caseDir, "case.yaml")
		data, err := os.ReadFile(manifestPath)
		if errors.Is(err, os.ErrNotExist) {
			continue // not a case directory
		}
		if err != nil {
			return nil, fmt.Errorf("golden: %s: %w", e.Name(), err)
		}
		var m manifest
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("golden: %s/case.yaml: %w", e.Name(), err)
		}
		c := Case{
			ID:       e.Name(),
			Dir:      caseDir,
			Note:     m.Note,
			Target:   cmpOr(m.Target, "."),
			Args:     m.Args,
			Formats:  m.Formats,
			Snapshot: cmpOr(m.Snapshot, "stdout"),
		}
		if m.Exit != nil {
			c.Exit = *m.Exit
		}
		if len(c.Formats) == 0 {
			c.Formats = defaultFormats
		}
		switch c.Snapshot {
		case "stdout", "stderr", "both":
		default:
			return nil, fmt.Errorf("golden: %s: snapshot must be stdout, stderr or both, got %q", c.ID, c.Snapshot)
		}
		cases = append(cases, c)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	return cases, nil
}

func cmpOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// RunResult is one binary invocation's observable outcome.
type RunResult struct {
	Exit   int
	Stdout []byte
	Stderr []byte
}

// Run executes `vanguard scan <target> <flags>` for one format with the
// process cwd at moduleRoot, so every path the binary sees — and therefore
// every path it prints — is relative and machine-independent. Placeholders
// in the target and manifest args expand to those same relative paths.
func (c Case) Run(bin, moduleRoot, format string) (RunResult, error) {
	relCase, err := relFrom(moduleRoot, c.Dir)
	if err != nil {
		return RunResult{}, err
	}
	expand := func(s string) string {
		s = strings.ReplaceAll(s, "{case}", relCase)
		s = strings.ReplaceAll(s, "{module}", ".")
		return s
	}
	target := c.Target
	switch {
	case target == ".":
		target = relCase // the default target is the case directory itself
	case strings.Contains(target, "{module}"):
		target = expand(target) // module-relative target, use as expanded
	default:
		target = filepath.Join(relCase, filepath.FromSlash(expand(target)))
	}
	args := []string{"scan", filepath.ToSlash(target), "--format", format, "--no-color"}
	for _, a := range c.Args {
		args = append(args, filepath.ToSlash(expand(a)))
	}
	return runBinary(bin, moduleRoot, args)
}

// runBinary execs bin with args in workDir, capturing both streams.
func runBinary(bin, workDir string, args []string) (RunResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = workDir
	// NO_COLOR pins the auto color matrix even if a future flag spelling
	// forgets to pass --no-color through (§6.6: env wins in auto mode).
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return RunResult{}, fmt.Errorf("golden: %s timed out after %s", bin, runTimeout)
	}
	res := RunResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err == nil {
		res.Exit = 0
		return res, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		res.Exit = ee.ExitCode()
		return res, nil
	}
	return RunResult{}, fmt.Errorf("golden: run %s %s: %w", bin, strings.Join(args, " "), err)
}

// relFrom returns target as a slash-separated path relative to base.
func relFrom(base, target string) (string, error) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", fmt.Errorf("golden: resolve %s under %s: %w", target, base, err)
	}
	return filepath.ToSlash(rel), nil
}

// ---------------------------------------------------------------------------
// Binary resolution — the real binary, built once per test process
// ---------------------------------------------------------------------------

var (
	binOnce sync.Once
	binPath string
	binErr  error
)

// BinaryPath returns the vanguard binary every fixture run uses: $VANGUARD_BIN
// when the environment provides one (CI may pin a make-bin artifact),
// otherwise a one-time `go build ./cmd/vanguard` into a per-process temp
// directory. The binary is built WITHOUT ldflags, so the version prints the
// source default — deterministic snapshot input (w2-06: never fail the
// build without metadata).
func BinaryPath() (string, error) {
	binOnce.Do(func() {
		if p := os.Getenv("VANGUARD_BIN"); p != "" {
			binPath = p
			return
		}
		goExe, err := exec.LookPath("go")
		if err != nil {
			binErr = fmt.Errorf("golden: no `go` on PATH and $VANGUARD_BIN is unset: %w", err)
			return
		}
		dir, err := os.MkdirTemp("", "vanguard-golden-bin-")
		if err != nil {
			binErr = fmt.Errorf("golden: temp dir for fixture binary: %w", err)
			return
		}
		out := filepath.Join(dir, "vanguard")
		root, err := ModuleRoot()
		if err != nil {
			binErr = err
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, goExe, "build", "-o", out, "github.com/Stellarhold170NT/vanguard/cmd/vanguard")
		cmd.Dir = root
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		if err := cmd.Run(); err != nil {
			binErr = fmt.Errorf("golden: build fixture binary: %v\n%s", err, outBuf.String())
			return
		}
		binPath = out
	})
	return binPath, binErr
}

// ModuleRoot returns the module root — the parent of this package's
// internal/golden directory (go test sets the cwd to the package dir).
func ModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("golden: working directory: %w", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if fi, err := os.Stat(filepath.Join(root, "go.mod")); err != nil || fi.IsDir() {
		return "", fmt.Errorf("golden: %s does not look like the module root (no go.mod)", root)
	}
	return root, nil
}
