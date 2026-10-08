package golden

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// -update rewrites the golden snapshots instead of comparing — the
// intentional-change path behind `make golden-update` (test strategy
// §2.1: the diff of a rewritten snapshot must appear in review).
func init() {
	flag.BoolVar(&updateFlag, "update", false, "rewrite the golden snapshot files")
}

// minCases is the W2 fixture-set floor (w2-07 brief deliverable 2 and test
// strategy §2.1: ≥ 10 cases covering the engine surface).
const minCases = 10

// TestGoldenCases runs every fixture case under testdata/golden through the
// real binary (exec, the CI invocation shape) and pins each requested
// format's bytes plus the exit code against the committed snapshots.
func TestGoldenCases(t *testing.T) {
	bin, err := BinaryPath()
	if err != nil {
		t.Fatalf("resolve fixture binary: %v", err)
	}
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	cases, err := LoadCases(filepath.Join(root, "testdata", "golden"))
	if err != nil {
		t.Fatalf("load golden cases: %v", err)
	}
	if len(cases) < minCases {
		t.Fatalf("golden: %d fixture cases found, want >= %d — the W2 set must cover the engine surface", len(cases), minCases)
	}
	for _, c := range cases {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			stderrDone := false
			for _, format := range c.Formats {
				res, err := c.Run(bin, root, format)
				if err != nil {
					t.Fatalf("run %s --format %s: %v", c.ID, format, err)
				}
				if res.Exit != c.Exit {
					t.Fatalf("case %s --format %s: exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s",
						c.ID, format, res.Exit, c.Exit, res.Stdout, res.Stderr)
				}
				if err := CheckStream(c, root, format, res, &stderrDone); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// TestMultiFormatConsistency cross-checks the machine-readable snapshots of
// the multi-format-consistency case: one scan rendered to json and sarif
// must carry the SAME finding identity set (rule, location, message) —
// charter §6.6 pins the pretty/SARIF order to the engine sort, and §6.7/§6.8
// pin the wire shapes; this test pins the three-way agreement (test
// strategy §2.1 case 9).
func TestMultiFormatConsistency(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	caseDir := filepath.Join(root, "testdata", "golden", "multi-format-consistency")

	jsonFindings, err := loadJSONFindings(filepath.Join(caseDir, "snapshot.json"))
	if err != nil {
		t.Fatalf("load snapshot.json: %v", err)
	}
	if len(jsonFindings) == 0 {
		t.Fatal("multi-format-consistency snapshot.json carries no findings — the cross-check needs a mixed finding set")
	}
	sarifFindings, err := loadSARIFFindings(filepath.Join(caseDir, "snapshot.sarif"))
	if err != nil {
		t.Fatalf("load snapshot.sarif: %v", err)
	}

	jsonSet := identitySet(jsonFindings)
	sarifSet := identitySet(sarifFindings)
	if len(jsonSet) != len(jsonFindings) {
		t.Fatalf("json findings contain duplicates by (rule,location,message): %v", jsonFindings)
	}
	if !setsEqual(jsonSet, sarifSet) {
		t.Fatalf("json and sarif disagree:\n  json only: %v\n  sarif only: %v",
			setDifference(jsonSet, sarifSet), setDifference(sarifSet, jsonSet))
	}
	// The case must exercise the severity taxonomy, not just one rule.
	sev := map[string]bool{}
	for _, f := range jsonFindings {
		sev[f.severity] = true
	}
	for _, want := range []string{"ERROR", "WARN", "INFO"} {
		if !sev[want] {
			t.Errorf("case must carry a %s finding for the taxonomy cross-check; severities present: %v", want, sortedKeys(sev))
		}
	}
}

// finding is the cross-format identity of one finding: the fields both wire
// formats carry (charter §6.7/§6.8).
type finding struct {
	rule      string
	severity  string
	location  string
	message   string
}

func loadJSONFindings(path string) ([]finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Findings []struct {
			RuleID   string `json:"ruleId"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Location struct {
				File   string `json:"file"`
				Line   int    `json:"line"`
				Column int    `json:"column"`
			} `json:"location"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse json snapshot: %w", err)
	}
	out := make([]finding, 0, len(doc.Findings))
	for _, f := range doc.Findings {
		out = append(out, finding{
			rule:     f.RuleID,
			severity: f.Severity,
			message:  f.Message,
			location: fmt.Sprintf("%s:%d:%d", f.Location.File, f.Location.Line, f.Location.Column),
		})
	}
	return out, nil
}

// loadSARIFFindings reads the §6.8 results the identity check needs.
func loadSARIFFindings(path string) ([]finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Runs []struct {
			Results []struct {
				RuleID  string `json:"ruleId"`
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine   int `json:"startLine"`
							StartColumn int `json:"startColumn"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse sarif snapshot: %w", err)
	}
	var out []finding
	for _, r := range doc.Runs {
		for _, res := range r.Results {
			loc := "?:0:0"
			if len(res.Locations) > 0 {
				p := res.Locations[0].PhysicalLocation
				loc = fmt.Sprintf("%s:%d:%d", p.ArtifactLocation.URI, p.Region.StartLine, p.Region.StartColumn)
			}
			out = append(out, finding{
				rule:     res.RuleID,
				message:  res.Message.Text,
				location: loc,
			})
		}
	}
	return out, nil
}

// identitySet maps findings to their identity; the location+message+rule
// triple must be unique within one scan's findings.
func identitySet(findings []finding) map[finding]bool {
	set := map[finding]bool{}
	for _, f := range findings {
		set[finding{rule: f.rule, location: f.location, message: f.message}] = true
	}
	return set
}

func setsEqual(a, b map[finding]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func setDifference(a, b map[finding]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k.rule+" @ "+k.location)
		}
	}
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
