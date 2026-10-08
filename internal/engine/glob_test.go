package engine

import "testing"

// TestMatchGlob pins the glob dialect used by config include/exclude and
// path suppressions (charter §6.4.1: globs relative to the directory that
// contains the config file). "**" spans whole segments only; "*" and "?"
// and "[...]" stay within one segment (path.Match semantics).
func TestMatchGlob(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		path    string
		want    bool
		wantErr bool
	}{
		{"match-all", "**", "a/b/c.java", true, false},
		{"match-all-single", "**", "a.java", true, false},
		{"doublestar-middle", "**/generated/**", "src/generated/x.java", true, false},
		{"doublestar-no-left", "**/generated/**", "generated/x.java", true, false},
		{"doublestar-no-right", "**/generated/**", "src/generated", true, false},
		{"doublestar-miss", "**/generated/**", "src/gen/x.java", false, false},
		{"dir-prefix", "src/**", "src/main/a.java", true, false},
		{"dir-prefix-self", "src/**", "src", true, false},
		{"dir-prefix-miss", "src/**", "other/a.java", false, false},
		{"star-no-cross-segment", "*.java", "a.java", true, false},
		{"star-no-cross-segment-miss", "*.java", "dir/a.java", false, false},
		{"doublestar-star", "**/*.java", "dir/sub/a.java", true, false},
		{"doublestar-star-root", "**/*.java", "a.java", true, false},
		{"question-mark", "src/?.java", "src/a.java", true, false},
		{"question-mark-miss", "src/?.java", "src/ab.java", false, false},
		{"class", "src/[ab].java", "src/a.java", true, false},
		{"class-miss", "src/[ab].java", "src/c.java", false, false},
		{"exact", "src/a.java", "src/a.java", true, false},
		{"exact-miss", "src/a.java", "src/b.java", false, false},
		{"malformed-class", "src/[ab", "src/a.java", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MatchGlob(tc.pattern, tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("MatchGlob(%q, %q) = %v, want error", tc.pattern, tc.path, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("MatchGlob(%q, %q) unexpected error: %v", tc.pattern, tc.path, err)
			}
			if got != tc.want {
				t.Fatalf("MatchGlob(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
			}
		})
	}
}

// TestValidateGlob checks pattern validation used at config load time so a
// malformed glob fails the config (exit 2) instead of silently matching
// nothing at scan time.
func TestValidateGlob(t *testing.T) {
	valid := []string{"**", "**/generated/**", "src/**", "*.java", "src/?.java", "src/[ab].java", "src/main"}
	for _, p := range valid {
		if err := ValidateGlob(p); err != nil {
			t.Errorf("ValidateGlob(%q) = %v, want nil", p, err)
		}
	}
	invalid := []string{"src/[ab", "a[b]c[d"}
	for _, p := range invalid {
		if err := ValidateGlob(p); err == nil {
			t.Errorf("ValidateGlob(%q) = nil, want error", p)
		}
	}
}
