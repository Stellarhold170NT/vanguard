package engine

import (
	"strings"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// Severity is the finding severity taxonomy (charter §3.0). Only ERROR
// affects the exit code (§6.3); WARN/INFO are display-only.
type Severity string

const (
	SeverityError Severity = "ERROR"
	SeverityWarn  Severity = "WARN"
	SeverityInfo  Severity = "INFO"
)

// ParseSeverity normalizes a severity spelling to its canonical constant.
// Config files and rule metadata may spell severities in any case ("warn",
// "Warn"); the canonical values are upper-case (§3.0). The second return
// value is false for anything that is not one of the three constants —
// callers turn that into a schema error (exit 2, §6.3).
func ParseSeverity(s string) (Severity, bool) {
	out := Severity(strings.ToUpper(strings.TrimSpace(s)))
	switch out {
	case SeverityError, SeverityWarn, SeverityInfo:
		return out, true
	default:
		return "", false
	}
}

// Finding is one lint result. Checks fill Message, Suggestion and Location;
// the engine stamps RuleID and Severity (rule metadata default, then config
// overrides) — rules never set them themselves (charter §5.4).
type Finding struct {
	RuleID     string      // R<F>xx-NN (charter §3.0), engine-stamped
	Severity   Severity    // engine-stamped: metadata default, then config
	Message    string      // one English sentence, reason first (§3.0)
	Suggestion string      // replacement text for the span; "" when none
	Location   ir.Location // the exact node the finding is about
}

// Selector filters which IR nodes a rule applies to — the framework-level
// predicate (e.g. "Method Verb=GET", "Type.IsEntity") spelled over IR
// concepts, never over a concrete language (charter §5.4: the engine must
// not know Java or Spring; selectors live with the rule data).
type Selector interface {
	Matches(node ir.Node) bool
}

// CheckFunc is the pure check signature every rule implements (charter
// §5.4): no I/O, no global state; findings derive only from ctx and node.
// Message/Suggestion/Location are the check's job; RuleID/Severity are
// stamped by the engine afterwards (§5.4), so checks never set them.
type CheckFunc func(ctx *LintContext, node ir.Node) []Finding

// Rule binds one rule's metadata to its check. Metadata fields are the
// Go projection of the rule's YAML record (rules package, w3-03): the
// engine consumes the joined result and never reads rule data files or
// imports single rules itself.
type Rule struct {
	ID       string   // R<F>xx-NN
	Slug     string   // kebab-case, e.g. "get-no-body"
	Category string   // family category, e.g. "methods"
	Severity Severity // metadata default severity

	// Summary is the one-line human description shown by explain/pretty;
	// DocPath points at the rule's documentation page (w3-07 owns the
	// files — the engine only carries the path).
	Summary string
	DocPath string

	// ExampleGood/ExampleBad are data-driven usage examples (brief w2-04:
	// "rule là data") rendered by explain and the rule docs.
	ExampleGood string
	ExampleBad  string

	// DefaultDisabled marks opt-in policy rules (charter §3.6, R6xx-01):
	// they stay off unless a config override enables them.
	DefaultDisabled bool

	// Options lists the option keys the check reads (e.g. "allow").
	// Config overrides may only set declared keys; an unknown key is a
	// tool error (exit 2) — enforced by Linter.Run.
	Options []string

	Selector Selector
	Check    CheckFunc
}

// Registry resolves rules by exact id or family prefix ("R1xx" → all R1xx
// rules, charter §6.4.2). The rules package (w3-03+) provides the
// production registry; the engine only consumes it — it never imports
// single rules.
type Registry interface {
	// Register adds a rule. Registration errors (bad id format, duplicate,
	// missing fixtures) fail tests, never runtime (charter §3.0, §5.4).
	Register(r Rule) error
	// ByID resolves one rule by exact id.
	ByID(id string) (Rule, bool)
	// All lists rules sorted by id, for deterministic iteration (B-6).
	All() []Rule
}

// RuleOverride is the per-rule .vanguard.yaml override (charter §6.4.1),
// keyed by rule id or family prefix. Disabled is a tri-state: nil means the
// config did not mention it (the metadata default stands), true/false are
// explicit — an exact `disabled: false` can re-enable a rule its family
// disables, and vice versa (§6.4.2 #4: disable matches first, enable after).
type RuleOverride struct {
	Disabled *bool
	Severity Severity       // "" = keep the metadata default
	Options  map[string]any // only keys declared in rule metadata (§6.4.1)
}

// PathSuppression is one path-scoped suppression entry (charter §6.4.1).
type PathSuppression struct {
	Rule   string   // rule id or family prefix
	Paths  []string // globs relative to the config file directory
	Reason string
}

// Config mirrors the .vanguard.yaml schema v1 (charter §6.4.1). Parsing and
// validation (unknown version/options → exit 2, §6.3) live in config.go;
// Load fills Path and Dir so consumers (suppression globs, renderers'
// "config:" header line) can anchor relative paths.
type Config struct {
	Version      int                     // only 1 is accepted in v0.1
	Frameworks   map[string][]string     // empty = auto-detect (§6.4.1)
	Include      []string                // globs relative to config dir; default ["**"]
	Exclude      []string                // applied after built-in ignores
	Rules        map[string]RuleOverride // by id or family prefix
	Suppressions []PathSuppression

	// Path is the config file as loaded; Dir the directory containing it —
	// the anchor every include/exclude/suppression glob is relative to
	// (§6.4.1). Both are zero for a hand-built Config (tests, embedders).
	Path string
	Dir  string
}

// LintContext is the per-run state handed to every check: the scanned
// surface, the effective config, the owning Service of the node being
// visited (nil outside services — Method.Params/Response/Pagination
// inherit their Method's Service; w2-02 §11 promised this), and the
// rule's merged options (family prefix, then exact id; §6.4.2 #4).
//
// Suppression is deliberately NOT in the context: it is applied by the
// engine after stamping, per finding (§6.4.2 #3, §5.4 "suppression
// per-problem"). A check cannot pre-check it — a finding has no RuleID
// until the engine stamps one.
//
// Checks must treat all fields as read-only.
type LintContext struct {
	Surface *ir.ApiSurface // the scanned surface (never nil inside Run)
	Config  *Config        // the config in effect (nil = defaults)
	Service *ir.Service    // owning Service, nil outside services
	Options map[string]any // merged options for the current rule, nil when none
}

// Suppression answers, per finding, whether an inline comment or a
// path-scoped config suppression covers it. Inline wins over config, and
// the check re-runs per finding, not per rule (charter §6.4.2, §6.5).
// Implementations: InlineSuppression and PathSuppressions (suppression.go).
type Suppression interface {
	// Suppressed reports whether f is suppressed, and why (for --verbose).
	Suppressed(f Finding) (bool, string)
}

// SuppressedNote records one suppressed finding for --verbose (charter
// §6.5: "số suppressions được đếm và in ở summary; --verbose note từng
// cái"). Source says which layer suppressed it: "inline" (comment
// directive) or "config" (path-scoped suppression).
type SuppressedNote struct {
	RuleID   string
	Source   string // "inline" | "config"
	Reason   string // the directive/config reason, "" when none given
	Location ir.Location
}

// Report is one scan's outcome, consumed by render (w2-05) and by the
// exit-code decision (§6.3: ≥1 ERROR → exit 1). Findings are sorted
// (file, line, col, ruleId) — renderers rely on that order (§5.4,
// §6.6–6.8). Suppressed findings are counted, not carried; each leaves a
// SuppressedNote. Diagnostics carry the surface's parse diagnostics, the
// inline-directive diagnostics and one diagnostic per panicking rule —
// none of them change the exit code (§5.3).
type Report struct {
	Source          ir.Source
	Target          string           // scan root as given on the command line
	Findings        []Finding        // sorted (file, line, col, ruleId)
	Suppressed      int              // count of suppressed findings
	SuppressedNotes []SuppressedNote // one per suppressed finding, verbose input
	Diagnostics     []ir.Diagnostic
	FilesScanned    int   // filled by the CLI layer, not the engine
	FilesSkipped    int   // filled by the CLI layer
	DurationMS      int64 // engine-side wall time, >= 0
}

// Engine applies the registered rules to an ApiSurface (charter §5.4).
// Implementations must: filter nodes by each rule's Selector, run checks,
// stamp Finding.RuleID/Severity, apply suppression per finding, isolate a
// panicking rule into a diagnostic while other rules continue, sort
// findings (file, line, col, ruleId), and iterate deterministically — never
// over bare maps (charter B-6). The engine knows nothing about Java or
// Spring (charter §12). Linter is the v0.1 implementation (run.go).
type Engine interface {
	// Run lints surface under cfg (nil cfg = all default-enabled rules at
	// metadata severities) and returns the report.
	Run(surface *ir.ApiSurface, cfg *Config) (*Report, error)
}
