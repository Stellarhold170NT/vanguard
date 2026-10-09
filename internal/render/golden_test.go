package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden rewrites the golden files when set:
//
//	go test ./internal/render/ -run TestGoldenSnapshots -update
//
// The diff of a rewritten golden must appear in the reviewing PR — that is
// the "intentional change" contract from the test strategy (tier 2.1).
var updateGolden = flag.Bool("update", false, "rewrite the golden snapshot files")

// TestGoldenSnapshots locks the exact bytes of all three formats for the one
// sample finding set (w2-05 deliverable 3). The render runs with ColorNever
// and no wall-clock input beyond the report's own DurationMS, so the files
// stay stable across machines, TTY states and runs.
func TestGoldenSnapshots(t *testing.T) {
	for _, tc := range []struct {
		format Format
		file   string
	}{
		{FormatPretty, "sample.pretty"},
		{FormatJSON, "sample.json"},
		{FormatSARIF, "sample.sarif"},
	} {
		t.Run(string(tc.format), func(t *testing.T) {
			got := renderSample(t, tc.format)
			path := filepath.Join("testdata", "golden", tc.file)
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir testdata/golden: %v", err)
				}
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("golden file missing (generate with `go test ./internal/render/ -update`): %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.file, got, want)
			}
		})
	}
}
