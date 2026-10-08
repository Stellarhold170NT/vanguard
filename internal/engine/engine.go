package engine

import "github.com/Stellarhold170NT/vanguard/internal/ir"

// Severity is the finding severity taxonomy (charter §3.0). Only ERROR
// affects the exit code (§6.3); WARN/INFO are display-only.
type Severity string

const (
	SeverityError Severity = "ERROR"
	SeverityWarn  Severity = "WARN"
	SeverityInfo  Severity = "INFO"
)

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
// predicates (e.g. "Method Verb=GET", "Type.IsEntity") that replace
// api-linter's OnlyIf, restated over IR concepts (charter §5.4).
type Selector interface {
	Matches(node ir.Node) bool
}

// CheckFunc is the pure check signature every rule implements (charter
// §5.4): no I/O, no global state; findings derive only from ctx and node.
type CheckFunc func(ctx *LintContext, node ir.Node) []Finding

// Rule binds one rule's metadata to its check. The metadata itself lives in
// the YAML data files under rules/ (go:embed, w3-03); the registry joins
// metadata and check by id (charter §3.0, §5.4).
type Rule struct {
	ID              string   // R<F>xx-NN
	Slug            string   // kebab-case, e.g. "get-no-body"
	Category        string   // family category, e.g. "methods"
	Severity        Severity // metadata default severity
	DefaultDisabled bool     // true for opt-in policy rules (R6xx-01)
	Selector        Selector
	Check           CheckFunc
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
// keyed by rule id or family prefix.
type RuleOverride struct {
	Disabled bool
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
// validation (unknown version/options → exit 2, §6.3) belong to w2-04.
type Config struct {
	Version      int                     // only 1 is accepted in v0.1
	Frameworks   map[string][]string     // empty = auto-detect (§6.4.1)
	Include      []string                // globs relative to config dir
	Exclude      []string                // applied after built-in ignores
	Rules        map[string]RuleOverride // by id or family prefix
	Suppressions []PathSuppression
}

// LintContext is the per-run state handed to every check: the effective
// config (rule options, severity overrides), the suppression state, and the
// diagnostic sink. TODO(w2-04): concrete fields.
type LintContext struct{}

// Suppression answers, per finding, whether an inline comment or a
// path-scoped config suppression covers it. Inline wins over config, and
// the check re-runs per finding, not per rule (charter §6.4.2, §6.5).
type Suppression interface {
	// Suppressed reports whether f is suppressed, and why (for --verbose).
	Suppressed(f Finding) (bool, string)
}

// Report is one scan's outcome, consumed by render (w2-05) and by the
// exit-code decision (§6.3: ≥1 ERROR → exit 1). Findings are sorted
// (file, line, col, ruleId) — renderers rely on that order (§5.4,
// §6.6–6.8). Suppressed findings are counted, not carried.
type Report struct {
	Source       ir.Source
	Target       string    // scan root as given on the command line
	Findings     []Finding // sorted (file, line, col, ruleId)
	Suppressed   int
	Diagnostics  []ir.Diagnostic
	FilesScanned int
	FilesSkipped int
	DurationMS   int64
}

// Engine applies the registered rules to an ApiSurface (charter §5.4).
// Implementations must: filter nodes by each rule's Selector, run checks,
// stamp Finding.RuleID/Severity, apply suppression per finding, isolate a
// panicking rule into a diagnostic while other rules continue, sort
// findings (file, line, col, ruleId), and iterate deterministically — never
// over bare maps (charter B-6). The engine knows nothing about Java or
// Spring (charter §12).
type Engine interface {
	// Run lints surface under cfg (nil cfg = all default-enabled rules at
	// metadata severities) and returns the report.
	Run(surface *ir.ApiSurface, cfg *Config) (*Report, error)
}
