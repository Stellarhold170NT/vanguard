package render

import (
	"io"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// jsonSchemaVersion is the "schema" field of the §6.7 wire format. Bump it
// on any shape change (charter §6.7: the field exists so machine readers can
// branch).
const jsonSchemaVersion = 1

// jsonRenderer emits the §6.7 schema-1 document. Findings and diagnostics
// stream element by element, so a 10k-finding report never builds one giant
// string (w2-05 trap list / w4-03).
type jsonRenderer struct{ cfg *config }

// jsonSummary mirrors §6.7 "summary". Struct field order — not map sorting —
// fixes the key order on the wire (deterministic by construction).
type jsonSummary struct {
	Files            int        `json:"files"`
	SkippedFiles     int        `json:"skippedFiles"`
	ParseDiagnostics int        `json:"parseDiagnostics"`
	Findings         jsonCounts `json:"findings"`
	Suppressed       int        `json:"suppressed"`
	DurationMS       int64      `json:"durationMs"`
}

// jsonCounts mirrors §6.7 summary.findings.
type jsonCounts struct {
	Error   int `json:"error"`
	Warning int `json:"warning"`
	Info    int `json:"info"`
}

// jsonFinding mirrors §6.7 one finding. Slug comes from rule metadata and is
// omitted when unknown; suggestion is omitted when empty (nil ≡ absent, the
// w2-02 serialization convention).
type jsonFinding struct {
	RuleID     string          `json:"ruleId"`
	Slug       string          `json:"slug,omitempty"`
	Severity   engine.Severity `json:"severity"`
	Message    string          `json:"message"`
	Suggestion string          `json:"suggestion,omitempty"`
	Location   ir.Location     `json:"location"`
}

func (r jsonRenderer) Render(w io.Writer, report *engine.Report) error {
	report = orEmpty(report)
	ew := &errWriter{w: w}
	errs, warns, infos := countSeverities(report.Findings)

	ew.writeString("{\n")
	ew.printf("  \"vanguard\": %s,\n", jstr(ew, r.cfg.version))
	ew.printf("  \"schema\": %d,\n", jsonSchemaVersion)
	ew.printf("  \"target\": %s,\n", jstr(ew, report.Target))

	ew.writeString("  \"source\": ")
	writeJSONValue(ew, report.Source, 1)
	ew.writeString(",\n")

	ew.writeString("  \"summary\": ")
	writeJSONValue(ew, jsonSummary{
		Files:            report.FilesScanned,
		SkippedFiles:     report.FilesSkipped,
		ParseDiagnostics: len(report.Diagnostics),
		Findings:         jsonCounts{Error: errs, Warning: warns, Info: infos},
		Suppressed:       report.Suppressed,
		DurationMS:       report.DurationMS,
	}, 1)
	ew.writeString(",\n")

	ew.writeString("  \"findings\": ")
	writeJSONFindings(ew, r.cfg, report)
	ew.writeString(",\n")

	ew.writeString("  \"diagnostics\": ")
	writeJSONDiagnostics(ew, report)
	ew.writeString("\n}\n")
	return ew.err
}

// writeJSONFindings streams the findings array in the engine contract order
// (file, line, col, ruleId), re-sorted via sortedFindings so the wire bytes
// do not depend on the caller's slice order. Empty output is [], never null.
func writeJSONFindings(ew *errWriter, cfg *config, report *engine.Report) {
	if len(report.Findings) == 0 {
		ew.writeString("[]")
		return
	}
	ew.writeString("[\n")
	for i, f := range sortedFindings(report) {
		if i > 0 {
			ew.writeString(",\n")
		}
		writeJSONValue(ew, jsonFinding{
			RuleID:     f.RuleID,
			Slug:       cfg.ruleSlug(f.RuleID),
			Severity:   f.Severity,
			Message:    f.Message,
			Suggestion: f.Suggestion,
			Location:   f.Location,
		}, 2)
	}
	ew.writeString("\n  ]")
}

// writeJSONDiagnostics streams the parse diagnostics (§5.3 best-effort
// contract). Empty output is [], never null.
func writeJSONDiagnostics(ew *errWriter, report *engine.Report) {
	if len(report.Diagnostics) == 0 {
		ew.writeString("[]")
		return
	}
	ew.writeString("[\n")
	for i, d := range report.Diagnostics {
		if i > 0 {
			ew.writeString(",\n")
		}
		writeJSONValue(ew, d, 2)
	}
	ew.writeString("\n  ]")
}
