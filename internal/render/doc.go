// Package render renders findings in the three output formats promised by
// the charter: pretty (human, grouped by rule), JSON (schema 1) and SARIF
// 2.1.0 (charter §6.6–6.8; implementation w2-05).
//
// Contract notes for maintainers and for the CLI (w2-06) that wires this
// package:
//
//   - Determinism: rendering the same Report twice is byte-identical. The
//     renderer re-sorts findings (file, line, col, ruleId) instead of
//     trusting the caller, groups sort severity desc (ERROR→WARN→INFO) then
//     rule id asc, and no output iteration walks a bare map. Output contains
//     no wall-clock timestamp, hostname, PID or absolute path (test strategy
//     §5.4). DurationMS is engine-measured state, not render-time clock
//     reading — but it is wall-clock, so the CLI only forwards it under
//     --timing (w4-03): the default rendered report is byte-identical across
//     runs, and the pretty header prints the duration segment only when a
//     real measurement was passed in.
//   - Streaming: all three renderers write incrementally to the io.Writer —
//     a 10k-finding report never materializes as one giant string (w2-05
//     trap list, perf work w4-03).
//   - Rule metadata (slug, descriptions, help link, default severity) is not
//     part of engine.Report; the CLI passes it via WithRules. Missing
//     metadata degrades gracefully: the JSON slug is omitted and the SARIF
//     shortDescription falls back to the rule id — never fabricated.
//   - Color (pretty only): auto = TTY && !NO_COLOR (no-color.org), over-
//     ridable with WithColor. Non-colored output replaces the colored
//     severity pill with the plain text pill "[ERROR]"/"[WARN]"/"[INFO]" so
//     CI logs stay free of escape bytes (§6.6).
//   - SARIF: severity maps ERROR→error, WARN→warning, INFO→note (GitHub
//     convention §6.8); uris are the scan-root-relative file paths with "/"
//     separators; regions carry UTF-16 code unit columns per the declared
//     columnKind (w2-02); partialFingerprints[vanguardFindingV1] =
//     sha256(ruleId|uri|startLine|startColumn|message)[:16] exists from day
//     one so GitHub dedups unchanged findings.
package render
