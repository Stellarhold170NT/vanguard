package perf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/discovery"
)

// The §5.5 edge-case matrix (test strategy): "đầu vào hợp pháp ở mức cực
// đoan" — each row pins the DEFINED behaviour (exit code, findings, notice)
// with the real binary. None of these cases may crash, hang (Run fails the
// test on its 90s timeout) or OOM. Deliberately hostile inputs (path
// traversal, symlinks out of the root, …) belong to w4-05, not here.

// hugeJava builds a syntactically valid Java file of at least minBytes out of
// trivial getter-style methods — big enough to exceed the walker's 1 MiB cap,
// still parseable content so the case isolates the size decision.
func hugeJava(t *testing.T, minBytes int) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("package bench.huge;\n\npublic class HugeFile {\n")
	for i := 0; b.Len() < minBytes; i++ {
		fmt.Fprintf(&b, "    public int value%d() {\n        return %d;\n    }\n\n", i, i)
	}
	b.WriteString("}\n")
	return b.String()
}

// TestEdgeHugeJavaFileSkippedWithDiagnostic pins §5.5 row 1: a ≥5 MB .java
// file is NOT read into memory — the walker skips it above the published cap
// (charter B-7) with a diagnostic on --verbose, the run exits normally and
// nothing crashes. The 90s run timeout is the no-hang proof.
func TestEdgeHugeJavaFileSkippedWithDiagnostic(t *testing.T) {
	root := ModuleRoot(t)
	dir := t.TempDir()
	content := hugeJava(t, 5<<20) // 5 MiB of valid Java
	if err := os.WriteFile(filepath.Join(dir, "HugeFile.java"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(filepath.Join(dir, "HugeFile.java")); err != nil || fi.Size() < int64(5<<20) {
		t.Fatalf("fixture too small: %d bytes (%v)", fi.Size(), err)
	}

	res := Run(t, root, "scan", dir, "--format", "json", "--no-color")
	if res.Exit != 0 {
		t.Fatalf("huge-file scan exit %d, want 0 (skip, no findings)\nstderr: %s", res.Exit, res.Stderr)
	}
	if strings.Contains(string(res.Stdout), "HugeFile.java") {
		t.Fatalf("skipped file leaked into findings:\n%s", res.Stdout)
	}

	verbose := Run(t, root, "scan", dir, "--format", "json", "--no-color", "--verbose")
	if !strings.Contains(string(verbose.Stderr), "exceeds max file size") {
		t.Fatalf("--verbose diagnostics missing the skip reason (stderr):\n%s", verbose.Stderr)
	}
}

// TestEdgeDeeplyNestedClassesScan pins §5.5 row 2: class nesting 10+ levels
// deep parses and scans normally — the controller still maps into the IR and
// produces findings (exit 1), i.e. the whole pipeline ran on the file.
func TestEdgeDeeplyNestedClassesScan(t *testing.T) {
	root := ModuleRoot(t)
	dir := t.TempDir()
	const depth = 12
	var b strings.Builder
	b.WriteString("package bench.deep;\n\n")
	b.WriteString("import org.springframework.web.bind.annotation.*;\n\n")
	b.WriteString("@RestController\n@RequestMapping(\"/legacy/deep\")\n")
	b.WriteString("public class DeepController {\n\n")
	// One mapped endpoint: rule shapes (R6xx-94 legacy path) fire per method,
	// so the finding below proves the mapper reached through the nesting.
	fmt.Fprintf(&b, "    @GetMapping(\"/{id}\")\n    public String get(long id) {\n        return \"ok\";\n    }\n\n")
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&b, "%*spublic static class Level%02d {\n\n", (i+1)*4, "", i)
	}
	fmt.Fprintf(&b, "%*spublic static final int LEAF = %d;\n\n", (depth+1)*4, "", depth)
	for i := depth - 1; i >= 0; i-- {
		fmt.Fprintf(&b, "%*s}\n\n", (i+1)*4, "")
	}
	b.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "DeepController.java"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Run(t, root, "scan", dir, "--format", "json", "--no-color")
	// §6.3 gates exit 1 on ERROR-severity findings; this file fires WARN/INFO
	// rule shapes, so the defined outcome is exit 0-or-1 (never 2, never a
	// crash) WITH findings — that is what "parses and scans normally" means.
	if res.Exit != 0 && res.Exit != 1 {
		t.Fatalf("deep-nesting scan exit %d, want 0 or 1 (§6.3 severity gating)\nstdout: %s\nstderr: %s",
			res.Exit, res.Stdout, res.Stderr)
	}
	if got := strings.Count(string(res.Stdout), "DeepController.java"); got == 0 {
		t.Fatalf("deeply nested file produced no findings:\n%s", res.Stdout)
	}
}

// TestEdgeZeroApiFilesExitsZero pins §5.5 row 3: a repo whose Java files are
// all non-API exits 0 with the §6.3 "nothing to check" notice — never exit 2,
// never a crash.
func TestEdgeZeroApiFilesExitsZero(t *testing.T) {
	root := ModuleRoot(t)
	dir := t.TempDir()
	src := "package bench.plain;\n\npublic class PlainHelper {\n    public int add(int a, int b) {\n        return a + b;\n    }\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "PlainHelper.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# bench\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Run(t, root, "scan", dir, "--format", "json", "--no-color")
	if res.Exit != 0 {
		t.Fatalf("zero-API scan exit %d, want 0\nstdout: %s\nstderr: %s", res.Exit, res.Stdout, res.Stderr)
	}
	if !strings.Contains(string(res.Stdout), "nothing to check") && !strings.Contains(string(res.Stderr), "nothing to check") {
		t.Fatalf("§6.3 notice missing:\nstdout: %s\nstderr: %s", res.Stdout, res.Stderr)
	}
}

// writeController writes the one-file Spring controller used by the walk
// comparisons below. It triggers the legacy-path finding deterministically
// (the same shape the w3-07 golden showcase pins), so its finding count is a
// precise "how often was this file visited" probe.
func writeController(t *testing.T, dir string) {
	t.Helper()
	src := "package bench.loop;\n\n" +
		"import org.springframework.web.bind.annotation.GetMapping;\n" +
		"import org.springframework.web.bind.annotation.RequestMapping;\n" +
		"import org.springframework.web.bind.annotation.RestController;\n\n" +
		"@RestController\n@RequestMapping(\"/legacy/loops\")\n" +
		"public class LoopController {\n\n" +
		"    @GetMapping(\"/{id}\")\n" +
		"    public String get(long id) {\n        return \"ok\";\n    }\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "LoopController.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestEdgeSymlinkLoopWalkedOnce pins §5.5 row 4: symlink loops inside the
// root must not loop the walker and must not duplicate files — the scan of a
// directory with a self-loop and a two-cycle symlink reports exactly the same
// findings as the identical directory without them, and returns (Run's 90s
// timeout is the backstop against a hang).
func TestEdgeSymlinkLoopWalkedOnce(t *testing.T) {
	root := ModuleRoot(t)
	plain, looped := t.TempDir(), t.TempDir()
	writeController(t, plain)
	writeController(t, looped)

	// A self-loop and a two-node cycle, both strictly inside the root.
	if err := os.Symlink(".", filepath.Join(looped, "selfloop")); err != nil {
		t.Skipf("symlinks unavailable in this environment: %v", err)
	}
	if err := os.Symlink("b", filepath.Join(looped, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a", filepath.Join(looped, "b")); err != nil {
		t.Fatal(err)
	}

	for name, dir := range map[string]string{"plain": plain, "looped": looped} {
		res := Run(t, root, "scan", dir, "--format", "json", "--no-color")
		if res.Exit != 0 && res.Exit != 1 {
			t.Fatalf("%s scan exit %d\nstderr: %s", name, res.Exit, res.Stderr)
		}
	}

	plainOut := Run(t, root, "scan", plain, "--format", "json", "--no-color").Stdout
	loopedOut := Run(t, root, "scan", looped, "--format", "json", "--no-color").Stdout
	plainHits := strings.Count(string(plainOut), "LoopController.java")
	loopedHits := strings.Count(string(loopedOut), "LoopController.java")
	if plainHits == 0 {
		t.Fatalf("plain scan did not report the controller:\n%s", plainOut)
	}
	if plainHits != loopedHits {
		t.Fatalf("symlink loop changed the walk: %d finding references without the loop, %d with",
			plainHits, loopedHits)
	}
}

// TestEdgeUnicodeFilename pins §5.5 row 5: a unicode filename scans normally
// and reaches the output as intact UTF-8 — the finding path names the file
// exactly as it is on disk.
func TestEdgeUnicodeFilename(t *testing.T) {
	root := ModuleRoot(t)
	dir := t.TempDir()
	src := "package bench.uni;\n\n" +
		"import org.springframework.web.bind.annotation.GetMapping;\n" +
		"import org.springframework.web.bind.annotation.RequestMapping;\n" +
		"import org.springframework.web.bind.annotation.RestController;\n\n" +
		"@RestController\n@RequestMapping(\"/legacy/tai-lieu\")\n" +
		"public class TàiLiệuController {\n\n" +
		"    @GetMapping(\"/{id}\")\n" +
		"    public String get(long id) {\n        return \"ok\";\n    }\n}\n"
	name := "TàiLiệuController.java"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Run(t, root, "scan", dir, "--format", "json", "--no-color")
	// Exit policy as in the deep-nesting case above: §6.3 gates exit 1 on
	// ERROR severity, and this fixture fires WARN/INFO shapes — the defined
	// outcome is 0-or-1 with findings that name the file in intact UTF-8.
	if res.Exit != 0 && res.Exit != 1 {
		t.Fatalf("unicode-filename scan exit %d, want 0 or 1 (§6.3 severity gating)\nstdout: %s\nstderr: %s",
			res.Exit, res.Stdout, res.Stderr)
	}
	if !strings.Contains(string(res.Stdout), name) {
		t.Fatalf("unicode filename not intact in output:\n%s", res.Stdout)
	}
}

// TestEdgeEmptyAndCommentOnlyJavaIgnored pins §5.5 row 6: an empty .java file
// and a comment-only .java file are ignored silently — no finding, no
// diagnostic on the default channel, clean exit 0 (with the §6.3 notice,
// since nothing else in the tree carries an API surface).
func TestEdgeEmptyAndCommentOnlyJavaIgnored(t *testing.T) {
	root := ModuleRoot(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Empty.java"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	comments := "// benchgen: comment-only file\n/* block comment\n   spanning lines */\n"
	if err := os.WriteFile(filepath.Join(dir, "OnlyComments.java"), []byte(comments), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Run(t, root, "scan", dir, "--format", "json", "--no-color")
	if res.Exit != 0 {
		t.Fatalf("empty/comment-only scan exit %d, want 0\nstdout: %s\nstderr: %s", res.Exit, res.Stdout, res.Stderr)
	}
	for _, out := range []string{string(res.Stdout), string(res.Stderr)} {
		if strings.Contains(out, "Empty.java") || strings.Contains(out, "OnlyComments.java") {
			t.Fatalf("ignored file leaked into output:\n%s", out)
		}
	}
	if !strings.Contains(string(res.Stdout), "nothing to check") && !strings.Contains(string(res.Stderr), "nothing to check") {
		t.Fatalf("§6.3 notice missing:\nstdout: %s\nstderr: %s", res.Stdout, res.Stderr)
	}
}

// The walker's published caps are part of the §5.5 contract: the 5 MB case
// above leans on the size cap being ON by default.
func TestPublishedCapsAreActive(t *testing.T) {
	if discovery.DefaultMaxFileSize != 1<<20 {
		t.Fatalf("DefaultMaxFileSize = %d, want %d (charter B-7)", discovery.DefaultMaxFileSize, 1<<20)
	}
	if discovery.DefaultMaxDepth != 32 {
		t.Fatalf("DefaultMaxDepth = %d, want 32", discovery.DefaultMaxDepth)
	}
}
