// Package engine implements the data-driven rule engine: selector and
// predicate filtering over the IR, the rule registry that binds rule
// metadata (YAML, go:embed) to check functions, config (.vanguard.yaml)
// parsing and discovery, suppression handling, and per-rule panic
// isolation.
//
// The engine is language-agnostic — it must not know anything about Java
// or Spring (charter §5.4; implementation: w2-04).
//
// # Exit codes and severity
//
// ExitCode implements the CI contract of charter §6.3 (the w2-06 CLI maps
// it 1:1):
//
//	┌──────┬────────────────────────────────────────────────────────────┐
//	│ code │ condition (after config + suppression)                     │
//	├──────┼────────────────────────────────────────────────────────────┤
//	│  0   │ clean scan, or findings only at WARN/INFO                  │
//	│  1   │ ≥ 1 finding with severity ERROR                            │
//	│  2   │ tool error (config unreadable/bad schema, unknown rule     │
//	│      │ option, nil registry/surface) — never prints findings      │
//	└──────┴────────────────────────────────────────────────────────────┘
//
// Per severity (§3.0):
//
//	ERROR — design mistakes that break the contract: counts toward exit 1
//	WARN  — convention violations with evidence: display only, exit 0
//	INFO  — soft recommendations: display only, exit 0
//
// WARN/INFO never change the exit code (the spectral behaviour adopted in
// w1-02); the --severity flag is a display threshold and likewise never
// changes it. Exit 2 outranks everything: a tool error means the scan did
// not happen, so there is nothing to judge.
//
// # Config discovery
//
// Load parses one .vanguard.yaml; Discover walks up from the start
// directory and returns the nearest one (§6.4.2 #1 — nearest wins, no
// multi-level merge in v0.1). Every glob in the file is relative to the
// directory that contains it (Config.Dir).
//
// # Suppression layers
//
// Applied per finding, in order (§6.4.2 #3, §6.5): inline
// vanguard:ignore comment directives win over path-scoped config
// suppressions; a suppressed finding is counted (Report.Suppressed) and
// noted (Report.SuppressedNotes) instead of carried. Disabled rules are a
// config decision, not a suppression — they produce no notes.
package engine
