package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/vanguard-lint/vanguard/internal/engine"
)

// ANSI escapes used by the pretty renderer (§6.6): the severity pill is the
// only colored token — ERROR red, WARN yellow, INFO cyan — plus bold for the
// rule id in group headers. Non-colored output emits none of these bytes.
const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiBold   = "\x1b[1m"
)

// ruleLineWidth is the width of the horizontal ─ rules, measured from the
// §6.6 mock (66 glyphs, 1-space left margin).
const ruleLineWidth = 66

// prettyRenderer draws the §6.6 mock: header, rule groups (severity desc,
// rule id asc), footer summary and next step. It groups findings by rule —
// not by file — because PR review fixes one rule at a time, and that order
// matches the SARIF rules[] order (§6.6 convention list).
type prettyRenderer struct{ cfg *config }

func (p prettyRenderer) Render(w io.Writer, report *engine.Report) error {
	report = orEmpty(report)
	ew := &errWriter{w: w}
	color := colorEnabled(p.cfg.color, writerIsTTY(w), noColorEnvSet())

	p.header(ew, report)
	ew.writeString("\n")

	groups := groupFindings(report)
	for _, g := range groups {
		p.groupHeader(ew, g, color)
		p.ruleLine(ew)
		p.findingBlocks(ew, g)
	}

	p.ruleLine(ew)
	p.footer(ew, report, groups)
	return ew.err
}

// header prints the two §6.6 header lines: identity + active config, then
// scan stats. Segments whose data is unavailable (no stats option, empty
// source field) are omitted — never printed as zeros.
func (p prettyRenderer) header(ew *errWriter, report *engine.Report) {
	segments := []string{toolName + " " + p.cfg.version}
	if report.Target != "" {
		segments = append(segments, report.Target)
	}
	if report.Source.Lang != "" {
		segments = append(segments, report.Source.Lang)
	}
	if framework := report.Source.Framework; framework != "" {
		if report.Source.FrameworkVersion != "" {
			framework += " " + report.Source.FrameworkVersion
		}
		segments = append(segments, framework)
	}
	if p.cfg.configPath != "" {
		segments = append(segments, "config: "+p.cfg.configPath)
	}
	ew.printf(" %s\n", strings.Join(segments, " · "))

	stats := p.cfg.stats
	statSegs := make([]string, 0, 5)
	if stats.Services > 0 {
		statSegs = append(statSegs, fmt.Sprintf("%d services", stats.Services))
	}
	if stats.Endpoints > 0 {
		statSegs = append(statSegs, fmt.Sprintf("%d endpoints", stats.Endpoints))
	}
	if stats.GrpcServices > 0 {
		statSegs = append(statSegs, fmt.Sprintf("%d grpc services", stats.GrpcServices))
	}
	if stats.RulesTotal > 0 {
		statSegs = append(statSegs, fmt.Sprintf("%d rules (%d active)", stats.RulesTotal, stats.RulesActive))
	}
	// Duration segment: only when the engine measured and the CLI passed a
	// real wall clock through (--timing, charter §6.6). The default
	// deterministic output (§5.4) renders without the segment rather than
	// printing a misleading "in 0ms".
	if report.DurationMS > 0 {
		statSegs = append(statSegs, fmt.Sprintf("%d files in %s", report.FilesScanned, formatDuration(report.DurationMS)))
	} else {
		statSegs = append(statSegs, fmt.Sprintf("%d files", report.FilesScanned))
	}
	ew.printf(" %s\n", strings.Join(statSegs, " · "))
}

// groupHeader prints e.g. " [ERROR] R2xx-01 get-no-body (3)" — severity pill
// (fixed width), rule id, slug from metadata when known, finding count.
func (p prettyRenderer) groupHeader(ew *errWriter, g ruleGroup, color bool) {
	line := " " + severityPill(g.severity, color)
	if color {
		line += ansiBold + g.ruleID + ansiReset
	} else {
		line += g.ruleID
	}
	if info, ok := p.cfg.ruleInfo(g.ruleID); ok && info.Slug != "" {
		line += " " + info.Slug
	}
	line += fmt.Sprintf(" (%d)", len(g.findings))
	ew.writeString(line + "\n")
}

// severityPill renders the severity token: colored word padded to the width
// of "ERROR" in color mode, bracket text in plain mode ("[ERROR]"/"[WARN]"/
// "[INFO]" — the CI-log-safe form, §6.6). Both include one trailing space.
func severityPill(s engine.Severity, color bool) string {
	switch s {
	case engine.SeverityError:
		if color {
			return ansiRed + "ERROR" + ansiReset + " "
		}
		return "[ERROR] "
	case engine.SeverityWarn:
		if color {
			return ansiYellow + "WARN" + ansiReset + "  "
		}
		return "[WARN]  "
	case engine.SeverityInfo:
		if color {
			return ansiCyan + "INFO" + ansiReset + "  "
		}
		return "[INFO]  "
	default:
		return "[" + string(s) + "] "
	}
}

func (prettyRenderer) ruleLine(ew *errWriter) {
	ew.printf(" %s\n", strings.Repeat("─", ruleLineWidth))
}

// findingBlocks prints each finding of the group: location, message,
// optional suggestion, and the suppression hint exactly once per group —
// on the first finding (§6.6 convention list).
func (prettyRenderer) findingBlocks(ew *errWriter, g ruleGroup) {
	for i, f := range g.findings {
		ew.printf(" %s:%d:%d\n", f.Location.File, f.Location.Line, f.Location.Column)
		ew.printf("   %s\n", f.Message)
		if f.Suggestion != "" {
			ew.printf("   suggest: %s\n", f.Suggestion)
		}
		if i == 0 {
			ew.printf("   suppress: // vanguard:ignore %s <reason>\n", g.ruleID)
		}
		ew.writeString("\n")
	}
}

// footer prints the §6.6 footer: separator, finding counts, file counts and
// the next-step line pointing at the densest ERROR group.
func (prettyRenderer) footer(ew *errWriter, report *engine.Report, groups []ruleGroup) {
	errs, warns, infos := countSeverities(report.Findings)
	ew.printf(" %d findings · %d ERROR · %d WARN · %d INFO · %d suppressed\n",
		len(report.Findings), errs, warns, infos, report.Suppressed)
	ew.printf(" %d files with findings · %d scanned · %d skipped · %d parse diagnostics\n",
		filesWithFindings(report.Findings), report.FilesScanned, report.FilesSkipped, len(report.Diagnostics))
	if rule := nextStepRule(groups); rule != "" {
		ew.printf(" next: vanguard explain %s\n", rule)
	}
}
