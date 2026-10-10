package engine

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/vanguard-lint/vanguard/internal/ir"
)

// Linter is the v0.1 Engine implementation: it applies every registered
// rule to an ApiSurface under an optional Config. The CLI layer (w2-06)
// fills Comments (from the adapter) and Target (the command-line root)
// before calling Run.
type Linter struct {
	Registry Registry
	Comments CommentIndex
	Target   string
}

// NewLinter returns a Linter over reg. A nil registry is accepted here and
// reported as a tool error by Run (the CLI maps it to exit 2).
func NewLinter(reg Registry) *Linter {
	return &Linter{Registry: reg}
}

// Run lints surface under cfg and returns the report. Errors (nil
// registry, nil surface, undeclared rule option, invalid hand-built
// suppression entries) are the exit-2 tool-error class: they never produce
// a partial report.
func (l *Linter) Run(surface *ir.ApiSurface, cfg *Config) (*Report, error) {
	if l == nil || l.Registry == nil {
		return nil, errors.New("engine: no rule registry configured")
	}
	if surface == nil {
		return nil, errors.New("engine: no API surface to lint")
	}
	var duration durationMeasure
	// start() must run BEFORE the lint work, not via defer: a deferred call
	// only fires after ms() was already read, leaving DurationMS at a
	// constant 0 (found by w4-03: the §5.4 determinism gate passed on a
	// dead timer, and the perf bench would have measured nothing).
	duration.start()

	inline, inlineDiags := NewInlineSuppression(l.Comments)
	var pathSup *PathSuppressions
	if cfg != nil && len(cfg.Suppressions) > 0 {
		ps, err := NewPathSuppressions(cfg.Suppressions)
		if err != nil {
			return nil, err
		}
		pathSup = ps
	}

	report := &Report{
		Source:          surface.Source,
		Target:          l.Target,
		Diagnostics:     append([]ir.Diagnostic(nil), surface.Diagnostics...),
		SuppressedNotes: []SuppressedNote{},
	}
	report.Diagnostics = append(report.Diagnostics, inlineDiags...)

	dead := map[string]bool{} // rules that panicked — skipped for the rest of the run
	for _, rule := range l.Registry.All() {
		res := resolveRule(cfg, rule)
		if res.Disabled {
			continue // disabled is a config decision, not a suppression
		}
		if err := checkOptions(rule, res); err != nil {
			return nil, err
		}
		ctx := &LintContext{Surface: surface, Config: cfg, Options: res.Options}
		walkSurface(surface, func(node ir.Node, owner *ir.Service) {
			if dead[rule.ID] || !rule.Selector.Matches(node) {
				return
			}
			ctx.Service = owner
			findings, panicValue, panicked := invokeCheck(rule, ctx, node)
			if panicked {
				dead[rule.ID] = true
				report.Diagnostics = append(report.Diagnostics, ir.Diagnostic{
					Message:  fmt.Sprintf("rule %s (%s) panicked: %v — rule skipped for the rest of this run", rule.ID, rule.Slug, panicValue),
					Location: node.Loc(),
				})
				return
			}
			for _, f := range findings {
				f.RuleID = rule.ID // engine stamps; checks never set them (§5.4)
				f.Severity = res.Severity
				if ok, reason := inline.Suppressed(f); ok {
					report.Suppressed++
					report.SuppressedNotes = append(report.SuppressedNotes, note(rule.ID, SourceInline, reason, f.Location))
					continue
				}
				if pathSup != nil {
					if ok, reason := pathSup.Suppressed(f); ok {
						report.Suppressed++
						report.SuppressedNotes = append(report.SuppressedNotes, note(rule.ID, SourceConfig, reason, f.Location))
						continue
					}
				}
				report.Findings = append(report.Findings, f)
			}
		})
	}

	sortFindings(report.Findings)
	report.DurationMS = duration.ms()
	return report, nil
}

func note(ruleID, source, reason string, loc ir.Location) SuppressedNote {
	return SuppressedNote{RuleID: ruleID, Source: source, Reason: reason, Location: loc}
}

// walkSurface visits every IR node exactly once, in a fixed order
// (services → methods → params → response → pagination → types → fields →
// grpc services → error handlers) so rule output and reports are
// deterministic (charter B-6). owner is the Service containing the node,
// nil outside services.
func walkSurface(surface *ir.ApiSurface, visit func(node ir.Node, owner *ir.Service)) {
	for i := range surface.Services {
		s := &surface.Services[i]
		visit(*s, s)
		for j := range s.Methods {
			m := &s.Methods[j]
			visit(*m, s)
			for k := range m.Params {
				visit(m.Params[k], s)
			}
			visit(m.Response, s)
			visit(m.Pagination, s)
		}
	}
	for i := range surface.Types {
		t := &surface.Types[i]
		visit(*t, nil)
		for j := range t.Fields {
			visit(t.Fields[j], nil)
		}
	}
	for i := range surface.GrpcServices {
		visit(surface.GrpcServices[i], nil)
	}
	for i := range surface.ErrorScheme.Handlers {
		visit(surface.ErrorScheme.Handlers[i], nil)
	}
}

// invokeCheck calls one check with panic isolation (charter §5.4): a
// panicking rule is reported, not fatal — the engine drops the rule's
// findings for that node and the caller stops running the rule entirely.
func invokeCheck(rule Rule, ctx *LintContext, node ir.Node) (findings []Finding, panicValue any, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			findings, panicValue, panicked = nil, r, true
		}
	}()
	return rule.Check(ctx, node), nil, false
}

// checkOptions rejects option keys the rule metadata does not declare
// (charter §6.4.1: "chỉ nhận options khai báo trong metadata — sai = exit 2").
func checkOptions(rule Rule, res resolvedRule) error {
	if len(res.Options) == 0 {
		return nil
	}
	declared := make(map[string]bool, len(rule.Options))
	for _, o := range rule.Options {
		declared[o] = true
	}
	unknown := make([]string, 0, len(res.Options))
	for k := range res.Options {
		if !declared[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("rule %s: unknown option %q (declared options: %v)", rule.ID, unknown[0], rule.Options)
}

// sortFindings applies the B-6 order (file, line, column, ruleId) that
// renderers and golden snapshots depend on.
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i].Location, findings[j].Location
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return findings[i].RuleID < findings[j].RuleID
	})
}

// ExitCode maps a Run outcome onto the CI contract (charter §6.3, matched
// by the w2-06 CLI):
//
//	2 — err != nil (tool error: unreadable/invalid config, unknown option,
//	    …) or a missing report; nothing is printed as findings on this path
//	1 — at least one surviving finding has severity ERROR
//	0 — clean scan, or only WARN/INFO findings
func ExitCode(report *Report, err error) int {
	if err != nil || report == nil {
		return 2
	}
	for _, f := range report.Findings {
		if f.Severity == SeverityError {
			return 1
		}
	}
	return 0
}

// durationMeasure is the wall-clock the engine reports per run; it is a
// distinct type so Run's timing stays out of the linting logic.
type durationMeasure struct {
	started time.Time
}

func (d *durationMeasure) start() { d.started = time.Now() }

func (d *durationMeasure) ms() int64 {
	if d.started.IsZero() {
		return 0
	}
	return time.Since(d.started).Milliseconds()
}
