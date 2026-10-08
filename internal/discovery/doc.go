// Package discovery walks a source tree, detects language and framework,
// and selects the adapter that parses the API surface — the machinery
// behind "vanguard scan /path auto-detects everything" (charter §5.3, R1).
//
// # Pipeline
//
// Scan composes the three pieces this package provides:
//
//	walk    FSWalker enumerates candidate files (caps and ignores below)
//	detect  Detect ranks language candidates from file names, with evidence
//	select  SelectAdapter asks every registered adapter; the first Detect
//	        win parses and returns (*ir.ApiSurface, []ir.Diagnostic)
//
// Scan is the w2-06 CLI's engine room: the cobra layer adds flags, output
// formats and the exit-code mapping on top.
//
// # Path conventions and security groundwork (w4-05 audits again)
//
// Every path this package produces or consumes is relative to the scan
// root, "/"-separated, and never contains "..". The walker structurally
// cannot produce one (paths come from WalkDir under an EvalSymlinks-
// resolved root, and relFrom re-checks anyway); StubAdapter.read re-checks
// and refuses such paths before opening anything. Symlinks are never
// followed: a symlinked directory can neither loop nor escape because the
// walk never enters one, and symlinked files/other non-regular entries are
// skipped with a recorded reason.
//
// # Performance guards (published constants, charter B-7)
//
// DefaultMaxDepth (32 path segments) and DefaultMaxFileSize (1 MiB) bound
// the walk; oversized files are skipped with a note, never read. Parse-level
// reads are independently bounded (StubMaxFileBytes) so no single file can
// dominate memory even when Parse is called directly.
//
// # Ignore rules
//
// The walker skips .git, vendor/, node_modules/, target/, build/ (by name,
// any depth) and hidden entries, and honors the scan root's
// .vanguardignore — a gitignore subset ('#' comments, trailing '/' =
// directory-only, '/' inside = anchored to the root, otherwise basename
// match at any depth; no '!' negation in v0.1). Nested ignore files are
// v0.1-out-of-scope.
//
// # "No API found" is success
//
// A repo where nothing is detected — no adapter claims the files, or the
// adapter parses zero services/types — is a normal outcome: ScanResult
// carries a human-readable NoAPI explanation, and the charter pins exit
// code 0 for it (§6.3: "repo không phát hiện API surface cũng là 0 — với
// thông báo rõ + summary"). internal/cli (w2-06) prints that message and
// returns 0. Only root-level failures (missing, unreadable or
// non-directory root) return an error — tool-error territory (§6.3, code 2).
//
// # Best-effort contract
//
// Discovery follows the best-effort contract: a file that fails to parse
// becomes a diagnostic — never a fatal error, and never a change of exit
// code (charter §5.3). Concretely: one unreadable tree entry never aborts
// the walk, one unparseable stub file never aborts the parse, and the stub
// adapter validates its own output against the w2-02 invariants so adapter
// bugs surface as diagnostics, not silent IR corruption.
package discovery
