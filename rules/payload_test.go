package rules

import (
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// The R4xx family tests (w3-05): payload & DTO schema rules over hand-built
// IR surfaces. R4xx-01's heuristic (the W4 attack surface) is pinned here:
// entity index from the IR, package disambiguation, shadowed names skipped.

// r4xxSurface assembles a surface from entity/DTO type specs and one
// controller exposing them. Types are addressed by simple name; entity
// flags come from the caller (the IR's IsEntity is adapter-derived).
type typeSpec struct {
	name     string
	pkg      string
	entity   bool
	fields   []ir.Field
	jsonName map[int]string // field index → JSONName override
}

func r4xxSurface(method ir.Method, types ...typeSpec) *ir.ApiSurface {
	method.Location = ir.Location{File: "src/YouthResource.java", Line: 30, Column: 10}
	if method.Response.Location == (ir.Location{}) {
		method.Response.Location = method.Location
	}
	surface := &ir.ApiSurface{
		Source: ir.Source{Lang: "java", Framework: "spring-boot"},
		Services: []ir.Service{{
			Name:     "YouthResource",
			BasePath: "/api/v1/youth",
			Location: ir.Location{File: "src/YouthResource.java", Line: 12, Column: 7},
			Methods:  []ir.Method{method},
		}},
	}
	for _, ts := range types {
		t := ir.Type{
			Name:     ts.name,
			Kind:     ir.KindPOJO,
			Package:  ts.pkg,
			IsEntity: ts.entity,
			Location: ir.Location{File: "src/" + ts.name + ".java", Line: 5, Column: 7},
		}
		for i, f := range ts.fields {
			if f.Location == (ir.Location{}) {
				f.Location = ir.Location{File: "src/" + ts.name + ".java", Line: 8 + i, Column: 12}
			}
			f.JSONName = ts.jsonName[i]
			t.Fields = append(t.Fields, f)
		}
		surface.Types = append(surface.Types, t)
	}
	return surface
}

func runPayloadRules(t *testing.T, surface *ir.ApiSurface) []engine.Finding {
	t.Helper()
	rs, err := PayloadRules()
	if err != nil {
		t.Fatalf("PayloadRules: %v", err)
	}
	reg := NewRegistry()
	for _, r := range rs {
		if err := reg.Register(r); err != nil {
			t.Fatalf("register %s: %v", r.ID, err)
		}
	}
	report, err := engine.NewLinter(reg).Run(surface, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return report.Findings
}

func findByRule(findings []engine.Finding, id string) []engine.Finding {
	var out []engine.Finding
	for _, f := range findings {
		if f.RuleID == id {
			out = append(out, f)
		}
	}
	return out
}

// TestR4xxRulesFromData pins the data-driven identity of the family.
func TestR4xxRulesFromData(t *testing.T) {
	rs, err := PayloadRules()
	if err != nil {
		t.Fatalf("PayloadRules: %v", err)
	}
	want := map[string]engine.Severity{
		"R4xx-01": engine.SeverityError,
		"R4xx-02": engine.SeverityWarn,
		"R4xx-03": engine.SeverityInfo,
	}
	if len(rs) != len(want) {
		t.Fatalf("rules = %+v, want %d members", rs, len(want))
	}
	for _, r := range rs {
		if r.Severity != want[r.ID] {
			t.Errorf("%s severity = %s, want %s", r.ID, r.Severity, want[r.ID])
		}
		if r.Category != "payload" {
			t.Errorf("%s category = %q, want payload", r.ID, r.Category)
		}
		if r.Slug == "" || r.DocPath == "" || r.Summary == "" || r.ExampleGood == "" || r.ExampleBad == "" {
			t.Errorf("%s metadata incomplete: %+v", r.ID, r)
		}
	}
}

// TestR4xx01NoEntityInPayload — the ERROR heuristic: a type the scan knows
// as an entity (IsEntity) must not appear as payload or response, at top
// level or inside a generic argument. Same-name DTOs in a different package
// (the FP acceptance) and shadowed names never fire.
func TestR4xx01NoEntityInPayload(t *testing.T) {
	entityPkg := "com.example.youth.domain"
	dtoPkg := "com.example.youth.web"
	customerEntity := typeSpec{name: "Customer", pkg: entityPkg, entity: true, fields: []ir.Field{{Name: "id", Type: ir.TypeRef{Name: "Long"}}}}
	youthEntity := typeSpec{name: "Youth", pkg: entityPkg, entity: true, fields: []ir.Field{{Name: "id", Type: ir.TypeRef{Name: "Long"}}}}
	youthDTO := typeSpec{name: "Youth", pkg: dtoPkg, fields: []ir.Field{{Name: "id", Type: ir.TypeRef{Name: "Long"}}}}

	tests := []struct {
		name     string
		method   ir.Method
		types    []typeSpec
		wantFire bool
	}{
		{
			name: "entity response fires",
			method: ir.Method{OperationName: "current", Verb: ir.VerbGet, Path: "/api/v1/youth/current",
				Response: ir.Response{Type: ir.TypeRef{Name: "Customer", Package: entityPkg}}},
			types:    []typeSpec{customerEntity},
			wantFire: true,
		},
		{
			name: "entity request body fires",
			method: func() ir.Method {
				m := ir.Method{OperationName: "save", Verb: ir.VerbPost, Path: "/api/v1/youth",
					Response: ir.Response{Type: ir.TypeRef{Name: "Customer", Package: entityPkg}}}
				m.Payload = &ir.TypeRef{Name: "Customer", Package: entityPkg}
				return m
			}(),
			types:    []typeSpec{customerEntity},
			wantFire: true,
		},
		{
			name: "entity inside generic argument fires (Page<Customer>)",
			method: ir.Method{OperationName: "page", Verb: ir.VerbGet, Path: "/api/v1/youth",
				Response: ir.Response{Type: ir.TypeRef{Name: "Page<Customer>"}, IsCollection: true}},
			types:    []typeSpec{customerEntity},
			wantFire: true,
		},
		{
			name: "entity inside bare list argument fires (ImportResult<Youth> shape)",
			method: ir.Method{OperationName: "import", Verb: ir.VerbPost, Path: "/api/v1/youth/import",
				Response: ir.Response{Type: ir.TypeRef{Name: "ImportResult<Youth>"}}},
			types:    []typeSpec{youthEntity},
			wantFire: true,
		},
		{
			name: "unresolved ref to a lone entity still fires",
			method: ir.Method{OperationName: "current", Verb: ir.VerbGet, Path: "/api/v1/youth/current",
				Response: ir.Response{Type: ir.TypeRef{Name: "Customer", Package: ""}}},
			types:    []typeSpec{customerEntity},
			wantFire: true,
		},
		{
			name: "same-name DTO in a clear different package does NOT fire (acceptance)",
			method: ir.Method{OperationName: "current", Verb: ir.VerbGet, Path: "/api/v1/youth/current",
				Response: ir.Response{Type: ir.TypeRef{Name: "Youth", Package: dtoPkg}}},
			types:    []typeSpec{youthEntity, youthDTO},
			wantFire: false,
		},
		{
			name: "shadowed name stays silent even when the adapter resolved the entity package",
			method: ir.Method{OperationName: "raw", Verb: ir.VerbGet, Path: "/api/v1/youth/raw",
				Response: ir.Response{Type: ir.TypeRef{Name: "Youth", Package: entityPkg}}},
			types:    []typeSpec{youthEntity, youthDTO},
			wantFire: false,
		},
		{
			name: "shadowed name without package info does NOT fire (anti-FP)",
			method: ir.Method{OperationName: "shadow", Verb: ir.VerbGet, Path: "/api/v1/youth/shadow",
				Response: ir.Response{Type: ir.TypeRef{Name: "Youth", Package: ""}}},
			types:    []typeSpec{youthEntity, youthDTO},
			wantFire: false,
		},
		{
			name: "type outside the scan scope is not an entity signal",
			method: ir.Method{OperationName: "external", Verb: ir.VerbGet, Path: "/api/v1/youth/external",
				Response: ir.Response{Type: ir.TypeRef{Name: "ExternalDto", Package: "com.thirdparty.api"}}},
			types:    nil,
			wantFire: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := findByRule(runPayloadRules(t, r4xxSurface(tc.method, tc.types...)), "R4xx-01")
			if tc.wantFire {
				if len(findings) != 1 {
					t.Fatalf("R4xx-01 findings = %+v, want exactly 1", findings)
				}
				f := findings[0]
				if f.Severity != engine.SeverityError {
					t.Errorf("severity = %s, want ERROR (engine-stamped)", f.Severity)
				}
				if !strings.Contains(f.Message, "entity") {
					t.Errorf("message %q should name the entity exposure (reason first)", f.Message)
				}
				if !strings.Contains(f.Suggestion, "DTO") {
					t.Errorf("suggestion %q should propose the DTO mapping", f.Suggestion)
				}
			} else if len(findings) != 0 {
				t.Fatalf("R4xx-01 must stay silent: %+v", findings)
			}
		})
	}
}

// TestR4xx02FieldCasing: the effective JSON name (JSONName when resolved,
// Name otherwise) must be lowerCamelCase on non-entity types.
func TestR4xx02FieldCasing(t *testing.T) {
	dto := typeSpec{name: "AuditDto", pkg: "com.example.dto", fields: []ir.Field{
		{Name: "created_by", Type: ir.TypeRef{Name: "String"}},
		{Name: "authorName", Type: ir.TypeRef{Name: "String"}, JSONName: "author_name"},
		{Name: "CreatedBy", Type: ir.TypeRef{Name: "String"}},
		{Name: "addressLine2", Type: ir.TypeRef{Name: "String"}},
	}}
	jsonName := map[int]string{1: "author_name"}
	m := ir.Method{OperationName: "audit", Verb: ir.VerbGet, Path: "/api/v1/audit",
		Response: ir.Response{Type: ir.TypeRef{Name: "AuditDto", Package: "com.example.dto"}}}
	findings := findByRule(runPayloadRules(t, r4xxSurface(m, withEntry(dto, jsonName))), "R4xx-02")

	if len(findings) != 3 {
		t.Fatalf("R4xx-02 findings = %+v, want exactly 3 (created_by + author_name + CreatedBy)", findings)
	}
	sawSnake, sawJSON, sawPascal := false, false, false
	for _, f := range findings {
		if !strings.Contains(f.Message, "lowerCamelCase") {
			t.Errorf("message %q should name the convention (reason first)", f.Message)
		}
		if f.Suggestion == "" {
			t.Errorf("finding %+v lacks replacement text", f)
		}
		if strings.Contains(f.Message, `"created_by"`) {
			sawSnake = true
			if f.Suggestion != "createdBy" {
				t.Errorf("suggestion = %q, want createdBy", f.Suggestion)
			}
		}
		if strings.Contains(f.Message, `"author_name"`) {
			sawJSON = true
			if f.Suggestion != "authorName" {
				t.Errorf("JSON-name case suggestion = %q, want authorName", f.Suggestion)
			}
		}
		if strings.Contains(f.Message, `"CreatedBy"`) {
			sawPascal = true
			if f.Suggestion != "createdBy" {
				t.Errorf("PascalCase suggestion = %q, want createdBy", f.Suggestion)
			}
		}
	}
	if !sawSnake || !sawJSON || !sawPascal {
		t.Errorf("expected findings for the source name, the resolved JSON name and the PascalCase head: %+v", findings)
	}
}

// TestR4xx02SkipsEntities: entity field casing is a persistence concern,
// not a payload concern — R4xx-02 reads DTO types only.
func TestR4xx02SkipsEntities(t *testing.T) {
	entity := typeSpec{name: "Customer", pkg: "com.example.domain", entity: true, fields: []ir.Field{
		{Name: "full_name", Type: ir.TypeRef{Name: "String"}},
	}}
	m := ir.Method{OperationName: "current", Verb: ir.VerbGet, Path: "/api/v1/customers/current",
		Response: ir.Response{Type: ir.TypeRef{Name: "Customer", Package: "com.example.domain"}}}
	findings := findByRule(runPayloadRules(t, r4xxSurface(m, entity)), "R4xx-02")
	if len(findings) != 0 {
		t.Fatalf("R4xx-02 fired on an entity field: %+v", findings)
	}
}

// TestR4xx03TimeFieldStandard: time-like names must carry RFC3339-ready
// types; date-only names accept LocalDate; non-time names and unknown
// types stay silent.
func TestR4xx03TimeFieldStandard(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		typeName string
		wantFire bool
	}{
		{"String createdAt fires", "createdAt", "String", true},
		{"Instant createdAt is silent", "createdAt", "Instant", false},
		{"OffsetDateTime createdAt is silent", "createdAt", "OffsetDateTime", false},
		{"legacy Date fires", "validUntil", "Date", true},
		{"zoneless LocalDateTime fires", "updatedAt", "LocalDateTime", true},
		{"epoch long fires", "expiresAt", "Long", true},
		{"date-only LocalDate allowed", "startDate", "LocalDate", false},
		{"String startDate fires", "startDate", "String", true},
		{"non-time name stays silent", "name", "String", false},
		{"substring traps stay silent", "candidate", "String", false},
		{"format stays silent", "format", "String", false},
		{"unknown custom type stays silent", "loggedAt", "Stopwatch", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dto := typeSpec{name: "EventDto", pkg: "com.example.dto", fields: []ir.Field{
				{Name: tc.field, Type: ir.TypeRef{Name: tc.typeName}},
			}}
			m := ir.Method{OperationName: "event", Verb: ir.VerbGet, Path: "/api/v1/events",
				Response: ir.Response{Type: ir.TypeRef{Name: "EventDto", Package: "com.example.dto"}}}
			findings := findByRule(runPayloadRules(t, r4xxSurface(m, dto)), "R4xx-03")
			if tc.wantFire != (len(findings) == 1) {
				t.Fatalf("R4xx-03 on %s %s: findings = %+v, want fired=%v", tc.field, tc.typeName, findings, tc.wantFire)
			}
			if tc.wantFire && (!strings.Contains(findings[0].Suggestion, "Instant") || !strings.Contains(findings[0].Suggestion, "RFC3339")) {
				t.Errorf("suggestion should name Instant/OffsetDateTime + RFC3339: %+v", findings[0])
			}
		})
	}
}

// TestR4xx03SkipsEntities: time standard applies to the wire shape (DTOs);
// entity columns are persistence territory.
func TestR4xx03SkipsEntities(t *testing.T) {
	entity := typeSpec{name: "AuditLog", pkg: "com.example.domain", entity: true, fields: []ir.Field{
		{Name: "createdAt", Type: ir.TypeRef{Name: "String"}},
	}}
	m := ir.Method{OperationName: "log", Verb: ir.VerbGet, Path: "/api/v1/logs",
		Response: ir.Response{Type: ir.TypeRef{Name: "AuditLog", Package: "com.example.domain"}}}
	findings := findByRule(runPayloadRules(t, r4xxSurface(m, entity)), "R4xx-03")
	if len(findings) != 0 {
		t.Fatalf("R4xx-03 fired on an entity field: %+v", findings)
	}
}

// withEntry attaches the JSONName overrides to the typeSpec builder calls.
func withEntry(ts typeSpec, jsonName map[int]string) typeSpec {
	ts.jsonName = jsonName
	return ts
}
