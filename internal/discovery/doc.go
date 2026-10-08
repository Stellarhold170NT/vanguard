// Package discovery walks a source tree, detects language and framework,
// and selects the adapter that parses the API surface.
//
// Discovery follows the best-effort contract: a file that fails to parse
// becomes a diagnostic — never a fatal error, and never a change of exit
// code (charter §5.3; implementation: w2-03).
package discovery
