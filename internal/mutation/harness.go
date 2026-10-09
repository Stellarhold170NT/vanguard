package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Stellarhold170NT/vanguard/internal/golden"
)

// selectionFormula documents the §4.4 case-selection recipe the JSON carries
// so an auditor can replay the exact case set.
const selectionFormula = "per (target rule, mutator) cell: pool = corpus manifest ok-cases " +
	"(expect: miss) of the pairing's pool rule, sorted by file path ascending; " +
	"apply the mutator to each until maxPerCell valid mutations are collected; " +
	"files the mutator cannot transform are counted as not_applicable. " +
	"No randomness — two runs select the same cases."

// manifestCase is one manifest.json entry (w4-01 §3.6 schema); only the
// fields the harness reads are declared.
type manifestCase struct {
	ID     string  `json:"id"`
	Rule   *string `json:"rule"`
	File   string  `json:"file"`
	Expect string  `json:"expect"`
	Config string  `json:"config"`
}

// manifest mirrors testdata/adversarial/manifest.json (w4-01 §3.6 schema).
type manifest struct {
	Schema string         `json:"schema"`
	Cases  []manifestCase `json:"cases"`
}

// scanReport is the subset of vanguard's JSON scan output the harness reads
// (same shape tools/corpus-check parses).
type scanReport struct {
	Findings []struct {
		RuleID string `json:"ruleId"`
	} `json:"findings"`
	Diagnostics []struct {
		Message string `json:"message"`
	} `json:"diagnostics"`
}

// caseResult is one mutation case's outcome, JSON-serialized verbatim.
type caseResult struct {
	ID          string   `json:"id"` // <mutator>/<pool rule>/<ok-case id>
	Mutator     string   `json:"mutator"`
	Kind        string   `json:"kind"`
	PoolRule    string   `json:"pool_rule"`
	Targets     []string `json:"target_rules"`
	File        string   `json:"file"` // corpus-relative ok-case path
	Config      string   `json:"config"`
	Verdict     string   `json:"verdict"`
	Fired       []string `json:"fired,omitempty"`
	DiagCount   int      `json:"diag_count"`
	Note        string   `json:"note,omitempty"`
	HonoredMiss bool     `json:"honored_expected_miss"`
	MissReason  string   `json:"miss_reason,omitempty"`
}

// Verdicts. hit/missed/honored enter the catch-rate denominators;
// broken-syntax, not-applicable and harness-error are logged counts only.
const (
	verdictHit        = "hit"
	verdictMissed     = "missed"
	verdictHonored    = "honored-expected-miss"
	verdictBroken     = "broken-syntax"
	verdictNotApp     = "not-applicable"
	verdictHarnessErr = "harness-error"
)

// tally counts one cell's cases.
type tally struct {
	Valid     int `json:"valid"`
	Hit       int `json:"hit"`
	Missed    int `json:"missed"`
	Honored   int `json:"honored_expected_miss"`
	Broken    int `json:"broken_syntax"`
	NotApp    int `json:"not_applicable"`
	HarnessEr int `json:"harness_error"`
}

func (t *tally) add(v string) {
	switch v {
	case verdictHit:
		t.Valid++
		t.Hit++
	case verdictMissed:
		t.Valid++
		t.Missed++
	case verdictHonored:
		t.Valid++
		t.Honored++
	case verdictBroken:
		t.Broken++
	case verdictNotApp:
		t.NotApp++
	default:
		t.HarnessEr++
	}
}

func (t tally) strictPct() float64 {
	if t.Valid == 0 {
		return 0
	}
	return 100 * float64(t.Hit) / float64(t.Valid)
}

// declaredPct counts pre-declared honored misses as caught: the two-number
// discipline carried over from tools/corpus-check (w4-01 §3) so the known
// unregistered rules cannot hide, and cannot drag the gate either. Both
// numbers are always reported.
func (t tally) declaredPct() float64 {
	if t.Valid == 0 {
		return 0
	}
	return 100 * float64(t.Hit+t.Honored) / float64(t.Valid)
}

// ruleTally is one rule row: every pairing targeting the rule contributes.
type ruleTally struct {
	Rule       string `json:"rule"`
	Registered bool   `json:"registered"`
	tally
	StrictPct   float64 `json:"strict_pct"`
	DeclaredPct float64 `json:"declared_pct"`
}

// mutatorTally is one mutator row (postmortem data, not gated — §4.2).
type mutatorTally struct {
	Mutator string `json:"mutator"`
	Kind    string `json:"kind"`
	tally
	StrictPct   float64 `json:"strict_pct"`
	DeclaredPct float64 `json:"declared_pct"`
}

// blindSpot is one (rule, mutator) cell with valid cases and zero hits that
// was NOT pre-declared — the §4.3 blind-spot entry feeding w5-06.
type blindSpot struct {
	Rule       string     `json:"rule"`
	Mutator    string     `json:"mutator"`
	Cases      []string   `json:"cases"`
	Fired      [][]string `json:"fired"`
	Hypothesis string     `json:"hypothesis"`
	Suggestion string     `json:"suggestion"`
}

// results is the machine-readable artifact (testdata/mutation-results.json).
// No timestamps, no absolute paths, no host data: two runs marshal
// byte-identical JSON (determinism acceptance, brief w4-02).
type results struct {
	Schema    string `json:"schema"`
	Tool      string `json:"tool"`
	Vanguard  string `json:"vanguard"`
	Selection struct {
		Formula      string `json:"formula"`
		MaxPerCell   int    `json:"max_per_cell"`
		ManifestCase int    `json:"manifest_cases"`
	} `json:"selection"`
	Totals    tally `json:"totals"`
	CatchRate struct {
		Strict   float64 `json:"strict"`
		Declared float64 `json:"declared"`
	} `json:"catch_rate"`
	PerRule    []ruleTally    `json:"per_rule"`
	PerMutator []mutatorTally `json:"per_mutator"`
	BlindSpots []blindSpot    `json:"blind_spots"`
	Cases      []caseResult   `json:"cases"`
}

// Options bounds and shapes one harness run.
type Options struct {
	ManifestPath string        // testdata/adversarial/manifest.json
	BinaryPath   string        // vanguard binary; "" → build via internal/golden
	OutPath      string        // results JSON; "" → no file written
	MaxPerCell   int           // §4.4 cap (default 3)
	CaseTimeout  time.Duration // per-scan timeout (default 30s)
	Deadline     time.Duration // whole-run deadline (default 4m — CI budget §4.4)
	MutatedDir   string        // optional dir to keep the mutated fixtures for inspection
}

func (o *Options) fillDefaults() {
	if o.MaxPerCell <= 0 {
		o.MaxPerCell = 3
	}
	if o.CaseTimeout <= 0 {
		o.CaseTimeout = 30 * time.Second
	}
	if o.Deadline <= 0 {
		o.Deadline = 4 * time.Minute
	}
}

// Run executes the whole harness and returns the results (it does not gate —
// the caller, usually TestMutationHarness, applies the §4.2 gate).
func Run(opts Options) (*results, error) {
	opts.fillDefaults()
	raw, err := os.ReadFile(opts.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("mutation: read manifest: %w", err)
	}
	var man manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return nil, fmt.Errorf("mutation: parse manifest: %w", err)
	}
	if len(man.Cases) == 0 {
		return nil, fmt.Errorf("mutation: manifest holds no cases")
	}
	advRoot := filepath.Dir(opts.ManifestPath)

	bin := opts.BinaryPath
	if bin == "" {
		bin, err = golden.BinaryPath()
		if err != nil {
			return nil, err
		}
	}

	res := &results{Schema: "vanguard-mutation-results/1", Tool: "internal/mutation"}
	res.Vanguard = vanguardVersion(bin, opts.CaseTimeout)
	res.Selection.Formula = selectionFormula
	res.Selection.MaxPerCell = opts.MaxPerCell
	res.Selection.ManifestCase = len(man.Cases)

	// Group the manifest's ok-cases by rule, file-sorted — the §4.4 pools.
	pool := map[string][]int{} // rule → indices into man.Cases
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

	start := time.Now()
	casesRun := 0
	for _, m := range mutators {
		for _, p := range m.Pairings {
			if time.Since(start) > opts.Deadline {
				return res, fmt.Errorf("mutation: run deadline %s exceeded after %d cases", opts.Deadline, casesRun)
			}
			poolRule := p.PoolRule
			indices, ok := pool[poolRule]
			if poolRule == "*" {
				indices, ok = allOK(man)
			}
			if !ok {
				continue // pool rule has no ok-cases — cell reported empty via tallies
			}
			perCell := tally{}
			taken := 0
			for _, ci := range indices {
				if taken >= opts.MaxPerCell {
					break
				}
				if time.Since(start) > opts.Deadline {
					return res, fmt.Errorf("mutation: run deadline %s exceeded after %d cases", opts.Deadline, casesRun)
				}
				c := man.Cases[ci]
				casesRun++
				src, err := os.ReadFile(filepath.Join(advRoot, c.File))
				if err != nil {
					res.Cases = append(res.Cases, caseResult{ID: m.Name + "/" + poolRule + "/" + c.ID,
						Mutator: m.Name, Kind: m.Kind, PoolRule: poolRule, Targets: p.Targets,
						File: c.File, Config: c.Config, Verdict: verdictHarnessErr,
						Note: "read ok-case: " + err.Error()})
					continue
				}
				mutated, applied := m.Apply(src)
				if !applied && opts.MutatedDir != "" {
					// Debug aid only: never write a file the mutator refused.
					continue
				}
				if opts.MutatedDir != "" {
					_ = os.WriteFile(filepath.Join(opts.MutatedDir, m.Name+"-"+filepath.Base(c.File)), mutated, 0o644)
				}
				cr := caseResult{ID: m.Name + "/" + poolRule + "/" + c.ID,
					Mutator: m.Name, Kind: m.Kind, PoolRule: poolRule, Targets: p.Targets,
					File: c.File, Config: c.Config}
				if !applied {
					cr.Verdict = verdictNotApp
					cr.Note = "mutator found no applicable pattern in this ok-case"
					res.Cases = append(res.Cases, cr)
					perCell.add(cr.Verdict)
					continue
				}
				// expectedMiss is declared BEFORE the scan: every target rule
				// unregistered → the case cannot hit by construction.
				expected := true
				for _, t := range p.Targets {
					if !unregisteredRules[t] {
						expected = false
						break
					}
				}
				cr.HonoredMiss = expected
				if expected {
					cr.MissReason = expectedMissReason
				}
				fired, diagCount, herr := scanMutated(opts, bin, advRoot, c, mutated)
				cr.Fired = fired
				cr.DiagCount = diagCount
				switch {
				case herr != "":
					cr.Verdict = verdictHarnessErr
					cr.Note = herr
				case diagCount > 0:
					cr.Verdict = verdictBroken
					cr.Note = "mutated file produced " + itoa(diagCount) + " parse diagnostics"
				case hitsAny(fired, p.Targets):
					cr.Verdict = verdictHit
				case expected:
					cr.Verdict = verdictHonored
				default:
					cr.Verdict = verdictMissed
					cr.Note = "no target rule fired; see fired list for cross-rule hits"
				}
				res.Cases = append(res.Cases, cr)
				perCell.add(cr.Verdict)
				switch cr.Verdict {
				case verdictHit, verdictMissed, verdictHonored:
					// §4.4: the cap counts collected valid mutations —
					// broken-syntax and harness-error cases do not consume
					// a cell slot (they say nothing about the rule).
					taken++
				}
			}
			// Fold this pairing's cell into the per-rule and per-mutator tallies.
			for _, t := range p.Targets {
				rule := res.ruleTally(t)
				mergeTally(&rule.tally, perCell)
			}
			mt := res.mutatorTally(m)
			mergeTally(&mt.tally, perCell)
			mergeTally(&res.Totals, perCell)
		}
	}

	// Percentages + ordering (sorted for stable JSON).
	sort.Slice(res.Cases, func(i, j int) bool { return res.Cases[i].ID < res.Cases[j].ID })
	for i := range res.PerRule {
		res.PerRule[i].StrictPct = res.PerRule[i].strictPct()
		res.PerRule[i].DeclaredPct = res.PerRule[i].declaredPct()
	}
	sort.Slice(res.PerRule, func(i, j int) bool { return res.PerRule[i].Rule < res.PerRule[j].Rule })
	for i := range res.PerMutator {
		res.PerMutator[i].StrictPct = res.PerMutator[i].strictPct()
		res.PerMutator[i].DeclaredPct = res.PerMutator[i].declaredPct()
	}
	res.CatchRate.Strict = res.Totals.strictPct()
	res.CatchRate.Declared = res.Totals.declaredPct()
	res.BlindSpots = collectBlindSpots(res)

	if opts.OutPath != "" {
		b, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return res, fmt.Errorf("mutation: marshal results: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(opts.OutPath), 0o755); err != nil {
			return res, fmt.Errorf("mutation: mkdir for results: %w", err)
		}
		if err := os.WriteFile(opts.OutPath, append(b, '\n'), 0o644); err != nil {
			return res, fmt.Errorf("mutation: write results: %w", err)
		}
	}
	return res, nil
}

// ruleTally returns (creating) the tally row for one rule.
func (r *results) ruleTally(rule string) *ruleTally {
	for i := range r.PerRule {
		if r.PerRule[i].Rule == rule {
			return &r.PerRule[i]
		}
	}
	r.PerRule = append(r.PerRule, ruleTally{Rule: rule, Registered: !unregisteredRules[rule]})
	return &r.PerRule[len(r.PerRule)-1]
}

func (r *results) mutatorTally(m *Mutator) *mutatorTally {
	for i := range r.PerMutator {
		if r.PerMutator[i].Mutator == m.Name {
			return &r.PerMutator[i]
		}
	}
	r.PerMutator = append(r.PerMutator, mutatorTally{Mutator: m.Name, Kind: m.Kind})
	return &r.PerMutator[len(r.PerMutator)-1]
}

// mergeTally adds every component of src into dst.
func mergeTally(dst *tally, src tally) {
	dst.Valid += src.Valid
	dst.Hit += src.Hit
	dst.Missed += src.Missed
	dst.Honored += src.Honored
	dst.Broken += src.Broken
	dst.NotApp += src.NotApp
	dst.HarnessEr += src.HarnessEr
}

// allOK returns every manifest ok-case index (the verb-swap wildcard pool).
func allOK(man manifest) ([]int, bool) {
	var idx []int
	for i, c := range man.Cases {
		if c.Rule != nil && c.Expect == "miss" {
			idx = append(idx, i)
		}
	}
	sort.Slice(idx, func(i, j int) bool { return man.Cases[idx[i]].File < man.Cases[idx[j]].File })
	return idx, len(idx) > 0
}

// hitsAny reports whether any target rule fired.
func hitsAny(fired, targets []string) bool {
	for _, f := range fired {
		for _, t := range targets {
			if f == t {
				return true
			}
		}
	}
	return false
}

// collectBlindSpots turns missed (valid, no hit, not honored) cases into
// §4.3 blind-spot entries grouped by (rule, mutator), each carrying the
// hypothesis class and a one-line improvement suggestion for w5-06.
func collectBlindSpots(res *results) []blindSpot {
	byCell := map[string]*blindSpot{}
	var order []string
	for _, c := range res.Cases {
		if c.Verdict != verdictMissed {
			continue
		}
		for _, t := range c.Targets {
			if unregisteredRules[t] {
				continue // pre-declared, never a blind spot
			}
			key := t + "|" + c.Mutator
			bs, ok := byCell[key]
			if !ok {
				bs = &blindSpot{Rule: t, Mutator: c.Mutator,
					Hypothesis: "predicate-gap",
					Suggestion: "verify the rule predicate against the mutated shape; " +
						"add a corpus case pinning the behavior if correct, or fix the rule if not (input for w5-06)"}
				byCell[key] = bs
				order = append(order, key)
			}
			bs.Cases = append(bs.Cases, c.ID)
			bs.Fired = append(bs.Fired, c.Fired)
		}
	}
	sort.Strings(order)
	out := make([]blindSpot, 0, len(order))
	for _, k := range order {
		out = append(out, *byCell[k])
	}
	return out
}

// scanMutated writes the mutated file into a fresh temp dir (plus the proto
// companion when needed), scans it with the ok-case's config pass, and
// returns the fired rule ids and diagnostic count.
func scanMutated(opts Options, bin, advRoot string, c manifestCase, mutated []byte) (fired []string, diagCount int, herr string) {
	tmp, err := os.MkdirTemp("", "vgmut-")
	if err != nil {
		return nil, 0, "tmpdir: " + err.Error()
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, filepath.Base(c.File)), mutated, 0o644); err != nil {
		return nil, 0, "write: " + err.Error()
	}
	if strings.HasSuffix(c.File, ".proto") {
		companion := `// mutation companion: gives the scanner a detected Java surface.
package probe;

import org.springframework.web.bind.annotation.RestController;

@RestController
class Companion {
}
`
		if err := os.WriteFile(filepath.Join(tmp, "Companion.java"), []byte(companion), 0o644); err != nil {
			return nil, 0, "companion: " + err.Error()
		}
	}
	cfg := filepath.Join(advRoot, ".vanguard.yaml")
	if c.Config == "enable-r6xx-01" {
		cfg = filepath.Join(advRoot, ".vanguard-r6xx01.yaml")
	}
	cmd := exec.Command(bin, "scan", tmp, "--format", "json", "--config", cfg)
	cmd.Dir = tmp
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return nil, 0, "start: " + err.Error()
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(opts.CaseTimeout):
		_ = cmd.Process.Kill()
		<-done
		return nil, 0, "scan timeout after " + opts.CaseTimeout.String()
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return nil, 0, "" // charter §6.3: empty stdout with exit 0 is a valid empty report
	}
	var rep scanReport
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		return nil, 0, "json: " + err.Error()
	}
	set := map[string]bool{}
	for _, f := range rep.Findings {
		set[f.RuleID] = true
	}
	for r := range set {
		fired = append(fired, r)
	}
	sort.Strings(fired)
	return fired, len(rep.Diagnostics), ""
}

// vanguardVersion reads the binary's version line for the results header
// (bounded by timeout — every exec gets a deadline, process amendment 2).
func vanguardVersion(bin string, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
