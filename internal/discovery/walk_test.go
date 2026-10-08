package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree materializes a fixture tree from rel-path → content (directories
// are implicit) and returns the root.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// walked runs the walker with stats and fails the test on a root-level error.
func walked(t *testing.T, root string, opts WalkOptions) *WalkResult {
	t.Helper()
	res, err := NewWalker().WalkWithStats(root, opts)
	if err != nil {
		t.Fatalf("walk %q: %v", root, err)
	}
	return res
}

// findSkip returns the skip record for one relative path, if any.
func findSkip(res *WalkResult, rel string) (SkipRecord, bool) {
	for _, sk := range res.Skipped {
		if sk.Path == rel {
			return sk, true
		}
	}
	return SkipRecord{}, false
}

// TestWalkSkipsExcludedAndHiddenDirs pins charter §5.3's exclusion list:
// .git, vendor, node_modules, target, build (by name, at any depth) and
// hidden directories are never entered. Hidden files are skipped too — they
// are config (like .vanguardignore itself), never API sources. Every
// exclusion must leave a skip record so verbose mode can explain it.
func TestWalkSkipsExcludedAndHiddenDirs(t *testing.T) {
	root := writeTree(t, map[string]string{
		".git/config":            "x",
		"vendor/v.go":            "x",
		"node_modules/m.js":      "x",
		"target/t.java":          "x",
		"build/b.go":             "x",
		".hidden/h.go":           "x",
		"sub/.secret.go":         "x",
		"main.go":                "x",
		"svc/dep/build/out.java": "x", // excluded at depth, not just at root
		"svc/Svc.java":           "x",
	})
	res := walked(t, root, WalkOptions{})
	want := []string{"main.go", "svc/Svc.java"}
	if strings.Join(res.Files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", res.Files, want)
	}
	for _, dir := range []string{".git", "vendor", "node_modules", "target", "build", ".hidden"} {
		sk, ok := findSkip(res, dir)
		if !ok {
			t.Errorf("excluded dir %q missing from skip log %v", dir, res.Skipped)
			continue
		}
		if sk.Reason != ReasonExcludedDir && sk.Reason != ReasonHidden {
			t.Errorf("skip %q reason = %q, want %q or %q", dir, sk.Reason, ReasonExcludedDir, ReasonHidden)
		}
	}
	if sk, ok := findSkip(res, "sub/.secret.go"); !ok || sk.Reason != ReasonHidden {
		t.Errorf("hidden file skip = %+v, want reason %q", sk, ReasonHidden)
	}
}

// TestWalkHonorsVanguardIgnore covers the documented gitignore subset:
// '#' comments, a basename rule matching at any depth, a trailing '/' as a
// directory-only rule, and a rule containing '/' anchored to the scan root.
func TestWalkHonorsVanguardIgnore(t *testing.T) {
	root := writeTree(t, map[string]string{
		".vanguardignore":   "# generated\nscratch/\nnotes.md\ndocs/gen/*.tmp\n",
		"root.go":           "x",
		"scratch/s.go":      "x",
		"deep/scratch/s.go": "x", // directory rule hits at any depth
		"notes.md":          "x",
		"sub/notes.md":      "x", // basename rule hits at any depth
		"docs/gen/a.tmp":    "x",
		"docs/gen/keep.go":  "x",
		"docs/other.tmp":    "x", // anchored glob must not leak outside docs/gen
	})
	res := walked(t, root, WalkOptions{IgnoreFileName: ".vanguardignore"})
	want := []string{"docs/gen/keep.go", "docs/other.tmp", "root.go"}
	if strings.Join(res.Files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", res.Files, want)
	}
	for _, ignored := range []string{"scratch", "deep/scratch", "notes.md", "sub/notes.md", "docs/gen/a.tmp"} {
		if sk, ok := findSkip(res, ignored); !ok || sk.Reason != ReasonIgnoreFile {
			t.Errorf("ignored %q: skip = %+v, want reason %q", ignored, sk, ReasonIgnoreFile)
		}
	}
}

// TestWalkWithoutIgnoreFile: the ignore file is opt-in per call
// (IgnoreFileName empty = none), so the same tree walks everything.
func TestWalkWithoutIgnoreFile(t *testing.T) {
	root := writeTree(t, map[string]string{
		".vanguardignore": "notes.md\n",
		"notes.md":        "x",
		"root.go":         "x",
	})
	res := walked(t, root, WalkOptions{})
	want := []string{"notes.md", "root.go"}
	if strings.Join(res.Files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v (no ignore file configured)", res.Files, want)
	}
}

// TestWalkDepthCap: MaxDepth counts path segments below the root (a root
// file is depth 1); directories at the cap are not descended, and both the
// pruned directory and nothing else appear in the skip log.
func TestWalkDepthCap(t *testing.T) {
	root := writeTree(t, map[string]string{
		"root.txt":    "x",
		"a/f.txt":     "x",
		"a/b/g.txt":   "x",
		"a/b/c/h.txt": "x",
	})
	res := walked(t, root, WalkOptions{MaxDepth: 2})
	want := []string{"a/f.txt", "root.txt"}
	if strings.Join(res.Files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", res.Files, want)
	}
	if sk, ok := findSkip(res, "a/b"); !ok || sk.Reason != ReasonTooDeep {
		t.Errorf("deep dir skip = %+v, want reason %q", sk, ReasonTooDeep)
	}
	if len(res.Skipped) != 1 {
		t.Fatalf("skip log = %v, want only a/b", res.Skipped)
	}
}

// TestWalkSkipsOversizedFiles: a file above MaxFileSize is never read —
// skipped with the cap spelled out in the reason (the brief's verbose note).
func TestWalkSkipsOversizedFiles(t *testing.T) {
	root := writeTree(t, map[string]string{
		"small.txt": "ok",
		"big.txt":   strings.Repeat("x", 128),
	})
	res := walked(t, root, WalkOptions{MaxFileSize: 8})
	want := []string{"small.txt"}
	if strings.Join(res.Files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", res.Files, want)
	}
	sk, ok := findSkip(res, "big.txt")
	if !ok || !strings.Contains(sk.Reason, "max file size") {
		t.Fatalf("big.txt skip = %+v, want a max-file-size reason", sk)
	}
	if !strings.Contains(sk.Reason, "8") {
		t.Errorf("skip reason %q should name the cap", sk.Reason)
	}
}

// TestWalkNeverFollowsSymlinks is the brief's mandatory security/perf test:
// a directory symlink looping back to an ancestor must not hang or explode
// the walk, and NO symlink — loop, plain file link, or dangling — is ever
// followed. Each leaves a skip record instead.
func TestWalkNeverFollowsSymlinks(t *testing.T) {
	root := writeTree(t, map[string]string{"src/real.go": "x"})
	mustSymlink := func(oldname, rel string) {
		t.Helper()
		if err := os.Symlink(oldname, filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
	}
	mustSymlink(root, "src/loop")                     // directory loop → root
	mustSymlink(filepath.Join(root, "src/real.go"), "linked.go") // file link
	mustSymlink(filepath.Join(root, "nowhere"), "dangling")      // broken link

	res := walked(t, root, WalkOptions{}) // must simply terminate
	want := []string{"src/real.go"}
	if strings.Join(res.Files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v (symlinks must not be walked)", res.Files, want)
	}
	for _, rel := range []string{"src/loop", "linked.go", "dangling"} {
		if sk, ok := findSkip(res, rel); !ok || sk.Reason != ReasonSymlink {
			t.Errorf("symlink %q: skip = %+v, want reason %q", rel, sk, ReasonSymlink)
		}
	}
}

// TestWalkRootFailures: a missing root and a file-as-root are the only walk
// errors — the root-level failures of the w2-01 Walker contract.
func TestWalkRootFailures(t *testing.T) {
	if _, err := NewWalker().Walk(filepath.Join(t.TempDir(), "missing"), WalkOptions{}); err == nil {
		t.Fatal("missing root must error")
	}
	single := writeTree(t, map[string]string{"f.txt": "x"})
	if _, err := NewWalker().Walk(filepath.Join(single, "f.txt"), WalkOptions{}); err == nil {
		t.Fatal("a file as root must error")
	}
}

// TestWalkPathsAreRelativeAndSafe pins the path contract every consumer
// relies on: relative to the root, "/" separators, never ".." (the security
// groundwork w4-05 will audit again).
func TestWalkPathsAreRelativeAndSafe(t *testing.T) {
	root := writeTree(t, map[string]string{"a/b/c.go": "x", "top.go": "x"})
	files, err := NewWalker().Walk(root, WalkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a/b/c.go", "top.go"}
	if strings.Join(files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", files, want)
	}
	for _, f := range files {
		if strings.HasPrefix(f, "/") || f == ".." || strings.HasPrefix(f, "../") || strings.Contains(f, "\\") {
			t.Fatalf("unsafe path %q", f)
		}
	}
}
