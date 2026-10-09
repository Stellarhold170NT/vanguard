package main

import (
	"strings"
	"testing"
)

// validDoc is the minimal SARIF document vanguard emits (§6.8 shape): one
// run, tool.driver with a rules catalog, one result with a full location
// and fingerprint. Every mutation test below breaks exactly one property
// of a deep copy of this document and expects one precise problem.
var validDoc = `{
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "vanguard",
          "version": "0.1.0",
          "informationUri": "https://github.com/Stellarhold170NT/vanguard",
          "rules": [
            {
              "id": "R1xx-01",
              "name": "plural-collection",
              "shortDescription": { "text": "Collection paths use plurals" },
              "defaultConfiguration": { "level": "warning" }
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "R1xx-01",
          "ruleIndex": 0,
          "level": "warning",
          "message": { "text": "Collection path /user uses a singular noun" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "src/com/example/UserController.java" },
                "region": { "startLine": 12, "startColumn": 5 }
              }
            }
          ],
          "partialFingerprints": { "vanguardFinding/v1": "abc123" }
        }
      ],
      "columnKind": "utf16CodeUnits"
    }
  ]
}`

func TestValidateAcceptsVanguardShape(t *testing.T) {
	if errs := Validate([]byte(validDoc)); len(errs) != 0 {
		t.Fatalf("valid SARIF rejected: %v", errs)
	}
}

// mutations: (name, json patch as string surgery, expected substring).
// The patches operate on validDoc so each failure is isolated to one field.
func TestValidateRejectsBrokenDocuments(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(string) string
		want   string // substring of the expected first problem
	}{
		{"bad-version", func(s string) string { return strings.Replace(s, `"2.1.0"`, `"2.0.0"`, 1) }, `version: "2.0.0"`},
		{"missing-version", func(s string) string { return strings.Replace(s, `"version": "2.1.0",`, ``, 1) }, "version: required property is missing"},
		{"no-runs", func(s string) string { return strings.Replace(s, `"runs":`, `"runz":`, 1) }, "runs: required property missing"},
		{"empty-runs", func(s string) string { return strings.Replace(s, `"runs": [`, `"runsz": [`, 1) }, "runs: required property missing"},
		{"driver-without-name", func(s string) string { return strings.Replace(s, `"name": "vanguard",`, ``, 1) }, "tool.driver.name"},
		{"rule-without-id", func(s string) string { return strings.Replace(s, `"id": "R1xx-01",`, ``, 1) }, "tool.driver.rules[0].id"},
		{"result-without-ruleid", func(s string) string { return strings.Replace(s, `"ruleId": "R1xx-01",`, ``, 1) }, "results[0].ruleId"},
		{"unknown-ruleid", func(s string) string { return strings.Replace(s, `"ruleId": "R1xx-01"`, `"ruleId": "R9xx-99"`, 1) }, "not present in tool.driver.rules"},
		{"ruleindex-out-of-range", func(s string) string { return strings.Replace(s, `"ruleIndex": 0`, `"ruleIndex": 5`, 1) }, "ruleIndex: 5 out of range"},
		{"bad-level", func(s string) string {
			return strings.Replace(s, `"ruleIndex": 0,`+"\n"+`          "level": "warning",`, `"ruleIndex": 0,`+"\n"+`          "level": "fatal",`, 1)
		}, `level: "fatal" is not a SARIF level`},
		{"message-without-text", func(s string) string {
			return strings.Replace(s, `{ "text": "Collection path /user uses a singular noun" }`, `{}`, 1)
		}, "message"},
		{"empty-message", func(s string) string {
			return strings.Replace(s, `"text": "Collection path /user uses a singular noun"`, `"text": ""`, 1)
		}, "message.text"},
		{"no-locations", func(s string) string { return strings.Replace(s, `"locations": [`, `"locationz": [`, 1) }, "locations: required non-empty array"},
		{"uri-absolute", func(s string) string {
			return strings.Replace(s, `"uri": "src/com/example/UserController.java"`, `"uri": "/abs/UserController.java"`, 1)
		}, "scan-root-relative"},
		{"uri-escape", func(s string) string {
			return strings.Replace(s, `"uri": "src/com/example/UserController.java"`, `"uri": "../outside/UserController.java"`, 1)
		}, "escapes the scan root"},
		{"uri-backslash", func(s string) string {
			return strings.Replace(s, `"uri": "src/com/example/UserController.java"`, `"uri": "src\\UserController.java"`, 1)
		}, "forward slashes"},
		{"startline-zero", func(s string) string { return strings.Replace(s, `"startLine": 12`, `"startLine": 0`, 1) }, "startLine: required integer ≥ 1"},
		{"startcolumn-zero", func(s string) string { return strings.Replace(s, `"startColumn": 5`, `"startColumn": 0`, 1) }, "startColumn: required integer ≥ 1"},
		{"no-fingerprint", func(s string) string {
			return strings.Replace(s, `"partialFingerprints": { "vanguardFinding/v1": "abc123" }`, `"partialFingerprints": {}`, 1)
		}, "partialFingerprints"},
		{"bad-columnkind", func(s string) string {
			return strings.Replace(s, `"columnKind": "utf16CodeUnits"`, `"columnKind": "utf8"`, 1)
		}, "not a SARIF columnKind"},
		{"not-json", func(string) string { return `{ nope` }, "not valid JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := Validate([]byte(tc.mutate(validDoc)))
			if len(errs) == 0 {
				t.Fatalf("broken SARIF accepted (want problem containing %q)", tc.want)
			}
			found := false
			for _, e := range errs {
				if strings.Contains(e, tc.want) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("problems %v do not mention %q", errs, tc.want)
			}
		})
	}
}

// TestValidateReportsAllProblems pins the "report everything" behaviour:
// one document broken in two independent places yields both reasons.
func TestValidateReportsAllProblems(t *testing.T) {
	broken := strings.Replace(validDoc, `"ruleIndex": 0,`+"\n"+`          "level": "warning",`, `"ruleIndex": 0,`+"\n"+`          "level": "fatal",`, 1)
	broken = strings.Replace(broken, `"startLine": 12`, `"startLine": 0`, 1)
	errs := Validate([]byte(broken))
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 problems (bad level + bad startLine), got %v", errs)
	}
}
