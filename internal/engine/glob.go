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
func MatchGlob(pattern, path string) (bool, error) {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(path, "/"))
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

// matchSegments matches pattern segments against path segments. "**"
// matches zero or more segments; every other segment delegates to
// path.Match, whose ErrBadPattern propagates.
func matchSegments(pattern, segments []string) (bool, error) {
	if len(pattern) == 0 {
		return len(segments) == 0, nil
	}
	if pattern[0] == "**" {
		// "**" may consume any number of segments, including none.
		for i := 0; i <= len(segments); i++ {
			ok, err := matchSegments(pattern[1:], segments[i:])
			if ok || err != nil {
				return ok, err
			}
		}
		return false, nil
	}
	if len(segments) == 0 {
		return false, nil
	}
	matched, err := path.Match(pattern[0], segments[0])
	if err != nil || !matched {
		return false, err
	}
	return matchSegments(pattern[1:], segments[1:])
}
