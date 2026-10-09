package corpus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The adversarial corpus manifest is the contract between w4-01 (author)
// and w4-02 (mutation harness reads manifest.yaml directly) and between the
// two machine forms. These tests pin the schema, the file discipline and
// the yaml/json agreement so a hand edit cannot silently break either
// reader (test-strategy §3.6/§3.1).

const advRoot = "../../testdata/adversarial"

// yamlEntry mirrors §3.6 manifest.yaml entry fields.
type yamlEntry struct {
	ID           string   `yaml:"id"`
	Rule         *string  `yaml:"rule"`
	File         string   `yaml:"file"`
	Expect       string   `yaml:"expect"`
	Tier         string   `yaml:"tier"`
	Config       string   `yaml:"config"`
	Note         string   `yaml:"note"`
	AlsoExpect   []string `yaml:"also_expect"`
	ExpectedMiss bool     `yaml:"expected_miss"`
	MissReason   string   `yaml:"miss_reason"`
}

// jsonEntry mirrors manifest.json (the corpus-check input).
type jsonEntry struct {
	ID           string   `json:"id"`
	Rule         *string  `json:"rule"`
	File         string   `json:"file"`
	Expect       string   `json:"expect"`
	Tier         string   `json:"tier"`
	Config       string   `json:"config"`
	Also         []string `json:"also"`
	ExpectedMiss bool     `json:"expected_miss"`
	MissReason   string   `json:"miss_reason"`
}

type jsonManifest struct {
	Schema string      `json:"schema"`
	Cases  []jsonEntry `json:"cases"`
}

var (
	ruleIDRe   = regexp.MustCompile(`^R[1-6]xx-[0-9]{2}$`)
	caseFileRe = regexp.MustCompile(`^(vio|ok)-[a-z0-9]+(-[a-z0-9]+)*\.(java|proto)$`)
)

func loadManifests(t *testing.T) ([]yamlEntry, []jsonEntry) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(advRoot, "manifest.yaml"))
	if err != nil {
		t.Fatalf("read manifest.yaml: %v", err)
	}
	var entries []yamlEntry
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("manifest.yaml is not valid YAML: %v", err)
	}
	rawJSON, err := os.ReadFile(filepath.Join(advRoot, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	var man jsonManifest
	if err := json.Unmarshal(rawJSON, &man); err != nil {
		t.Fatalf("manifest.json is not valid JSON: %v", err)
	}
	return entries, man.Cases
}

// TestManifestSchemaPins checks the §3.6 field contract entry by entry.
func TestManifestSchemaPins(t *testing.T) {
	entries, _ := loadManifests(t)
	seenID := map[string]bool{}
	seenFile := map[string]bool{}
	for _, e := range entries {
		if e.ID == "" {
			t.Fatalf("entry without id: %+v", e)
		}
		if seenID[e.ID] {
			t.Errorf("duplicate manifest id %q", e.ID)
		}
		seenID[e.ID] = true
		if seenFile[e.File] {
			t.Errorf("duplicate file %q", e.File)
		}
		seenFile[e.File] = true
		switch {
		case e.Rule == nil: // _global engine-only case, expect: parse-ok
			if e.Expect != "parse-ok" {
				t.Errorf("%s: rule null requires expect parse-ok, got %q", e.ID, e.Expect)
			}
		case !ruleIDRe.MatchString(*e.Rule):
			t.Errorf("%s: rule %q is not an R<F>xx-NN id", e.ID, *e.Rule)
		}
		switch e.Expect {
		case "hit", "miss", "parse-ok":
		default:
			t.Errorf("%s: expect %q (want hit|miss|parse-ok)", e.ID, e.Expect)
		}
		switch e.Tier {
		case "baseline", "near-miss":
		default:
			t.Errorf("%s: tier %q (want baseline|near-miss)", e.ID, e.Tier)
		}
		switch e.Config {
		case "", "default", "enable-r6xx-01":
		default:
			t.Errorf("%s: config %q (want default|enable-r6xx-01)", e.ID, e.Config)
		}
		// R6xx-01 is the charter's opt-in rule: its cases must run on the
		// enable pass, everything else on the default pass.
		if e.Rule != nil && *e.Rule == "R6xx-01" && e.Config != "enable-r6xx-01" {
			t.Errorf("%s: R6xx-01 case must declare config enable-r6xx-01", e.ID)
		}
		if e.Rule != nil && *e.Rule != "R6xx-01" && e.Config == "enable-r6xx-01" {
			t.Errorf("%s: only R6xx-01 cases run on the opt-in pass", e.ID)
		}
		if e.ExpectedMiss && e.MissReason == "" {
			t.Errorf("%s: expected_miss without miss_reason (§3.6)", e.ID)
		}
		if e.ExpectedMiss && e.Expect != "hit" {
			t.Errorf("%s: expected_miss only applies to expect hit", e.ID)
		}
		if e.Expect == "hit" && e.Rule == nil {
			t.Errorf("%s: expect hit with no rule", e.ID)
		}
	}
}

// TestManifestJSONAgrees checks that the machine twin carries the same
// contract — corpus-check reads the JSON, w4-02 reads the YAML.
func TestManifestJSONAgrees(t *testing.T) {
	entries, jentries := loadManifests(t)
	if len(entries) != len(jentries) {
		t.Fatalf("yaml has %d entries, json has %d", len(entries), len(jentries))
	}
	jm := map[string]jsonEntry{}
	for _, j := range jentries {
		jm[j.ID] = j
	}
	for _, e := range entries {
		j, ok := jm[e.ID]
		if !ok {
			t.Errorf("id %q missing from manifest.json", e.ID)
			continue
		}
		if deref(e.Rule) != deref(j.Rule) || e.File != j.File || e.Expect != j.Expect ||
			e.Tier != j.Tier || e.Config != j.Config || e.ExpectedMiss != j.ExpectedMiss {
			t.Errorf("id %q: yaml/json disagree (yaml %+v json %+v)", e.ID, e, j)
		}
		if len(e.AlsoExpect) != len(j.Also) {
			t.Errorf("id %q: also_expect differs (yaml %v json %v)", e.ID, e.AlsoExpect, j.Also)
			continue
		}
		for i := range e.AlsoExpect {
			if e.AlsoExpect[i] != j.Also[i] {
				t.Errorf("id %q: also_expect[%d] differs", e.ID, i)
			}
		}
	}
}

// TestCorpusFileDiscipline pins §3.1: naming, machine header, lure line and
// the ≤ 40-line budget for every case file on disk.
func TestCorpusFileDiscipline(t *testing.T) {
	entries, _ := loadManifests(t)
	if len(entries) < 160 {
		t.Fatalf("per-task allocation: got %d entries, want ≥ 160 (161 per-rule + 10 global)", len(entries))
	}
	perRule := map[string]map[string]int{} // rule -> {vio, ok}
	for _, e := range entries {
		path := filepath.Join(advRoot, e.File)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: case file missing: %v", e.ID, err)
			continue
		}
		base := filepath.Base(e.File)
		if !caseFileRe.MatchString(base) {
			t.Errorf("%s: file %q violates {ok,vio}-<slug>.{java,proto}", e.ID, base)
		}
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		if len(lines) > 40 {
			t.Errorf("%s: %s is %d lines (budget 40, §3.1)", e.ID, base, len(lines))
		}
		dir := strings.SplitN(e.File, "/", 2)[0]
		kind := "vio"
		if strings.HasPrefix(base, "ok-") {
			kind = "ok"
		}
		wantHeader := "// corpus: " + dir + " " + kind + " " +
			strings.TrimSuffix(strings.TrimPrefix(base, kind+"-"), filepath.Ext(base))
		if len(lines) < 2 || lines[0] != wantHeader {
			t.Errorf("%s: first line %q, want %q", e.ID, first(lines), wantHeader)
		}
		if len(lines) < 2 || !strings.HasPrefix(lines[1], "// lure: ") {
			t.Errorf("%s: second line must be the // lure: line", e.ID)
		}
		if dir != "_global" {
			if perRule[dir] == nil {
				perRule[dir] = map[string]int{}
			}
			perRule[dir][kind]++
		}
	}
	// §3.3: every rule ≥ 3 vio + ≥ 2 ok; the pack holds ≥ 150 per-rule cases.
	total := 0
	for rule, n := range perRule {
		if ruleIDRe.MatchString(rule) {
			total += n["vio"] + n["ok"]
		}
		if n["vio"] < 3 || n["ok"] < 2 {
			t.Errorf("%s: %d vio / %d ok (minimum 3/2, §3.3)", rule, n["vio"], n["ok"])
		}
	}
	if total < 150 {
		t.Errorf("per-rule case total %d < 150", total)
	}
}

func first(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

func deref(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}
