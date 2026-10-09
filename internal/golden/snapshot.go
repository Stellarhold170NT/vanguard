package golden

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Snapshot file names. stdout snapshots carry the format in the name; the
// stderr snapshot is format-independent (tool errors do not vary with
// --format), so one file serves all formats.
const (
	stderrSnapshot = "snapshot.stderr"
	snapshotPrefix = "snapshot."
)

// updateFlag is wired to the test binary's -update flag by golden_test.go:
// when set, matching snapshots are rewritten instead of compared — the
// intentional-change path behind `make golden-update`.
var updateFlag = false

// Check compares one normalized output against its snapshot file. In verify
// mode a missing snapshot is a FAILURE with the regeneration command (the
// golden contract: an expectation must exist and be reviewed, never
// silently created by a test run). In update mode the file is written.
func Check(caseDir, name string, got []byte) error {
	path := filepath.Join(caseDir, name)
	if updateFlag {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			return fmt.Errorf("golden: write snapshot %s: %w", path, err)
		}
		return nil
	}
	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("golden: snapshot %s is missing — regenerate intentionally with `make golden-update` (go test ./internal/golden/ -run TestGoldenCases -update)", path)
	}
	if err != nil {
		return fmt.Errorf("golden: read snapshot %s: %w", path, err)
	}
	if bytes.Equal(want, got) {
		return nil
	}
	return mismatchError(path, want, got)
}

// EnsureExpectedSnapshots pre-creates (empty) the snapshot files a case is
// expected to produce, unless they already exist.
//
// Why: the scan walk reports every ignored file in its "skipped" summary,
// and the snapshot files themselves are ignored (each case's
// .vanguardignore lists snapshot.*). If the snapshots exist at verify time
// but not yet at update time, the two runs would pin different skip
// counts — the first `make ci` after an update would fail against the
// freshly generated snapshots (self-pollution, found by exactly that
// sequence). Pre-creating the exact expected set makes the walked file
// set identical in both modes: empty files carry no content, and the walk
// only ever sees their names. A verify-mode run finds them already
// committed, so this is a no-op there; a failed update leaves empty files
// that the next update overwrites (and a verify run still fails loudly on
// their emptiness vs the pinned output).
func EnsureExpectedSnapshots(c Case) error {
	for _, name := range expectedSnapshotNames(c) {
		path := filepath.Join(c.Dir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("golden: stat %s: %w", path, err)
		}
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("golden: pre-create %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("golden: pre-create %s: %w", path, err)
		}
	}
	return nil
}

// expectedSnapshotNames lists the snapshot files CheckStream may write for
// this case: one per format when stdout is captured, plus the single
// format-independent stderr snapshot when stderr is captured.
func expectedSnapshotNames(c Case) []string {
	var names []string
	if c.Snapshot == "stdout" || c.Snapshot == "both" {
		for _, f := range c.Formats {
			names = append(names, snapshotPrefix+f)
		}
	}
	if c.Snapshot == "stderr" || c.Snapshot == "both" {
		names = append(names, stderrSnapshot)
	}
	return names
}

// mismatchError renders the first divergent line pair so a red CI log
// points at the change instead of dumping two whole documents.
func mismatchError(path string, want, got []byte) error {
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")
	diff := "snapshot mismatch: " + path + "\n"
	n := max(len(wantLines), len(gotLines))
	shown := 0
	for i := 0; i < n && shown < 6; i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			diff += fmt.Sprintf("  line %d\n    want: %s\n    got:  %s\n", i+1, w, g)
			shown++
		}
	}
	diff += fmt.Sprintf("  (%d want lines, %d got lines) — if the change is intentional, run make golden-update and review the diff", len(wantLines), len(gotLines))
	return fmt.Errorf("%s", diff)
}

// CheckStream snapshots one stream of a run for one format, applying the
// case substitutions first. Stderr is snapshotted once per case (the first
// format that asks for it) because tool errors are format-independent.
func CheckStream(c Case, moduleRoot, format string, res RunResult, stderrDone *bool) error {
	subs := Substitutions(c.Dir, moduleRoot)
	if c.Snapshot == "stdout" || c.Snapshot == "both" {
		if err := Check(c.Dir, snapshotPrefix+format, Normalize(res.Stdout, subs)); err != nil {
			return err
		}
	}
	if (c.Snapshot == "stderr" || c.Snapshot == "both") && !*stderrDone {
		if err := Check(c.Dir, stderrSnapshot, Normalize(res.Stderr, subs)); err != nil {
			return err
		}
		*stderrDone = true
	}
	return nil
}
