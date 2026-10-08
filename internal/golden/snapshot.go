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
