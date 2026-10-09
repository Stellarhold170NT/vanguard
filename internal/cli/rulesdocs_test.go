package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The w3-07 docs cross-check (brief deliverable 4): the rule metadata
// catalog and the docs/rules/ pages must agree 1-1, so a rule that lands
// without its reference page (or a doc left behind by a removed rule)
// fails the build. The catalog is read through defaultRegistry() — the
// same production set --list-rules prints — so the check cannot drift from
// what the binary ships.

// docFilePattern pins the page naming contract: docs/rules/<ID>-<slug>.md
// with the id in the registry's R<F>xx-NN shape.
var docPagePattern = regexp.MustCompile(`^(R[1-6]xx-[0-9]{2})-([a-z0-9]+(-[a-z0-9]+)*)\.md$`)

// requiredDocHeadings are the sections every reference page must carry —
// the format the w6-03 demo pack consumes without rewriting (w3-07
// acceptance). Titles follow the w3-07 standard: `# <ID> — <slug>`.
var requiredDocHeadings = []string{
	"## What fires",
	"## Suppression",
}

// exampleFences counts the example code blocks of one doc — java for the
// Spring rules, protobuf for the gRPC rule.
func exampleFences(doc string) int {
	return strings.Count(doc, "```java") + strings.Count(doc, "```protobuf")
}

// moduleRootForDocs resolves the module root from this package's test cwd
// (internal/cli → two levels up), mirroring golden.ModuleRoot without the
// import cycle.
func moduleRootForDocs(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("%s does not look like the module root (no go.mod)", root)
	}
	return root
}

func TestRuleDocsMatchCatalog(t *testing.T) {
	reg, err := defaultRegistry()
	if err != nil {
		t.Fatalf("defaultRegistry: %v", err)
	}
	docsDir := filepath.Join(moduleRootForDocs(t), "docs", "rules")

	// 1. Every registered rule has its page, matching id AND slug.
	rules := reg.All()
	if len(rules) == 0 {
		t.Fatal("defaultRegistry returned no rules")
	}
	seen := make(map[string]bool, len(rules))
	for _, r := range rules {
		want := "docs/rules/" + r.ID + "-" + r.Slug + ".md"
		if r.DocPath != want {
			t.Errorf("rule %s DocPath = %q, want %q (docs 1-1 contract)", r.ID, r.DocPath, want)
			continue
		}
		path := filepath.Join(docsDir, r.ID+"-"+r.Slug+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("rule %s has no reference page: %v", r.ID, err)
			continue
		}
		seen[r.ID+"-"+r.Slug+".md"] = true
		doc := string(data)
		lines := strings.Split(doc, "\n")
		if len(lines) < 30 {
			t.Errorf("doc %s: %d lines — a reference page, not a stub (w3-07: examples > prose)", path, len(lines))
		}
		if !strings.HasPrefix(lines[0], "# "+r.ID+" — "+r.Slug) {
			t.Errorf("doc %s: title %q must start with \"# %s — %s\"", path, lines[0], r.ID, r.Slug)
		}
		for _, heading := range requiredDocHeadings {
			if !strings.Contains(doc, heading) {
				t.Errorf("doc %s: missing required section %q", path, heading)
			}
		}
		if !strings.Contains(doc, "AIP-") {
			t.Errorf("doc %s: no AIP reference — every rule page cites its AIP grounding", path)
		}
		if !strings.Contains(doc, string(r.Severity)) {
			t.Errorf("doc %s: default severity %s not stated", path, r.Severity)
		}
		if !strings.Contains(doc, "vanguard:ignore") {
			t.Errorf("doc %s: missing the vanguard:ignore suppression syntax", path)
		}
		// Spring rules cite their AIP grounding; the demo family (w2-07
		// acceptance fixtures) is grounded in charter §3.0 instead.
		if r.Category != "demo" && !strings.Contains(doc, "AIP-") {
			t.Errorf("doc %s: no AIP reference — every rule page cites its AIP grounding", path)
		}
		// Both a compliant and a violating example. gRPC pages show the
		// contract in protobuf (the surface the rule judges) — Spring
		// pages in Java.
		if exampleFences(doc) < 2 {
			t.Errorf("doc %s: needs both a compliant and a violating example (```java or ```protobuf)", path)
		}
	}

	// 2. Every page maps back to a registered rule — no orphans.
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatalf("read %s: %v", docsDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if e.Name() == "README.md" {
			continue
		}
		if !docPagePattern.MatchString(e.Name()) {
			t.Errorf("docs/rules/%s: file name does not match <Rxx-NN>-<slug>.md", e.Name())
			continue
		}
		if !seen[e.Name()] {
			t.Errorf("docs/rules/%s: page exists but no registered rule maps to it (orphan doc)", e.Name())
		}
	}
}

func TestRuleDocsIndexListsEveryRule(t *testing.T) {
	reg, err := defaultRegistry()
	if err != nil {
		t.Fatalf("defaultRegistry: %v", err)
	}
	docsDir := filepath.Join(moduleRootForDocs(t), "docs", "rules")
	data, err := os.ReadFile(filepath.Join(docsDir, "README.md"))
	if err != nil {
		t.Fatalf("read docs/rules/README.md: %v", err)
	}
	index := string(data)
	for _, r := range reg.All() {
		if !strings.Contains(index, r.ID) {
			t.Errorf("docs/rules/README.md: rule %s missing from the index", r.ID)
		}
	}
}
