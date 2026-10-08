package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
)

// DefaultVersion is the tool version used when WithVersion is not given
// (charter §1: vanguard 0.1.0). The CLI (w2-06) overrides it with the
// ldflags-injected build version, falling back to "dev".
const DefaultVersion = "0.1.0"

// Tool identity and SARIF constants, verbatim from the §6.8 mock.
const (
	sarifSchemaURI     = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"
	sarifVersion       = "2.1.0"
	toolName           = "vanguard"
	toolInformationURI = "https://github.com/Stellarhold170NT/vanguard"
	fingerprintKey     = "vanguardFindingV1"
)

// Format is one output format (charter §6.2 flags).
type Format string

const (
	FormatPretty Format = "pretty"
	FormatJSON   Format = "json"
	FormatSARIF  Format = "sarif"
)

// errUnknownFormat marks a --format value outside §6.2 (the CLI maps it to
// exit 2, §6.3).
var errUnknownFormat = errors.New("unknown output format")

// Renderer writes one report in one format. Implementations must be
// deterministic — rendering the same report twice yields byte-identical
// output (charter §6.6, w4-03) — and write only findings + summary to w;
// tool errors go to stderr at the CLI layer (§6.3). All three renderers
// stream to w: a 10k-finding report never materializes as one string.
type Renderer interface {
	Render(w io.Writer, report *engine.Report) error
}

// RuleInfo is the rule metadata the machine-readable formats need but
// engine.Report does not carry (charter §6.7 slug, §6.8 rule descriptors).
// The CLI (w2-06) fills it from the rule registry (w3-03); the renderers
// never fabricate metadata that is missing.
type RuleInfo struct {
	ID              string          // R<F>xx-NN, matches Finding.RuleID
	Slug            string          // kebab-case name, e.g. "get-no-body"
	Summary         string          // one-line summary → SARIF shortDescription
	Description     string          // long text → SARIF fullDescription
	HelpURI         string          // docs link → SARIF helpUri
	Category        string          // rule family category, e.g. "methods"
	AIP             string          // AIP reference, e.g. "131"
	DefaultSeverity engine.Severity // metadata default → SARIF defaultConfiguration
}

// ScanStats carries the scan-wide counters the pretty header shows
// (§6.6 line 2) that the engine Report does not carry yet. Zero-valued
// fields are omitted from the header, so w2-06 can pass only what the
// engine provides at the time.
type ScanStats struct {
	Services     int
	Endpoints    int
	GrpcServices int
	RulesTotal   int // 0 = unknown → the rules segment is omitted
	RulesActive  int
}

// ColorMode selects how the pretty renderer treats ANSI color (§6.6):
// auto = TTY && !NO_COLOR; the explicit modes win over both.
type ColorMode int

const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

// config is the resolved option set shared by the three renderers.
type config struct {
	version    string
	rules      map[string]RuleInfo
	stats      ScanStats
	configPath string
	color      ColorMode
}

// Option configures the renderers returned by For.
type Option func(*config)

// WithVersion overrides the tool version printed in the pretty header and
// written to the JSON "vanguard" field and the SARIF driver version.
func WithVersion(v string) Option { return func(c *config) { c.version = v } }

// WithRules supplies rule metadata for slug/description enrichment. Missing
// entries degrade gracefully (JSON omits the slug; SARIF falls back to the
// rule id as shortDescription).
func WithRules(rules map[string]RuleInfo) Option { return func(c *config) { c.rules = rules } }

// WithStats passes the header counters (§6.6 line 2).
func WithStats(s ScanStats) Option { return func(c *config) { c.stats = s } }

// WithConfigPath prints the active config in the pretty header (§6.6 via
// §6.4.2: --verbose surfaces the config path there).
func WithConfigPath(p string) Option { return func(c *config) { c.configPath = p } }

// WithColor forces the color mode; default is ColorAuto.
func WithColor(m ColorMode) Option { return func(c *config) { c.color = m } }

// For returns the renderer for format. Unknown formats fail loudly instead
// of silently rendering nothing.
func For(format Format, opts ...Option) (Renderer, error) {
	cfg := &config{version: DefaultVersion, color: ColorAuto}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	switch format {
	case FormatPretty:
		return prettyRenderer{cfg: cfg}, nil
	case FormatJSON:
		return jsonRenderer{cfg: cfg}, nil
	case FormatSARIF:
		return sarifRenderer{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("render: %q: %w", format, errUnknownFormat)
	}
}

// ---------------------------------------------------------------------------
// Shared helpers (grouping, counting, JSON writing, color, fingerprint)
// ---------------------------------------------------------------------------

// ruleGroup is one (severity, rule) bucket of findings. The engine stamps
// one severity per rule (config overrides are per rule), so in practice a
// rule owns exactly one group.
type ruleGroup struct {
	severity engine.Severity
	ruleID   string
	findings []engine.Finding
}

// sortedFindings returns a copy of the report's findings in the engine
// contract order (file, line, col, ruleId) without mutating the report
// (renderers never mutate their input). Every streamed findings/results
// array walks this order, so determinism does not depend on the caller's
// slice order (charter B-6, §6.7).
func sortedFindings(report *engine.Report) []engine.Finding {
	sorted := make([]engine.Finding, len(report.Findings))
	copy(sorted, report.Findings)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].Location, sorted[j].Location
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return sorted[i].RuleID < sorted[j].RuleID
	})
	return sorted
}

// groupFindings buckets the report's findings per rule. Groups sort severity
// desc (ERROR→WARN→INFO) then rule id asc (charter §6.6).
func groupFindings(report *engine.Report) []ruleGroup {
	sorted := sortedFindings(report)
	groups := make([]ruleGroup, 0, 8)
	pos := make(map[string]int, 8) // severity\x00ruleID → index in groups
	for _, f := range sorted {
		key := string(f.Severity) + "\x00" + f.RuleID
		if p, ok := pos[key]; ok {
			groups[p].findings = append(groups[p].findings, f)
			continue
		}
		pos[key] = len(groups)
		groups = append(groups, ruleGroup{severity: f.Severity, ruleID: f.RuleID, findings: []engine.Finding{f}})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if rank := severityRank(groups[i].severity) - severityRank(groups[j].severity); rank != 0 {
			return rank < 0
		}
		return groups[i].ruleID < groups[j].ruleID
	})
	return groups
}

// severityRank orders the §3.0 taxonomy: ERROR first (exit-code severity),
// then WARN, INFO; unknown severities sink to the end deterministically.
func severityRank(s engine.Severity) int {
	switch s {
	case engine.SeverityError:
		return 0
	case engine.SeverityWarn:
		return 1
	case engine.SeverityInfo:
		return 2
	default:
		return 3
	}
}

// countSeverities tallies the three taxonomy severities for summaries.
func countSeverities(findings []engine.Finding) (errs, warns, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case engine.SeverityError:
			errs++
		case engine.SeverityWarn:
			warns++
		case engine.SeverityInfo:
			infos++
		}
	}
	return errs, warns, infos
}

// filesWithFindings counts distinct files carrying at least one finding.
func filesWithFindings(findings []engine.Finding) int {
	set := make(map[string]struct{}, len(findings))
	for _, f := range findings {
		set[f.Location.File] = struct{}{}
	}
	return len(set)
}

// nextStepRule picks the rule for the §6.6 "next:" line: the ERROR group
// with the most findings (ties → lowest rule id, i.e. first in group
// order); with no ERRORs, the first group overall.
func nextStepRule(groups []ruleGroup) string {
	if len(groups) == 0 {
		return ""
	}
	best := ""
	bestCount := -1
	for _, g := range groups {
		if g.severity != engine.SeverityError {
			break // ERROR groups precede all others (groupFindings order)
		}
		if len(g.findings) > bestCount {
			bestCount = len(g.findings)
			best = g.ruleID
		}
	}
	if best != "" {
		return best
	}
	return groups[0].ruleID
}

// errWriter funnels all writes through one error slot so renderer code can
// stay linear: the first write error wins and later writes become no-ops.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) writeString(s string) {
	if ew.err != nil {
		return
	}
	_, ew.err = io.WriteString(ew.w, s)
}

// marshalValue encodes v with 2-space indentation and HTML escaping off —
// stable, diffable wire bytes ("<", "&" stay readable inside messages).
func marshalValue(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// marshalCompact encodes v on one line (scalar key values).
func marshalCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// writeJSONValue writes v so that its first line sits at the given depth
// (2 spaces per level) and nested lines keep their relative indent. This is
// what lets the renderers stream one array element at a time while keeping
// a fully indented document.
func writeJSONValue(ew *errWriter, v any, depth int) {
	b, err := marshalValue(v)
	if err != nil {
		if ew.err == nil {
			ew.err = err
		}
		return
	}
	pad := strings.Repeat("  ", depth)
	for i, line := range strings.Split(string(b), "\n") {
		if i > 0 {
			ew.writeString("\n")
		}
		ew.writeString(pad)
		ew.writeString(line)
	}
}

// writeJSONValueInline writes v so that its first line continues the current
// line (after a hand-written `"key": `) and the remaining lines indent to
// depth. The keyed members of a streamed object ("source", "summary", the
// SARIF "tool" block) use this variant: writeJSONValue would pad the first
// line too and leave stray spaces between the key and the value.
func writeJSONValueInline(ew *errWriter, v any, depth int) {
	b, err := marshalValue(v)
	if err != nil {
		if ew.err == nil {
			ew.err = err
		}
		return
	}
	pad := strings.Repeat("  ", depth)
	for i, line := range strings.Split(string(b), "\n") {
		if i > 0 {
			ew.writeString("\n")
			ew.writeString(pad)
		}
		ew.writeString(line)
	}
}

// jstr encodes one string as a JSON literal (for hand-written key lines).
func jstr(ew *errWriter, s string) string {
	b, err := marshalCompact(s)
	if err != nil {
		if ew.err == nil {
			ew.err = err
		}
		return `""`
	}
	return string(b)
}

// findingFingerprint implements the §6.8 partialFingerprints formula:
// hash(ruleId|uri|startLine|startColumn|message)[:16] with SHA-256 — stable
// across platforms and runs, so GitHub dedups unchanged findings.
func findingFingerprint(ruleID, uri string, line, column int, message string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%d|%d|%s", ruleID, uri, line, column, message)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// formatDuration renders a duration the way the §6.6 footer shows it:
// raw milliseconds below 1s, one decimal in seconds above.
func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

// writerIsTTY reports whether w is a terminal character device (the only
// case where §6.6 allows color in auto mode).
func writerIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// noColorEnvSet honors the NO_COLOR convention (no-color.org): any non-empty
// value disables auto color.
func noColorEnvSet() bool { return os.Getenv("NO_COLOR") != "" }

// colorEnabled resolves the §6.6 color matrix: auto = TTY && !NO_COLOR;
// the explicit modes always win.
func colorEnabled(mode ColorMode, tty, noColorEnv bool) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		return tty && !noColorEnv
	}
}

// sarifLevel maps the §3.0 severity taxonomy onto the GitHub/SARIF levels
// (§6.8): ERROR→error, WARN→warning, INFO→note.
func sarifLevel(s engine.Severity) string {
	switch s {
	case engine.SeverityError:
		return "error"
	case engine.SeverityWarn:
		return "warning"
	case engine.SeverityInfo:
		return "note"
	default:
		return "none"
	}
}

// orEmpty guards against a nil report so renderers degrade to the empty
// shape instead of panicking.
func orEmpty(report *engine.Report) *engine.Report {
	if report == nil {
		return &engine.Report{}
	}
	return report
}

// ruleSlug returns the metadata slug for a rule id, "" when unknown.
func (c *config) ruleSlug(id string) string {
	if info, ok := c.rules[id]; ok {
		return info.Slug
	}
	return ""
}

// ruleInfo returns the metadata entry for a rule id.
func (c *config) ruleInfo(id string) (RuleInfo, bool) {
	info, ok := c.rules[id]
	return info, ok
}
