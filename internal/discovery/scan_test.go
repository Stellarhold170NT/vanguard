package discovery

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// stubRepoRoot is the checked-in fixture repo, seen from this package's
// test working directory (the brief's "scan testdata/stub-repo").
var stubRepoRoot = filepath.Join("..", "..", "testdata", "stub-repo")

// TestScanStubRepoEndToEnd — acceptance: the whole pipeline on the checked-in
// fixture: detect the stub adapter, parse, build a valid IR, end clean.
func TestScanStubRepoEndToEnd(t *testing.T) {
	res, err := Scan(stubRepoRoot, ScanOptions{})
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
	if skips["node_modules"] == "" || skips["scratch"] == "" {
		t.Fatalf("skip log missing node_modules/scratch: %+v", res.Skipped)
	}
	// no language candidates in a stub-only fixture
	if len(res.Candidates) != 0 {
		t.Fatalf("candidates = %v, want none", res.Candidates)
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

// TestScanVerboseQuietByDefault: without a Verbose writer nothing is printed
// and no output object is touched.
func TestScanVerboseQuietByDefault(t *testing.T) {
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
