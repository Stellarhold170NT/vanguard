package engine

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/vanguard-lint/vanguard/internal/ir"
)

// ruleRefPattern is the id/prefix vocabulary shared by config rule keys,
// config suppressions and inline directives: an exact id (R1xx-02) or a
// family prefix (R1xx), families 1..6 per charter §3.0.
var ruleRefPattern = regexp.MustCompile(`^R[1-6]xx(-[0-9]{2})?$`)

// ValidRuleRef reports whether s is a well-formed rule id or family
// prefix. References that fail this check never suppress anything
// (fail-open, §6.5) and surface a diagnostic instead.
func ValidRuleRef(s string) bool {
	return ruleRefPattern.MatchString(s)
}

// ruleRefMatches reports whether a directive/config reference covers a
// finding's rule id: exact id, or family prefix of that id (§6.5 "Match id
// đầy đủ hoặc prefix họ").
func ruleRefMatches(ref, ruleID string) bool {
	if ref == ruleID {
		return true
	}
	return len(ref) == 4 && strings.HasPrefix(ruleID, ref)
}

// Comment is one source comment extracted by the adapter (charter §5.2:
// the adapter strips comments with their Location so the engine never
// re-reads files). Text is the raw text including any "//" marker.
type Comment struct {
	Location ir.Location
	Text     string
}

// CommentIndex indexes the adapter's comments by file (relative to the
// scan root, "/"-separated — the same form as ir.Location.File).
type CommentIndex map[string][]Comment

// SuppressedNote sources.
const (
	SourceInline = "inline"
	SourceConfig = "config"
)

// directive is one parsed vanguard:ignore comment.
type directive struct {
	kind   string // "line" | "begin" | "end"
	ref    string // rule id or family prefix (validated)
	reason string // everything after the ref, "" when absent
	loc    ir.Location
	bad    string // non-empty = malformed directive (message, never suppresses)
}

// directivePrefix is the marker every suppression directive starts with.
const directivePrefix = "vanguard:ignore"

// parseDirective extracts a suppression directive from a comment. Anything
// that is not a vanguard:ignore directive returns ok=false and is ignored.
// A text that starts like a directive but is malformed returns ok=true with
// bad set — the caller surfaces a diagnostic and the directive suppresses
// nothing (fail-open, §6.5).
func parseDirective(c Comment) (directive, bool) {
	text := strings.TrimSpace(c.Text)
	// Strip the language's comment marker; the directive itself is
	// language-agnostic (Java "//" today, other languages reuse it).
	for _, marker := range []string{"//", "/*", "*"} {
		if strings.HasPrefix(text, marker) {
			text = strings.TrimSpace(text[len(marker):])
			break
		}
	}
	if !strings.HasPrefix(text, directivePrefix) {
		return directive{}, false
	}
	d := directive{loc: c.Location}
	rest := text[len(directivePrefix):]

	// Block forms first: their leading "-" is part of the directive word
	// ("-begin"/"-end"), not a missing whitespace separator.
	d.kind = "line"
	for _, form := range []struct{ suffix, kind string }{{"-begin", "begin"}, {"-end", "end"}} {
		if strings.HasPrefix(rest, form.suffix) {
			after := rest[len(form.suffix):]
			if after == "" || (after[0] != ' ' && after[0] != '\t') {
				d.bad = fmt.Sprintf("rule id must be separated from %q by whitespace", directivePrefix+form.suffix)
				return d, true
			}
			d.kind = form.kind
			rest = after[1:]
			break
		}
	}
	if d.kind == "line" {
		switch {
		case rest == "":
			d.bad = `missing rule id — want "// vanguard:ignore <rule-id> [reason]"`
			return d, true
		case rest[0] != ' ' && rest[0] != '\t':
			// "vanguard:ignoreR6xx-99" — the rule id must be separated by space.
			d.bad = fmt.Sprintf("rule id must be separated from the directive by whitespace (got %q)", directivePrefix+rest)
			return d, true
		}
		rest = rest[1:]
		if strings.HasPrefix(rest, "-") {
			// "-begin"/"-end" were consumed above; any other "-xxx" is an
			// unknown directive form.
			d.bad = fmt.Sprintf("unknown directive form %q", directivePrefix+rest)
			return d, true
		}
	}

	rest = strings.TrimSpace(rest)
	if rest == "" {
		d.bad = `missing rule id — want "// vanguard:ignore <rule-id> [reason]"`
		return d, true
	}
	fields := strings.Fields(rest)
	d.ref = fields[0]
	if !ValidRuleRef(d.ref) {
		d.bad = fmt.Sprintf("%q is not a rule id or family prefix (R<1-6>xx[-NN])", d.ref)
		return d, true
	}
	if idx := strings.Index(rest, d.ref); idx >= 0 {
		d.reason = strings.TrimSpace(rest[idx+len(d.ref):])
	}
	return d, true
}

// InlineSuppression answers suppression queries from the adapter's comment
// index (charter §6.5): a line directive covers findings on its own line
// and the line immediately below; a begin/end block covers every line in
// between, inclusive. Matching is per file — directives never leak across
// files.
type InlineSuppression struct {
	byFile map[string]fileDirectives
}

type fileDirectives struct {
	lines  []directive
	blocks []lineBlock
}

type lineBlock struct {
	ref, reason string
	from, to    int // to == math.MaxInt for an unterminated begin (lenient)
}

// NewInlineSuppression parses every comment into directives. Problems are
// returned as diagnostics anchored at the offending comment — malformed
// directives, an ignore-begin without a matching ignore-end, and a stray
// ignore-end. None of them abort the scan (best-effort, §5.3) and none of
// them suppress (fail-open).
func NewInlineSuppression(idx CommentIndex) (*InlineSuppression, []ir.Diagnostic) {
	sup := &InlineSuppression{byFile: make(map[string]fileDirectives, len(idx))}
	var diags []ir.Diagnostic
	for file, comments := range idx {
		var fd fileDirectives
		var open []directive
		for _, c := range comments {
			d, ok := parseDirective(c)
			if !ok {
				continue // an ordinary comment
			}
			if d.bad != "" {
				diags = append(diags, ir.Diagnostic{
					Message:  "invalid vanguard:ignore directive: " + d.bad,
					Location: c.Location,
				})
				continue
			}
			switch d.kind {
			case "line":
				fd.lines = append(fd.lines, d)
			case "begin":
				open = append(open, d)
			case "end":
				match := -1
				for i := len(open) - 1; i >= 0; i-- {
					if open[i].ref == d.ref {
						match = i
						break
					}
				}
				if match < 0 {
					diags = append(diags, ir.Diagnostic{
						Message:  fmt.Sprintf("vanguard:ignore-end for %s has no matching vanguard:ignore-begin in this file", d.ref),
						Location: d.loc,
					})
					continue
				}
				begin := open[match]
				fd.blocks = append(fd.blocks, lineBlock{ref: begin.ref, reason: begin.reason, from: begin.loc.Line, to: d.loc.Line})
				open = append(open[:match], open[match+1:]...)
			}
		}
		// Unterminated begins keep suppressing to end of file (lenient —
		// the scan stays useful) but stay visible as diagnostics.
		for _, begin := range open {
			diags = append(diags, ir.Diagnostic{
				Message:  fmt.Sprintf("vanguard:ignore-begin for %s is never closed — block suppresses to end of file", begin.ref),
				Location: begin.loc,
			})
			fd.blocks = append(fd.blocks, lineBlock{ref: begin.ref, reason: begin.reason, from: begin.loc.Line, to: math.MaxInt})
		}
		if len(fd.lines) > 0 || len(fd.blocks) > 0 {
			sup.byFile[file] = fd
		}
	}
	return sup, diags
}

// Suppressed implements Suppression. Inline is consulted before the config
// layer, so it wins whenever both cover a finding (§6.4.2 #3).
func (s *InlineSuppression) Suppressed(f Finding) (bool, string) {
	fd, ok := s.byFile[f.Location.File]
	if !ok {
		return false, ""
	}
	line := f.Location.Line
	for _, d := range fd.lines {
		if ruleRefMatches(d.ref, f.RuleID) && (line == d.loc.Line || line == d.loc.Line+1) {
			return true, d.reason
		}
	}
	for _, b := range fd.blocks {
		if ruleRefMatches(b.ref, f.RuleID) && line >= b.from && line <= b.to {
			return true, b.reason
		}
	}
	return false, ""
}

// PathSuppressions is the config-side, path-scoped suppression layer
// (charter §6.4.1: buf ignore_only style). An entry covers a finding when
// its rule reference matches AND one of its path globs matches the
// finding's file.
type PathSuppressions struct {
	entries []PathSuppression
}

// NewPathSuppressions validates every entry up front: rule references must
// be well-formed and every glob must parse (ValidateGlob) so a typo fails
// loudly instead of silently suppressing nothing.
func NewPathSuppressions(entries []PathSuppression) (*PathSuppressions, error) {
	for _, e := range entries {
		if !ValidRuleRef(e.Rule) {
			return nil, fmt.Errorf("suppression rule reference %q is not a rule id or family prefix (R<1-6>xx[-NN])", e.Rule)
		}
		for _, g := range e.Paths {
			if err := ValidateGlob(g); err != nil {
				return nil, fmt.Errorf("suppression for %s: invalid glob %q: %w", e.Rule, g, err)
			}
		}
	}
	return &PathSuppressions{entries: entries}, nil
}

// Suppressed implements Suppression for the config layer.
func (p *PathSuppressions) Suppressed(f Finding) (bool, string) {
	for _, e := range p.entries {
		if !ruleRefMatches(e.Rule, f.RuleID) {
			continue
		}
		for _, g := range e.Paths {
			if ok, err := MatchGlob(g, f.Location.File); err == nil && ok {
				return true, e.Reason
			}
		}
	}
	return false, ""
}
