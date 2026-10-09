// Package java adapts Java sources into raw API candidates — the
// intermediate extraction layer between the file walk (internal/discovery)
// and the Spring-aware IR mapping (w3-02, the architecture boundary this
// package must not cross).
//
// Parsing uses tree-sitter-java through github.com/smacker/go-tree-sitter:
// the grammar ships as bundled C sources compiled via cgo, so no external
// grammar build step is needed (choice rationale in reports/w3-01.md).
//
// One pass per file extracts, in source order:
//   - package; class/interface/record/enum declarations, nested included,
//     with their modifiers and annotations;
//   - methods with modifiers, type parameters, return type, parameters
//     (varargs included) and annotations — constructors are not captured;
//   - fields and record components with their annotations;
//   - annotation arguments raw: strings keep their quotes, arrays keep
//     their braces, nested annotations keep their full text.
//
// Deliberately absent: any annotation semantics. Reading @GetMapping as
// GET, merging request paths and translating to the final ir.ApiSurface
// is w3-02, which consumes the Result.Candidates view; un-annotated DTO
// records and POJOs stay reachable through Result.Files.
//
// Behavior contract (the stub adapter's, kept): Parse is best-effort —
// unreadable, oversized and syntactically broken files each yield one
// ir.Diagnostic naming the file and the reason, and the remaining files
// keep scanning; nothing ever aborts the pass. Files above the discovery
// size cap are refused without being fully loaded. Locations follow the
// IR contract: 1-based line, 1-based column in UTF-16 code units.
package java
