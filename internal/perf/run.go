// Package perf hosts the w4-03 tier-4 harness tests (test strategy §2.4):
// determinism of the real binary's output across repeated runs (§5.4) and
// the edge-case behaviour contract (§5.5).
//
// Everything here exercises the REAL vanguard binary via exec — the CI
// invocation shape — never a mock (the same rule internal/golden applies).
// Determinism is checked on the RAW bytes: §5.4 forbids normalizing the
// comparison in the harness, so any nondeterministic byte is an engine or
// renderer bug, not a test concern.
package perf

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Stellarhold170NT/vanguard/internal/golden"
)

// runTimeout bounds ONE binary invocation; a hung scan fails the test
// instead of hanging the suite (process amendment 2: no unbounded waits).
const runTimeout = 90 * time.Second

// RunResult is one invocation's observable outcome.
type RunResult struct {
	Exit   int
	Stdout []byte
	Stderr []byte
}

// BinaryPath resolves the real vanguard binary (built once per test
// process, the internal/golden contract).
func BinaryPath(t *testing.T) string {
	t.Helper()
	bin, err := golden.BinaryPath()
	if err != nil {
		t.Fatalf("resolve vanguard binary: %v", err)
	}
	return bin
}

// ModuleRoot returns the module root (the parent of internal/perf).
func ModuleRoot(t *testing.T) string {
	t.Helper()
	root, err := golden.ModuleRoot()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	return root
}

// Run executes the binary with args in dir and fails the test on a launch
// error. Exit codes are returned as-is: several edge cases legitimately
// exit 1, so the assertions, not the helper, decide what is expected.
func Run(t *testing.T, dir string, args ...string) RunResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, tBinaryPath(t), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("vanguard %s timed out after %s (hung run is a failure, not a wait)", args, runTimeout)
	}
	res := RunResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err == nil {
		return res
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		res.Exit = ee.ExitCode()
		return res
	}
	t.Fatalf("run vanguard %s in %s: %v", args, dir, err)
	return RunResult{}
}

var binOnce struct {
	path string
	err  error
}

// tBinaryPath caches the resolved binary across the package's tests.
func tBinaryPath(t *testing.T) string {
	t.Helper()
	if binOnce.path == "" && binOnce.err == nil {
		binOnce.path, binOnce.err = golden.BinaryPath()
	}
	if binOnce.err != nil {
		t.Fatalf("resolve vanguard binary: %v", binOnce.err)
	}
	return binOnce.path
}
