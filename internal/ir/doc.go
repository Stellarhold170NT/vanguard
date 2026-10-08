// Package ir defines the ApiSurface intermediate representation that every
// adapter produces and every rule consumes.
//
// The IR is pure Go: it must not import tree-sitter or any language-specific
// parser. Every node carries a full Location{File, Line, Column} so findings
// can point at source, and the tree marshals to stable JSON for golden
// snapshots (charter §5.2; implementation: w2-02).
package ir
