package mutation

import (
	"encoding/json"
	"flag"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Stellarhold170NT/vanguard/internal/golden"
)

// Gate constants (test-strategy §4.2): total declared catch-rate >= 90%,
// every rule with >= 1 valid case > 0% (declared), and no harness errors.
// The strict number is reported next to it, never gated: the four chartered
// but unregistered rules (w4-01 §7.1) cannot fire by construction, and their
// pre-declared misses are counted only in the declared number.
const (
	gateTotalPct   = 90.0
	defaultMaxCell = 3
)

var (
	flagWriteResults = flag.Bool("write-results", false, "write the results JSON (make mutation sets this)")
	flagOut          = flag.String("out", "", "results JSON path when -write-results is set")
	flagMaxPerCell   = flag.Int("max-per-cell", defaultMaxCell, "§4.4 case cap per (rule, mutator) cell")
)

const itemController = `package com.example.books;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}
`

// TestMutatorTransforms pins each mutator's transform: exact output for the
// applicable shapes, refusal for the shapes the mutation cannot make
// well-formed. These run without the binary — fast, deterministic.
func TestMutatorTransforms(t *testing.T) {
	tests := []struct {
		name    string
		mutator *Mutator
		src     string
		wantIn  string // substring the mutated source must contain; "" → must NOT apply
	}{
		{"singularize-path", mutSingularizePath,
			strings.Replace(itemController, "/books/{id}", "/books", 1),
			`@GetMapping("/book")`},
		{"singularize-path skips uncountable", mutSingularizePath,
			strings.Replace(itemController, "/books/{id}", "/equipment/{id}", 1),
			""},
		{"inject-verb-path", mutInjectVerbPath, itemController,
			`@GetMapping("/books/delete-item/{id}")`},
		{"flatten-resource-path", mutFlattenResourcePath, itemController,
			`@GetMapping("/find-by-id/{id}")`},
		{"mis-case-path", mutMisCasePath, itemController,
			`@GetMapping("/Books/{id}")`},
		{"body-into-get", mutBodyIntoGet, itemController,
			`public BookDto get(@PathVariable Long id, @RequestBody String body) {`},
		{"body-into-delete", mutBodyIntoDelete,
			strings.Replace(itemController, "@GetMapping", "@DeleteMapping", 1),
			`public BookDto get(@PathVariable Long id, @RequestBody String body) {`},
		{"verb-swap item-template not applicable", mutVerbSwap, itemController, ""},
		{"verb-swap collection not applicable", mutVerbSwap,
			strings.Replace(itemController,
				"public BookDto get(@PathVariable Long id) {", "public Page<BookDto> list() {", 1),
			""},
		{"verb-swap literal read", mutVerbSwap,
			strings.Replace(itemController, `"/books/{id}"`, `"/books/default"`, 1),
			`@PostMapping("/books/default")`},
		{"remove-status-mapping skips bare post", mutRemoveStatusMapping,
			strings.Replace(itemController, "@GetMapping", "@PostMapping", 1), ""},
		{"swap-put-patch", mutSwapPutPatch,
			strings.Replace(itemController, "@GetMapping", "@PutMapping", 1),
			`@PatchMapping("/books/{id}")`},
		{"drop-pagination-param", mutDropPaginationParam,
			strings.Replace(itemController,
				"get(@PathVariable Long id) {", "list(Pageable pageable) {", 1),
			`list() {`},
		{"entity-into-response", mutEntityIntoResponse,
			itemController + "\n@Entity\nclass Book {\n    Long id;\n}\n",
			"public Book get("},
		{"entity-into-response skips shadowed", mutEntityIntoResponse,
			strings.Replace(itemController, "BookDto get", "Book get", 1) +
				"\nrecord Book(Long id) {\n}\n@Entity\nclass Book {\n    Long id;\n}\n",
			""},
		{"snake-case-field", mutSnakeCaseField,
			"record BookDto(Long id, Instant createdAt) {\n}\n",
			"created_at"},
		{"error-map-handler", mutErrorMapHandler,
			"@ControllerAdvice\nclass Advice {\n" +
				"    @ExceptionHandler(AException.class)\n    Envelope one(AException ex) {\n        return new Envelope(1, \"a\");\n    }\n" +
				"    @ExceptionHandler(BException.class)\n    Envelope two(BException ex) {\n        return new Envelope(2, \"b\");\n    }\n" +
				"}\nrecord Envelope(int status, String detail) {\n}\n",
			"Map<String, Object> one("},
		{"error-map-handler skips no-consensus", mutErrorMapHandler,
			"@ControllerAdvice\nclass Advice {\n" +
				"    @ExceptionHandler(AException.class)\n    Envelope one(AException ex) {\n        return new Envelope(1, \"a\");\n    }\n" +
				"    @ExceptionHandler(BException.class)\n    Other two(BException ex) {\n        return new Other(2, \"b\");\n    }\n" +
				"}\nrecord Envelope(int status, String detail) {\n}\nrecord Other(int code) {\n}\n",
			""},
		{"business-exception-500", mutBusinessException500,
			"@ControllerAdvice\nclass Advice {\n" +
				"    @ExceptionHandler(BookNotFoundException.class)\n    @ResponseStatus(HttpStatus.NOT_FOUND)\n    Envelope one(BookNotFoundException ex) {\n        return new Envelope(1, \"a\");\n    }\n" +
				"}\n",
			"HttpStatus.INTERNAL_SERVER_ERROR"},
		{"add-second-advice", mutAddSecondAdvice,
			"@ControllerAdvice\nclass Advice {\n}\n",
			"@ControllerAdvice\nclass SecondAdvice {"},
		{"unversioned-path", mutUnversionedPath,
			strings.Replace(itemController, "@RestController", "@RestController\n@RequestMapping(\"/api/v1/orders\")", 1),
			`@RequestMapping("/api/orders")`},
		{"mix-id-field", mutMixIdField,
			"record LoanDto(Long id, Long bookId, Long memberId) {\n}\n",
			"memberID"},
		{"action-get-swap", mutActionGetSwap,
			strings.ReplaceAll(strings.Replace(itemController, `"/books/{id}"`, `"/books/search"`, 1),
				"@GetMapping", "@PostMapping"),
			`@GetMapping("/books/search")`},
		{"unwrap-envelope", mutUnwrapEnvelope,
			strings.Replace(itemController,
				"public BookDto get(@PathVariable Long id) {", "public Page<BookDto> list(Pageable pageable) {", 1),
			"public List<BookDto> list("},
		{"remove-size-cap", mutRemoveSizeCap,
			"    public Page<BookDto> list(@RequestParam(\"size\") @Max(100) int size) {\n",
			"public Page<BookDto> list(@RequestParam(\"size\") int size) {"},
		{"time-field-to-string", mutTimeFieldToString,
			"record PostDto(Long id, OffsetDateTime createdAt) {\n}\n",
			"String createdAt"},
		{"dto-suffix-mix", mutDtoSuffixMix,
			"record OrderRequest(String reference) {\n}\nrecord OrderResponse(Long id, String reference) {\n}\n",
			"record Order("},
		{"rename-rpc-nonstandard", mutRenameRpcNonstandard,
			"service BookService {\n  rpc GetBook(GetBookRequest) returns (Book);\n}\n",
			"rpc FetchBook("},
		{"bump-one-path-version", mutBumpOnePathVersion,
			strings.Replace(itemController, `"/books/{id}"`, `"/api/v1/books"`, 1),
			`"/api/v2/books"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, applied := tt.mutator.Apply([]byte(tt.src))
			if tt.wantIn == "" {
				if applied {
					t.Fatalf("mutator %s applied to non-applicable source:\n%s", tt.mutator.Name, got)
				}
				return
			}
			if !applied {
				t.Fatalf("mutator %s did not apply; want %q in output", tt.mutator.Name, tt.wantIn)
			}
			if !strings.Contains(string(got), tt.wantIn) {
				t.Fatalf("mutator %s: output missing %q\n--- mutated ---\n%s", tt.mutator.Name, tt.wantIn, got)
			}
		})
	}
}

// TestRemoveStatusMappingTransform pins remove-status-mapping's exact
// behavior on a realistic POST-201 fixture: the status line is dropped,
// everything else stays byte-identical.
func TestRemoveStatusMappingTransform(t *testing.T) {
	src := `package com.example.books;

@RestController
public class BookCreateController {

    @PostMapping("/books")
    @ResponseStatus(HttpStatus.CREATED)
    public BookDto create(@RequestBody BookDto draft) {
        return draft;
    }
}
`
	got, ok := mutRemoveStatusMapping.Apply([]byte(src))
	if !ok {
		t.Fatal("remove-status-mapping did not apply to a POST-201 fixture")
	}
	out := string(got)
	if strings.Contains(out, "@ResponseStatus") {
		t.Fatalf("status line not removed:\n%s", out)
	}
	if !strings.Contains(out, "@PostMapping(\"/books\")") || !strings.Contains(out, "public BookDto create") {
		t.Fatalf("non-status lines were touched:\n%s", out)
	}
}

// TestMutatorCatalog pins the §4.1 catalog: 17 MUST + 8 OPTIONAL mutators,
// each with at least one pairing, a pool and targets.
func TestMutatorCatalog(t *testing.T) {
	must, optional := 0, 0
	seen := map[string]bool{}
	for _, m := range mutators {
		if seen[m.Name] {
			t.Fatalf("duplicate mutator %q", m.Name)
		}
		seen[m.Name] = true
		if len(m.Pairings) == 0 {
			t.Fatalf("mutator %s has no pairings", m.Name)
		}
		for _, p := range m.Pairings {
			if len(p.Targets) == 0 {
				t.Fatalf("mutator %s pairing has no targets", m.Name)
			}
			if p.PoolRule == "" {
				t.Fatalf("mutator %s pairing has no pool rule", m.Name)
			}
		}
		switch m.Kind {
		case "must":
			must++
		case "optional":
			optional++
		default:
			t.Fatalf("mutator %s has unknown kind %q", m.Name, m.Kind)
		}
	}
	if must != 17 {
		t.Errorf("MUST mutators = %d, want 17 (test-strategy §4.1)", must)
	}
	if optional != 8 {
		t.Errorf("OPTIONAL mutators = %d, want 8 (test-strategy §4.1)", optional)
	}
}

// TestMutatorsApplyToCorpus feeds every mutator its real corpus pool and
// checks the pairing still produces at least one applicable mutation in the
// first maxPerCell applicable files — the guard that keeps mutator ↔ corpus
// drift from silently emptying cells (a mutator that applies nowhere
// measures nothing).
func TestMutatorsApplyToCorpus(t *testing.T) {
	root, err := golden.ModuleRoot()
	if err != nil {
		t.Fatalf("module root: %v", err)
	}
	man, err := loadManifest(filepath.Join(root, "testdata", "adversarial", "manifest.json"))
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	pool := poolsByRule(man)
	for _, m := range mutators {
		for _, p := range m.Pairings {
			indices, ok := pool[p.PoolRule]
			if p.PoolRule == "*" {
				indices, ok = allOK(man)
			}
			if !ok {
				t.Errorf("mutator %s pool %s: no ok-cases in manifest", m.Name, p.PoolRule)
				continue
			}
			applied := 0
			for _, ci := range indices {
				if applied >= defaultMaxCell {
					break
				}
				src, err := os.ReadFile(filepath.Join(root, "testdata", "adversarial", man.Cases[ci].File))
				if err != nil {
					t.Fatalf("read %s: %v", man.Cases[ci].File, err)
				}
				if _, changed := m.Apply(src); changed {
					applied++
				}
			}
			if applied == 0 {
				t.Errorf("mutator %s (pool %s): no applicable ok-case in %d pool files — mutator/corpus drift",
					m.Name, p.PoolRule, len(indices))
			}
		}
	}
}

// TestMutationHarness runs the whole harness against the real binary and
// applies the §4.2 gate. `make mutation` runs exactly this (with
// -write-results); the committed testdata/mutation-results.json is its
// output.
func TestMutationHarness(t *testing.T) {
	if testing.Short() {
		t.Skip("mutation harness needs the real binary (skipped in -short)")
	}
	root, err := golden.ModuleRoot()
	if err != nil {
		t.Fatalf("module root: %v", err)
	}
	opts := Options{
		ManifestPath: filepath.Join(root, "testdata", "adversarial", "manifest.json"),
		MaxPerCell:   *flagMaxPerCell,
		CaseTimeout:  30 * time.Second,
		Deadline:     4 * time.Minute,
	}
	if *flagWriteResults {
		out := *flagOut
		if out == "" {
			out = filepath.Join(root, "testdata", "mutation-results.json")
		}
		opts.OutPath = out
	}
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("mutation run failed: %v", err)
	}

	t.Logf("totals: valid=%d hit=%d missed=%d honored=%d broken=%d notApp=%d harnessErr=%d",
		res.Totals.Valid, res.Totals.Hit, res.Totals.Missed, res.Totals.Honored,
		res.Totals.Broken, res.Totals.NotApp, res.Totals.HarnessEr)
	t.Logf("catch-rate: strict %.1f%% · declared %.1f%%", res.CatchRate.Strict, res.CatchRate.Declared)
	for _, rt := range res.PerRule {
		t.Logf("  %-8s valid=%-2d hit=%-2d honored=%-2d missed=%-2d broken=%-2d napp=%-2d strict=%5.1f%% declared=%5.1f%%",
			rt.Rule, rt.Valid, rt.Hit, rt.Honored, rt.Missed, rt.Broken, rt.NotApp, rt.StrictPct, rt.DeclaredPct)
	}
	for _, bs := range res.BlindSpots {
		t.Logf("  BLIND SPOT %s × %s: cases %v fired %v", bs.Rule, bs.Mutator, bs.Cases, bs.Fired)
	}

	if res.Totals.HarnessEr > 0 {
		var first caseResult
		for _, c := range res.Cases {
			if c.Verdict == verdictHarnessErr {
				first = c
				break
			}
		}
		t.Fatalf("harness errors: %d — first: %s (%s)", res.Totals.HarnessEr, first.ID, first.Note)
	}
	if res.Totals.Valid == 0 {
		t.Fatal("no valid mutation cases — harness is not measuring anything")
	}
	if res.CatchRate.Declared < gateTotalPct {
		t.Errorf("GATE: declared catch-rate %.1f%% < %.1f%% (strict %.1f%%)",
			res.CatchRate.Declared, gateTotalPct, res.CatchRate.Strict)
	}
	for _, rt := range res.PerRule {
		if rt.Valid > 0 && rt.DeclaredPct <= 0 {
			t.Errorf("per-rule gate: %s has %d valid cases but declared catch-rate %.1f%%",
				rt.Rule, rt.Valid, rt.DeclaredPct)
		}
	}
}

// TestCatchRateMath pins the percentage arithmetic of the audited number.
func TestCatchRateMath(t *testing.T) {
	tl := tally{}
	tl.add(verdictHit)
	tl.add(verdictHit)
	tl.add(verdictMissed)
	tl.add(verdictHonored)
	tl.add(verdictBroken)
	if tl.Valid != 4 {
		t.Fatalf("valid = %d, want 4 (broken-syntax excluded from the denominator)", tl.Valid)
	}
	if got := tl.strictPct(); math.Abs(got-50.0) > 1e-9 {
		t.Fatalf("strict = %.1f, want 50.0", got)
	}
	if got := tl.declaredPct(); math.Abs(got-75.0) > 1e-9 {
		t.Fatalf("declared = %.1f, want 75.0", got)
	}
}

// loadManifest reads and parses the corpus manifest.json.
func loadManifest(path string) (manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}
	var man manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return manifest{}, err
	}
	return man, nil
}

// poolsByRule groups ok-case indices by rule, file-sorted (the §4.4 pools).
func poolsByRule(man manifest) map[string][]int {
	pool := map[string][]int{}
	for i, c := range man.Cases {
		if c.Rule == nil || c.Expect != "miss" {
			continue
		}
		pool[*c.Rule] = append(pool[*c.Rule], i)
	}
	for r := range pool {
		idx := pool[r]
		sort.Slice(idx, func(a, b int) bool {
			return man.Cases[idx[a]].File < man.Cases[idx[b]].File
		})
		pool[r] = idx
	}
	return pool
}
