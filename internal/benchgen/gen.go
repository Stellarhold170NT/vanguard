// Package benchgen deterministically generates the vanguard performance bench
// repository (w4-03; test strategy §5.1).
//
// Contract: the same Plan always produces the same bytes. File names, package
// layout and file content are pure functions of (Seed, Files, Mix) — no clock,
// no global rand, no map iteration (charter B-6). The generated tree is
// measurement material and is never committed (§5.1: only the generator, the
// formula and the default seed live in the repo — tools/gen-bench-repo is the
// CLI entry point, this package the logic).
//
// The mix (§5.1) mirrors a real Spring service tree: REST controllers with
// ~5 endpoints each covering all six rule families (R1xx paths, R2xx methods,
// R3xx pagination, R4xx payload, R5xx errors, R6xx versioning), @Service
// classes, DTO/entity types and non-API support code. Every file is
// syntactically valid Java — a bench file that fails to parse would measure
// the diagnostic path, not the scan.
package benchgen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultSeed is the committed bench seed (§5.1: the generator ships with a
// default seed so every environment regenerates the identical repo).
const DefaultSeed int64 = 20261008

// DefaultFiles is the bench target: at least 10,000 .java files (§5.1).
const DefaultFiles = 10000

// Mix is the §5.1 generation formula, in whole percentages of the file count.
// The four weights must sum to exactly 100.
type Mix struct {
	Controllers int // Spring @RestController files, ~5 endpoints each
	Services    int // @Service business classes
	Types       int // DTO / entity records and POJOs the API references
	Noise       int // non-API support code (utils, constants, exceptions)
}

// DefaultMix is the committed formula: controllers dominate because they are
// the API surface the rules, the IR mapping and the renderers work on; the
// rest keeps the walk and the parse honest (real trees are not all API code).
var DefaultMix = Mix{Controllers: 40, Services: 20, Types: 25, Noise: 15}

// Plan pins one generation run. Two runs with equal Plans produce identical
// trees — that equality is the generator's own test (gen_test.go) and the
// reason the digest below exists.
type Plan struct {
	Seed  int64
	Files int
	Mix   Mix
}

// Default returns the committed plan: default seed, 10k files, default mix.
func Default() Plan { return Plan{Seed: DefaultSeed, Files: DefaultFiles, Mix: DefaultMix} }

// Validate rejects formulas that cannot become a repo.
func (p Plan) Validate() error {
	if p.Files <= 0 {
		return fmt.Errorf("benchgen: file count must be positive, got %d", p.Files)
	}
	if total := p.Mix.Controllers + p.Mix.Services + p.Mix.Types + p.Mix.Noise; total != 100 {
		return fmt.Errorf("benchgen: mix weights must sum to 100, got %d (%+v)", total, p.Mix)
	}
	if p.Mix.Controllers < 0 || p.Mix.Services < 0 || p.Mix.Types < 0 || p.Mix.Noise < 0 {
		return fmt.Errorf("benchgen: mix weights must be non-negative, got %+v", p.Mix)
	}
	return nil
}

// Result reports one generation. Digest is a SHA-256 over the sorted
// "relPath\n<sha256(content)>\n" records, so two identical trees share a
// digest and one changed byte breaks it — the evidence the determinism report
// quotes (§5.1: same seed = same bytes).
type Result struct {
	Dir       string
	JavaFiles int
	Files     int // .java + pom.xml
	Bytes     int64
	Lines     int
	Digest    string
}

// Generate writes the plan's tree under dir. A non-empty existing directory
// is refused unless force is set: silently mixing a stale tree into a fresh
// measurement would corrupt every number downstream.
func Generate(plan Plan, dir string, force bool) (Result, error) {
	if err := plan.Validate(); err != nil {
		return Result{}, err
	}
	if !force {
		if err := refuseNonEmpty(dir); err != nil {
			return Result{}, err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("benchgen: create %s: %w", dir, err)
	}

	res := Result{Dir: dir}
	sum := sha256.New()
	record := func(rel, content string) {
		line := strings.Count(content, "\n") + 1
		fileSum := sha256.Sum256([]byte(content))
		fmt.Fprintf(sum, "%s\n%s\n", rel, hex.EncodeToString(fileSum[:]))
		res.Files++
		res.Bytes += int64(len(content))
		res.Lines += line
	}

	pom := pomXML()
	if err := writeFile(dir, "pom.xml", pom); err != nil {
		return Result{}, err
	}
	record("pom.xml", pom)

	for i := 0; i < plan.Files; i++ {
		rel, content := File(plan, i)
		if err := writeFile(dir, rel, content); err != nil {
			return Result{}, err
		}
		record(rel, content)
		res.JavaFiles++
	}
	res.Files = res.JavaFiles + 1
	res.Digest = hex.EncodeToString(sum.Sum(nil))
	return res, nil
}

// File returns the relPath ("/" separators) and content of generated file
// index i — the pure half of Generate, so tests can pin single files without
// a filesystem.
func File(plan Plan, i int) (string, string) {
	kind, _ := kindOf(plan, i)
	name := typeName(i)
	// pkg is the directory path below src/main/java; pkgDecl is its dotted
	// form for the package statements (bench/app00 → bench.app00).
	pkg := fmt.Sprintf("bench/app%02d", i%packageCount)
	pkgDecl := strings.ReplaceAll(pkg, "/", ".")
	switch kind {
	case kindController:
		return filepath.ToSlash(filepath.Join("src", "main", "java", pkg, "web", name+"Controller.java")),
			controller(name, pkgDecl, i, plan.Seed)
	case kindService:
		return filepath.ToSlash(filepath.Join("src", "main", "java", pkg, "service", name+"Service.java")),
			service(name, pkgDecl, i, plan.Seed)
	case kindType:
		return filepath.ToSlash(filepath.Join("src", "main", "java", pkg, "model", name+".java")),
			model(name, pkgDecl, i, plan.Seed)
	default:
		return filepath.ToSlash(filepath.Join("src", "main", "java", pkg, "support", name+".java")),
			support(name, pkgDecl, i, plan.Seed)
	}
}

// kindOf splits the file indexes into the four §5.1 families. The split is a
// plain prefix partition of [0, Files): controllers first (they dominate the
// API surface the bench measures), then services, types and noise.
func kindOf(plan Plan, i int) (kind, int) {
	total := plan.Files
	c := total * plan.Mix.Controllers / 100
	s := c + total*plan.Mix.Services/100
	t := s + total*plan.Mix.Types/100
	switch {
	case i < c:
		return kindController, i
	case i < s:
		return kindService, i - c
	case i < t:
		return kindType, i - s
	default:
		return kindNoise, i - t
	}
}

type kind int

const (
	kindController kind = iota
	kindService
	kindType
	kindNoise
)

func refuseNonEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("benchgen: read %s: %w", dir, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("benchgen: %s is not empty (%d entries) — pass -force to overwrite", dir, len(entries))
	}
	return nil
}

func writeFile(dir, rel, content string) error {
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("benchgen: create dir for %s: %w", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return fmt.Errorf("benchgen: write %s: %w", rel, err)
	}
	return nil
}
