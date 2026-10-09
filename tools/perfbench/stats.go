package main

import (
	"sort"
	"time"
)

// Summary aggregates one benchmark series. Every field is derived from the
// raw per-run numbers — nothing here invents or rounds away data
// (anti-gaming rule §9.2).
type Summary struct {
	N               int       `json:"n"`
	Files           int       `json:"files"`
	RawSeconds      []float64 `json:"rawSeconds"`
	P50Seconds      float64   `json:"p50Seconds"`
	P95Seconds      float64   `json:"p95Seconds"`
	P99Seconds      float64   `json:"p99Seconds"`
	MinSeconds      float64   `json:"minSeconds"`
	MaxSeconds      float64   `json:"maxSeconds"`
	MeanSeconds     float64   `json:"meanSeconds"`
	FilesPerSecP95  float64   `json:"filesPerSecP95"`
	FilesPerSecMax  float64   `json:"filesPerSecMax"`
	TargetSeconds   float64   `json:"targetSeconds"`
	P95UnderTarget  bool      `json:"p95UnderTarget"`
	MeasurementNote string    `json:"measurementNote,omitempty"`
}

// Percentile returns the nearest-rank percentile p of a NON-EMPTY sample of
// wall-clock durations (test strategy §5.3: rank = ⌈p/100·n⌉, clamped to
// [1, n] — so with the n=5 floor, p95 is conservatively the slowest run).
// The input slice is not modified; the sort happens on a copy, so the raw
// evidence always stays printable in the order it happened.
func Percentile(sample []time.Duration, p float64) time.Duration {
	if len(sample) == 0 {
		panic("bench: Percentile of empty sample")
	}
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	sorted := append([]time.Duration(nil), sample...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	rank := int(mathCeil(float64(len(sorted)) * p / 100))
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

func mathCeil(f float64) float64 {
	if float64(int64(f)) == f {
		return f
	}
	return float64(int64(f)) + 1
}

// Summarize folds the raw per-run wall-times into the §5.3 aggregates:
// nearest-rank p50/p95/p99, min/max/mean, and per-run throughput derived
// from the file count (files / seconds). Raw seconds stay in RawSeconds so
// the summary never replaces its own evidence (§9.5: every number has a
// source, every raw number is quotable).
func Summarize(runs []time.Duration, files int, targetSeconds float64) Summary {
	if len(runs) == 0 {
		panic("bench: Summarize of empty run set")
	}
	raws := make([]float64, len(runs))
	var total float64
	for i, d := range runs {
		raws[i] = d.Seconds()
		total += raws[i]
	}
	p95 := Percentile(runs, 95).Seconds()
	s := Summary{
		N:              len(runs),
		Files:          files,
		RawSeconds:     raws,
		P50Seconds:     Percentile(runs, 50).Seconds(),
		P95Seconds:     p95,
		P99Seconds:     Percentile(runs, 99).Seconds(),
		MinSeconds:     Percentile(runs, 0).Seconds(),
		MaxSeconds:     Percentile(runs, 100).Seconds(),
		MeanSeconds:    total / float64(len(runs)),
		TargetSeconds:  targetSeconds,
		P95UnderTarget: p95 <= targetSeconds,
	}
	if s.P95Seconds > 0 {
		s.FilesPerSecP95 = float64(files) / s.P95Seconds
	}
	if s.MinSeconds > 0 {
		s.FilesPerSecMax = float64(files) / s.MinSeconds
	}
	return s
}
