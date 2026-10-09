package engine

import (
	"path"
	"strings"
)

// MatchGlob reports whether path (always "/"-separated, relative to the
// config directory — charter §6.4.1) matches the glob pattern.
//
// Dialect (pinned by TestMatchGlob): "**" spans whole path segments only
// — "**/generated/**", "src/**", "**" — while "*", "?" and "[...]" stay
// within one segment with path.Match semantics. A malformed pattern
// (unclosed character class) is an error, never a silent non-match:
// ValidateGlob runs the same code at config-load time so a typo fails the
// config (exit 2) instead of matching nothing at scan time.
//
// Evaluation is memoized on (pattern, segment) index pairs: a hostile
// config glob — dozens of `**` segments that never match — otherwise
// fork-walks every distribution of path segments across the wildcards,
// exponential in practice (w4-05 measurement: 8 `**` segments ≈ 9.2 s and
// 10+ beyond 15 s on a 32-segment path; suppression matching runs this per
// finding, so a hostile .vanguard.yaml hung the scan). The memo preserves
// the dialect and the first-match-wins semantics exactly, capping the work
// at O(len(pattern) · len(path)²).
func MatchGlob(pattern, path string) (bool, error) {
	m := &globMatcher{
		pattern:  strings.Split(pattern, "/"),
		segments: strings.Split(path, "/"),
		memo:     make(map[globKey]globOutcome),
	}
	return m.match(0, 0)
}

// ValidateGlob checks a pattern without a subject: every "/"-segment is
// parsed by path.Match, so a malformed segment (an unterminated "[ab")
// is an error even when the pattern's segment count could never match a
// real path.
func ValidateGlob(pattern string) error {
	for _, segment := range strings.Split(pattern, "/") {
		if _, err := path.Match(segment, ""); err != nil {
			return err
		}
	}
	return nil
}

// globMatcher is the matchSegments recursion over a memo: match(pi, si) is
// a pure function of the remaining pattern and path slices, so each state
// is computed once and every revisit is a map hit.
type globMatcher struct {
	pattern  []string
	segments []string
	memo     map[globKey]globOutcome
}

type globKey struct{ pattern, segment int }

type globOutcome struct {
	matched bool
	err     error
}

func (m *globMatcher) match(pi, si int) (bool, error) {
	if pi == len(m.pattern) {
		return si == len(m.segments), nil
	}
	key := globKey{pi, si}
	if out, ok := m.memo[key]; ok {
		return out.matched, out.err
	}
	out := m.compute(pi, si)
	m.memo[key] = out
	return out.matched, out.err
}

// compute is the verbatim matchSegments body, indexed instead of sliced:
// "**" may consume any number of segments (first match wins, errors
// propagate immediately); every other segment delegates to path.Match,
// whose ErrBadPattern propagates.
func (m *globMatcher) compute(pi, si int) globOutcome {
	if m.pattern[pi] == "**" {
		for i := si; i <= len(m.segments); i++ {
			ok, err := m.match(pi+1, i)
			if ok || err != nil {
				return globOutcome{ok, err}
			}
		}
		return globOutcome{false, nil}
	}
	if si == len(m.segments) {
		return globOutcome{false, nil}
	}
	matched, err := path.Match(m.pattern[pi], m.segments[si])
	if err != nil || !matched {
		return globOutcome{false, err}
	}
	ok, err := m.match(pi+1, si+1)
	return globOutcome{ok, err}
}
