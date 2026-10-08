package ir

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// at is a shorthand for building locations in fixtures.
func at(file string, line, column int) Location {
	return Location{File: file, Line: line, Column: column}
}

// fixtureSurface builds a surface exercising every node kind of the charter
// §5.2 inventory with realistic Spring/gRPC shapes. It must stay
// Validate-clean: the round-trip test relies on the fixture being
// well-formed, and validate_test.go reuses it as its positive case.
func fixtureSurface() ApiSurface {
	return ApiSurface{
		Source: Source{Lang: "java", Framework: "spring-boot", FrameworkVersion: "4.0.2"},
		Services: []Service{{
			Name:     "YouthResource",
			BasePath: "/api/military-youth",
			Annotations: map[string]string{
				"RestController": "@RestController",
				"RequestMapping": "@RequestMapping(\"/api/military-youth\")",
			},
			Location: at("src/main/java/vn/mil/youth/YouthResource.java", 20, 1),
			Methods: []Method{
				{
					OperationName: "autoComplete",
					Verb:          VerbGet,
					Path:          "/api/military-youth/auto-complete",
					Params: []Param{
						{
							Name:     "q",
							In:       ParamInQuery,
							Type:     TypeRef{Name: "String"},
							Location: at("src/main/java/vn/mil/youth/YouthResource.java", 24, 34),
						},
						{
							Name:       "limit",
							In:         ParamInQuery,
							Type:       TypeRef{Name: "Integer"},
							Validation: "@Max(100)",
							Location:   at("src/main/java/vn/mil/youth/YouthResource.java", 25, 34),
						},
					},
					Response: Response{
						Type:       TypeRef{Name: "Page<YouthResponse>", IsCollection: true},
						StatusCode: 200,
						Envelope:   "ApiResponse",
						Location:   at("src/main/java/vn/mil/youth/YouthResource.java", 23, 12),
					},
					Pagination: Pagination{
						Style:          PaginationPageable,
						PageSizeCapped: true,
						Location:       at("src/main/java/vn/mil/youth/YouthResource.java", 26, 30),
					},
					Annotations: map[string]string{
						"ResponseStatus": "@ResponseStatus(HttpStatus.OK)",
					},
					Location: at("src/main/java/vn/mil/youth/YouthResource.java", 23, 5),
				},
				{
					OperationName: "register",
					Verb:          VerbPost,
					Path:          "/api/military-youth/register",
					Params: []Param{
						{
							Name:     "request",
							In:       ParamInBody,
							Type:     TypeRef{Name: "YouthRequest", Package: "vn.mil.youth.dto"},
							Location: at("src/main/java/vn/mil/youth/YouthResource.java", 41, 30),
						},
					},
					Payload: &TypeRef{Name: "YouthRequest", Package: "vn.mil.youth.dto"},
					Response: Response{
						Type:       TypeRef{Name: "YouthResponse"},
						StatusCode: 201,
						Location:   at("src/main/java/vn/mil/youth/YouthResource.java", 40, 12),
					},
					Pagination: Pagination{
						Style:    PaginationNone,
						Location: at("src/main/java/vn/mil/youth/YouthResource.java", 40, 5),
					},
					Location: at("src/main/java/vn/mil/youth/YouthResource.java", 40, 5),
				},
			},
		}},
		Types: []Type{
			{
				Name:    "YouthRequest",
				Kind:    KindPOJO,
				Package: "vn.mil.youth.dto",
				Fields: []Field{{
					Name:     "fullName",
					JSONName: "full_name",
					Type:     TypeRef{Name: "String"},
					Annotations: map[string]string{
						"JsonProperty": "@JsonProperty(\"full_name\")",
					},
					Location: at("src/main/java/vn/mil/youth/dto/YouthRequest.java", 9, 12),
				}},
				Location: at("src/main/java/vn/mil/youth/dto/YouthRequest.java", 7, 1),
			},
			{
				Name:     "YouthEntity",
				Kind:     KindPOJO,
				Package:  "vn.mil.youth.domain",
				IsEntity: true,
				Location: at("src/main/java/vn/mil/youth/domain/YouthEntity.java", 10, 1),
			},
		},
		ErrorScheme: ErrorScheme{Handlers: []ErrorHandler{{
			ExceptionType: "BusinessException",
			ResponseType:  "ErrorResponse",
			StatusCode:    400,
			Location:      at("src/main/java/vn/mil/youth/web/GlobalExceptionHandler.java", 30, 5),
		}}},
		GrpcServices: []GrpcService{{
			Name:     "YouthService",
			Rpcs:     []string{"GetYouth", "ListYouth"},
			Location: at("proto/youth/v1/youth.proto", 12, 1),
		}},
		Diagnostics: []Diagnostic{{
			Message:  "unresolved annotation @Audit on YouthResource.autoComplete",
			Location: at("src/main/java/vn/mil/youth/YouthResource.java", 23, 5),
		}},
	}
}

// TestRoundTripByteIdentical is the serialization acceptance of w2-02:
// marshal → unmarshal → marshal must give byte-identical output, and the
// unmarshalled value must be indistinguishable from the original.
func TestRoundTripByteIdentical(t *testing.T) {
	surface := fixtureSurface()
	first, err := json.Marshal(surface)
	if err != nil {
		t.Fatalf("marshal ApiSurface: %v", err)
	}
	var back ApiSurface
	if err := json.Unmarshal(first, &back); err != nil {
		t.Fatalf("unmarshal ApiSurface: %v", err)
	}
	second, err := json.Marshal(back)
	if err != nil {
		t.Fatalf("re-marshal ApiSurface: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("round-trip is not byte-identical:\nfirst:  %s\nsecond: %s", first, second)
	}
	if !reflect.DeepEqual(surface, back) {
		t.Fatal("round-trip lost data: unmarshalled surface differs from the original")
	}
}

// TestJSONTagsAreExplicitLowerCamel pins the wire format: every field has
// an explicit lowerCamelCase tag, so golden snapshots never depend on Go
// field-name defaults.
func TestJSONTagsAreExplicitLowerCamel(t *testing.T) {
	raw, err := json.Marshal(fixtureSurface())
	if err != nil {
		t.Fatalf("marshal ApiSurface: %v", err)
	}
	for _, tag := range []string{
		`"lang"`, `"framework"`, `"frameworkVersion"`,
		`"services"`, `"basePath"`, `"annotations"`, `"methods"`, `"location"`,
		`"operationName"`, `"verb"`, `"path"`, `"params"`, `"in"`, `"validation"`,
		`"payload"`, `"package"`, `"response"`, `"statusCode"`, `"isCollection"`, `"envelope"`,
		`"pagination"`, `"style"`, `"pageSizeCapped"`,
		`"types"`, `"kind"`, `"isEntity"`, `"fields"`, `"jsonName"`, `"type"`,
		`"errorScheme"`, `"handlers"`, `"exceptionType"`, `"responseType"`,
		`"grpcServices"`, `"rpcs"`, `"diagnostics"`, `"message"`,
		`"file"`, `"line"`, `"column"`,
	} {
		if !strings.Contains(string(raw), tag+":") {
			t.Errorf("marshalled JSON does not contain %s — field tag missing or omitempty surprised us", tag)
		}
	}
	for _, goName := range []string{
		`"OperationName"`, `"BasePath"`, `"PageSizeCapped"`, `"IsCollection"`,
		`"FrameworkVersion"`, `"ExceptionType"`, `"JSONName"`,
	} {
		if strings.Contains(string(raw), goName+":") {
			t.Errorf("marshalled JSON leaks the Go field name %s — explicit tag required", goName)
		}
	}
}

// TestAnnotationMapsSerializeSorted pins the B-6 determinism rule on maps:
// encoding/json must emit annotation keys sorted, whatever order the
// adapter built the map in.
func TestAnnotationMapsSerializeSorted(t *testing.T) {
	surface := fixtureSurface()
	surface.Services[0].Annotations = map[string]string{
		"zulu":  "z",
		"mike":  "m",
		"alpha": "a",
	}
	raw, err := json.Marshal(surface)
	if err != nil {
		t.Fatalf("marshal ApiSurface: %v", err)
	}
	ia := strings.Index(string(raw), `"alpha":"a"`)
	im := strings.Index(string(raw), `"mike":"m"`)
	iz := strings.Index(string(raw), `"zulu":"z"`)
	if ia < 0 || im < 0 || iz < 0 {
		t.Fatalf("expected annotation entries missing from output: %s", raw)
	}
	if !(ia < im && im < iz) {
		t.Fatalf("annotation keys are not serialized in sorted order: %s", raw)
	}
}

// TestNilAndEmptyCollectionsMarshalAlike lets adapters use nil and empty
// slices/maps interchangeably: both must serialize to the same bytes, so
// golden snapshots cannot flake on that choice.
func TestNilAndEmptyCollectionsMarshalAlike(t *testing.T) {
	nilSurface := ApiSurface{Source: Source{Lang: "java"}}
	emptySurface := ApiSurface{
		Source:       Source{Lang: "java"},
		Services:     []Service{},
		Types:        []Type{},
		GrpcServices: []GrpcService{},
		Diagnostics:  []Diagnostic{},
	}
	a, err := json.Marshal(nilSurface)
	if err != nil {
		t.Fatalf("marshal nil surface: %v", err)
	}
	b, err := json.Marshal(emptySurface)
	if err != nil {
		t.Fatalf("marshal empty surface: %v", err)
	}
	if string(a) != string(b) {
		t.Fatalf("nil and empty collections marshal differently:\nnil:   %s\nempty: %s", a, b)
	}
}

// TestUnmarshalPinsWireFormat decodes a hand-written JSON document, so the
// tag names are locked from the decode side too — renaming a tag now fails
// here and in golden diffs, not silently.
func TestUnmarshalPinsWireFormat(t *testing.T) {
	const wire = `{
 "source": {"lang": "java", "framework": "spring-boot", "frameworkVersion": "3.2.0"},
 "services": [{
   "name": "S", "basePath": "/api/s",
   "annotations": {"RestController": "@RestController"},
   "location": {"file": "a/S.java", "line": 10, "column": 1},
   "methods": [{
     "operationName": "create", "verb": "POST", "path": "/api/s/items",
     "params": [{"name": "body", "in": "body", "type": {"name": "Req", "package": "app.dto"},
                 "location": {"file": "a/S.java", "line": 12, "column": 9}}],
     "payload": {"name": "Req", "package": "app.dto"},
     "response": {"type": {"name": "Res"}, "statusCode": 201,
                  "location": {"file": "a/S.java", "line": 11, "column": 12}},
     "pagination": {"style": "params", "pageSizeCapped": true,
                    "location": {"file": "a/S.java", "line": 13, "column": 5}},
     "annotations": {"ResponseStatus": "@ResponseStatus(HttpStatus.CREATED)"},
     "location": {"file": "a/S.java", "line": 11, "column": 5}
   }]
 }],
 "types": [{"name": "Req", "kind": "pojo", "package": "app.dto",
   "fields": [{"name": "id", "jsonName": "maSo", "type": {"name": "String"},
               "location": {"file": "a/Req.java", "line": 5, "column": 12}}],
   "location": {"file": "a/Req.java", "line": 3, "column": 1}}],
 "errorScheme": {"handlers": [{"exceptionType": "BusinessException", "responseType": "ErrorResponse",
   "statusCode": 400, "location": {"file": "a/Advice.java", "line": 9, "column": 5}}]},
 "grpcServices": [{"name": "G", "rpcs": ["DoIt"], "location": {"file": "a/g.proto", "line": 4, "column": 1}}],
 "diagnostics": [{"message": "unresolved import", "location": {"file": "a/S.java", "line": 2, "column": 1}}]
}`
	var s ApiSurface
	if err := json.Unmarshal([]byte(wire), &s); err != nil {
		t.Fatalf("unmarshal wire format: %v", err)
	}
	if s.Source != (Source{Lang: "java", Framework: "spring-boot", FrameworkVersion: "3.2.0"}) {
		t.Errorf("source decoded wrong: %+v", s.Source)
	}
	svc := s.Services[0]
	if svc.Name != "S" || svc.BasePath != "/api/s" || svc.Annotations["RestController"] != "@RestController" {
		t.Errorf("service decoded wrong: %+v", svc)
	}
	m := svc.Methods[0]
	if m.Verb != VerbPost || m.Path != "/api/s/items" {
		t.Errorf("method decoded wrong: %+v", m)
	}
	if m.Payload == nil || m.Payload.Package != "app.dto" {
		t.Errorf("payload decoded wrong: %+v", m.Payload)
	}
	if m.Params[0].In != ParamInBody || m.Params[0].Type.Package != "app.dto" {
		t.Errorf("param decoded wrong: %+v", m.Params[0])
	}
	if m.Response.StatusCode != 201 {
		t.Errorf("response status decoded wrong: %+v", m.Response)
	}
	if m.Pagination.Style != PaginationParams || !m.Pagination.PageSizeCapped {
		t.Errorf("pagination decoded wrong: %+v", m.Pagination)
	}
	ty := s.Types[0]
	if ty.Kind != KindPOJO || ty.Fields[0].JSONName != "maSo" {
		t.Errorf("type decoded wrong: %+v", ty)
	}
	if h := s.ErrorScheme.Handlers[0]; h.ExceptionType != "BusinessException" || h.StatusCode != 400 {
		t.Errorf("error handler decoded wrong: %+v", h)
	}
	if g := s.GrpcServices[0]; g.Name != "G" || len(g.Rpcs) != 1 || g.Rpcs[0] != "DoIt" {
		t.Errorf("grpc service decoded wrong: %+v", g)
	}
	if d := s.Diagnostics[0]; d.Message != "unresolved import" {
		t.Errorf("diagnostic decoded wrong: %+v", d)
	}
}
