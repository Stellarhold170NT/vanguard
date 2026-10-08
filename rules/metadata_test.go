package rules

import (
	"strings"
	"testing"
)

// TestParseMetadata pins the metadata data-file schema (charter §3.0
// "Metadata là data"): a rule is a YAML record, not Go code.
func TestParseMetadata(t *testing.T) {
	yaml := `id: R1xx-02
slug: no-verb-path
category: resources
severity: ERROR
summary: Path must not contain a CRUD verb — the HTTP verb carries the action.
docPath: docs/rules/R1xx-02-no-verb-path.md
exampleGood: |
  @DeleteMapping("/drafts/{id}")
exampleBad: |
  @DeleteMapping("/delete")
defaultDisabled: false
options:
  - verbWhitelist
`
	m, err := ParseMetadata([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if m.ID != "R1xx-02" || m.Slug != "no-verb-path" || m.Category != "resources" {
		t.Fatalf("identity fields = %+v", m)
	}
	if m.Severity != "ERROR" || m.Summary == "" || m.DocPath != "docs/rules/R1xx-02-no-verb-path.md" {
		t.Fatalf("descriptor fields = %+v", m)
	}
	if m.ExampleGood == "" || m.ExampleBad == "" {
		t.Fatal("examples missing")
	}
	if m.DefaultDisabled {
		t.Fatal("defaultDisabled should default to false")
	}
	if len(m.Options) != 1 || m.Options[0] != "verbWhitelist" {
		t.Fatalf("Options = %v", m.Options)
	}
}

// TestParseMetadataMinimal: only the required keys are enough; optional
// flags may be omitted.
func TestParseMetadataMinimal(t *testing.T) {
	m, err := ParseMetadata([]byte("id: R6xx-99\nslug: demo\nseverity: WARN\n"))
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if m.ID != "R6xx-99" || m.Slug != "demo" || m.Severity != "WARN" {
		t.Fatalf("m = %+v", m)
	}
}

// TestParseMetadataErrors: malformed metadata must fail with the YAML line
// number so the offending record is fixable at a glance (charter §6.3
// error-message discipline, applied to rule data).
func TestParseMetadataErrors(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		contains string
	}{
		{"unknown-key", "id: R1xx-02\nseveri: WARN\n", "line 2"},
		{"bad-type", "id: R1xx-02\ndefaultDisabled: maybe\n", "line 2"},
		{"not-a-mapping", "- a\n- b\n", "metadata"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMetadata([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("ParseMetadata succeeded, want error containing %q", tc.contains)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("error %q does not mention %q", err.Error(), tc.contains)
			}
		})
	}
}
