package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// sha256hex hashes whatever the callback read into buf.
func sha256hex(buf []byte) string {
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

// runChild launches the vanguard binary with args and waits for it to finish
// under the hard timeout mandated by process amendment 2. It returns the exit
// code (or -1 when the child was killed).
func runChild(bin, workDir string, timeout time.Duration, args ...string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	err := cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return -1, fmt.Errorf("child %v timed out after %s", args, timeout)
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}

// scopedScanRun runs one full scan of target with --format format --output
// tmp --no-config, capturing wall time and the child's peak RSS (Linux
// ru_maxrss). All output goes to a temp file — printf-ing a 10k-finding
// report to a pipe would measure stdio, not vanguard.
func scopedScanRun(rec *scanRun, bin, moduleRoot, target, tmp, format string, timeout time.Duration) error {
	if err := os.Truncate(tmp, 0); err != nil {
		return err
	}
	start := time.Now()
	exit, rss, err := scanToFile(rec, bin, moduleRoot, target, tmp, format, timeout)
	if err != nil {
		return err
	}
	rec.WallSeconds = time.Since(start).Seconds()
	rec.MaxRssKiB = rss
	rec.Exit = exit
	return nil
}

func scanToFile(rec *scanRun, bin, moduleRoot, target, tmp, format string, timeout time.Duration) (int, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "scan", target, "--format", format, "--output", tmp, "--no-color", "--no-config")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	if err := cmd.Start(); err != nil {
		return -1, 0, err
	}
	err := cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return -1, 0, fmt.Errorf("scan timed out after %s", timeout)
	}
	var exit int
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			return -1, 0, err
		}
		exit = ee.ExitCode()
	} else {
		exit = 0
	}
	rss := int64(-1)
	if sys := cmd.ProcessState.SysUsage(); sys != nil {
		if ru, ok := sys.(*syscall.Rusage); ok {
			rss = ru.Maxrss // KiB on Linux
		}
	}
	return exit, rss, nil
}

func determinismPair(bin, moduleRoot, target, format string, timeout time.Duration) (pair, error) {
	var rec pair
	f, err := os.CreateTemp("", "vanguard-perfbench-pair-*")
	if err != nil {
		return rec, err
	}
	defer os.Remove(f.Name())
	f.Close()

	var first string
	for run := 0; run < 2; run++ {
		if err := os.Truncate(f.Name(), 0); err != nil {
			return rec, err
		}
		exit, _, err := scanToFile(nil, bin, moduleRoot, target, f.Name(), format, timeout)
		if err != nil {
			return rec, err
		}
		if exit != 0 && exit != 1 {
			return rec, fmt.Errorf("unexpected exit code %d (%s)", exit, format)
		}
		buf, err := os.ReadFile(f.Name())
		if err != nil {
			return rec, err
		}
		sum := sha256hex(buf)
		if run == 0 {
			first = sum
			rec.Sha256 = sum
			rec.Bytes = int64(len(buf))
		} else {
			rec.Sha256Run = sum
			rec.Identical = first == sum
		}
	}
	return rec, nil
}

// countJava counts the .java files under dir — the §5.2 reported file count
// and the denominator of every throughput figure.
func countJava(dir string) (int, error) {
	n := 0
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(p) == ".java" {
			n++
		}
		return nil
	})
	return n, err
}

// printReport echoes the human summary quoting §5.2/§5.3: raw times beside
// the aggregates, so a report can quote either without rerunning the bench.
func printReport(res *result, bench benchRepo) {
	fmt.Println("== perfbench (w4-03, test-strategy §5) ==")
	fmt.Printf("environment image=%s cpu=%s mem=%s\n", res.Environment.Image, res.Environment.CPU, res.Environment.Mem)
	fmt.Printf("bench repo  %s\n", bench.Dir)
	fmt.Printf("java files  %d · digest %s\n", bench.JavaFiles, bench.Digest)
	fmt.Println("-- cold start (`vanguard version`, target ≤ 3s) --")
	for _, c := range res.ColdStart {
		fmt.Printf("  run %d: %.3fs (exit %d)\n", c.Index, c.Seconds, c.Exit)
	}
	fmt.Println("-- raw runs (wall clock) --")
	for _, r := range res.Runs {
		fmt.Printf("  run %d: %.2fs (%.0f files/s, peak RSS %d KiB, exit %d)\n",
			r.Index, r.WallSeconds, r.FilesPerSec, r.MaxRssKiB, r.Exit)
	}
	fmt.Printf("p50 %.2fs · p95 %.2fs · p99 %.2fs · target %.0fs\n",
		res.Summary.P50Seconds, res.Summary.P95Seconds, res.Summary.P99Seconds, targetSeconds)
	fmt.Printf("throughput at p95: %.0f files/s (max %.0f files/s)\n",
		res.Summary.FilesPerSecP95, res.Summary.FilesPerSecMax)
	fmt.Println("-- determinism pairs (§5.4) --")
	for _, p := range res.Determinism {
		fmt.Printf("  %-6s %-5s %s (%d bytes) identical=%v\n",
			p.Source, p.Format, p.Sha256[:16], p.Bytes, p.Identical)
	}
	fmt.Printf("identical pairs: %d/%d\n", res.IdenticalPairs, res.TotalPairs)
}
