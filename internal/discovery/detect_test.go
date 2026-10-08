package discovery

import (
	"reflect"
	"strings"
	"testing"
)

// langs extracts the ranking from candidates.
func langs(cands []Candidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.Lang
	}
	return out
}

// TestDetectBuildFileBeatsStraySources: one pom.xml outranks any number of
// stray python files (build evidence weighs 100, one source file 1), and
// the winner's evidence explains every signal so verbose mode can show why
// a language won or lost.
func TestDetectBuildFileBeatsStraySources(t *testing.T) {
	files := []string{"pom.xml", "src/A.java", "src/B.java", "app.py", "lib.py", "x.py", "y.py"}
	cands := Detect(files)
	if got := langs(cands); !reflect.DeepEqual(got, []string{"java", "python"}) {
		t.Fatalf("ranking = %v, want [java python]", got)
	}
	java := cands[0]
	if java.Framework != "" {
		t.Errorf("Framework = %q, want empty in v0.1 (framework pinning is adapter work, w3)", java.Framework)
	}
	if !strings.Contains(java.Evidence.Reason, "java") {
		t.Errorf("evidence reason %q should name the language", java.Evidence.Reason)
	}
	joined := strings.Join(java.Evidence.Details, "\n")
	if !strings.Contains(joined, "pom.xml") || !strings.Contains(joined, ".java") {
		t.Errorf("evidence details %v should name the build file and the extension", java.Evidence.Details)
	}
}

// TestDetectBuildFileList pins the charter §5.3 mapping table verbatim.
func TestDetectBuildFileList(t *testing.T) {
	cases := map[string]string{
		"pom.xml":          "java",
		"build.gradle":     "java",
		"go.mod":           "go",
		"requirements.txt": "python",
		"pyproject.toml":   "python",
	}
	for build, wantLang := range cases {
		cands := Detect([]string{"svc/" + build})
		if got := langs(cands); !reflect.DeepEqual(got, []string{wantLang}) {
			t.Errorf("Detect(%q) = %v, want [%s]", build, got, wantLang)
		}
	}
}

// TestDetectNestedBuildFileCounts: build files count anywhere in the tree —
// a monorepo module is still a signal.
func TestDetectNestedBuildFileCounts(t *testing.T) {
	if got := langs(Detect([]string{"services/orders/pom.xml", "README.md"})); !reflect.DeepEqual(got, []string{"java"}) {
		t.Fatalf("candidates = %v, want [java]", got)
	}
}

// TestDetectExtensionOnlyLanguages: without build files, source extensions
// alone rank, more files first.
func TestDetectExtensionOnlyLanguages(t *testing.T) {
	cands := Detect([]string{"a.go", "b.go", "c.go", "x.py"})
	if got := langs(cands); !reflect.DeepEqual(got, []string{"go", "python"}) {
		t.Fatalf("ranking = %v, want [go python]", got)
	}
}

// TestDetectDeterministicTieBreak: equal scores tie-break by language name —
// repeated detection yields byte-identical rankings (charter B-6).
func TestDetectDeterministicTieBreak(t *testing.T) {
	files := []string{"a.go", "b.py"}
	first, second := langs(Detect(files)), langs(Detect(files))
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, []string{"go", "python"}) {
		t.Fatalf("tie-break not deterministic: %v vs %v", first, second)
	}
}

// TestDetectNoSignals: an empty, nil or unrelated file set yields no
// candidates — "no language found" is an empty answer, not an error.
func TestDetectNoSignals(t *testing.T) {
	for name, files := range map[string][]string{
		"nil":       nil,
		"empty":     {},
		"unrelated": {"README.md", "api/orders.stub.json", "logo.svg"},
	} {
		if got := Detect(files); len(got) != 0 {
			t.Errorf("%s: candidates = %v, want none", name, got)
		}
	}
}
