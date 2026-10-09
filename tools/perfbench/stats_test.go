package main

import (
	"testing"
	"time"
)

// TestPercentileNearestRank pins §5.3's nearest-rank formula: rank is
// rounded UP, and p stays clamped to [0,100].
func TestPercentileNearestRank(t *testing.T) {
	// n=5: rank(50)=ceil(2.5)=3 → third smallest; rank(95)=ceil(4.75)=5 → max.
	sample := []time.Duration{2, 4, 6, 8, 10}
	for _, tc := range []struct {
		p    float64
		want time.Duration
	}{
		{50, 6}, {0, 2}, {100, 10}, {95, 10}, {99, 10}, {-5, 2}, {200, 10},
	} {
		if got := Percentile(sample, tc.p); got != tc.want {
			t.Errorf("p=%.0f of %v = %v, want %v", tc.p, sample, got, tc.want)
		}
	}

	// n=4: rank(50)=2 → 2nd of 4; rank(51)=ceil(2.04)=3.
	four := []time.Duration{10, 20, 30, 40}
	if got := Percentile(four, 50); got != 20 {
		t.Errorf("p50 of 4-element sample = %v, want 20", got)
	}
	if got := Percentile(four, 51); got != 30 {
		t.Errorf("p51 of 4-element sample = %v, want 30", got)
	}

	// Unsorted input must behave identically (and stay unchanged).
	shuffled := []time.Duration{40, 10, 30, 20}
	if got := Percentile(shuffled, 50); got != 20 {
		t.Errorf("p50 must not depend on input order, got %v", got)
	}
	if shuffled[0] != 40 || shuffled[1] != 10 {
		t.Fatalf("Percentile mutated its input: %v", shuffled)
	}

	// Degenerate: n=1 returns the single sample at every percentile.
	if got := Percentile([]time.Duration{7}, 95); got != 7 {
		t.Errorf("p95 of one sample = %v, want 7", got)
	}
}

// TestSummarize checks the aggregate fields on a known series, including the
// §5.3 minimum-five-runs consequence p95 == max.
func TestSummarize(t *testing.T) {
	const files = 10000
	raws := []float64{10, 30, 20, 40, 25}
	ds := make([]time.Duration, len(raws))
	for i, r := range raws {
		ds[i] = time.Duration(r * float64(time.Second))
	}
	s := Summarize(ds, files, 60)
	if s.N != 5 || s.Files != files {
		t.Fatalf("N/Files = %d/%d", s.N, s.Files)
	}
	if s.MinSeconds != 10 || s.MaxSeconds != 40 {
		t.Errorf("min/max = %.1f/%.1f, want 10/40", s.MinSeconds, s.MaxSeconds)
	}
	if s.P50Seconds != 25 {
		t.Errorf("p50 = %.1f, want 25", s.P50Seconds)
	}
	if s.P95Seconds != 40 {
		t.Errorf("p95 = %.1f, want 40 (nearest-rank, n=5 → slowest run)", s.P95Seconds)
	}
	if s.P99Seconds != 40 {
		t.Errorf("p99 = %.1f, want 40", s.P99Seconds)
	}
	if s.MeanSeconds != 25 {
		t.Errorf("mean = %.1f, want 25", s.MeanSeconds)
	}
	if !s.P95UnderTarget {
		t.Error("p95 40s must be under the 60s target")
	}
	// Throughput at p95: files / slowest run, down-converted from seconds.
	if got := files / s.P95Seconds; s.FilesPerSecP95 != got {
		t.Errorf("filesPerSecP95 = %.2f, want %.2f", s.FilesPerSecP95, got)
	}
	if s.FilesPerSecMax != files/s.MinSeconds {
		t.Errorf("filesPerSecMax = %.2f, want %.2f", s.FilesPerSecMax, files/s.MinSeconds)
	}
}

func TestSummarizeReportsMissedTargetUntouched(t *testing.T) {
	ds := []time.Duration{120 * time.Second, 130 * time.Second, 125 * time.Second}
	s := Summarize(ds, 10000, 60)
	if s.P95UnderTarget {
		t.Error("p95 above target misreported as passing")
	}
	if s.P95Seconds != 130 {
		t.Errorf("p95 = %.1f, want 130 (raw, unmodified)", s.P95Seconds)
	}
}
