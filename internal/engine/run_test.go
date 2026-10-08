package engine

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// demoTrailSelector matches only Method nodes — the framework-level
// predicate (charter §5.4) spelled over IR concepts, no Java knowledge.
type demoTrailSelector struct{}

func (demoTrailSelector) Matches(node ir.Node) bool {
	_, ok := node.(ir.Method)
	return ok
}

// demoTrailCheck flags a Method whose Path ends with a trailing slash —
// the w2-04 simulated (demo) rule body. It reads ctx.Service to prove the
// engine hands the owning Service to checks (charter §5.4, w2-02 note).
func demoTrailCheck(ctx *LintContext, node ir.Node) []Finding {
	m, ok := node.(ir.Method)
	if !ok {
		return nil
	}
	if !strings.HasSuffix(m.Path, "/") {
		return nil
	}
	return []Finding{{
		Message:    "Path ends with a trailing slash — normalize to one canonical form.",
		Suggestion: strings.TrimSuffix(m.Path, "/"),
		Location:   node.Loc(),
	}}
}

// demoTrailRule assembles the demo rule the way rules/Compile does.
func demoTrailRule() Rule {
	return Rule{
		ID:          "R6xx-99",
		Slug:        "demo-trailing-slash",
		Category:    "demo",
		Severity:    SeverityError,
		Summary:     "Path must not end with a trailing slash.",
		DocPath:     "docs/rules/R6xx-99-demo-trailing-slash.md",
		ExampleGood: "GET /api/v1/youth",
		ExampleBad:  "GET /api/v1/youth/",
		Options:     []string{"allow"},
		Selector:    demoTrailSelector{},
		Check:       demoTrailCheck,
	}
}

// demoSurface is a small Java-flavoured ApiSurface built from the w2-02 IR:
// one controller (line 10) with a trailing-slash GET (line 12) and a clean
// POST (line 18), plus one DTO.
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
		Types: []ir.Type{{
			Name:     "YouthDTO",
			Kind:     ir.KindPOJO,
			Location: ir.Location{File: "src/main/java/YouthDTO.java", Line: 3, Column: 14},
		}},
	}
}

// sortedTestRegistry is the engine-side test registry: it implements
// engine.Registry with deterministic (sorted) iteration. The production
// registry with rule validation lives in the rules package (w3-03).
type sortedTestRegistry struct {
	rules []Rule
}

func (r *sortedTestRegistry) Register(rule Rule) error {
	r.rules = append(r.rules, rule)
	return nil
}

func (r *sortedTestRegistry) ByID(id string) (Rule, bool) {
	for _, rule := range r.rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return Rule{}, false
}

func (r *sortedTestRegistry) All() []Rule {
	out := append([]Rule(nil), r.rules...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ID < out[j-1].ID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func newTestLinter(rules ...Rule) *Linter {
	reg := &sortedTestRegistry{}
	for _, r := range rules {
		if err := reg.Register(r); err != nil {
			panic(err)
		}
	}
	return NewLinter(reg)
}

// TestRunDemoRuleFindingAtExactLocation is acceptance 1: a demo rule
// registered through the data path produces a finding with the exact
// Location of the offending Method node.
func TestRunDemoRuleFindingAtExactLocation(t *testing.T) {
	report, err := newTestLinter(demoTrailRule()).Run(demoSurface(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %+v, want exactly 1", report.Findings)
	}
	f := report.Findings[0]
	wantLoc := ir.Location{File: "src/main/java/YouthResource.java", Line: 12, Column: 10}
	if f.Location != wantLoc {
		t.Fatalf("Location = %+v, want %+v", f.Location, wantLoc)
	}
	if f.RuleID != "R6xx-99" || f.Severity != SeverityError {
		t.Fatalf("stamps = %s/%s, want R6xx-99/ERROR", f.RuleID, f.Severity)
	}
	if f.Suggestion != "/api/v1/youth" {
		t.Fatalf("Suggestion = %q, want /api/v1/youth", f.Suggestion)
	}
	if report.Source != (ir.Source{Lang: "java", Framework: "spring-boot", FrameworkVersion: "4.0.2"}) {
		t.Fatalf("Source = %+v", report.Source)
	}
}

// TestRunDisableViaConfig is acceptance 2 (first half): disabling the rule
// through .vanguard.yaml makes the finding disappear.
func TestRunDisableViaConfig(t *testing.T) {
	cfg := &Config{Rules: map[string]RuleOverride{
		"R6xx-99": {Disabled: boolPtr(true)},
	}}
	report, err := newTestLinter(demoTrailRule()).Run(demoSurface(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Findings) != 0 || report.Suppressed != 0 {
		t.Fatalf("findings = %+v suppressed = %d, want none (rule disabled, not suppressed)",
			report.Findings, report.Suppressed)
	}
}

// TestRunSuppressedInline is acceptance 2 (second half): an inline
// vanguard:ignore comment removes the finding and leaves a verbose note.
func TestRunSuppressedInline(t *testing.T) {
	linter := newTestLinter(demoTrailRule())
	linter.Comments = CommentIndex{"src/main/java/YouthResource.java": []Comment{
		commentAt("src/main/java/YouthResource.java", 12, "// vanguard:ignore R6xx-99 legacy route"),
	}}
	report, err := linter.Run(demoSurface(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %+v, want none (suppressed)", report.Findings)
	}
	if report.Suppressed != 1 || len(report.SuppressedNotes) != 1 {
		t.Fatalf("Suppressed = %d notes = %+v, want 1 note", report.Suppressed, report.SuppressedNotes)
	}
	note := report.SuppressedNotes[0]
	if note.RuleID != "R6xx-99" || note.Source != "inline" || note.Reason != "legacy route" {
		t.Fatalf("note = %+v, want R6xx-99/inline/legacy route", note)
	}
	if note.Location.Line != 12 {
		t.Fatalf("note line = %d, want 12", note.Location.Line)
	}
}

// TestRunSuppressedByConfigPath: a path-scoped config suppression removes
// the finding and records a config-sourced note; inline wins when both cover.
func TestRunSuppressedByConfigPath(t *testing.T) {
	cfg := &Config{
		Suppressions: []PathSuppression{{
			Rule:   "R6xx-99",
			Paths:  []string{"src/main/java/**"},
			Reason: "tracked separately",
		}},
	}
	report, err := newTestLinter(demoTrailRule()).Run(demoSurface(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Findings) != 0 || report.Suppressed != 1 {
		t.Fatalf("findings = %+v suppressed = %d, want 0/1", report.Findings, report.Suppressed)
	}
	if note := report.SuppressedNotes[0]; note.Source != "config" || note.Reason != "tracked separately" {
		t.Fatalf("note = %+v, want config/tracked separately", note)
	}

	// Inline wins over config (§6.4.2 #3): the note credits inline.
	linter := newTestLinter(demoTrailRule())
	linter.Comments = CommentIndex{"src/main/java/YouthResource.java": []Comment{
		commentAt("src/main/java/YouthResource.java", 12, "// vanguard:ignore R6xx-99 inline reason"),
	}}
	report, err = linter.Run(demoSurface(), cfg)
	if err != nil {
		t.Fatalf("Run with inline+config: %v", err)
	}
	if note := report.SuppressedNotes[0]; note.Source != "inline" || note.Reason != "inline reason" {
		t.Fatalf("note = %+v, want inline/inline reason", note)
	}
}

// TestRunSeverityOverrideAffectsExitCode pins the §6.3 table: only ERROR
// findings drive exit 1; a rule downgraded to WARN by config exits 0.
func TestRunSeverityOverrideAffectsExitCode(t *testing.T) {
	linter := newTestLinter(demoTrailRule())

	errorReport, err := linter.Run(demoSurface(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code := ExitCode(errorReport, nil); code != 1 {
		t.Fatalf("ExitCode(ERROR finding) = %d, want 1", code)
	}

	warnCfg := &Config{Rules: map[string]RuleOverride{"R6xx-99": {Severity: SeverityWarn}}}
	warnReport, err := linter.Run(demoSurface(), warnCfg)
	if err != nil {
		t.Fatalf("Run with WARN override: %v", err)
	}
	if len(warnReport.Findings) != 1 || warnReport.Findings[0].Severity != SeverityWarn {
		t.Fatalf("findings = %+v, want 1 WARN finding", warnReport.Findings)
	}
	if code := ExitCode(warnReport, nil); code != 0 {
		t.Fatalf("ExitCode(WARN finding) = %d, want 0", code)
	}

	infoCfg := &Config{Rules: map[string]RuleOverride{"R6xx-99": {Severity: SeverityInfo}}}
	infoReport, err := linter.Run(demoSurface(), infoCfg)
	if err != nil {
		t.Fatalf("Run with INFO override: %v", err)
	}
	if code := ExitCode(infoReport, nil); code != 0 {
		t.Fatalf("ExitCode(INFO finding) = %d, want 0", code)
	}
}

// TestExitCodeTable pins the remaining §6.3 rows.
func TestExitCodeTable(t *testing.T) {
	if code := ExitCode(nil, errors.New("bad config")); code != 2 {
		t.Fatalf("ExitCode(nil, err) = %d, want 2", code)
	}
	if code := ExitCode(nil, nil); code != 2 {
		t.Fatalf("ExitCode(nil, nil) = %d, want 2 (no report is a tool failure)", code)
	}
	clean := &Report{}
	if code := ExitCode(clean, nil); code != 0 {
		t.Fatalf("ExitCode(clean) = %d, want 0", code)
	}
}

// TestRunSelectorFiltersNodes: a selector that rejects every node yields no
// findings — the selector, not the check, decides applicability.
func TestRunSelectorFiltersNodes(t *testing.T) {
	rule := demoTrailRule()
	rule.Selector = stubSelector{match: false}
	report, err := newTestLinter(rule).Run(demoSurface(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %+v, want none (selector rejects all)", report.Findings)
	}
}

// TestRunPanicIsolation: a panicking rule becomes one diagnostic; the other
// rules keep running (charter §5.4).
func TestRunPanicIsolation(t *testing.T) {
	panicky := demoTrailRule()
	panicky.ID = "R6xx-98"
	panicky.Slug = "demo-panic"
	// Match every node so the panic lands on the first visited node (the
	// Service, line 10): the demo selector would only let the check run at
	// the first Method (line 12), which this assertion does not pin.
	panicky.Selector = allSelector{}
	panicky.Check = func(*LintContext, ir.Node) []Finding {
		panic("boom")
	}
	report, err := newTestLinter(panicky, demoTrailRule()).Run(demoSurface(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The healthy rule still produced its finding on line 12.
	if len(report.Findings) != 1 || report.Findings[0].RuleID != "R6xx-99" {
		t.Fatalf("findings = %+v, want the healthy rule's finding", report.Findings)
	}
	var panicDiags []ir.Diagnostic
	for _, d := range report.Diagnostics {
		if strings.Contains(d.Message, "R6xx-98") {
			panicDiags = append(panicDiags, d)
		}
	}
	if len(panicDiags) != 1 {
		t.Fatalf("panic diagnostics = %+v, want exactly 1 naming R6xx-98", report.Diagnostics)
	}
	if !strings.Contains(panicDiags[0].Message, "boom") || !strings.Contains(panicDiags[0].Message, "panicked") {
		t.Fatalf("panic diagnostic = %q, want reason + 'panicked'", panicDiags[0].Message)
	}
	if panicDiags[0].Location.Line != 10 {
		t.Fatalf("panic diagnostic line = %d, want 10 (the Service node)", panicDiags[0].Location.Line)
	}
}

// TestRunFindingsSorted pins the B-6 order: (file, line, column, ruleId) —
// renderers and golden snapshots rely on it.
func TestRunFindingsSorted(t *testing.T) {
	late := demoTrailRule()
	late.ID = "R6xx-98"
	late.Slug = "demo-late"
	// Same node as the demo rule but a different rule id on the same
	// location: ordering must fall through to the ruleId tiebreak.
	early := demoTrailRule()
	surface := demoSurface()
	surface.Services[0].Methods = surface.Services[0].Methods[:1]

	otherFile := demoTrailRule()
	otherFile.ID = "R1xx-98"
	otherFile.Slug = "demo-other-file"
	otherFile.Check = func(_ *LintContext, node ir.Node) []Finding {
		loc := node.Loc()
		loc.File = "src/main/java/AAAFirst.java"
		return []Finding{{Message: "other file", Location: loc}}
	}

	report, err := newTestLinter(late, early, otherFile).Run(surface, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Findings) != 3 {
		t.Fatalf("findings = %d, want 3", len(report.Findings))
	}
	got := []string{}
	for _, f := range report.Findings {
		got = append(got, fmt.Sprintf("%s:%d:%d:%s", f.Location.File, f.Location.Line, f.Location.Column, f.RuleID))
	}
	want := []string{
		"src/main/java/AAAFirst.java:12:10:R1xx-98",
		"src/main/java/YouthResource.java:12:10:R6xx-98",
		"src/main/java/YouthResource.java:12:10:R6xx-99",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("findings[%d] = %s, want %s (full order %v)", i, got[i], want[i], got)
		}
	}
}

// TestRunContextCarriesServiceAndOptions proves the LintContext contract:
// checks see the owning Service for nodes inside it and the merged rule
// options; nodes outside any Service see a nil Service.
func TestRunContextCarriesServiceAndOptions(t *testing.T) {
	type observation struct {
		kind    string
		service string
		options map[string]any
	}
	var got []observation
	recorder := demoTrailRule()
	recorder.ID = "R6xx-97"
	recorder.Slug = "demo-observer"
	recorder.Selector = allSelector{}
	// Declare the option keys the config below sets — this test pins context
	// carrying, not option validation (TestRunUnknownOptionIsToolError does).
	recorder.Options = []string{"family", "exact"}
	recorder.Check = func(ctx *LintContext, node ir.Node) []Finding {
		obs := observation{kind: kindName(node), options: ctx.Options}
		if ctx.Service != nil {
			obs.service = ctx.Service.Name
		}
		if ctx.Surface == nil {
			t.Error("ctx.Surface is nil")
		}
		got = append(got, obs)
		return nil
	}
	// The walk must cover every node kind exactly once, so the fixture needs
	// the full node inventory: one method (with a param, response and
	// pagination) plus one type with a field. demoSurface alone has neither
	// Param nor Field nodes and carries two Methods, which would repeat
	// kinds and defeat the visit-once check below.
	surface := demoSurface()
	surface.Services[0].Methods = surface.Services[0].Methods[:1]
	method := &surface.Services[0].Methods[0]
	method.Params = []ir.Param{{
		Name:     "id",
		In:       ir.ParamInPath,
		Type:     ir.TypeRef{Name: "long"},
		Location: ir.Location{File: "src/main/java/YouthResource.java", Line: 13, Column: 30},
	}}
	method.Pagination = ir.Pagination{Style: ir.PaginationNone, Location: method.Location}
	surface.Types[0].Fields = []ir.Field{{
		Name:     "fullName",
		Type:     ir.TypeRef{Name: "String"},
		Location: ir.Location{File: "src/main/java/YouthDTO.java", Line: 5, Column: 12},
	}}
	cfg := &Config{Rules: map[string]RuleOverride{
		"R6xx":    {Options: map[string]any{"family": "fam"}},
		"R6xx-97": {Options: map[string]any{"exact": "ex"}},
	}}
	if _, err := newTestLinter(recorder).Run(surface, cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}
	byKind := map[string]observation{}
	for _, o := range got {
		if _, dup := byKind[o.kind]; dup {
			t.Fatalf("kind %s observed twice; selector must see each node once", o.kind)
		}
		byKind[o.kind] = o
	}
	for _, kind := range []string{"Service", "Method", "Param", "Response", "Pagination", "Type", "Field"} {
		if _, ok := byKind[kind]; !ok {
			t.Fatalf("node kind %s never visited (visited: %v)", kind, got)
		}
	}
	if s := byKind["Method"].service; s != "YouthResource" {
		t.Fatalf("Method ctx.Service = %q, want YouthResource", s)
	}
	if s := byKind["Param"].service; s != "YouthResource" {
		t.Fatalf("Param ctx.Service = %q, want YouthResource (method subtree)", s)
	}
	if s := byKind["Type"].service; s != "" {
		t.Fatalf("Type ctx.Service = %q, want empty (outside any service)", s)
	}
	opts := byKind["Method"].options
	if opts["family"] != "fam" || opts["exact"] != "ex" {
		t.Fatalf("options = %v, want merged family+exact", opts)
	}
}

// allSelector matches every IR node — for walk-coverage tests.
type allSelector struct{}

func (allSelector) Matches(ir.Node) bool { return true }

// kindName names the IR node kind of node — the vocabulary walkSurface
// visits, used to pin walk coverage without depending on struct printing.
func kindName(node ir.Node) string {
	switch node.(type) {
	case ir.Service:
		return "Service"
	case ir.Method:
		return "Method"
	case ir.Param:
		return "Param"
	case ir.Response:
		return "Response"
	case ir.Pagination:
		return "Pagination"
	case ir.Type:
		return "Type"
	case ir.Field:
		return "Field"
	case ir.ErrorHandler:
		return "ErrorHandler"
	case ir.GrpcService:
		return "GrpcService"
	default:
		return "Node"
	}
}

// TestRunUnknownOptionIsToolError: an option key the rule metadata does not
// declare is a config misuse — Run must fail (the CLI maps it to exit 2).
func TestRunUnknownOptionIsToolError(t *testing.T) {
	cfg := &Config{Rules: map[string]RuleOverride{
		"R6xx-99": {Options: map[string]any{"bogus": true}},
	}}
	_, err := newTestLinter(demoTrailRule()).Run(demoSurface(), cfg)
	if err == nil || !strings.Contains(err.Error(), "bogus") || !strings.Contains(err.Error(), "R6xx-99") {
		t.Fatalf("err = %v, want unknown-option error naming rule and key", err)
	}
}

// TestRunSurfaceDiagnosticsCopied: diagnostics recorded by the adapter ride
// into the report untouched (charter §5.3 best-effort).
func TestRunSurfaceDiagnosticsCopied(t *testing.T) {
	surface := demoSurface()
	surface.Diagnostics = []ir.Diagnostic{{
		Message:  "could not parse FileX.java",
		Location: ir.Location{File: "FileX.java", Line: 4, Column: 1},
	}}
	report, err := newTestLinter(demoTrailRule()).Run(surface, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Message != "could not parse FileX.java" {
		t.Fatalf("diagnostics = %+v, want the surface diagnostic copied", report.Diagnostics)
	}
}

// TestRunErrorsOnBadInput pins the tool-error path (exit 2 class).
func TestRunErrorsOnBadInput(t *testing.T) {
	if _, err := NewLinter(nil).Run(demoSurface(), nil); err == nil {
		t.Fatal("Run with nil registry succeeded, want error")
	}
	if _, err := newTestLinter(demoTrailRule()).Run(nil, nil); err == nil {
		t.Fatal("Run with nil surface succeeded, want error")
	}
}

// TestRunNilReportFields: the engine fills what it owns (Source, findings,
// suppression counts, duration) and leaves file counts to the CLI layer.
func TestRunNilReportFields(t *testing.T) {
	linter := newTestLinter(demoTrailRule())
	linter.Target = "./demo-repo"
	report, err := linter.Run(demoSurface(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Target != "./demo-repo" {
		t.Fatalf("Target = %q, want ./demo-repo", report.Target)
	}
	if report.DurationMS < 0 {
		t.Fatalf("DurationMS = %d, want >= 0", report.DurationMS)
	}
	if report.FilesScanned != 0 || report.FilesSkipped != 0 {
		t.Fatalf("file counts = %d/%d, want 0 (filled by the CLI layer)",
			report.FilesScanned, report.FilesSkipped)
	}
}
