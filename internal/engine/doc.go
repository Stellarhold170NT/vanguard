// Package engine implements the data-driven rule engine: selector and
// predicate filtering over the IR, the rule registry that binds rule
// metadata (YAML, go:embed) to check functions, suppression handling, and
// per-rule panic isolation.
//
// The engine is language-agnostic — it must not know anything about Java or
// Spring (charter §5.4; implementation: w2-04).
package engine
