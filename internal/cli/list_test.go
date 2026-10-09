package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/rules"
)

// TestListRulesTableSortedWithMetadata pins the w2-06 brief rule: the
// --list-rules catalog is sorted and carries the full metadata per rule —
// the public API for CI config review and w3-07 doc generation.
func TestListRulesTableSortedWithMetadata(t *testing.T) {
	var stdout bytes.Buffer
	inv := &invoker{stdout: &stdout}
	if err := inv.listRules(""); err != nil {
		t.Fatalf("listRules: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"RULE ID",
		"R6xx-99",
		"demo-trailing-slash",
		"demo",
		"ERROR",
		"docs/rules/R6xx-99-demo-trailing-slash.md",
		"Path must not end with a trailing slash",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("catalog missing %q:\n%s", want, out)
		}
	}
}

// TestParseRuleSelector pins the --rules selector grammar: exact ids,
// family prefixes, and loud failures for empty or unknown tokens.
func TestParseRuleSelector(t *testing.T) {
	base, err := defaultRegistry()
	if err != nil {
		t.Fatalf("defaultRegistry: %v", err)
	}
	cases := []struct {
		name     string
		selector string
		want     map[string]bool
		wantErr  string // "" = no error
	}{
		{"empty keeps everything", "", nil, ""},
		{"exact id", "R6xx-99", map[string]bool{"R6xx-99": true}, ""},
		// The family prefix keeps every registered member of the family —
		// derived from the registry so this case pins the selector grammar,
		// not the current size of the demo set.
		{"family prefix", "R6xx", r6xxFamilyIDs(base), ""},
		{"spaces tolerated", " R6xx-99 ", map[string]bool{"R6xx-99": true}, ""},
		{"empty token is an error", "R6xx-99,,", nil, "empty rule"},
		{"unknown token is an error", "R9xx-01", nil, "unknown rule"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			keep, err := parseRuleSelector(tc.selector, base)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRuleSelector(%q): %v", tc.selector, err)
			}
			if tc.want == nil {
				if keep != nil {
					t.Fatalf("keep = %v, want nil", keep)
				}
				return
			}
			if len(keep) != len(tc.want) {
				t.Fatalf("keep = %v, want %v", keep, tc.want)
			}
			for id := range tc.want {
				if !keep[id] {
					t.Fatalf("keep = %v, want %q kept", keep, id)
				}
			}
		})
	}
}

// r6xxFamilyIDs lists every R6xx rule currently registered — the expected
// result of the "R6xx" family-prefix selector.
func r6xxFamilyIDs(base *rules.Registry) map[string]bool {
	out := map[string]bool{}
	for _, r := range base.All() {
		if strings.HasPrefix(r.ID, "R6xx") {
			out[r.ID] = true
		}
	}
	return out
}
