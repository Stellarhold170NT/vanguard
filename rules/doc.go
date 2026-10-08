// Package rules hosts the R1xx–R6xx rule families: one metadata record per
// rule (id, slug, category, default severity, AIP reference, good/bad
// examples) stored as data and embedded into the binary at build time, plus
// the Go check functions and the registry that binds them (charter §3 and
// §5.1; implementation: w3-03..w3-06).
package rules
