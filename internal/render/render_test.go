package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/vanguard-lint/vanguard/internal/engine"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

// ---------------------------------------------------------------------------
// Fixtures — one deterministic sample finding set shared by unit and golden
// tests. The findings are pre-sorted (file, line, col, ruleId) exactly as the
// engine contract guarantees (engine.go §Report), so the fixtures also pin
// the renderer's reliance on that order.
// ---------------------------------------------------------------------------

func sampleReport() *engine.Report {
	return &engine.Report{
		Source: ir.Source{Lang: "java", Framework: "spring-boot", FrameworkVersion: "4.0.2"},
		Target: "./military-youth",
		Findings: []engine.Finding{
			{
				RuleID:   "R4xx-02",
				Severity: engine.SeverityInfo,
				Message:  `Field "createdAt" uses the type String — prefer a temporal type.`,
				Location: ir.Location{File: "src/main/java/com/viettel/tcct/model/Youth.java", Line: 27, Column: 11},
			},
			{
				RuleID:     "R1xx-02",
				Severity:   engine.SeverityWarn,
				Message:    `Path "/update/draft" contains the verb "update" — use PUT/PATCH on the resource path.`,
				Suggestion: "PUT /api/military-youth/movement-reports/{id}",
				Location:   ir.Location{File: "src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java", Line: 134, Column: 24},
			},
			{
				RuleID:     "R2xx-01",
				Severity:   engine.SeverityError,
				Message:    "GET endpoint declares @RequestBody — GET must not have a body.",
				Suggestion: "use @RequestParam for the fields, or switch to POST /auto-complete",
				Location:   ir.Location{File: "src/main/java/com/viettel/tcct/web/rest/OrderResource.java", Line: 58, Column: 30},
			},
			{
				RuleID:     "R2xx-01",
				Severity:   engine.SeverityError,
				Message:    "GET endpoint declares @RequestBody — GET must not have a body.",
				Suggestion: "use @RequestParam for the fields, or switch to POST /auto-complete",
				Location:   ir.Location{File: "src/main/java/com/viettel/tcct/web/rest/YouthResource.java", Line: 113, Column: 44},
			},
		},
		Suppressed:   2,
		Diagnostics:  []ir.Diagnostic{{Message: "trailing tokens ignored after class body", Location: ir.Location{File: "src/main/java/com/viettel/tcct/model/Youth.java", Line: 210, Column: 1}}},
		FilesScanned: 598,
		FilesSkipped: 3,
		DurationMS:   3200,
	}
}

func sampleRules() map[string]RuleInfo {
	return map[string]RuleInfo{
		"R1xx-02": {
			ID:              "R1xx-02",
			Slug:            "no-verb-path",
			Summary:         "No CRUD verb in path",
			Description:     "Resource paths must name the resource, not the operation. Move the verb into the HTTP method or split the endpoint.",
			HelpURI:         "https://github.com/vanguard-lint/vanguard/blob/main/docs/rules/R1xx-02-no-verb-path.md",
			Category:        "resource",
			AIP:             "131",
			DefaultSeverity: engine.SeverityError,
		},
		"R2xx-01": {
			ID:              "R2xx-01",
			Slug:            "get-no-body",
			Summary:         "GET endpoints must not declare @RequestBody",
			Description:     "GET endpoints must not declare @RequestBody parameters. Move the payload to @RequestParam or switch to POST.",
			HelpURI:         "https://github.com/vanguard-lint/vanguard/blob/main/docs/rules/R2xx-01-get-no-body.md",
			Category:        "methods",
			AIP:             "131",
			DefaultSeverity: engine.SeverityError,
		},
		"R4xx-02": {
			ID:              "R4xx-02",
			Slug:            "string-timestamp",
			Summary:         "Timestamp fields must not use String",
			Description:     "A timestamp-shaped field is typed String. Use a dedicated temporal type so serialization and validation apply.",
			HelpURI:         "https://github.com/vanguard-lint/vanguard/blob/main/docs/rules/R4xx-02-string-timestamp.md",
			Category:        "dto",
			AIP:             "121",
			DefaultSeverity: engine.SeverityInfo,
		},
	}
}

func sampleStats() ScanStats {
	return ScanStats{Services: 24, Endpoints: 185, GrpcServices: 11, RulesTotal: 26, RulesActive: 25}
}

// fullOptions returns the option set the CLI (w2-06) is expected to pass.
func fullOptions() []Option {
	return []Option{
		WithVersion("0.1.0"),
		WithRules(sampleRules()),
		WithStats(sampleStats()),
		WithConfigPath("./.vanguard.yaml"),
		WithColor(ColorNever),
	}
}

func renderSample(t *testing.T, format Format, opts ...Option) []byte {
	t.Helper()
	options := append([]Option{}, fullOptions()...)
	options = append(options, opts...)
	r, err := For(format, options...)
	if err != nil {
		t.Fatalf("For(%q): %v", format, err)
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, sampleReport()); err != nil {
		t.Fatalf("Render(%q): %v", format, err)
	}
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// Constructor contract
// ---------------------------------------------------------------------------

// TestRendererContractCompiles pins the interface the CLI (w2-06) consumes.
func TestRendererContractCompiles(t *testing.T) {
	r, err := For(FormatJSON)
	if err != nil {
		t.Fatalf("For(json): %v", err)
	}
	if err := r.Render(io.Discard, &engine.Report{}); err != nil {
		t.Fatalf("render empty report: %v", err)
	}
}

// TestForUnknownFormatFails keeps unknown --format values loud (§6.3 exit 2
// territory) instead of silently falling back to a default.
func TestForUnknownFormatFails(t *testing.T) {
	if _, err := For(Format("xml")); !errors.Is(err, errUnknownFormat) {
		t.Fatalf("For(xml) error = %v, want errUnknownFormat", err)
	}
}

// TestForReturnsRendererPerFormat replaces the w2-01 stub behaviour: all
// three charter formats must now render.
func TestForReturnsRendererPerFormat(t *testing.T) {
	for _, format := range []Format{FormatPretty, FormatJSON, FormatSARIF} {
		r, err := For(format)
		if err != nil {
			t.Fatalf("For(%q): %v", format, err)
		}
		if r == nil {
			t.Fatalf("For(%q) returned nil renderer", format)
		}
	}
}

// TestFormatConstantsAreDistinct guards the §6.2 flag values.
func TestFormatConstantsAreDistinct(t *testing.T) {
	formats := []Format{FormatPretty, FormatJSON, FormatSARIF}
	seen := make(map[Format]bool, len(formats))
	for _, f := range formats {
		if f == "" {
			t.Fatal("format constant must not be empty")
		}
		if seen[f] {
			t.Fatalf("duplicate format %q", f)
		}
		seen[f] = true
	}
}

// TestRenderDeterministicByteIdentical is the w2-05 determinism acceptance:
// rendering the same report twice must produce byte-identical output.
func TestRenderDeterministicByteIdentical(t *testing.T) {
	for _, format := range []Format{FormatPretty, FormatJSON, FormatSARIF} {
		first := renderSample(t, format)
		second := renderSample(t, format)
		if !bytes.Equal(first, second) {
			t.Fatalf("%s: render is not byte-identical across two runs", format)
		}
	}
}

// TestRendererSortsCallerOrder: the engine hands findings pre-sorted
// (engine.Report contract), but the renderers must not depend on it — a
// shuffled input slice must not change one byte of any format ("sort mọi
// iteration", w2-05 deliverable 2).
func TestRendererSortsCallerOrder(t *testing.T) {
	reversed := sampleReport()
	for i, j := 0, len(reversed.Findings)-1; i < j; i, j = i+1, j-1 {
		reversed.Findings[i], reversed.Findings[j] = reversed.Findings[j], reversed.Findings[i]
	}
	for _, format := range []Format{FormatPretty, FormatJSON, FormatSARIF} {
		r, err := For(format, fullOptions()...)
		if err != nil {
			t.Fatalf("For(%q): %v", format, err)
		}
		var buf bytes.Buffer
		if err := r.Render(&buf, reversed); err != nil {
			t.Fatalf("Render(%q): %v", format, err)
		}
		if !bytes.Equal(buf.Bytes(), renderSample(t, format)) {
			t.Fatalf("%s: output depends on the caller's finding order", format)
		}
	}
}

// ---------------------------------------------------------------------------
// Pretty
// ---------------------------------------------------------------------------

// TestPrettyHeaderSegments pins both §6.6 stats-line shapes (w4-03): with a
// real measured duration the line carries "N files in X"; without one (the
// default deterministic output, §5.4) it degrades to "N files" instead of a
// misleading "in 0ms".
func TestPrettyHeaderSegments(t *testing.T) {
	out := string(renderSample(t, FormatPretty))
	wantHeader := " vanguard 0.1.0 · ./military-youth · java · spring-boot 4.0.2 · config: ./.vanguard.yaml"
	wantStats := " 24 services · 185 endpoints · 11 grpc services · 26 rules (25 active) · 598 files in 3.2s"
	if !strings.Contains(out, wantHeader+"\n") {
		t.Fatalf("header line missing:\nwant: %q\ngot:\n%s", wantHeader, out)
	}
	if !strings.Contains(out, wantStats+"\n") {
		t.Fatalf("stats line missing:\nwant: %q\ngot:\n%s", wantStats, out)
	}
}

// TestPrettyOmitsDurationWhenNotMeasured pins the deterministic default
// (test strategy §5.4): no "in 0ms" segment; every other byte stays
// identical to the timed render.
func TestPrettyOmitsDurationWhenNotMeasured(t *testing.T) {
	report := sampleReport()
	report.DurationMS = 0
	r, err := For(FormatPretty, fullOptions()...)
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, report); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "in 0ms") || strings.Contains(out, "files in") {
		t.Fatalf("untimed render must not carry a duration segment:\n%s", out)
	}
	wantStats := " 24 services · 185 endpoints · 11 grpc services · 26 rules (25 active) · 598 files"
	if !strings.Contains(out, wantStats+"\n") {
		t.Fatalf("stats line without duration missing:\nwant: %q\ngot:\n%s", wantStats, out)
	}
}

// TestPrettyGroupsBySeverityThenRule pins the §6.6 group order: severity
// desc (ERROR→WARN→INFO), then rule id asc, regardless of finding order.
func TestPrettyGroupsBySeverityThenRule(t *testing.T) {
	out := string(renderSample(t, FormatPretty))
	iErr := strings.Index(out, "[ERROR] R2xx-01 get-no-body (2)")
	iWarn := strings.Index(out, "[WARN]  R1xx-02 no-verb-path (1)")
	iInfo := strings.Index(out, "[INFO]  R4xx-02 string-timestamp (1)")
	if iErr < 0 || iWarn < 0 || iInfo < 0 {
		t.Fatalf("group headers missing:\n%s", out)
	}
	if !(iErr < iWarn && iWarn < iInfo) {
		t.Fatalf("group order wrong: error@%d warn@%d info@%d", iErr, iWarn, iInfo)
	}
}

// TestPrettyFindingsSortedWithinGroup pins (file, line, col) order inside a
// rule group.
func TestPrettyFindingsSortedWithinGroup(t *testing.T) {
	out := string(renderSample(t, FormatPretty))
	iOrder := strings.Index(out, "OrderResource.java:58:30")
	iYouth := strings.Index(out, "YouthResource.java:113:44")
	if iOrder < 0 || iYouth < 0 || iOrder > iYouth {
		t.Fatalf("findings inside the R2xx-01 group are not sorted by file:\n%s", out)
	}
}

// TestPrettySuppressHintOncePerGroup pins "suppress: 1 lần mỗi nhóm".
func TestPrettySuppressHintOncePerGroup(t *testing.T) {
	out := string(renderSample(t, FormatPretty))
	// Three groups → exactly three suppress hints.
	if got := strings.Count(out, "suppress: // vanguard:ignore "); got != 3 {
		t.Fatalf("suppress hints = %d, want 3 (one per group):\n%s", got, out)
	}
	if got := strings.Count(out, "suppress: // vanguard:ignore R2xx-01"); got != 1 {
		t.Fatalf("R2xx-01 suppress hints = %d, want 1:\n%s", got, out)
	}
}

// TestPrettySuggestOnlyWhenPresent: findings without a suggestion must not
// emit an empty suggest line.
func TestPrettySuggestOnlyWhenPresent(t *testing.T) {
	out := string(renderSample(t, FormatPretty))
	if got := strings.Count(out, "suggest: "); got != 3 {
		t.Fatalf("suggest lines = %d, want 3 (only findings with a suggestion):\n%s", got, out)
	}
	if strings.Contains(out, "suggest: \n") {
		t.Fatalf("empty suggest line emitted:\n%s", out)
	}
}

// TestPrettyFooterAndNextStep pins the footer summary and the "next:" line
// pointing at the densest ERROR group.
func TestPrettyFooterAndNextStep(t *testing.T) {
	out := string(renderSample(t, FormatPretty))
	for _, want := range []string{
		" 4 findings · 2 ERROR · 1 WARN · 1 INFO · 2 suppressed",
		" 4 files with findings · 598 scanned · 3 skipped · 1 parse diagnostics",
		" next: vanguard explain R2xx-01",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Fatalf("footer line missing:\nwant: %q\ngot:\n%s", want, out)
		}
	}
}

// TestPrettyEmptyReport: a clean scan still renders a readable summary and
// no next-step line.
func TestPrettyEmptyReport(t *testing.T) {
	r, err := For(FormatPretty, WithVersion("0.1.0"), WithColor(ColorNever))
	if err != nil {
		t.Fatalf("For(pretty): %v", err)
	}
	var buf bytes.Buffer
	report := &engine.Report{Target: "./empty-repo", Source: ir.Source{Lang: "java", Framework: "spring-boot"}}
	if err := r.Render(&buf, report); err != nil {
		t.Fatalf("render empty: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "next:") {
		t.Fatalf("next step printed for an empty report:\n%s", out)
	}
	if !strings.Contains(out, " 0 findings · 0 ERROR · 0 WARN · 0 INFO · 0 suppressed\n") {
		t.Fatalf("empty footer missing:\n%s", out)
	}
}

// TestPrettyNonTTYPlainPills: without color the pills are bare bracket text
// and no escape byte ever reaches the writer (CI log stays clean).
func TestPrettyNonTTYPlainPills(t *testing.T) {
	out := string(renderSample(t, FormatPretty))
	if strings.Contains(out, "\x1b") {
		t.Fatalf("escape byte in non-colored output:\n%q", out)
	}
	for _, pill := range []string{"[ERROR]", "[WARN]", "[INFO]"} {
		if !strings.Contains(out, pill) {
			t.Fatalf("pill %q missing:\n%s", pill, out)
		}
	}
}

// TestPrettyTTYColoredPills: color mode wraps the severity word in ANSI
// colors (ERROR red, WARN yellow, INFO cyan); brackets are the non-TTY
// substitute for color, so they disappear in colored output.
func TestPrettyTTYColoredPills(t *testing.T) {
	out := string(renderSample(t, FormatPretty, WithColor(ColorAlways)))
	for _, want := range []string{"\x1b[31mERROR\x1b[0m", "\x1b[33mWARN\x1b[0m", "\x1b[36mINFO\x1b[0m"} {
		if !strings.Contains(out, want) {
			t.Fatalf("colored pill %q missing:\n%q", want, out)
		}
	}
	if strings.Contains(out, "[ERROR]") {
		t.Fatalf("bracket pill leaked into colored output:\n%q", out)
	}
}

// TestColorEnabledDecision pins the color matrix: auto = TTY && !NO_COLOR;
// explicit modes win.
func TestColorEnabledDecision(t *testing.T) {
	cases := []struct {
		mode    ColorMode
		tty     bool
		noColor bool
		want    bool
	}{
		{ColorAuto, true, false, true},
		{ColorAuto, true, true, false},
		{ColorAuto, false, false, false},
		{ColorAlways, false, true, true},
		{ColorNever, true, false, false},
	}
	for _, tc := range cases {
		if got := colorEnabled(tc.mode, tc.tty, tc.noColor); got != tc.want {
			t.Fatalf("colorEnabled(%v, tty=%v, noColor=%v) = %v, want %v", tc.mode, tc.tty, tc.noColor, got, tc.want)
		}
	}
}

// TestPrettyAutoColorNeverForNonFileWriters: a bytes.Buffer is not a TTY, so
// auto mode must produce plain text (this is the golden-test path too).
func TestPrettyAutoColorNeverForNonFileWriters(t *testing.T) {
	r, err := For(FormatPretty, WithColor(ColorAuto))
	if err != nil {
		t.Fatalf("For(pretty): %v", err)
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, sampleReport()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf.String(), "\x1b") {
		t.Fatalf("auto color enabled for a non-TTY writer:\n%q", buf.String())
	}
}

// ---------------------------------------------------------------------------
// JSON
// ---------------------------------------------------------------------------

func decodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, data)
	}
	return doc
}

// TestJSONShape pins the §6.7 schema-1 wire format end to end.
func TestJSONShape(t *testing.T) {
	doc := decodeJSON(t, renderSample(t, FormatJSON))
	if doc["vanguard"] != "0.1.0" {
		t.Fatalf("vanguard = %v, want 0.1.0", doc["vanguard"])
	}
	if doc["schema"] != float64(1) {
		t.Fatalf("schema = %v, want 1", doc["schema"])
	}
	if doc["target"] != "./military-youth" {
		t.Fatalf("target = %v", doc["target"])
	}
	source := doc["source"].(map[string]any)
	if source["lang"] != "java" || source["framework"] != "spring-boot" || source["frameworkVersion"] != "4.0.2" {
		t.Fatalf("source = %v", source)
	}
	summary := doc["summary"].(map[string]any)
	if summary["files"] != float64(598) || summary["skippedFiles"] != float64(3) ||
		summary["parseDiagnostics"] != float64(1) || summary["suppressed"] != float64(2) ||
		summary["durationMs"] != float64(3200) {
		t.Fatalf("summary = %v", summary)
	}
	counts := summary["findings"].(map[string]any)
	if counts["error"] != float64(2) || counts["warning"] != float64(1) || counts["info"] != float64(1) {
		t.Fatalf("finding counts = %v", counts)
	}
	findings := doc["findings"].([]any)
	if len(findings) != 4 {
		t.Fatalf("findings = %d, want 4", len(findings))
	}
	first := findings[0].(map[string]any)
	loc := first["location"].(map[string]any)
	if first["ruleId"] != "R4xx-02" || first["severity"] != "INFO" ||
		loc["file"] != "src/main/java/com/viettel/tcct/model/Youth.java" ||
		loc["line"] != float64(27) || loc["column"] != float64(11) {
		t.Fatalf("first finding = %v", first)
	}
	if diags, ok := doc["diagnostics"].([]any); !ok || len(diags) != 1 {
		t.Fatalf("diagnostics = %v, want 1 entry", doc["diagnostics"])
	}
}

// TestJSONFindingsSorted pins (file, line, col, ruleId) order on the wire.
func TestJSONFindingsSorted(t *testing.T) {
	doc := decodeJSON(t, renderSample(t, FormatJSON))
	findings := doc["findings"].([]any)
	want := []string{
		"src/main/java/com/viettel/tcct/model/Youth.java",
		"src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java",
		"src/main/java/com/viettel/tcct/web/rest/OrderResource.java",
		"src/main/java/com/viettel/tcct/web/rest/YouthResource.java",
	}
	for i, f := range findings {
		got := f.(map[string]any)["location"].(map[string]any)["file"]
		if got != want[i] {
			t.Fatalf("findings[%d].location.file = %v, want %s", i, got, want[i])
		}
	}
}

// TestJSONSlugFollowsMetadata: slug comes from the rule metadata option; it
// is omitted when the CLI could not supply metadata (never fabricated).
func TestJSONSlugFollowsMetadata(t *testing.T) {
	with := decodeJSON(t, renderSample(t, FormatJSON))
	first := with["findings"].([]any)[0].(map[string]any)
	if first["slug"] != "string-timestamp" {
		t.Fatalf("slug = %v, want string-timestamp", first["slug"])
	}

	plain, err := For(FormatJSON, WithVersion("0.1.0"), WithColor(ColorNever))
	if err != nil {
		t.Fatalf("For(json): %v", err)
	}
	var buf bytes.Buffer
	if err := plain.Render(&buf, sampleReport()); err != nil {
		t.Fatalf("render: %v", err)
	}
	doc := decodeJSON(t, buf.Bytes())
	for i, f := range doc["findings"].([]any) {
		if _, ok := f.(map[string]any)["slug"]; ok {
			t.Fatalf("findings[%d] carries slug without metadata", i)
		}
	}
}

// TestJSONSuggestionOmittedWhenEmpty: §6.7 shows suggestion as optional.
func TestJSONSuggestionOmittedWhenEmpty(t *testing.T) {
	doc := decodeJSON(t, renderSample(t, FormatJSON))
	first := doc["findings"].([]any)[0].(map[string]any)
	if _, ok := first["suggestion"]; ok {
		t.Fatalf("empty suggestion serialized: %v", first)
	}
	second := doc["findings"].([]any)[1].(map[string]any)
	if second["suggestion"] == "" {
		t.Fatalf("non-empty suggestion lost: %v", second)
	}
}

// TestJSONEmptyArraysNeverNull: nil slices must marshal as [], never null
// (same convention as the IR serialization, w2-02).
func TestJSONEmptyArraysNeverNull(t *testing.T) {
	r, err := For(FormatJSON)
	if err != nil {
		t.Fatalf("For(json): %v", err)
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, &engine.Report{}); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "null") {
		t.Fatalf("null leaked into JSON output:\n%s", out)
	}
	for _, want := range []string{`"findings": []`, `"diagnostics": []`} {
		if !strings.Contains(out, want) {
			t.Fatalf("empty array %s missing:\n%s", want, out)
		}
	}
}

// ---------------------------------------------------------------------------
// SARIF
// ---------------------------------------------------------------------------

var fingerprintPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

func decodeSARIF(t *testing.T, data []byte) map[string]any {
	t.Helper()
	doc := decodeJSON(t, data)
	runs, ok := doc["runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("runs = %v, want exactly 1 run", doc["runs"])
	}
	return doc
}

func sarifRun(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	return doc["runs"].([]any)[0].(map[string]any)
}

// TestSARIFHeader pins version, schema URI and tool driver identity.
func TestSARIFHeader(t *testing.T) {
	doc := decodeSARIF(t, renderSample(t, FormatSARIF))
	if doc["version"] != "2.1.0" {
		t.Fatalf("version = %v, want 2.1.0", doc["version"])
	}
	if doc["$schema"] != "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json" {
		t.Fatalf("$schema = %v", doc["$schema"])
	}
	run := sarifRun(t, doc)
	if run["columnKind"] != "utf16CodeUnits" {
		t.Fatalf("columnKind = %v, want utf16CodeUnits (w2-02 decision)", run["columnKind"])
	}
	driver := run["tool"].(map[string]any)["driver"].(map[string]any)
	if driver["name"] != "vanguard" || driver["version"] != "0.1.0" {
		t.Fatalf("driver = %v", driver)
	}
}

// TestSARIFRulesOnlyWithResults: rules[] lists exactly the rules that have
// ≥1 result, ordered severity desc then rule id (matching pretty groups),
// and every result's ruleIndex points back at the right entry.
func TestSARIFRulesOnlyWithResults(t *testing.T) {
	doc := decodeSARIF(t, renderSample(t, FormatSARIF))
	run := sarifRun(t, doc)
	rules := run["tool"].(map[string]any)["driver"].(map[string]any)["rules"].([]any)
	var ids []string
	for _, r := range rules {
		ids = append(ids, r.(map[string]any)["id"].(string))
	}
	want := []string{"R2xx-01", "R1xx-02", "R4xx-02"}
	if len(ids) != len(want) {
		t.Fatalf("rules = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("rules[%d] = %s, want %s", i, ids[i], want[i])
		}
	}
	results := run["results"].([]any)
	if len(results) != 4 {
		t.Fatalf("results = %d, want 4", len(results))
	}
	for _, res := range results {
		r := res.(map[string]any)
		idx := int(r["ruleIndex"].(float64))
		if idx < 0 || idx >= len(rules) {
			t.Fatalf("ruleIndex %d out of range", idx)
		}
		if rules[idx].(map[string]any)["id"] != r["ruleId"] {
			t.Fatalf("ruleIndex %d = %v, want ruleId %v", idx, rules[idx].(map[string]any)["id"], r["ruleId"])
		}
	}
}

// TestSARIFLevelMapping pins the GitHub severity convention §6.8.
func TestSARIFLevelMapping(t *testing.T) {
	doc := decodeSARIF(t, renderSample(t, FormatSARIF))
	results := sarifRun(t, doc)["results"].([]any)
	levels := map[string]string{}
	for _, res := range results {
		r := res.(map[string]any)
		levels[r["ruleId"].(string)] = r["level"].(string)
	}
	if levels["R2xx-01"] != "error" || levels["R1xx-02"] != "warning" || levels["R4xx-02"] != "note" {
		t.Fatalf("levels = %v, want error/warning/note", levels)
	}
}

// TestSARIFLocations pins file:line:column under physicalLocation and the
// relative, slash-separated URI convention.
func TestSARIFLocations(t *testing.T) {
	doc := decodeSARIF(t, renderSample(t, FormatSARIF))
	results := sarifRun(t, doc)["results"].([]any)
	var sawTarget bool
	for _, res := range results {
		r := res.(map[string]any)
		locs := r["locations"].([]any)
		if len(locs) != 1 {
			t.Fatalf("result %v has %d locations, want 1", r["ruleId"], len(locs))
		}
		phys := locs[0].(map[string]any)["physicalLocation"].(map[string]any)
		uri := phys["artifactLocation"].(map[string]any)["uri"].(string)
		if strings.HasPrefix(uri, "/") || strings.Contains(uri, "\\") {
			t.Fatalf("uri %q is not scan-root-relative with / separators", uri)
		}
		region := phys["region"].(map[string]any)
		if region["startLine"].(float64) <= 0 || region["startColumn"].(float64) <= 0 {
			t.Fatalf("region missing start line/column: %v", region)
		}
		if uri == "src/main/java/com/viettel/tcct/web/rest/YouthResource.java" {
			if region["startLine"] != float64(113) || region["startColumn"] != float64(44) {
				t.Fatalf("YouthResource region = %v, want line 113 column 44", region)
			}
			sawTarget = true
		}
	}
	if !sawTarget {
		t.Fatal("YouthResource result missing")
	}
}

// TestSARIFPartialFingerprints: every result carries vanguardFindingV1, a
// stable 16-hex-digit hash that differs across findings (GitHub dedup).
func TestSARIFPartialFingerprints(t *testing.T) {
	first := renderSample(t, FormatSARIF)
	second := renderSample(t, FormatSARIF)
	if !bytes.Equal(first, second) {
		t.Fatal("SARIF fingerprints differ between two renders of the same report")
	}
	doc := decodeSARIF(t, first)
	results := sarifRun(t, doc)["results"].([]any)
	seen := map[string]bool{}
	for _, res := range results {
		r := res.(map[string]any)
		fp, ok := r["partialFingerprints"].(map[string]any)["vanguardFindingV1"].(string)
		if !ok {
			t.Fatalf("result %v missing vanguardFindingV1", r["ruleId"])
		}
		if !fingerprintPattern.MatchString(fp) {
			t.Fatalf("fingerprint %q is not 16 lowercase hex chars", fp)
		}
		if seen[fp] {
			t.Fatalf("duplicate fingerprint %q across distinct findings", fp)
		}
		seen[fp] = true
	}
	if len(seen) != 4 {
		t.Fatalf("expected 4 distinct fingerprints, got %d", len(seen))
	}
}

// TestSARIFRuleDescriptionsFollowMetadata: metadata fills shortDescription /
// fullDescription / helpUri; without metadata the entry degrades to the rule
// id (schema-valid, deterministic) instead of fabricating text.
func TestSARIFRuleDescriptionsFollowMetadata(t *testing.T) {
	doc := decodeSARIF(t, renderSample(t, FormatSARIF))
	rules := sarifRun(t, doc)["tool"].(map[string]any)["driver"].(map[string]any)["rules"].([]any)
	first := rules[0].(map[string]any)
	if first["shortDescription"].(map[string]any)["text"] != "GET endpoints must not declare @RequestBody" {
		t.Fatalf("shortDescription = %v", first["shortDescription"])
	}
	if _, ok := first["fullDescription"]; !ok {
		t.Fatal("fullDescription missing although metadata provides it")
	}
	if first["helpUri"] == "" || first["properties"].(map[string]any)["category"] != "methods" {
		t.Fatalf("helpUri/properties = %v", first)
	}

	plain, err := For(FormatSARIF, WithVersion("0.1.0"), WithColor(ColorNever))
	if err != nil {
		t.Fatalf("For(sarif): %v", err)
	}
	var buf bytes.Buffer
	if err := plain.Render(&buf, sampleReport()); err != nil {
		t.Fatalf("render: %v", err)
	}
	doc2 := decodeSARIF(t, buf.Bytes())
	rules2 := sarifRun(t, doc2)["tool"].(map[string]any)["driver"].(map[string]any)["rules"].([]any)
	for _, r := range rules2 {
		entry := r.(map[string]any)
		if entry["shortDescription"].(map[string]any)["text"] != entry["id"] {
			t.Fatalf("fallback shortDescription = %v, want the rule id", entry["shortDescription"])
		}
		if _, ok := entry["helpUri"]; ok {
			t.Fatalf("helpUri fabricated without metadata: %v", entry)
		}
	}
}

// TestSARIFEmptyReport: a clean scan still emits a schema-shaped log.
func TestSARIFEmptyReport(t *testing.T) {
	r, err := For(FormatSARIF)
	if err != nil {
		t.Fatalf("For(sarif): %v", err)
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, &engine.Report{}); err != nil {
		t.Fatalf("render: %v", err)
	}
	doc := decodeSARIF(t, buf.Bytes())
	run := sarifRun(t, doc)
	if results, ok := run["results"].([]any); !ok || len(results) != 0 {
		t.Fatalf("results = %v, want []", run["results"])
	}
}
