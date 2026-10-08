package rules

import (
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// demoSurface is the acceptance fixture: one Spring-flavoured controller
// with a trailing-slash GET method — exactly what the demo rule flags.
func demoSurface() *ir.ApiSurface {
	return &ir.ApiSurface{
		Source: ir.Source{Lang: "java", Framework: "spring-boot", FrameworkVersion: "4.0.2"},
		Services: []ir.Service{{
			Name:     "YouthResource",
			BasePath: "/api/v1/youth",
			Location: ir.Location{File: "src/main/java/YouthResource.java", Line: 10, Column: 7},
			Methods: []ir.Method{
				{
					OperationName: "list",
					Verb:          ir.VerbGet,
					Path:          "/api/v1/youth/",
					Location:      ir.Location{File: "src/main/java/YouthResource.java", Line: 12, Column: 10},
				},
				{
					OperationName: "create",
					Verb:          ir.VerbPost,
					Path:          "/api/v1/youth",
					Location:      ir.Location{File: "src/main/java/YouthResource.java", Line: 18, Column: 10},
				},
			},
		}},
	}
}

// TestDemoRuleFromData proves the data-driven registration path (acceptance
// 1, first half): the demo rule's metadata lives in an embedded YAML file,
// not in Go code, and it passes full registry validation.
func TestDemoRuleFromData(t *testing.T) {
	rule, err := DemoRule()
	if err != nil {
		t.Fatalf("DemoRule: %v", err)
	}
	if rule.ID != "R6xx-99" || rule.Slug != "demo-trailing-slash" {
		t.Fatalf("rule identity = %s/%s, want R6xx-99/demo-trailing-slash", rule.ID, rule.Slug)
	}
	if rule.Severity != engine.SeverityError || rule.Summary == "" || rule.DocPath == "" {
		t.Fatalf("rule metadata incomplete: %+v", rule)
	}
	if rule.ExampleGood == "" || rule.ExampleBad == "" {
		t.Fatal("demo rule lacks good/bad examples")
	}
	// Registration re-validates everything a w3-0x rule will face.
	reg := NewRegistry()
	if err := reg.Register(rule); err != nil {
		t.Fatalf("register demo rule: %v", err)
	}
}

// TestDemoScanFindingAtExactLocation is acceptance 1 (second half): rule
// registered via data → engine scan → finding with the exact Location of
// the flagged node.
func TestDemoScanFindingAtExactLocation(t *testing.T) {
	rule, err := DemoRule()
	if err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	if err := reg.Register(rule); err != nil {
		t.Fatal(err)
	}
	report, err := engine.NewLinter(reg).Run(demoSurface(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %+v, want exactly 1", report.Findings)
	}
	f := report.Findings[0]
	want := ir.Location{File: "src/main/java/YouthResource.java", Line: 12, Column: 10}
	if f.Location != want {
		t.Fatalf("Location = %+v, want %+v", f.Location, want)
	}
	if f.RuleID != "R6xx-99" || f.Severity != engine.SeverityError {
		t.Fatalf("stamps = %s/%s, want R6xx-99/ERROR (engine-stamped)", f.RuleID, f.Severity)
	}
	if f.Suggestion != "/api/v1/youth" {
		t.Fatalf("Suggestion = %q, want /api/v1/youth", f.Suggestion)
	}
}

// TestDemoScanConfigDisableAndSuppress is acceptance 2 end to end: config
// disable removes the finding; with the rule re-enabled, an inline
// vanguard:ignore removes it again and leaves a verbose note.
func TestDemoScanConfigDisableAndSuppress(t *testing.T) {
	rule, err := DemoRule()
	if err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	if err := reg.Register(rule); err != nil {
		t.Fatal(err)
	}

	disabled := &engine.Config{Rules: map[string]engine.RuleOverride{
		"R6xx-99": {Disabled: boolPtr(true)},
	}}
	report, err := engine.NewLinter(reg).Run(demoSurface(), disabled)
	if err != nil {
		t.Fatalf("Run disabled: %v", err)
	}
	if len(report.Findings) != 0 || report.Suppressed != 0 {
		t.Fatalf("findings = %+v suppressed = %d, want none (disabled at config level)",
			report.Findings, report.Suppressed)
	}

	// Re-enable, then suppress inline on the finding line.
	enabled := &engine.Config{Rules: map[string]engine.RuleOverride{
		"R6xx-99": {Disabled: boolPtr(false)},
	}}
	linter := engine.NewLinter(reg)
	linter.Comments = engine.CommentIndex{"src/main/java/YouthResource.java": []engine.Comment{
		{Location: ir.Location{File: "src/main/java/YouthResource.java", Line: 12, Column: 60},
			Text: "// vanguard:ignore R6xx-99 demo"},
	}}
	report, err = linter.Run(demoSurface(), enabled)
	if err != nil {
		t.Fatalf("Run suppressed: %v", err)
	}
	if len(report.Findings) != 0 || report.Suppressed != 1 {
		t.Fatalf("findings = %+v suppressed = %d, want 0/1", report.Findings, report.Suppressed)
	}
	if n := report.SuppressedNotes[0]; n.Source != "inline" || n.Reason != "demo" {
		t.Fatalf("note = %+v, want inline/demo", n)
	}
}

func boolPtr(b bool) *bool { return &b }
