// corpus-check runs the adversarial corpus manifest (testdata/adversarial,
// test-strategy §3.6) against a vanguard binary and reports agreement.
//
// For every manifest entry the harness scans the case file in an isolated
// temp directory with the entry's config pass and evaluates the primary-rule
// expectation:
//
//	expect hit      -> the rule reports >= 1 finding for the case
//	                   (a declared expected_miss still honored counts as
//	                   declared-ok — see the two agreement numbers below)
//	expect miss     -> the rule reports 0 findings for the case
//	                   (cross-rule hits are ignored by design: the corpus
//	                   is per-aspect, one case aims at one rule)
//	expect parse-ok -> the file parses with zero diagnostics
//
// Two agreement numbers are reported (test-strategy §2.2):
//
//	strict   = cases whose verdict is ok / all cases — the literal §2.2
//	           reading ("kết quả thực khớp expect"), where a declared
//	           expected-miss that did miss is a mismatch, classified.
//	declared = (ok + expected-miss-honored) / all — the manifest as a whole
//	           (expect + pre-declared expected_miss + miss_reason) matched
//	           what the binary actually did.
//
// The -min-agreement gate applies to the declared number; both print so
// neither can hide behind the other.
//
// Go, not Python: the strategy §8 table names a python script, but the
// measurement environment (dind-sandbox:29) ships no python3; the same
// inputs (manifest) and outputs (JSON + per-rule table) are kept.
//
// Usage:
//
//	go run ./tools/corpus-check -repo . [-bin ./bin/vanguard] \
//		[-out testdata/corpus-results.json] [-min-agreement 85]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type manifestEntry struct {
	ID           string   `json:"id"`
	Rule         *string  `json:"rule"`
	File         string   `json:"file"`
	Expect       string   `json:"expect"`
	Tier         string   `json:"tier"`
	Config       string   `json:"config"`
	Also         []string `json:"also"`
	ExpectedMiss bool     `json:"expected_miss"`
	MissReason   string   `json:"miss_reason,omitempty"`
}

type manifest struct {
	Schema string          `json:"schema"`
	Cases  []manifestEntry `json:"cases"`
}

type finding struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Location struct {
		File string `json:"file"`
		Line int    `json:"line"`
	} `json:"location"`
}

type diag struct {
	Message  string `json:"message"`
	Location struct {
		File string `json:"file"`
		Line int    `json:"line"`
	} `json:"location"`
}

type scanReport struct {
	Findings    []finding `json:"findings"`
	Diagnostics []diag    `json:"diagnostics"`
	Summary     struct {
		Files            int `json:"files"`
		SkippedFiles     int `json:"skippedFiles"`
		ParseDiagnostics int `json:"parseDiagnostics"`
	} `json:"summary"`
}

type caseResult struct {
	ID           string   `json:"id"`
	Rule         *string  `json:"rule"`
	File         string   `json:"file"`
	Expect       string   `json:"expect"`
	Tier         string   `json:"tier"`
	Config       string   `json:"config"`
	ExpectedMiss bool     `json:"expected_miss"`
	Fired        []string `json:"fired"`
	FireCount    int      `json:"fire_count"`
	DiagCount    int      `json:"diag_count"`
	Verdict      string   `json:"verdict"`
	Note         string   `json:"note,omitempty"`
}

type results struct {
	Schema    string       `json:"schema"`
	Tool      string       `json:"tool"`
	Vanguard  string       `json:"vanguard"`
	Total     int          `json:"total"`
	Summary   tallyJSON    `json:"summary"`
	PerRule   []ruleTally  `json:"per_rule"`
	Cases     []caseResult `json:"cases"`
	Agreement struct {
		Strict   float64 `json:"strict"`
		Declared float64 `json:"declared"`
	} `json:"agreement"`
}

type tallyJSON struct {
	OK              int `json:"ok"`
	DeclaredMiss    int `json:"declared_miss_honored"`
	ViolationMissed int `json:"violation_missed"`
	FalsePositive   int `json:"false_positive"`
	UnexpectedDiag  int `json:"unexpected_diag"`
	HarnessError    int `json:"harness_error"`
}

type ruleTally struct {
	Rule  string `json:"rule"`
	Total int    `json:"total"`
	tallyJSON
	StrictAgreement   float64 `json:"strict_agreement"`
	DeclaredAgreement float64 `json:"declared_agreement"`
}

type tally struct {
	total int
	tallyJSON
}

func (t *tally) add(v string) {
	t.total++
	switch v {
	case "ok":
		t.OK++
	case "honored-expected-miss":
		t.DeclaredMiss++
	case "violation-missed":
		t.ViolationMissed++
	case "false-positive":
		t.FalsePositive++
	case "unexpected-diag":
		t.UnexpectedDiag++
	default:
		t.HarnessError++
	}
}

func (t tally) strictPct() float64 {
	if t.total == 0 {
		return 100
	}
	return 100 * float64(t.OK) / float64(t.total)
}

func (t tally) declaredPct() float64 {
	if t.total == 0 {
		return 100
	}
	return 100 * float64(t.OK+t.DeclaredMiss) / float64(t.total)
}

// companionJava gives .proto-only scan roots a detected Java surface: the
// java adapter needs at least one Java file before it walks the tree, and
// the proto reader reads .proto files from the same walk. Findings from the
// companion are cross-rule hits and never enter the primary-rule gate.
const companionJava = `// corpus-check companion: gives the scanner a detected Java surface.
// Findings from this file are cross-rule hits, ignored by the manifest gate.
package probe;

import org.springframework.web.bind.annotation.RestController;

@RestController
class Companion {
}
`

func main() {
	repo := flag.String("repo", ".", "repository root (holds testdata/adversarial)")
	bin := flag.String("bin", "", "vanguard binary (default <repo>/bin/vanguard)")
	out := flag.String("out", "", "results JSON path (default <repo>/testdata/corpus-results.json)")
	caseTimeout := flag.Duration("case-timeout", 30*time.Second, "per-case scan timeout")
	deadline := flag.Duration("deadline", 8*time.Minute, "whole-run deadline (A2: bounded work)")
	minAgreement := flag.Float64("min-agreement", 85, "gate on the declared agreement percent")
	flag.Parse()

	// Resolve defaults BEFORE absolutizing: filepath.Abs("") silently
	// returns the CWD, which would defeat the -bin/-out defaults below.
	advRoot := filepath.Join(*repo, "testdata", "adversarial")
	if *bin == "" {
		*bin = filepath.Join(*repo, "bin", "vanguard")
	}
	if *out == "" {
		*out = filepath.Join(*repo, "testdata", "corpus-results.json")
	}

	// The per-case scan runs with cmd.Dir inside a temp dir; every path the
	// harness passes on (binary, configs, case files) must be absolute.
	if abs, err := filepath.Abs(*repo); err == nil {
		*repo = abs
	}
	if abs, err := filepath.Abs(*bin); err == nil {
		*bin = abs
	}
	if abs, err := filepath.Abs(*out); err == nil {
		*out = abs
	}
	advRoot = filepath.Join(*repo, "testdata", "adversarial")

	raw, err := os.ReadFile(filepath.Join(advRoot, "manifest.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "read manifest.json:", err)
		os.Exit(2)
	}
	var man manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		fmt.Fprintln(os.Stderr, "parse manifest.json:", err)
		os.Exit(2)
	}
	if len(man.Cases) == 0 {
		fmt.Fprintln(os.Stderr, "manifest.json holds no cases")
		os.Exit(2)
	}

	version := vanguardVersion(*bin)

	start := time.Now()
	res := results{Schema: "vanguard-corpus-results/1", Tool: "tools/corpus-check", Vanguard: version}
	perRule := map[string]*tally{}
	var order []string
	get := func(rule string) *tally {
		if _, ok := perRule[rule]; !ok {
			perRule[rule] = &tally{}
			order = append(order, rule)
		}
		return perRule[rule]
	}
	var grand tally
	harnessFail := 0

	for _, e := range man.Cases {
		if time.Since(start) > *deadline {
			fmt.Println("deadline reached — stopping early; results cover", grand.total, "of", len(man.Cases), "cases")
			break
		}
		rule := "(_global)"
		if e.Rule != nil {
			rule = *e.Rule
		}
		rep, herr := scanCase(advRoot, e, *bin, *caseTimeout)
		var res1 caseResult
		if herr != "" {
			harnessFail++
			res1 = caseResult{ID: e.ID, Rule: e.Rule, File: e.File, Expect: e.Expect,
				Tier: e.Tier, Config: e.Config, ExpectedMiss: e.ExpectedMiss,
				Verdict: "harness-error", Note: herr}
		} else {
			verdict, note, fired := evaluate(e, rep)
			res1 = caseResult{ID: e.ID, Rule: e.Rule, File: e.File, Expect: e.Expect,
				Tier: e.Tier, Config: e.Config, ExpectedMiss: e.ExpectedMiss,
				Fired: fired, FireCount: len(rep.Findings), DiagCount: len(rep.Diagnostics),
				Verdict: verdict, Note: note}
		}
		res.Cases = append(res.Cases, res1)
		get(rule).add(res1.Verdict)
		grand.add(res1.Verdict)
	}

	sort.Strings(order)
	for _, rule := range order {
		t := perRule[rule]
		res.PerRule = append(res.PerRule, ruleTally{Rule: rule, Total: t.total,
			tallyJSON: t.tallyJSON, StrictAgreement: t.strictPct(), DeclaredAgreement: t.declaredPct()})
	}
	res.Total = grand.total
	res.Summary = grand.tallyJSON
	res.Agreement.Strict = grand.strictPct()
	res.Agreement.Declared = grand.declaredPct()

	if b, err := json.MarshalIndent(res, "", "  "); err == nil {
		if err := os.MkdirAll(filepath.Dir(*out), 0o755); err == nil {
			if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write results:", err)
				os.Exit(2)
			}
		}
	}

	fmt.Println("per-rule agreement (strict = ok/total; declared = ok + pre-declared expected-miss):")
	for _, rt := range res.PerRule {
		fmt.Printf("  %-8s total=%-3d ok=%-3d declaredMiss=%-2d missed=%-2d fp=%-2d diag=%-2d herr=%-2d strict=%5.1f%% declared=%5.1f%%\n",
			rt.Rule, rt.Total, rt.OK, rt.DeclaredMiss, rt.ViolationMissed, rt.FalsePositive,
			rt.UnexpectedDiag, rt.HarnessError, rt.StrictAgreement, rt.DeclaredAgreement)
	}
	fmt.Printf("OVERALL: strict %.1f%% (%d/%d) · declared %.1f%% · missed=%d fp=%d diag=%d herr=%d\n",
		res.Agreement.Strict, grand.OK, grand.total, res.Agreement.Declared,
		grand.ViolationMissed, grand.FalsePositive, grand.UnexpectedDiag, grand.HarnessError)
	fmt.Println("\nnon-ok cases:")
	for _, c := range res.Cases {
		if c.Verdict != "ok" && c.Verdict != "honored-expected-miss" {
			fmt.Printf("  %-8s %-38s expect=%-8s verdict=%-16s fired=%v %s\n",
				deref(c.Rule), c.ID, c.Expect, c.Verdict, c.Fired, c.Note)
		}
	}
	fmt.Printf("results: %s (harness failures: %d)\n", *out, harnessFail)

	switch {
	case harnessFail > 0:
		os.Exit(2)
	case res.Agreement.Declared < *minAgreement:
		fmt.Printf("GATE: declared agreement %.1f%% < %.1f%%\n", res.Agreement.Declared, *minAgreement)
		os.Exit(1)
	default:
		fmt.Printf("GATE: declared agreement %.1f%% >= %.1f%% (strict %.1f%%)\n",
			res.Agreement.Declared, *minAgreement, res.Agreement.Strict)
	}
}

func deref(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func vanguardVersion(bin string) string {
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return "unknown (" + err.Error() + ")"
	}
	return strings.TrimSpace(string(out))
}

func scanCase(advRoot string, e manifestEntry, bin string, timeout time.Duration) (scanReport, string) {
	var rep scanReport
	src := filepath.Join(advRoot, e.File)
	tmp, err := os.MkdirTemp("", "vgcase-")
	if err != nil {
		return rep, "tmpdir: " + err.Error()
	}
	defer os.RemoveAll(tmp)
	if err := copyFile(src, filepath.Join(tmp, filepath.Base(e.File))); err != nil {
		return rep, "copy: " + err.Error()
	}
	if strings.HasSuffix(e.File, ".proto") {
		if err := os.WriteFile(filepath.Join(tmp, "Companion.java"), []byte(companionJava), 0o644); err != nil {
			return rep, "companion: " + err.Error()
		}
	}
	cfg := filepath.Join(advRoot, ".vanguard.yaml")
	if e.Config == "enable-r6xx-01" {
		cfg = filepath.Join(advRoot, ".vanguard-r6xx01.yaml")
	}
	cmd := exec.Command(bin, "scan", tmp, "--format", "json", "--config", cfg)
	cmd.Dir = tmp
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		return rep, "start: " + err.Error()
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var waitErr error
	timedOut := false
	select {
	case waitErr = <-done:
	case <-time.After(timeout):
		timedOut = true
		_ = cmd.Process.Kill()
		<-done
	}
	if timedOut {
		return rep, "scan timeout after " + timeout.String()
	}
	if strings.TrimSpace(stdout.String()) == "" {
		// Empty stdout with exit 0 is the charter §6.3 "no API surface"
		// report — a valid empty result for a case with no findings.
		if waitErr != nil {
			return rep, "empty stdout; wait=" + fmt.Sprint(waitErr) +
				" stderr=" + tail(stderr.String(), 400)
		}
		return rep, ""
	}
	if err := json.Unmarshal([]byte(stdout.String()), &rep); err != nil {
		return rep, "json: " + err.Error()
	}
	return rep, ""
}

func tail(s string, n int) string {
	if len(s) <= n {
		return strings.TrimSpace(s)
	}
	return "…" + strings.TrimSpace(s[len(s)-n:])
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func firedRules(rep scanReport) []string {
	set := map[string]bool{}
	for _, f := range rep.Findings {
		set[f.RuleID] = true
	}
	out := make([]string, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func evaluate(e manifestEntry, rep scanReport) (verdict, note string, fired []string) {
	fired = firedRules(rep)
	primary := ""
	if e.Rule != nil {
		primary = *e.Rule
	}
	hasPrimary := false
	for _, r := range fired {
		if r == primary {
			hasPrimary = true
			break
		}
	}
	switch e.Expect {
	case "parse-ok":
		if len(rep.Diagnostics) > 0 {
			return "unexpected-diag", rep.Diagnostics[0].Message, fired
		}
		return "ok", "", fired
	case "hit":
		if hasPrimary {
			return "ok", "", fired
		}
		if e.ExpectedMiss {
			return "honored-expected-miss", "", fired
		}
		return "violation-missed", "", fired
	default: // "miss"
		if hasPrimary {
			return "false-positive", "", fired
		}
		return "ok", "", fired
	}
}
