// audit-check — the FP/FN audit harness (test-strategy §7, task w4-04).
//
// It scans the audit sample (testdata/audit-sample — military-youth
// patterns rebuilt on a fictional domain, w1-03) with a real vanguard
// binary, merges the findings with the pre-registered expectations
// (expectations.json, authored before the first scan — anti-gaming §9.1),
// and materializes the labeling protocol:
//
//	generate (default)
//	  - scan the sample once
//	  - align findings <-> expectation slots (same rule, same file,
//	    line within +-4 — annotation vs method-line anchoring)
//	  - write the machine sheet testdata/audit-labeling.csv (one row per
//	    §7.3 line; same-span findings merged, multi-rule rows carry both)
//	  - write the printable sheet reports/w4-04-labeling-sheet.md
//	  - write testdata/audit-findings.json (raw scan evidence) and
//	    testdata/audit-results.json with labeled=false (§7.5 shape,
//	    metrics n/a until the gate labels)
//	  - exit 0
//
//	merge (-merge <labeled.csv>)
//	  - re-scan the sample (the scanner is deterministic; any drift
//	    between the labeled CSV and the fresh scan is a stale sheet,
//	    not a measurement)
//	  - read the gate's verdicts (TP/FP on finding rows, FN/
//	    CLAIM-REJECTED on fn-candidate rows, UNCLEAR/DISPUTED mid-flight)
//	  - compute per-rule precision/recall + the aggregate FP share
//	    (precision = tp/(tp+fp), recall-audit = tp/(tp+fn), 0/0 -> n/a)
//	  - write testdata/audit-results.json with labeled=true
//	  - exit 0 when the sheet is fully resolved and the FP share is
//	    below -max-fp; exit 1 while rows are unresolved (NE must not
//	    remain, §7.1) or the FP share breaches the ratchet threshold;
//	    exit 2 on harness failure (stale sheet, invalid labels, scan
//	    failure)
//
// The harness never labels: verdicts belong to the gate CH3 person
// (§7.2 — the agent only prepares sheet + harness). A provisional
// dry-run labeling used to validate this pipeline lives in the w4-04
// report and is explicitly NOT the audit result.
//
// Go, not Python: the measurement environment (dind-sandbox:29) ships no
// python3 — same inputs (sample + expectations) and outputs (CSV, sheet,
// JSON) as the strategy's tool table (§8).
//
// Usage:
//
//	go run ./tools/audit-check -repo . [-bin ./bin/vanguard] [-merge <csv>]
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Inputs

type slot struct {
	ID         string `json:"id"`
	Rule       string `json:"rule"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	PatternRef string `json:"pattern_ref,omitempty"`
	Note       string `json:"note,omitempty"`
	MissReason string `json:"miss_reason,omitempty"`
}

type expectations struct {
	Schema   string `json:"schema"`
	Sample   string `json:"sample"`
	Slots    []slot `json:"slots"`
	OkSlots  []slot `json:"ok_slots"`
	Declared []slot `json:"declared"`
}

func loadExpectations(path string) (expectations, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return expectations{}, err
	}
	var exp expectations
	if err := json.Unmarshal(raw, &exp); err != nil {
		return expectations{}, err
	}
	if len(exp.Slots) == 0 {
		return expectations{}, fmt.Errorf("%s: no expectation slots", path)
	}
	return exp, nil
}

// ---------------------------------------------------------------------------
// Scan report (vanguard --format json; the corpus-check shapes)

type finding struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Location struct {
		File string `json:"file"`
		Line int    `json:"line"`
	} `json:"location"`
	Message string `json:"message"`
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

// ---------------------------------------------------------------------------
// Sheet rows — one row per §7.3 line: same rule+span findings merge into
// one row; a row may carry several rules when two rules hit one span.

type row struct {
	ID        string // F-001… (finding) / slot id (fn candidate, traceable to expectations.json)
	Kind      string // finding | fn-candidate
	Rules     []string
	File      string
	Line      int
	Message   string
	FromSlots []string // expectation slot ids matched (empty = unexpected finding)
	AipRef    string
}

// labeledRow is one labeler-filled verdict line from the machine sheet.
type labeledRow struct {
	GateVerdict string
	ReasonCode  string
	Note        string
	AipRef      string
}

// ---------------------------------------------------------------------------
// Results (test-strategy §7.5)

type ruleMetrics struct {
	Rule      string `json:"rule"`
	Claimed   int    `json:"claimed"`
	Observed  int    `json:"observed"`
	TP        int    `json:"tp"`
	FP        int    `json:"fp"`
	FN        int    `json:"fn"`
	Precision string `json:"precision"`
	Recall    string `json:"recall"`
}

type results struct {
	Schema   string `json:"schema"`
	Tool     string `json:"tool"`
	Vanguard string `json:"vanguard"`
	Labeled  bool   `json:"labeled"`
	Meta     struct {
		Sample          string `json:"sample"`
		HTTPEndpoints   int    `json:"http_endpoints"`
		RpcDeclarations int    `json:"rpc_declarations"`
		FindingRows     int    `json:"finding_rows"`
		FnCandidateRows int    `json:"fn_candidate_rows"`
		UnexpectedRows  int    `json:"unexpected_rows"`
		DeclaredGaps    int    `json:"declared_gaps"`
		ParseDiags      int    `json:"parse_diagnostics"`
	} `json:"meta"`
	PerRule   []ruleMetrics `json:"per_rule"`
	Aggregate struct {
		TP        int     `json:"tp"`
		FP        int     `json:"fp"`
		FN        int     `json:"fn"`
		Precision string  `json:"precision"`
		Recall    string  `json:"recall"`
		FPShare   float64 `json:"fp_share_pct"`
	} `json:"aggregate"`
	Disputes   []string `json:"disputes"`
	Unresolved []string `json:"unresolved"`
	MaxFPShare float64  `json:"max_fp_share_pct"`
}

// aipByRule is the charter §3.1–3.6 "AIP" column, for the sheet's aip-ref
// prefill (the labeler replaces it with the exact quoted sentence + URL
// when a dispute reaches aip.dev, §7.4).
var aipByRule = map[string]string{
	"R1xx-01": "https://google.aip.dev/131, https://google.aip.dev/122",
	"R1xx-02": "https://google.aip.dev/131, https://google.aip.dev/133, https://google.aip.dev/135",
	"R1xx-03": "https://google.aip.dev/127, https://google.aip.dev/131",
	"R1xx-04": "https://google.aip.dev/122",
	"R1xx-05": "https://google.aip.dev/122, https://google.aip.dev/140",
	"R2xx-01": "https://google.aip.dev/131",
	"R2xx-02": "https://google.aip.dev/133",
	"R2xx-03": "https://google.aip.dev/134",
	"R2xx-04": "https://google.aip.dev/135",
	"R2xx-05": "https://google.aip.dev/136",
	"R2xx-06": "https://google.aip.dev/134",
	"R3xx-01": "https://google.aip.dev/158",
	"R3xx-02": "https://google.aip.dev/158",
	"R3xx-03": "https://google.aip.dev/158",
	"R4xx-01": "https://google.aip.dev/121, https://google.aip.dev/203",
	"R4xx-02": "https://google.aip.dev/140",
	"R4xx-03": "https://google.aip.dev/142",
	"R5xx-01": "https://google.aip.dev/193",
	"R5xx-02": "https://google.aip.dev/193",
	"R5xx-03": "https://google.aip.dev/133, https://google.aip.dev/193",
	"R6xx-02": "https://google.aip.dev/131 (applied to rpc, AIP-131..136)",
}

func main() {
	repo := flag.String("repo", ".", "repository root (holds testdata/audit-sample)")
	bin := flag.String("bin", "", "vanguard binary (default <repo>/bin/vanguard)")
	sample := flag.String("sample", "testdata/audit-sample", "sample directory (relative to repo)")
	csvOut := flag.String("csv", "testdata/audit-labeling.csv", "machine labeling sheet (relative to repo)")
	sheetOut := flag.String("sheet", "reports/w4-04-labeling-sheet.md", "printable labeling sheet (relative to repo)")
	findingsOut := flag.String("findings", "testdata/audit-findings.json", "raw scan evidence (relative to repo)")
	out := flag.String("out", "testdata/audit-results.json", "results JSON (relative to repo)")
	merge := flag.String("merge", "", "labeled CSV to merge (default: generate mode)")
	maxFP := flag.Float64("max-fp", 15, "gate: aggregate FP share percent must stay below this (charter ratchet level 2)")
	scanTimeout := flag.Duration("scan-timeout", 120*time.Second, "single scan timeout")
	flag.Parse()

	// Resolve defaults BEFORE absolutizing (corpus-check lesson: Abs("") = CWD).
	if *bin == "" {
		*bin = filepath.Join(*repo, "bin", "vanguard")
	}
	if a, err := filepath.Abs(*repo); err == nil {
		*repo = a
	}
	if a, err := filepath.Abs(*bin); err == nil {
		*bin = a
	}
	abs := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(*repo, p)
	}
	*sample, *csvOut, *sheetOut, *findingsOut, *out =
		abs(*sample), abs(*csvOut), abs(*sheetOut), abs(*findingsOut), abs(*out)

	exp, err := loadExpectations(filepath.Join(*sample, "expectations.json"))
	if err != nil {
		exitf(2, "load expectations: %v", err)
	}

	rep, herr := runScan(*bin, *repo, *sample, *scanTimeout)
	if herr != "" {
		exitf(2, "scan: %s", herr)
	}
	version := vanguardVersion(*bin)

	files := collectSampleFiles(*sample)
	rows, fnRows := align(rep, exp, files)

	if *merge == "" {
		writeJSON(*findingsOut, findingsPayload{"vanguard-audit-findings/1", "tools/audit-check", version, rep})
		writeCSVFile(*csvOut, rows, fnRows)
		writeSheet(*sheetOut, rows, fnRows, exp, version)
		res := buildResults(version, rep, exp, rows, fnRows, *sample, false, *maxFP)
		writeJSON(*out, res)
		fmt.Printf("generated: %d finding rows (%d unexpected), %d fn candidates, %d declared gaps, %d parse diags\n",
			len(rows), res.Meta.UnexpectedRows, len(fnRows), len(exp.Declared), len(rep.Diagnostics))
		fmt.Printf("sheet:     %s\ncsv:       %s\nresults:   %s (labeled=false)\n", *sheetOut, *csvOut, *out)
		return
	}

	labeled, err := readLabeledCSV(*merge)
	if err != nil {
		exitf(2, "merge: %v", err)
	}
	if err := validateLabels(labeled, rows, fnRows); err != nil {
		exitf(2, "merge: %v", err)
	}
	res := buildResults(version, rep, exp, rows, fnRows, *sample, true, *maxFP)
	applyVerdicts(res, rows, fnRows, labeled)
	writeJSON(*out, res)
	printSummary(res)

	switch {
	case len(res.Unresolved) > 0:
		fmt.Printf("GATE: %d row(s) without a final verdict — NE must not remain (test-strategy §7.1)\n", len(res.Unresolved))
		os.Exit(1)
	case res.Aggregate.FPShare >= *maxFP:
		fmt.Printf("GATE: FP share %.1f%% >= %.1f%% (ratchet level 2)\n", res.Aggregate.FPShare, *maxFP)
		os.Exit(1)
	default:
		fmt.Printf("GATE: FP share %.1f%% < %.1f%% (ratchet level 2) — precision %s, recall-audit %s\n",
			res.Aggregate.FPShare, *maxFP, res.Aggregate.Precision, res.Aggregate.Recall)
	}
}

func exitf(code int, format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(code)
}

// ---------------------------------------------------------------------------
// Scan + version

func runScan(bin, repo, sample string, timeout time.Duration) (scanReport, string) {
	var rep scanReport
	cfg := filepath.Join(sample, ".vanguard.yaml")
	cmd := exec.Command(bin, "scan", sample, "--format", "json", "--config", cfg)
	cmd.Dir = repo
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
		return rep, fmt.Sprintf("scan timeout after %s (process amendment 2: bounded work)", timeout)
	}
	if strings.TrimSpace(stdout.String()) == "" {
		if waitErr != nil {
			return rep, "empty stdout; wait=" + fmt.Sprint(waitErr) + " stderr=" + tail(stderr.String(), 400)
		}
		return rep, ""
	}
	if err := json.Unmarshal([]byte(stdout.String()), &rep); err != nil {
		return rep, "json: " + err.Error()
	}
	// A non-zero exit with a parsed report is the §6.3 ERROR-findings
	// contract — the findings are the measurement, not a harness error.
	return rep, ""
}

func vanguardVersion(bin string) string {
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return "unknown (" + err.Error() + ")"
	}
	return strings.TrimSpace(string(out))
}

func tail(s string, n int) string {
	if len(s) <= n {
		return strings.TrimSpace(s)
	}
	return "…" + strings.TrimSpace(s[len(s)-n:])
}

// ---------------------------------------------------------------------------
// Sample inventory + path normalization

func collectSampleFiles(sample string) []string {
	var out []string
	_ = filepath.Walk(sample, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if rel, err := filepath.Rel(sample, p); err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out
}

// normalizeFile maps a finding path onto a sample-relative path: vanguard
// reports paths relative to the scan root, but absolute or ./-prefixed
// spellings are tolerated via unique-suffix match.
func normalizeFile(f string, files []string) string {
	f = filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(f), "./"))
	for _, cand := range files {
		if cand == f {
			return cand
		}
	}
	for _, cand := range files {
		if strings.HasSuffix(f, cand) {
			return cand
		}
	}
	return f
}

var (
	mappingRe = regexp.MustCompile(`@(Get|Post|Put|Patch|Delete)Mapping`)
	rpcRe     = regexp.MustCompile(`^\s*rpc\s+`)
)

// countEndpoints is the sample inventory (a labeling-population number,
// NOT a discovery claim — recall-discovery is §6.2's separate metric).
// Only source files count: expectations/README prose may mention
// annotation spellings, and notes are not endpoints.
func countEndpoints(sample string) (http int, rpc int) {
	for _, f := range collectSampleFiles(sample) {
		if !strings.HasSuffix(f, ".java") && !strings.HasSuffix(f, ".proto") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(sample, f))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			if mappingRe.MatchString(strings.TrimSpace(line)) {
				http++
			}
			if strings.HasSuffix(f, ".proto") && rpcRe.MatchString(line) {
				rpc++
			}
		}
	}
	return http, rpc
}

// ---------------------------------------------------------------------------
// Alignment

// lineTolerance absorbs annotation-vs-method-line anchoring differences.
const lineTolerance = 4

// align merges findings with expectations into sheet rows:
//   - findings grouped by exact (file, line); every rule on that span
//     joins the row (§7.3: one line may carry several rules)
//   - a finding matches a slot when rule and file agree and the line sits
//     within +-4 of the slot line
//   - slots with no matching finding become fn-candidate rows (their id
//     stays the slot id for traceability)
//   - declared gaps never become rows (coverage transparency, excluded
//     from recall — the w4-01 declared-agreement discipline)
func align(rep scanReport, exp expectations, files []string) (rows []row, fnRows []row) {
	for i := range rep.Findings {
		rep.Findings[i].Location.File = normalizeFile(rep.Findings[i].Location.File, files)
	}

	bySpan := map[string]*row{}
	var order []string
	for _, f := range rep.Findings {
		k := f.Location.File + "|" + strconv.Itoa(f.Location.Line)
		r, ok := bySpan[k]
		if !ok {
			r = &row{File: f.Location.File, Line: f.Location.Line, Kind: "finding"}
			bySpan[k] = r
			order = append(order, k)
		}
		if !contains(r.Rules, f.RuleID) {
			r.Rules = append(r.Rules, f.RuleID)
		}
		if r.Message == "" {
			r.Message = f.Message
		}
	}

	matchSlot := func(s slot) *row {
		for _, k := range order {
			r := bySpan[k]
			for _, rule := range r.Rules {
				if rule == s.Rule && r.File == s.File && absDiff(r.Line, s.Line) <= lineTolerance {
					return r
				}
			}
		}
		return nil
	}

	for _, s := range exp.Slots {
		if r := matchSlot(s); r != nil {
			if !contains(r.FromSlots, s.ID) {
				r.FromSlots = append(r.FromSlots, s.ID)
			}
		} else {
			fnRows = append(fnRows, row{
				ID: s.ID, Kind: "fn-candidate", Rules: []string{s.Rule},
				File: s.File, Line: s.Line, Message: s.Note, FromSlots: []string{s.ID},
				AipRef: aipByRule[s.Rule],
			})
		}
	}

	for _, k := range order {
		r := bySpan[k]
		r.AipRef = aipForRules(r.Rules)
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].File != rows[j].File {
			return rows[i].File < rows[j].File
		}
		if rows[i].Line != rows[j].Line {
			return rows[i].Line < rows[j].Line
		}
		return strings.Join(rows[i].Rules, ",") < strings.Join(rows[j].Rules, ",")
	})
	for i := range rows {
		rows[i].ID = fmt.Sprintf("F-%03d", i+1)
	}
	return rows, fnRows
}

func aipForRules(rules []string) string {
	var out []string
	for _, r := range rules {
		if aip := aipByRule[r]; aip != "" && !contains(out, aip) {
			out = append(out, aip)
		}
	}
	return strings.Join(out, " · ")
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// CSV (machine sheet) — the gate edits the verdict columns here

var csvHeader = []string{"id", "kind", "rules", "file", "line", "message", "from_slots", "gate_verdict", "reason_code", "note", "aip_ref"}

func writeCSVFile(path string, rows, fnRows []row) error {
	all := append(append([]row{}, rows...), fnRows...)
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write(csvHeader); err != nil {
		return err
	}
	for _, r := range all {
		if err := w.Write([]string{
			r.ID, r.Kind, strings.Join(r.Rules, ", "), r.File, strconv.Itoa(r.Line),
			r.Message, strings.Join(r.FromSlots, ", "), "", "", "", r.AipRef,
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return writeFile(path, []byte(b.String()))
}

func readLabeledCSV(path string) (map[string]labeledRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%s: empty csv", path)
	}
	want := map[string]int{}
	for i, h := range records[0] {
		want[strings.TrimSpace(h)] = i
	}
	for _, col := range []string{"id", "gate_verdict"} {
		if _, ok := want[col]; !ok {
			return nil, fmt.Errorf("%s: missing column %q", path, col)
		}
	}
	out := map[string]labeledRow{}
	for _, rec := range records[1:] {
		if len(rec) == 0 || strings.TrimSpace(rec[want["id"]]) == "" {
			continue
		}
		id := strings.TrimSpace(rec[want["id"]])
		out[id] = labeledRow{
			GateVerdict: strings.TrimSpace(rec[want["gate_verdict"]]),
			ReasonCode:  csvField(rec, want, "reason_code"),
			Note:        csvField(rec, want, "note"),
			AipRef:      csvField(rec, want, "aip_ref"),
		}
	}
	return out, nil
}

func csvField(rec []string, want map[string]int, col string) string {
	i, ok := want[col]
	if !ok || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

// validateLabels checks the verdict vocabulary against the row kind
// (§7.1/§7.3: TP/FP belong to findings, FN/CLAIM-REJECTED to misses).
func validateLabels(labeled map[string]labeledRow, rows, fnRows []row) error {
	kind := map[string]string{}
	for _, r := range rows {
		kind[r.ID] = "finding"
	}
	for _, r := range fnRows {
		kind[r.ID] = "fn-candidate"
	}
	forFinding := map[string]bool{"TP": true, "FP": true, "UNCLEAR": true, "DISPUTED": true}
	forFn := map[string]bool{"FN": true, "CLAIM-REJECTED": true, "UNCLEAR": true, "DISPUTED": true}
	for id, l := range labeled {
		v := strings.ToUpper(strings.TrimSpace(l.GateVerdict))
		if v == "" {
			continue
		}
		k, ok := kind[id]
		if !ok {
			return fmt.Errorf("row %q: unknown id (stale sheet? regenerate)", id)
		}
		if k == "finding" && !forFinding[v] {
			return fmt.Errorf("row %s (finding): verdict %q not allowed here (TP/FP/UNCLEAR/DISPUTED)", id, v)
		}
		if k == "fn-candidate" && !forFn[v] {
			return fmt.Errorf("row %s (fn-candidate): verdict %q not allowed here (FN/CLAIM-REJECTED/UNCLEAR/DISPUTED)", id, v)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Printable sheet (§7.3 format)

func writeSheet(path string, rows, fnRows []row, exp expectations, version string) {
	var b strings.Builder
	fmt.Fprintf(&b, "# w4-04 — FP/FN labeling sheet (gate CH3)\n\n")
	fmt.Fprintf(&b, "Generated by `tools/audit-check` from `testdata/audit-sample` · vanguard `%s` · protocol: `docs/fpfn-protocol.md` (test-strategy §7).\n\n", version)
	fmt.Fprintf(&b, "**How to label** (self-contained — no other context needed):\n\n")
	fmt.Fprintf(&b, "1. **verdict** per row — on finding rows: `TP` (real violation per charter §3: right rule, right span, right context) or `FP` (code does not violate; pick a reason-code); on FN candidate rows: `FN` (real violation the scanner missed) or `CLAIM-REJECTED` (the pre-registered claim itself is wrong). `UNCLEAR`/`DISPUTED` only with a reason — **no NE may remain on the final sheet** (§7.1).\n")
	fmt.Fprintf(&b, "2. **reason-code** — FP: `heuristic-context` (heuristic wrong in this context) · `app-convention` (the app's own convention; tune config, rule is fine) · `wrong-span` (right rule, wrong location/message) · `severity-mismatch` (technically a violation but contract-harmless — propose lowering severity). FN: `parser-gap` · `predicate-gap` · `config-off` · `expected-miss`.\n")
	fmt.Fprintf(&b, "3. **disputes** (§7.4): read charter §3's rule text first — if it is clear, charter wins. Then open the rule's aip.dev page (pre-filled in `aip-ref`) and quote the exact decision sentence into `lý do`; remember vanguard re-states AIP concepts in Spring terms, note the mapping when it matters. Still undecided → mark `DISPUTED` + one-line note. Tie-break: **FP-first** — when unsure whether the code is wrong, label FP and propose tuning; never label TP \"for safety\".\n")
	fmt.Fprintf(&b, "4. **submit**: fill `gate_verdict`, `reason_code`, `note`, `aip_ref` in `testdata/audit-labeling.csv` (same row ids), then run:\n   `go run ./tools/audit-check -repo . -merge testdata/audit-labeling.csv`\n\n")

	fmt.Fprintf(&b, "## Findings (scanner output — %d rows)\n\n", len(rows))
	fmt.Fprintf(&b, "| # | rule | file:line | message (rút gọn) | verdict | reason-code | lý do (≤1 câu) | aip-ref |\n")
	fmt.Fprintf(&b, "|---|---|---|---|---|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | `%s:%d` | %s | | | | %s |\n",
			r.ID, strings.Join(r.Rules, ", "), r.File, r.Line, shorten(r.Message, 90), r.AipRef)
	}

	fmt.Fprintf(&b, "\n## FN candidates (verifier manual read — %d rows)\n\n", len(fnRows))
	fmt.Fprintf(&b, "Pre-registered claims with **no** matching finding (the scanner stayed silent where the claim expects a violation).\n\n")
	fmt.Fprintf(&b, "| # | rule | file:line | claim | verdict | reason-code | lý do (≤1 câu) | aip-ref |\n")
	fmt.Fprintf(&b, "|---|---|---|---|---|---|---|---|\n")
	for _, r := range fnRows {
		fmt.Fprintf(&b, "| %s | %s | `%s:%d` | %s | | | | %s |\n",
			r.ID, strings.Join(r.Rules, ", "), r.File, r.Line, shorten(r.Message, 90), r.AipRef)
	}

	fmt.Fprintf(&b, "\n## Expected-silent anchors (informational — a finding on any of these is an FP candidate)\n\n")
	fmt.Fprintf(&b, "| id | file:line | why silent |\n|---|---|---|\n")
	for _, s := range exp.OkSlots {
		fmt.Fprintf(&b, "| %s | `%s:%d` | %s |\n", s.ID, s.File, s.Line, s.Note)
	}

	fmt.Fprintf(&b, "\n## Declared coverage gaps (chartered rules not registered in v0.1 — excluded from recall)\n\n")
	fmt.Fprintf(&b, "| id | rule | trigger present in sample | reason |\n|---|---|---|---|\n")
	for _, s := range exp.Declared {
		fmt.Fprintf(&b, "| %s | %s | `%s:%d` | %s |\n", s.ID, s.Rule, s.File, s.Line, s.MissReason)
	}

	_ = writeFile(path, []byte(b.String()))
}

func shorten(s string, n int) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// ---------------------------------------------------------------------------
// Findings + results

type findingsPayload struct {
	Schema   string     `json:"schema"`
	Tool     string     `json:"tool"`
	Vanguard string     `json:"vanguard"`
	Report   scanReport `json:"report"`
}

func buildResults(version string, rep scanReport, exp expectations, rows, fnRows []row, sample string, labeled bool, maxFP float64) *results {
	res := &results{Schema: "vanguard-audit-results/1", Tool: "tools/audit-check", Vanguard: version, Labeled: labeled, MaxFPShare: maxFP}
	res.Meta.Sample = exp.Sample
	res.Meta.FindingRows = len(rows)
	res.Meta.FnCandidateRows = len(fnRows)
	res.Meta.DeclaredGaps = len(exp.Declared)
	res.Meta.ParseDiags = len(rep.Diagnostics)
	for _, r := range rows {
		if len(r.FromSlots) == 0 {
			res.Meta.UnexpectedRows++
		}
	}
	http, rpc := countEndpoints(sample)
	res.Meta.HTTPEndpoints = http
	res.Meta.RpcDeclarations = rpc
	res.Aggregate.Precision = "n/a"
	res.Aggregate.Recall = "n/a"

	claimed := map[string]int{}
	for _, s := range exp.Slots {
		claimed[s.Rule]++
	}
	observed := map[string]int{}
	for _, r := range rows {
		for _, rule := range r.Rules {
			observed[rule]++
		}
	}
	ruleSet := appendUnique(nil, keys(claimed), keys(observed))
	sort.Strings(ruleSet)
	for _, rule := range ruleSet {
		res.PerRule = append(res.PerRule, ruleMetrics{
			Rule: rule, Claimed: claimed[rule], Observed: observed[rule],
			Precision: "n/a", Recall: "n/a",
		})
	}
	return res
}

func keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func appendUnique(dst []string, lists ...[]string) []string {
	for _, list := range lists {
		for _, v := range list {
			if !contains(dst, v) {
				dst = append(dst, v)
			}
		}
	}
	return dst
}

// applyVerdicts folds the gate's labels into the metrics. A row's verdict
// contributes to every rule the row carries (§7.3: to split a call, the
// labeler splits the row). Final verdicts: TP/FP on finding rows,
// FN/CLAIM-REJECTED on fn-candidate rows. Rows without a label, or marked
// UNCLEAR, are unresolved; DISPUTED rows are additionally listed as
// disputes. CLAIM-REJECTED removes a claim from the audit universe (no
// denominator impact).
func applyVerdicts(res *results, rows, fnRows []row, labeled map[string]labeledRow) {
	idRules := map[string][]string{}
	idKind := map[string]string{}
	for _, r := range append(append([]row{}, rows...), fnRows...) {
		idRules[r.ID] = r.Rules
		idKind[r.ID] = r.Kind
	}

	metrics := map[string]*ruleMetrics{}
	get := func(rule string) *ruleMetrics {
		if m, ok := metrics[rule]; ok {
			return m
		}
		m := &ruleMetrics{Rule: rule, Precision: "n/a", Recall: "n/a"}
		metrics[rule] = m
		return m
	}
	for _, existing := range res.PerRule {
		m := get(existing.Rule)
		m.Claimed, m.Observed = existing.Claimed, existing.Observed
	}

	resolve := func(list []row) {
		for _, r := range list {
			l, ok := labeled[r.ID]
			v := ""
			if ok {
				v = strings.ToUpper(strings.TrimSpace(l.GateVerdict))
			}
			switch v {
			case "":
				res.Unresolved = append(res.Unresolved, r.ID)
				continue
			case "UNCLEAR":
				res.Unresolved = append(res.Unresolved, r.ID)
				continue
			case "DISPUTED":
				res.Unresolved = append(res.Unresolved, r.ID)
				res.Disputes = append(res.Disputes, r.ID)
				continue
			case "CLAIM-REJECTED":
				continue // claim removed from the audit universe
			}
			for _, rule := range idRules[r.ID] {
				m := get(rule)
				switch v {
				case "TP":
					m.TP++
					res.Aggregate.TP++
				case "FP":
					m.FP++
					res.Aggregate.FP++
				case "FN":
					m.FN++
					res.Aggregate.FN++
				}
			}
		}
	}
	resolve(rows)
	resolve(fnRows)

	ruleSet := make([]string, 0, len(metrics))
	for rule := range metrics {
		ruleSet = append(ruleSet, rule)
	}
	sort.Strings(ruleSet)
	res.PerRule = nil
	for _, rule := range ruleSet {
		m := metrics[rule]
		if m.TP+m.FP > 0 {
			m.Precision = pct(100 * float64(m.TP) / float64(m.TP+m.FP))
		}
		if m.TP+m.FN > 0 {
			m.Recall = pct(100 * float64(m.TP) / float64(m.TP+m.FN))
		}
		res.PerRule = append(res.PerRule, *m)
	}
	if res.Aggregate.TP+res.Aggregate.FP > 0 {
		res.Aggregate.Precision = pct(100 * float64(res.Aggregate.TP) / float64(res.Aggregate.TP+res.Aggregate.FP))
		res.Aggregate.FPShare = 100 * float64(res.Aggregate.FP) / float64(res.Aggregate.TP+res.Aggregate.FP)
	}
	if res.Aggregate.TP+res.Aggregate.FN > 0 {
		res.Aggregate.Recall = pct(100 * float64(res.Aggregate.TP) / float64(res.Aggregate.TP+res.Aggregate.FN))
	}
	sort.Strings(res.Unresolved)
	sort.Strings(res.Disputes)
}

func pct(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

func writeJSON(path string, payload any) error {
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(b, '\n'))
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func printSummary(res *results) {
	fmt.Println("per-rule metrics (precision = tp/(tp+fp), recall-audit = tp/(tp+fn)):")
	for _, m := range res.PerRule {
		fmt.Printf("  %-8s claimed=%-2d observed=%-2d tp=%-2d fp=%-2d fn=%-2d precision=%-7s recall=%-7s\n",
			m.Rule, m.Claimed, m.Observed, m.TP, m.FP, m.FN, m.Precision, m.Recall)
	}
	fmt.Printf("AGGREGATE: tp=%d fp=%d fn=%d precision=%s recall=%s fp_share=%.1f%%\n",
		res.Aggregate.TP, res.Aggregate.FP, res.Aggregate.FN,
		res.Aggregate.Precision, res.Aggregate.Recall, res.Aggregate.FPShare)
}
