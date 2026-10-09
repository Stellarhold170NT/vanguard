// sarif-validate checks that SARIF documents are valid SARIF 2.1.0 — the
// GATE CH3 re-validation of vanguard's SARIF output (w4-05 deliverable 2)
// over the adversarial corpus and the military-youth audit sample.
//
// The validator walks the decoded document and reports EVERY violation,
// not just the first: each scanned case must "schema pass" with its own
// reason list, so one broken field cannot hide the rest. Beyond the SARIF
// 2.1.0 required-property checks it enforces the invariants vanguard's own
// emission locks in (charter §6.8, golden snapshots):
//
//   - result URIs are scan-root-relative, forward-slashed, and never
//     absolute or escaping (".." segments) — the SARIF view of the
//     walker's no-escape guarantee;
//   - every result's ruleId resolves to an entry of tool.driver.rules —
//     vanguard builds that catalog from the findings themselves;
//   - result.ruleIndex stays inside the catalog.
//
// Usage:
//
//	sarif-validate FILE [FILE...]
//
// Exit codes (repo §6.3 discipline): 0 every file valid · 1 at least one
// file invalid · 2 usage or read error (nothing validated).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sarif-validate FILE [FILE...]")
		os.Exit(2)
	}
	files := os.Args[1:]
	invalid := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: read error: %v\n", f, err)
			invalid++
			continue
		}
		if errs := Validate(data); len(errs) > 0 {
			fmt.Fprintf(os.Stdout, "%s: INVALID (%d problem(s))\n", f, len(errs))
			for _, e := range errs {
				fmt.Fprintf(os.Stdout, "  - %s\n", e)
			}
			invalid++
			continue
		}
		fmt.Fprintf(os.Stdout, "%s: OK\n", f)
	}
	if invalid > 0 {
		os.Exit(1)
	}
}

// Validate parses data as one SARIF 2.1.0 document and returns every
// structural problem found (empty slice = valid). A parse failure is
// reported as the first problem.
func Validate(data []byte) []string {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return []string{fmt.Sprintf("document is not valid JSON: %v", err)}
	}
	var errs []string

	switch v := doc["version"].(type) {
	case string:
		if v != "2.1.0" {
			errs = append(errs, fmt.Sprintf("version: %q, want \"2.1.0\"", v))
		}
	case nil:
		errs = append(errs, "version: required property is missing")
	default:
		errs = append(errs, fmt.Sprintf("version: must be a string, got %T", doc["version"]))
	}

	runs, ok := doc["runs"].([]any)
	if !ok {
		errs = append(errs, "runs: required property missing or not an array")
		return errs
	}
	if len(runs) == 0 {
		errs = append(errs, "runs: must contain at least one run")
	}
	for i, r := range runs {
		run, ok := r.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Sprintf("runs[%d]: must be an object", i))
			continue
		}
		errs = append(errs, validateRun(run, i)...)
	}
	return errs
}

// validateRun checks one run object. ruleCount is threaded back to the
// caller through the rules catalog built here.
func validateRun(run map[string]any, runIdx int) []string {
	var errs []string
	at := func(field string) string { return fmt.Sprintf("runs[%d].%s", runIdx, field) }

	tool, ok := run["tool"].(map[string]any)
	if !ok {
		errs = append(errs, at("tool")+": required property missing or not an object")
		return errs
	}
	driver, ok := tool["driver"].(map[string]any)
	if !ok {
		errs = append(errs, at("tool.driver")+": required property missing or not an object")
		return errs
	}
	name, ok := driver["name"].(string)
	if !ok || name == "" {
		errs = append(errs, at("tool.driver.name")+": required non-empty string")
	}

	ruleIDs := map[string]bool{}
	ruleCount := 0
	if rawRules, ok := driver["rules"].([]any); ok {
		ruleCount = len(rawRules)
		for ri, rr := range rawRules {
			rule, ok := rr.(map[string]any)
			if !ok {
				errs = append(errs, fmt.Sprintf("%s.rules[%d]: must be an object", at("tool.driver"), ri))
				continue
			}
			id, ok := rule["id"].(string)
			if !ok || id == "" {
				errs = append(errs, fmt.Sprintf("%s.rules[%d].id: required non-empty string", at("tool.driver"), ri))
				continue
			}
			ruleIDs[id] = true
		}
	}

	if ck, ok := run["columnKind"].(string); ok && ck != "utf16CodeUnits" && ck != "unicodeCodePoints" {
		errs = append(errs, fmt.Sprintf("%s: %q is not a SARIF columnKind", at("columnKind"), ck))
	}

	rawResults, ok := run["results"].([]any)
	if !ok {
		// SARIF allows a run without results; vanguard always emits [].
		return errs
	}
	for i, rr := range rawResults {
		result, ok := rr.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Sprintf("runs[%d].results[%d]: must be an object", runIdx, i))
			continue
		}
		errs = append(errs, validateResult(result, fmt.Sprintf("runs[%d].results[%d]", runIdx, i), ruleIDs, ruleCount)...)
	}
	return errs
}

// sarifLevels are the SARIF 2.1.0 result levels.
var sarifLevels = map[string]bool{"none": true, "note": true, "warning": true, "error": true}

func validateResult(result map[string]any, at string, ruleIDs map[string]bool, ruleCount int) []string {
	var errs []string

	ruleID, ok := result["ruleId"].(string)
	if !ok || ruleID == "" {
		errs = append(errs, at+".ruleId: required non-empty string")
	} else if len(ruleIDs) > 0 && !ruleIDs[ruleID] {
		errs = append(errs, fmt.Sprintf("%s.ruleId: %q not present in tool.driver.rules (vanguard emits one catalog entry per firing rule)", at, ruleID))
	}
	if idx, ok := result["ruleIndex"].(float64); ok {
		if idx < 0 || int(idx) >= ruleCount {
			errs = append(errs, fmt.Sprintf("%s.ruleIndex: %d out of range (%d rules)", at, int(idx), ruleCount))
		}
	}
	if level, ok := result["level"].(string); ok && !sarifLevels[level] {
		errs = append(errs, fmt.Sprintf("%s.level: %q is not a SARIF level", at, level))
	}

	msg, ok := result["message"].(map[string]any)
	if !ok {
		errs = append(errs, at+".message: required property missing or not an object")
	} else if txt, ok := msg["text"].(string); !ok || txt == "" {
		errs = append(errs, at+".message.text: required non-empty string")
	}

	locations, ok := result["locations"].([]any)
	if !ok || len(locations) == 0 {
		errs = append(errs, at+".locations: required non-empty array (vanguard always emits one)")
		return errs
	}
	loc, ok := locations[0].(map[string]any)
	if !ok {
		errs = append(errs, at+".locations[0]: must be an object")
		return errs
	}
	phys, ok := loc["physicalLocation"].(map[string]any)
	if !ok {
		errs = append(errs, at+".locations[0].physicalLocation: required property missing or not an object")
		return errs
	}
	artifact, ok := phys["artifactLocation"].(map[string]any)
	if !ok {
		errs = append(errs, at+".locations[0].physicalLocation.artifactLocation: required property missing or not an object")
		return errs
	}
	uri, ok := artifact["uri"].(string)
	if !ok || uri == "" {
		errs = append(errs, at+".locations[0].physicalLocation.artifactLocation.uri: required non-empty string")
	} else {
		errs = append(errs, validateURI(uri, at+".locations[0].physicalLocation.artifactLocation.uri")...)
	}

	if region, ok := phys["region"].(map[string]any); ok {
		if line, ok := region["startLine"].(float64); !ok || line < 1 {
			errs = append(errs, at+".locations[0].physicalLocation.region.startLine: required integer ≥ 1")
		}
		if col, ok := region["startColumn"].(float64); !ok || col < 1 {
			errs = append(errs, at+".locations[0].physicalLocation.region.startColumn: required integer ≥ 1")
		}
	}

	if fps, ok := result["partialFingerprints"].(map[string]any); !ok || len(fps) == 0 {
		errs = append(errs, at+".partialFingerprints: vanguard emits a non-empty fingerprint map (stable GitHub alert identities)")
	}
	return errs
}

// validateURI enforces the vanguard URI contract: forward slashes,
// root-relative, no escaping segments.
func validateURI(uri, at string) []string {
	var errs []string
	if strings.Contains(uri, "\\") {
		errs = append(errs, at+": URI must use forward slashes (SARIF columnKind contract), got backslash")
	}
	if strings.HasPrefix(uri, "/") {
		errs = append(errs, at+": URI must be scan-root-relative, got absolute "+uri)
	}
	for _, seg := range strings.Split(uri, "/") {
		if seg == ".." {
			errs = append(errs, at+": URI escapes the scan root: "+uri)
			break
		}
	}
	return errs
}
