package discovery

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// stubRepoRoot is the checked-in fixture repo, seen from this package's
// test working directory (the brief's "scan testdata/stub-repo").
var stubRepoRoot = filepath.Join("..", "..", "testdata", "stub-repo")

// writeTreeAt writes files into an existing root (rel-path → content,
// directories implicit).
func writeTreeAt(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// copyFixture copies a directory tree into a fresh temp dir (go.mod is
// Go 1.22, so no os.CopyFS) and returns the copy's root.
func copyFixture(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, p)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// TestScanStubRepoEndToEnd — acceptance: the whole pipeline on the checked-in
// fixture (scanned through a hermetic copy): detect the stub adapter, parse,
// build a valid IR, end clean.
//
// The copy exists because the sandbox VFS sync does not carry node_modules
// directories into the test container (CAN lesson in the w2-03 report), so
// the test adds the excluded-dir case itself to stay deterministic.
func TestScanStubRepoEndToEnd(t *testing.T) {
	root := copyFixture(t, stubRepoRoot)
	writeTreeAt(t, root, map[string]string{
		"node_modules/dep.stub.json": `{"service":"ShouldNeverAppear","methods":[]}`,
	})
	res, err := Scan(root, ScanOptions{})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Selected != StubLanguage {
		t.Fatalf("selected = %q, want %q", res.Selected, StubLanguage)
	}
	if res.Surface == nil {
		t.Fatal("nil surface")
	}
	s := res.Surface
	if s.Source != (ir.Source{Lang: StubLanguage, Framework: "stub-http"}) {
		t.Fatalf("source = %+v", s.Source)
	}
	if len(s.Services) != 1 || s.Services[0].Name != "Orders" || len(s.Services[0].Methods) != 3 {
		t.Fatalf("services = %+v", s.Services)
	}
	if len(s.Types) != 2 {
		t.Fatalf("types = %d, want 2", len(s.Types))
	}
	if v := s.Validate(); len(v) != 0 {
		t.Fatalf("fixture surface fails ir.Validate: %v", v)
	}
	// exactly one diagnostic: the deliberately broken fixture file
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Location.File != "src/broken.stub.json" {
		t.Fatalf("diagnostics = %+v, want exactly the broken fixture", res.Diagnostics)
	}
	// walker evidence: fixture junk is skipped by the built-in rules
	skips := map[string]string{}
	for _, sk := range res.Skipped {
		skips[sk.Path] = sk.Reason
	}
	if skips["node_modules"] != ReasonExcludedDir {
		t.Fatalf("node_modules skip = %q (log %+v), want %q", skips["node_modules"], res.Skipped, ReasonExcludedDir)
	}
	if skips["scratch"] != ReasonIgnoreFile {
		t.Fatalf("scratch skip = %q (log %+v), want %q", skips["scratch"], res.Skipped, ReasonIgnoreFile)
	}
	// no language candidates in a stub-only fixture
	if len(res.Candidates) != 0 {
		t.Fatalf("candidates = %v, want none", res.Candidates)
	}
	if res.NoAPI != "" {
		t.Fatalf("NoAPI set on a found surface: %q", res.NoAPI)
	}
}

// TestScanCheckedInFixtureDirect scans the checked-in testdata/stub-repo in
// place — the literal acceptance command — asserting the facts that do not
// depend on how a sandbox syncs the tree (node_modules may be absent there;
// the hermetic copy above covers that case).
func TestScanCheckedInFixtureDirect(t *testing.T) {
	res, err := Scan(stubRepoRoot, ScanOptions{})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Selected != StubLanguage || res.Surface == nil {
		t.Fatalf("selected = %q, surface = %+v", res.Selected, res.Surface)
	}
	s := res.Surface
	if len(s.Services) != 1 || s.Services[0].Name != "Orders" || len(s.Services[0].Methods) != 3 {
		t.Fatalf("services = %+v", s.Services)
	}
	if len(s.Types) != 2 {
		t.Fatalf("types = %d, want 2", len(s.Types))
	}
	if v := s.Validate(); len(v) != 0 {
		t.Fatalf("fixture surface fails ir.Validate: %v", v)
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Location.File != "src/broken.stub.json" {
		t.Fatalf("diagnostics = %+v, want exactly the broken fixture", res.Diagnostics)
	}
	skips := map[string]string{}
	for _, sk := range res.Skipped {
		skips[sk.Path] = sk.Reason
	}
	if skips["scratch"] != ReasonIgnoreFile {
		t.Fatalf("scratch skip = %q (log %+v), want %q", skips["scratch"], res.Skipped, ReasonIgnoreFile)
	}
	if res.NoAPI != "" {
		t.Fatalf("NoAPI set on a found surface: %q", res.NoAPI)
	}
}

// TestScanVerboseCarriesEvidence — acceptance: detection evidence lands in
// the verbose stream, so a wrong adapter pick is debuggable from the CLI
// output alone (charter §5.3).
func TestScanVerboseCarriesEvidence(t *testing.T) {
	root := writeTree(t, map[string]string{
		"pom.xml":              "<project/>",
		"src/A.java":           "x",
		"api/orders.stub.json": `{"service":"S","methods":[]}`,
	})
	var verbose bytes.Buffer
	res, err := Scan(root, ScanOptions{Verbose: &verbose})
	if err != nil {
		t.Fatal(err)
	}
	out := verbose.String()
	for _, want := range []string{
		"walk: 3 file(s)",
		"language candidate java",
		"build file: pom.xml",
		"adapter stub: detect=true",
		"selected adapter: stub",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("verbose output missing %q\ngot:\n%s", want, out)
		}
	}
	if res.Selected != StubLanguage {
		t.Fatalf("selected = %q, want stub", res.Selected)
	}
}

// TestScanQuietByDefault: without a Verbose writer the scan works and no
// output is produced anywhere (verbose is strictly opt-in).
func TestScanQuietByDefault(t *testing.T) {
	res, err := Scan(stubRepoRoot, ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Selected != StubLanguage {
		t.Fatalf("selected = %q", res.Selected)
	}
}

// TestScanNoAdapterDetectedIsCleanSuccess — deliverable 5: a repo where no
// adapter detects anything gets a clear explanation, not an error (charter
// §6.3 pins exit code 0 for this outcome).
func TestScanNoAdapterDetectedIsCleanSuccess(t *testing.T) {
	root := writeTree(t, map[string]string{"README.md": "just docs", "util.py": "print(1)"})
	res, err := Scan(root, ScanOptions{})
	if err != nil {
		t.Fatalf("no-adapter scan must not error: %v", err)
	}
	if res.Surface != nil {
		t.Fatalf("surface = %+v, want nil", res.Surface)
	}
	if res.NoAPI == "" {
		t.Fatal("NoAPI must explain the empty outcome")
	}
	// the language layer still did its job: python is a candidate, there is
	// just no adapter for it yet
	if len(res.Candidates) != 1 || res.Candidates[0].Lang != "python" {
		t.Fatalf("candidates = %+v, want [python]", res.Candidates)
	}
}

// TestScanEmptySurfaceIsCleanSuccess: an adapter detects and parses but the
// tree describes no API — same clean-success contract, different reason.
func TestScanEmptySurfaceIsCleanSuccess(t *testing.T) {
	root := writeTree(t, map[string]string{"docs/notes.md": "no api here"})
	reg := NewRegistry()
	if err := reg.Register(&fakeAdapter{lang: "empty", detects: true}); err != nil {
		t.Fatal(err)
	}
	res, err := Scan(root, ScanOptions{Registry: reg})
	if err != nil {
		t.Fatal(err)
	}
	if res.Selected != "empty" || res.Surface == nil {
		t.Fatalf("selected = %q, surface = %+v", res.Selected, res.Surface)
	}
	if res.NoAPI == "" {
		t.Fatal("empty surface must be explained in NoAPI")
	}
}

// TestScanHonorsWalkOverrides: WalkOptions flow through Scan — disabling the
// ignore file pulls previously-ignored files (and their content) back in.
func TestScanHonorsWalkOverrides(t *testing.T) {
	res, err := Scan(stubRepoRoot, ScanOptions{Walk: &WalkOptions{IgnoreFileName: ""}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Surface == nil || len(res.Surface.Services) != 2 {
		t.Fatalf("services = %+v, want 2 with the ignore file disabled", res.Surface)
	}
}

// TestScanRootFailuresAreErrors: root-level failures stay errors — the tool
// error path (§6.3 code 2 territory), unlike "nothing found".
func TestScanRootFailuresAreErrors(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "missing"), ScanOptions{}); err == nil {
		t.Fatal("missing root must error")
	}
}
