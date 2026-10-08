package discovery

import "github.com/Stellarhold170NT/vanguard/internal/ir"

// Evidence explains a Detect decision. Verbose mode prints it so a wrong
// adapter choice is debuggable from the CLI alone (charter §5.3).
type Evidence struct {
	Reason  string   // one-line verdict, e.g. "pom.xml present"
	Details []string // supporting facts, e.g. counts per extension
}

// Candidate is one ranked language/framework hypothesis produced by Detect.
type Candidate struct {
	Lang      string
	Framework string
	Evidence  Evidence
}

// Adapter is the charter §5.3 contract verbatim. Adapters (adapters/*, W3)
// implement it and register in a Registry; internal/cli (w2-06) drives
// walk → detect → pick adapter → Parse.
type Adapter interface {
	// Language names the adapter, e.g. "java" — the registry key and the
	// verbose-mode tag.
	Language() string
	// Detect reports whether this adapter should parse the given file set,
	// plus the evidence shown in verbose mode.
	Detect(files []string) (bool, Evidence)
	// Parse builds the API surface best-effort: per-file failures come back
	// as diagnostics — never an error, never an abort (charter §5.3, the
	// deliberate break from api-linter's compile-gate).
	Parse(files []string) (*ir.ApiSurface, []ir.Diagnostic)
}

// Registry stores adapters by language. w2-03 implements it together with
// the selection logic (scan picks the adapter whose Detect wins; sparse
// matches surface their evidence in verbose mode).
type Registry interface {
	// Register adds an adapter; a duplicate language is an error.
	Register(a Adapter) error
	// Adapters lists registered adapters in registration order.
	Adapters() []Adapter
}

// WalkOptions bounds the tree walk (charter §5.3). Zero values mean the
// published defaults that w2-03 chooses and documents.
type WalkOptions struct {
	MaxDepth       int    // directory depth cap; 0 = default
	MaxFileSize    int64  // per-file size cap in bytes; 0 = default
	IgnoreFileName string // e.g. ".vanguardignore"; empty = none
}

// Walker enumerates candidate source files under root as paths relative to
// root. Per charter §5.3 it skips .git, vendor/, node_modules/, target/,
// build/ and hidden directories, never follows symlink loops, and never
// returns a path escaping the root via "..". The returned error covers
// root-level failures only (missing/unreadable root); individual unreadable
// entries are counted and surfaced in verbose mode, not fatal.
type Walker interface {
	Walk(root string, opts WalkOptions) ([]string, error)
}

// Detect ranks language/framework candidates for a walked file set using
// extensions plus build files (pom.xml/build.gradle → java, go.mod → go,
// requirements.txt/pyproject.toml → python — charter §5.3). Callers print
// the winner's evidence in verbose mode.
//
// TODO(w2-03): implement; the nil return keeps the pipeline callable until
// then.
func Detect(files []string) []Candidate {
	return nil // TODO(w2-03): ranked candidates with evidence
}
