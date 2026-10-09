// Package robustness hosts the W4-05 hostile-input tests (GATE CH3,
// brief deliverable 1): the scanner's behaviour against malicious inputs,
// as opposed to the merely extreme ones of w4-03's edge cases
// (test-strategy §5.5 draws exactly this boundary).
//
// Contract under test — for every hostile input vanguard must
//
//   - not crash: exit code is the §6.3 tool code (0/1/2) with no Go panic
//     in stderr;
//   - not read outside the scan root: no finding, path or output byte may
//     reference content beyond the root (the walker's relFrom defense in
//     depth, internal/discovery walk.go — this package audits it end to
//     end through the real binary);
//   - finish in bounded time: a hostile config (glob bombs, deep trees,
//     parse bombs) may be slow, never unbounded.
//
// Hostile fixtures are generated at runtime in t.TempDir(): symlinks and
// deep chains do not survive git, and a committed hostile fixture is a
// foot-gun. Every binary invocation carries an explicit per-case timeout
// (process amendment 2); a timeout fails the test — that IS the finding.
package robustness
