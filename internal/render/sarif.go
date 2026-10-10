package render

import (
	"io"
	"path/filepath"

	"github.com/vanguard-lint/vanguard/internal/engine"
)

// sarifRenderer emits SARIF 2.1.0 (charter §6.8). Rules and results stream
// element by element; only the small tool/driver block is marshaled whole.
type sarifRenderer struct{ cfg *config }

// The SARIF object model below covers exactly the fields vanguard emits —
// no generic SARIF library, no dependency. Field order is declaration order
// and is part of the locked wire format (golden snapshots).

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

// sarifDriver carries the rules[] catalog of §6.8 — only rules with ≥1
// result; the full catalog belongs to --list-rules (w2-06).
type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name,omitempty"`
	ShortDescription     sarifText       `json:"shortDescription"`
	FullDescription      *sarifText      `json:"fullDescription,omitempty"`
	HelpURI              string          `json:"helpUri,omitempty"`
	DefaultConfiguration *sarifLevelConf `json:"defaultConfiguration,omitempty"`
	Properties           *sarifRuleProps `json:"properties,omitempty"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifLevelConf struct {
	Level string `json:"level"`
}

// sarifRuleProps is a struct (not a map) so the key order on the wire is the
// §6.8 mock order (category, aip) and stays deterministic.
type sarifRuleProps struct {
	Category string `json:"category,omitempty"`
	AIP      string `json:"aip,omitempty"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	RuleIndex           int               `json:"ruleIndex"`
	Level               string            `json:"level"`
	Message             sarifText         `json:"message"`
	Locations           []sarifLocation   `json:"locations"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

// sarifRegion carries the start of the finding span. The engine Report has
// no end location yet, so endLine/endColumn and the snippet are omitted —
// SARIF makes them optional and the §6.8 acceptance only demands a complete
// file:line location.
type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

func (r sarifRenderer) Render(w io.Writer, report *engine.Report) error {
	report = orEmpty(report)
	ew := &errWriter{w: w}

	groups := groupFindings(report)
	rules, ruleIndex := r.ruleCatalog(groups)

	ew.writeString("{\n")
	ew.printf("  \"$schema\": %s,\n", jstr(ew, sarifSchemaURI))
	ew.printf("  \"version\": %s,\n", jstr(ew, sarifVersion))
	ew.writeString("  \"runs\": [\n")
	ew.writeString("    {\n")

	ew.writeString("      \"tool\": ")
	writeJSONValueInline(ew, sarifTool{Driver: sarifDriver{
		Name:           toolName,
		Version:        r.cfg.version,
		InformationURI: toolInformationURI,
		Rules:          rules,
	}}, 3)
	ew.writeString(",\n")

	ew.writeString("      \"results\": ")
	if len(report.Findings) == 0 {
		ew.writeString("[]")
	} else {
		ew.writeString("[\n")
		for i, f := range sortedFindings(report) {
			if i > 0 {
				ew.writeString(",\n")
			}
			writeJSONValue(ew, r.buildResult(f, ruleIndex[f.RuleID]), 4)
		}
		ew.writeString("\n      ]")
	}
	ew.printf(",\n      \"columnKind\": %s\n", jstr(ew, "utf16CodeUnits"))
	ew.writeString("    }\n  ]\n}\n")
	return ew.err
}

// ruleCatalog builds the driver rules[] — one entry per rule that has at
// least one result, in group order (severity desc, rule id asc), matching
// the pretty group order (§6.6). The returned map gives each rule id its
// index for result.ruleIndex. A rule that somehow carries two severities
// (not producible by the engine today) keeps its first group's position.
func (r sarifRenderer) ruleCatalog(groups []ruleGroup) ([]sarifRule, map[string]int) {
	rules := make([]sarifRule, 0, len(groups))
	index := make(map[string]int, len(groups))
	for _, g := range groups {
		if _, seen := index[g.ruleID]; seen {
			continue
		}
		index[g.ruleID] = len(rules)
		rules = append(rules, r.buildRuleDescriptor(g))
	}
	return rules, index
}

// buildRuleDescriptor fills the §6.8 rule entry from RuleInfo. Without
// metadata the entry degrades to the rule id (shortDescription text = id,
// default level = the observed finding severity) — schema-valid and
// deterministic, never fabricated prose.
func (r sarifRenderer) buildRuleDescriptor(g ruleGroup) sarifRule {
	entry := sarifRule{ID: g.ruleID, ShortDescription: sarifText{Text: g.ruleID}}
	entry.DefaultConfiguration = &sarifLevelConf{Level: sarifLevel(g.severity)}
	info, ok := r.cfg.ruleInfo(g.ruleID)
	if !ok {
		return entry
	}
	if info.Slug != "" {
		entry.Name = info.Slug
	}
	if info.Summary != "" {
		entry.ShortDescription = sarifText{Text: info.Summary}
	}
	if info.Description != "" {
		entry.FullDescription = &sarifText{Text: info.Description}
	}
	entry.HelpURI = info.HelpURI
	if level := sarifLevel(info.DefaultSeverity); level != "none" {
		entry.DefaultConfiguration = &sarifLevelConf{Level: level}
	}
	if info.Category != "" || info.AIP != "" {
		entry.Properties = &sarifRuleProps{Category: info.Category, AIP: info.AIP}
	}
	return entry
}

// buildResult maps one finding onto a §6.8 result. The URI is the engine's
// scan-root-relative path with "/" separators (SARIF requires forward
// slashes even on Windows); columns are UTF-16 code units per the
// columnKind declaration (w2-02 decision).
func (r sarifRenderer) buildResult(f engine.Finding, ruleIndex int) sarifResult {
	uri := filepath.ToSlash(f.Location.File)
	return sarifResult{
		RuleID:    f.RuleID,
		RuleIndex: ruleIndex,
		Level:     sarifLevel(f.Severity),
		Message:   sarifText{Text: f.Message},
		Locations: []sarifLocation{{
			PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifact{URI: uri},
				Region:           sarifRegion{StartLine: f.Location.Line, StartColumn: f.Location.Column},
			},
		}},
		PartialFingerprints: map[string]string{
			fingerprintKey: findingFingerprint(f.RuleID, uri, f.Location.Line, f.Location.Column, f.Message),
		},
	}
}
