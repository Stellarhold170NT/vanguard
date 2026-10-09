package rules

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/discovery"
	"github.com/Stellarhold170NT/vanguard/internal/engine"
)

// The adversarial fixtures under testdata/adversarial/R1xx-0N are the
// corpus seed w4-01 grows (charter §3.8, CONTRIBUTING "Process for adding a
// rule"): every vio-*.java must trip its rule at least once, every
// ok-*.java must stay silent for it, and every fixture must parse cleanly
// (test-strategy §3.1: a parse-broken case is not an adversarial case).
//
// The pipeline is the production one — discovery (spring adapter) → engine
// → R1xx rules — so the test pins the whole path, not the check in
// isolation. Findings are bucketed per (file, rule): other families may
// legitimately fire on a case aimed at one rule.
func TestR1xxAdversarialFixtures(t *testing.T) {
	dirs := []string{"R1xx-01", "R1xx-02", "R1xx-03", "R1xx-04", "R1xx-05"}
	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			root := filepath.Join("..", "testdata", "adversarial", dir)
			res, err := discovery.Scan(root, discovery.ScanOptions{})
			if err != nil {
				t.Fatalf("scan %s: %v", root, err)
			}
			if res.NoAPI != "" {
				t.Fatalf("no API surface in %s: %s", root, res.NoAPI)
			}
			if len(res.Diagnostics) != 0 {
				t.Fatalf("fixtures must parse cleanly (test-strategy §3.1), got diagnostics: %+v", res.Diagnostics)
			}

			reg := NewRegistry()
			rulesSet, err := ResourceRules()
			if err != nil {
				t.Fatalf("ResourceRules: %v", err)
			}
			for _, r := range rulesSet {
				if err := reg.Register(r); err != nil {
					t.Fatalf("register %s: %v", r.ID, err)
				}
			}
			rep, err := engine.NewLinter(reg).Run(res.Surface, nil)
			if err != nil {
				t.Fatalf("lint: %v", err)
			}

			// ruleID → file → count
			byRuleFile := map[string]map[string]int{}
			for _, f := range rep.Findings {
				if byRuleFile[f.RuleID] == nil {
					byRuleFile[f.RuleID] = map[string]int{}
				}
				byRuleFile[f.RuleID][f.Location.File]++
			}

			files, err := filepath.Glob(filepath.Join(root, "*.java"))
			if err != nil || len(files) == 0 {
				t.Fatalf("no fixtures under %s", root)
			}
			for _, file := range files {
				base := filepath.Base(file)
				rel := base // scan root is the rule dir, so findings carry the base name
				switch {
				case strings.HasPrefix(base, "vio-"):
					if byRuleFile[dir][rel] == 0 {
						t.Errorf("vio case %s must trip %s — no finding (FP-side miss / FN)", dir, dir)
					}
				case strings.HasPrefix(base, "ok-"):
					if n := byRuleFile[dir][rel]; n != 0 {
						t.Errorf("ok case %s must stay silent for %s — %d finding(s) (false positive)", base, dir, n)
					}
				default:
					t.Errorf("fixture %s does not follow the {ok,vio}-<slug>.java naming", base)
				}
			}
		})
	}
}

// Acceptance gate: the w3-02 "clean" sample repos produce ZERO R1xx
// findings — the rules must not cry wolf on compliant code (brief w3-03,
// acceptance bullet 1).
func TestR1xxCleanSpringFixturesAreSilent(t *testing.T) {
	for _, root := range []string{
		filepath.Join("..", "testdata", "spring-repo"),
		filepath.Join("..", "testdata", "spring-overlay"),
	} {
		t.Run(filepath.Base(root), func(t *testing.T) {
			res, err := discovery.Scan(root, discovery.ScanOptions{})
			if err != nil {
				t.Fatalf("scan %s: %v", root, err)
			}
			reg := NewRegistry()
			rulesSet, err := ResourceRules()
			if err != nil {
				t.Fatalf("ResourceRules: %v", err)
			}
			for _, r := range rulesSet {
				if err := reg.Register(r); err != nil {
					t.Fatalf("register %s: %v", r.ID, err)
				}
			}
			rep, err := engine.NewLinter(reg).Run(res.Surface, nil)
			if err != nil {
				t.Fatalf("lint: %v", err)
			}
			if len(rep.Findings) != 0 {
				var lines []string
				for _, f := range rep.Findings {
					lines = append(lines, f.RuleID+" "+f.Location.File+":"+strconv.Itoa(f.Location.Line)+" "+f.Message)
				}
				sort.Strings(lines)
				t.Fatalf("R1xx false positives on the clean fixture %s:\n%s", root, strings.Join(lines, "\n"))
			}
		})
	}
}
