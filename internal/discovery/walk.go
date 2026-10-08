package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Published performance caps (charter §5.3: "giới hạn depth/size — hằng số
// công bố"). Callers override both per scan via WalkOptions; zero or
// negative values fall back to these defaults.
const (
	// DefaultMaxDepth caps how far below the scan root the walker goes,
	// counted in path segments: a file directly in the root is at depth 1,
	// "a/b.go" at depth 2. 32 segments comfortably covers hand-written
	// trees while cutting pathological ones (and the truly deep generated
	// trees are excluded by name anyway).
	DefaultMaxDepth = 32

	// DefaultMaxFileSize caps the size of one file in bytes. Files above
	// the cap are skipped and reported — never read into memory (charter
	// B-7). 1 MiB is far above any hand-written source file, so the cap
	// only bites on generated/minified artifacts.
	DefaultMaxFileSize = 1 << 20 // 1 MiB
)

// Directories the walker never descends into (charter §5.3, verbatim list),
// matched by name at any depth — like gitignore, "build/" means every
// directory named build, not just one at the root.
var excludedDirs = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
	"target":       true,
	"build":        true,
}

// Skip reasons — stable strings printed in verbose mode, so every exclusion
// is explainable from the output alone (and matched by tests).
const (
	ReasonExcludedDir = "excluded directory"
	ReasonHidden      = "hidden"
	ReasonIgnoreFile  = "matched .vanguardignore"
	ReasonTooDeep     = "exceeds max depth"
	ReasonSymlink     = "symlink or special file (not followed)"
	ReasonUnreadable  = "unreadable"
	ReasonEscapesRoot = "path escapes scan root"
)

// SkipRecord notes one path the walker deliberately excluded. Skips are
// data, not log noise: Scan prints them in verbose mode, and tests assert
// on the reasons.
type SkipRecord struct {
	Path   string // relative to the scan root, "/" separators
	Reason string // a Reason* constant (the size reason names the cap)
}

// WalkResult is what a walk produced: the files to consider and the record
// of everything deliberately left out.
type WalkResult struct {
	Files   []string     // relative to root, "/" separators, sorted (B-6)
	Skipped []SkipRecord // sorted by path (B-6)
}

// FSWalker is the production Walker: it enumerates the candidate source
// files under a root on the local filesystem.
//
// Guarantees (all tested, charter §5.3):
//   - never descends into .git, vendor, node_modules, target, build, hidden
//     directories, or directories matched by the ignore file;
//   - skips hidden FILES too (config like .vanguardignore is never a
//     source; the charter names hidden directories, this is the honest
//     superset — a rule needs to know its ignore file is not scanned);
//   - never follows symlinks or other non-regular entries: a directory
//     symlink cannot loop or escape because the walk never enters one
//     (security + perf, the brief's mandatory test);
//   - never returns a path escaping the root: paths come from WalkDir under
//     an EvalSymlinks-resolved root and relFrom re-checks anyway (w4-05
//     audits again);
//   - files above the size cap and paths below the depth cap are skipped
//     with a recorded reason, never read;
//   - one unreadable entry never aborts the walk (best-effort contract) —
//     only root-level failures (missing, unreadable, not a directory) are
//     errors, per the w2-01 Walker contract.
type FSWalker struct{}

// NewWalker returns the filesystem walker.
func NewWalker() *FSWalker { return &FSWalker{} }

// Compile-time pin: FSWalker satisfies the w2-01 Walker contract.
var _ Walker = (*FSWalker)(nil)

// Walk implements Walker, returning just the files.
func (w *FSWalker) Walk(root string, opts WalkOptions) ([]string, error) {
	res, err := w.WalkWithStats(root, opts)
	if err != nil {
		return nil, err
	}
	return res.Files, nil
}

// WalkWithStats walks root and additionally reports what was skipped and
// why. See FSWalker for the guarantees.
func (w *FSWalker) WalkWithStats(root string, opts WalkOptions) (*WalkResult, error) {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = DefaultMaxDepth
	}
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = DefaultMaxFileSize
	}
	// Resolve the root up front: the walk must never see a symlinked root
	// (a loop through it would otherwise be reachable).
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("discovery: scan root %q: %w", root, err)
	}
	if info, err := os.Stat(resolved); err != nil {
		return nil, fmt.Errorf("discovery: scan root %q: %w", root, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("discovery: scan root %q is not a directory", root)
	}

	res := &WalkResult{Files: []string{}, Skipped: []SkipRecord{}}
	ignore := loadIgnoreRules(filepath.Join(resolved, opts.IgnoreFileName), opts.IgnoreFileName)

	err = filepath.WalkDir(resolved, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// The root itself being unreadable is a root-level failure;
			// any deeper entry is a recorded skip, never an abort.
			if rel, relErr := relFrom(resolved, p); relErr == nil && rel == "." {
				return walkErr
			}
			res.record(resolved, p, ReasonUnreadable+": "+walkErr.Error())
			return nil
		}
		rel, err := relFrom(resolved, p)
		if err != nil {
			res.record(resolved, p, ReasonEscapesRoot)
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if rel == "." {
			return nil // the root itself
		}
		segs := strings.Count(rel, "/") + 1 // path segments below the root

		if d.IsDir() {
			if segs >= opts.MaxDepth { // nothing inside can be ≤ MaxDepth
				res.record(resolved, p, ReasonTooDeep)
				return fs.SkipDir
			}
			if excludedDirs[d.Name()] {
				res.record(resolved, p, ReasonExcludedDir)
				return fs.SkipDir
			}
			if isHidden(d.Name()) {
				res.record(resolved, p, ReasonHidden)
				return fs.SkipDir
			}
			if ignore != nil && ignore.match(rel, true) {
				res.record(resolved, p, ReasonIgnoreFile)
				return fs.SkipDir
			}
			return nil
		}

		// Everything WalkDir yields as a non-directory that is not a
		// regular file: symlinks (never followed — see FSWalker), devices,
		// sockets, fifos.
		if !d.Type().IsRegular() {
			res.record(resolved, p, ReasonSymlink)
			return nil
		}
		if isHidden(d.Name()) {
			res.record(resolved, p, ReasonHidden)
			return nil
		}
		if segs > opts.MaxDepth {
			res.record(resolved, p, ReasonTooDeep)
			return nil
		}
		if ignore != nil && ignore.match(rel, false) {
			res.record(resolved, p, ReasonIgnoreFile)
			return nil
		}
		fi, err := d.Info() // lstat info: no file content is read here
		if err != nil {
			res.record(resolved, p, ReasonUnreadable+": "+err.Error())
			return nil
		}
		if fi.Size() > opts.MaxFileSize {
			res.record(resolved, p, fmt.Sprintf("exceeds max file size (%d bytes)", opts.MaxFileSize))
			return nil
		}
		res.Files = append(res.Files, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discovery: walking %q: %w", root, err)
	}
	// Deterministic output regardless of filesystem quirks (charter B-6).
	sort.Strings(res.Files)
	sort.Slice(res.Skipped, func(i, j int) bool { return res.Skipped[i].Path < res.Skipped[j].Path })
	return res, nil
}

// record appends a skip note, keeping the path relative when possible.
func (r *WalkResult) record(root, p, reason string) {
	rel, err := relFrom(root, p)
	if err != nil {
		rel = filepath.ToSlash(p) // last resort: the full path in the note
	}
	r.Skipped = append(r.Skipped, SkipRecord{Path: rel, Reason: reason})
}

// relFrom makes p relative to root with "/" separators, and refuses paths
// that leave the root. WalkDir only produces paths under root, so this is
// defense in depth for the w4-05 security audit — and it is what keeps
// "everything is relative to the scan root" true by construction.
func relFrom(root, p string) (string, error) {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
		return "", fmt.Errorf("%q escapes the scan root", rel)
	}
	return rel, nil
}

// isHidden: a leading "." — WalkDir never yields "." or ".." as names.
// Directories AND files: hidden entries are config, not API sources.
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// ignoreRules is the compiled .vanguardignore — a documented gitignore
// subset, deliberately small for v0.1:
//
//   - blank lines and '#' comments are ignored;
//   - a trailing '/' marks a directory-only rule;
//   - a rule containing '/' is anchored to the scan root ("docs/gen/*.tmp");
//   - any other rule matches one path segment at ANY depth ("notes.md",
//     "scratch/") — with glob syntax per path.Match;
//   - negation ('!') is not supported in v0.1;
//   - only the scan root's file is read; nested ignore files are out of
//     scope for v0.1.
//
// A missing (or unreadable) ignore file simply means "no rules" — the file
// is optional by design.
type ignoreRules struct {
	rules []ignoreRule
}

type ignoreRule struct {
	pattern  string
	dirOnly  bool
	anchored bool
}

func loadIgnoreRules(fullPath, name string) *ignoreRules {
	if name == "" {
		return nil
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil
	}
	var rs ignoreRules
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r := ignoreRule{pattern: line}
		if strings.HasSuffix(r.pattern, "/") {
			r.dirOnly = true
			r.pattern = strings.TrimSuffix(r.pattern, "/")
		}
		r.anchored = strings.Contains(r.pattern, "/")
		r.pattern = strings.TrimPrefix(r.pattern, "/") // a leading slash only anchors
		if r.pattern == "" {
			continue
		}
		rs.rules = append(rs.rules, r)
	}
	return &rs
}

// match reports whether rel (relative to the scan root, "/" separators)
// is excluded. isDir tells directory rules from file rules.
func (rs *ignoreRules) match(rel string, isDir bool) bool {
	for _, r := range rs.rules {
		if r.dirOnly && !isDir {
			continue
		}
		if r.anchored {
			if r.matchesPath(rel) {
				return true
			}
			continue
		}
		for _, seg := range strings.Split(rel, "/") {
			if ok, err := path.Match(r.pattern, seg); err == nil && ok {
				return true
			}
		}
	}
	return false
}

// matchesPath reports whether an anchored rule hits rel or one of its
// ancestor directories — "docs/gen" must also cover "docs/gen/x.txt".
func (r ignoreRule) matchesPath(rel string) bool {
	for {
		if ok, err := path.Match(r.pattern, rel); err == nil && ok {
			return true
		}
		slash := strings.LastIndex(rel, "/")
		if slash < 0 {
			return false
		}
		rel = rel[:slash]
	}
}
