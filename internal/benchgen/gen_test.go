package benchgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanValidateRejectsBadFormulas(t *testing.T) {
	good := Default()
	if err := good.Validate(); err != nil {
		t.Fatalf("Default() plan rejected: %v", err)
	}
	badMix := good
	badMix.Mix.Noise = 16 // weights now sum to 101
	if err := badMix.Validate(); err == nil {
		t.Fatal("mix summing to 101 accepted")
	}
	badCount := good
	badCount.Files = 0
	if err := badCount.Validate(); err == nil {
		t.Fatal("zero file count accepted")
	}
}

// TestSamePlanSameBytes is the generator's determinism contract (§5.1: same
// seed = same bytes): two runs into two fresh directories must agree on every
// file, byte for byte, and therefore on the digest.
func TestSamePlanSameBytes(t *testing.T) {
	plan := Plan{Seed: 42, Files: 120, Mix: DefaultMix}
	first, err := Generate(plan, t.TempDir(), true)
	if err != nil {
		t.Fatalf("generate first: %v", err)
	}
	second, err := Generate(plan, t.TempDir(), true)
	if err != nil {
		t.Fatalf("generate second: %v", err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("digest mismatch for equal plans:\n first %s\nsecond %s", first.Digest, second.Digest)
	}
	if first.JavaFiles != second.JavaFiles || first.Bytes != second.Bytes || first.Lines != second.Lines {
		t.Fatalf("counts differ: %+v vs %+v", first, second)
	}
	if first.JavaFiles != plan.Files {
		t.Fatalf("JavaFiles = %d, want %d", first.JavaFiles, plan.Files)
	}
	if first.Files != plan.Files+1 { // + pom.xml
		t.Fatalf("Files = %d, want %d (.java + pom.xml)", first.Files, plan.Files+1)
	}
}

func TestDifferentSeedDifferentBytes(t *testing.T) {
	base := Plan{Seed: 7, Files: 60, Mix: DefaultMix}
	one, err := Generate(base, t.TempDir(), true)
	if err != nil {
		t.Fatalf("generate seed 7: %v", err)
	}
	other := base
	other.Seed = 8
	two, err := Generate(other, t.TempDir(), true)
	if err != nil {
		t.Fatalf("generate seed 8: %v", err)
	}
	if one.Digest == two.Digest {
		t.Fatal("different seeds produced identical trees")
	}
}

// TestMixPartition pins the §5.1 formula: the family split lands on the
// committed weights, every file is valid-looking Java (package line, one
// public type) and non-API support code stays annotation-free.
func TestMixPartition(t *testing.T) {
	plan := Plan{Seed: DefaultSeed, Files: 100, Mix: DefaultMix}
	counts := map[kind]int{}
	for i := 0; i < plan.Files; i++ {
		rel, content := File(plan, i)
		if !strings.HasSuffix(rel, ".java") {
			t.Fatalf("file %d: non-java path %q", i, rel)
		}
		if !strings.Contains(content, "package ") || !strings.Contains(content, "public ") {
			t.Fatalf("file %d (%s): not Java-shaped", i, rel)
		}
		if !strings.HasPrefix(rel, "src/main/java/bench/") {
			t.Fatalf("file %d: unexpected layout %q", i, rel)
		}
		switch {
		case strings.Contains(rel, "/web/"):
			counts[kindController]++
			if !strings.Contains(content, "@RestController") {
				t.Fatalf("controller %s missing @RestController", rel)
			}
		case strings.Contains(rel, "/service/"):
			counts[kindService]++
			if !strings.Contains(content, "@Service") {
				t.Fatalf("service %s missing @Service", rel)
			}
		case strings.Contains(rel, "/model/"):
			counts[kindType]++
		default:
			counts[kindNoise]++
			if strings.Contains(content, "org.springframework") {
				t.Fatalf("support file %s must be non-API (no Spring imports)", rel)
			}
		}
	}
	want := map[kind]int{kindController: 40, kindService: 20, kindType: 25, kindNoise: 15}
	for k, n := range want {
		if counts[k] != n {
			t.Fatalf("mix family %d: got %d files, want %d", k, counts[k], n)
		}
	}
}

func TestRefusesNonEmptyDirectoryWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stale.java"), []byte("class Stale {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(Default(), dir, false); err == nil {
		t.Fatal("non-empty directory accepted without -force")
	}
	if _, err := Generate(Default(), dir, true); err != nil {
		t.Fatalf("force generation over a non-empty directory failed: %v", err)
	}
}

// TestPomPinsSpringBoot keeps the framework evidence stable: the generated
// pom must name the Boot parent so scans report spring-boot + version, the
// same shape the golden fixtures show (§5.2 environment note).
func TestPomPinsSpringBoot(t *testing.T) {
	pom := pomXML()
	for _, want := range []string{"spring-boot-starter-parent", "3.2.4", "spring-boot-starter-web"} {
		if !strings.Contains(pom, want) {
			t.Fatalf("pom.xml missing %q", want)
		}
	}
}
