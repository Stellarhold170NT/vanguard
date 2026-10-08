package ir

import (
	"encoding/json"
	"strings"
	"testing"
)

// expectDiagnostics asserts that s.Validate() yields exactly one diagnostic
// per wanted substring, in order.
func expectDiagnostics(t *testing.T, s *ApiSurface, want ...string) {
	t.Helper()
	ds := s.Validate()
	if len(ds) != len(want) {
		t.Fatalf("Validate returned %d diagnostics, want %d: %+v", len(ds), len(want), ds)
	}
	for i := range want {
		if !strings.Contains(ds[i].Message, want[i]) {
			t.Fatalf("diagnostic %d = %q, want substring %q", i, ds[i].Message, want[i])
		}
	}
}

// httpMethod builds a minimal valid HTTP method for invariant tests; tests
// break one field at a time starting from this shape.
func httpMethod(name string, loc Location) Method {
	return Method{
		OperationName: name,
		Verb:          VerbGet,
		Path:          "/api/x",
		Response:      Response{Location: loc},
		Pagination:    Pagination{Style: PaginationNone, Location: loc},
		Location:      loc,
	}
}

// serviceWith wraps methods in one valid service.
func serviceWith(methods ...Method) ApiSurface {
	return ApiSurface{
		Services: []Service{{
			Name:     "S",
			Location: at("a.java", 1, 1),
			Methods:  methods,
		}},
	}
}

func TestValidateAcceptsWellFormedSurface(t *testing.T) {
	s := fixtureSurface()
	if ds := s.Validate(); len(ds) != 0 {
		t.Fatalf("well-formed surface must validate clean, got: %+v", ds)
	}
}

func TestValidateNilAndEmptySurface(t *testing.T) {
	var nilSurface *ApiSurface
	if ds := nilSurface.Validate(); len(ds) != 0 {
		t.Fatalf("nil surface must validate to no diagnostics, got %+v", ds)
	}
	if ds := (&ApiSurface{}).Validate(); len(ds) != 0 {
		t.Fatalf("empty surface must validate to no diagnostics, got %+v", ds)
	}
}

func TestValidateDetectsDuplicates(t *testing.T) {
	cases := []struct {
		name    string
		surface ApiSurface
		want    string
	}{
		{
			name: "duplicate service name",
			surface: ApiSurface{Services: []Service{
				{Name: "S", Location: at("a.java", 1, 1)},
				{Name: "S", Location: at("b.java", 2, 2)},
			}},
			want: `duplicate service name "S"`,
		},
		{
			name: "duplicate type in same package",
			surface: ApiSurface{Types: []Type{
				{Name: "D", Kind: KindPOJO, Package: "p", Location: at("a.java", 1, 1)},
				{Name: "D", Kind: KindPOJO, Package: "p", Location: at("b.java", 2, 2)},
			}},
			want: `duplicate type "D"`,
		},
		{
			name: "duplicate method name in service",
			surface: serviceWith(
				httpMethod("get", at("a.java", 2, 5)),
				httpMethod("get", at("a.java", 6, 5)),
			),
			want: `service "S": duplicate method "get"`,
		},
		{
			name: "duplicate parameter name in method",
			surface: serviceWith(Method{
				OperationName: "get",
				Verb:          VerbGet,
				Path:          "/x",
				Params: []Param{
					{Name: "q", In: ParamInQuery, Type: TypeRef{Name: "String"}, Location: at("a.java", 3, 10)},
					{Name: "q", In: ParamInQuery, Type: TypeRef{Name: "String"}, Location: at("a.java", 4, 10)},
				},
				Pagination: Pagination{Style: PaginationNone, Location: at("a.java", 2, 5)},
				Response:   Response{Location: at("a.java", 2, 5)},
				Location:   at("a.java", 2, 5),
			}),
			want: `duplicate parameter "q"`,
		},
		{
			name: "duplicate field name in type",
			surface: ApiSurface{Types: []Type{{Name: "D", Kind: KindPOJO, Location: at("a.java", 1, 1), Fields: []Field{
				{Name: "a", Type: TypeRef{Name: "String"}, Location: at("a.java", 2, 12)},
				{Name: "a", Type: TypeRef{Name: "String"}, Location: at("a.java", 3, 12)},
			}}}},
			want: `duplicate field "a"`,
		},
		{
			name:    "duplicate rpc name in grpc service",
			surface: ApiSurface{GrpcServices: []GrpcService{{Name: "G", Rpcs: []string{"Do", "Do"}, Location: at("g.proto", 1, 1)}}},
			want:    `duplicate rpc "Do"`,
		},
		{
			name: "duplicate grpc service name",
			surface: ApiSurface{GrpcServices: []GrpcService{
				{Name: "G", Location: at("g.proto", 1, 1)},
				{Name: "G", Location: at("h.proto", 1, 1)},
			}},
			want: `duplicate gRPC service name "G"`,
		},
		{
			name: "duplicate error handler for the same exception",
			surface: ApiSurface{ErrorScheme: ErrorScheme{Handlers: []ErrorHandler{
				{ExceptionType: "E", StatusCode: 400, Location: at("a.java", 1, 5)},
				{ExceptionType: "E", StatusCode: 404, Location: at("a.java", 5, 5)},
			}}},
			want: `duplicate error handler for exception "E"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expectDiagnostics(t, &tc.surface, tc.want)
		})
	}
}

func TestValidateRejectsUnknownVerb(t *testing.T) {
	cases := []struct {
		verb Verb
		want string
	}{
		{Verb("FETCH"), `unknown verb "FETCH"`},
		{Verb(""), `unknown verb ""`},
	}
	for _, tc := range cases {
		m := httpMethod("fetch", at("a.java", 2, 5))
		m.Verb = tc.verb
		s := serviceWith(m)
		expectDiagnostics(t, &s, tc.want)
	}
}

func TestValidateChecksLocations(t *testing.T) {
	cases := []struct {
		name string
		loc  Location
		want string
	}{
		{"missing file", Location{Line: 1, Column: 1}, "Location.File must not be empty"},
		{"half-synthetic line only", Location{File: "a.java", Line: 1, Column: 0}, "must be both zero (synthetic) or both positive"},
		{"half-synthetic column only", Location{File: "a.java", Line: 0, Column: 3}, "must be both zero (synthetic) or both positive"},
		{"negative line", Location{File: "a.java", Line: -2, Column: 1}, "got line -2 column 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := ApiSurface{Services: []Service{{Name: "S", Location: tc.loc}}}
			expectDiagnostics(t, &s, tc.want)
		})
	}
}

func TestValidateAllowsSyntheticLocation(t *testing.T) {
	s := ApiSurface{Services: []Service{{
		Name:     "S",
		Location: Location{File: "synthetic", Line: 0, Column: 0},
	}}}
	if ds := s.Validate(); len(ds) != 0 {
		t.Fatalf("fully synthetic location must be allowed, got %+v", ds)
	}
}

func TestValidatePayloadBodyConsistency(t *testing.T) {
	t.Run("http payload without body parameter", func(t *testing.T) {
		m := httpMethod("create", at("a.java", 2, 5))
		m.Verb = VerbPost
		m.Payload = &TypeRef{Name: "Req"}
		s := serviceWith(m)
		expectDiagnostics(t, &s, `Payload "Req" is set but no in=body parameter declares it`)
	})
	t.Run("body parameter without payload", func(t *testing.T) {
		m := httpMethod("create", at("a.java", 2, 5))
		m.Verb = VerbPost
		m.Params = []Param{{Name: "b", In: ParamInBody, Type: TypeRef{Name: "Req"}, Location: at("a.java", 3, 9)}}
		s := serviceWith(m)
		expectDiagnostics(t, &s, `body parameter "b" is declared but Payload is nil`)
	})
	t.Run("payload does not match body parameter type", func(t *testing.T) {
		m := httpMethod("create", at("a.java", 2, 5))
		m.Verb = VerbPost
		m.Payload = &TypeRef{Name: "Req"}
		m.Params = []Param{{Name: "b", In: ParamInBody, Type: TypeRef{Name: "Other"}, Location: at("a.java", 3, 9)}}
		s := serviceWith(m)
		expectDiagnostics(t, &s, `Payload "Req" does not match the body parameter type "Other"`)
	})
	t.Run("two body parameters", func(t *testing.T) {
		m := httpMethod("create", at("a.java", 2, 5))
		m.Verb = VerbPost
		m.Params = []Param{
			{Name: "a", In: ParamInBody, Type: TypeRef{Name: "Req"}, Location: at("a.java", 3, 9)},
			{Name: "b", In: ParamInBody, Type: TypeRef{Name: "Req"}, Location: at("a.java", 4, 9)},
		}
		s := serviceWith(m)
		expectDiagnostics(t, &s, `2 in=body parameters declared`)
	})
	t.Run("rpc payload without body parameter is fine", func(t *testing.T) {
		m := httpMethod("DoIt", at("a.proto", 2, 3))
		m.Verb = VerbRPC
		m.Payload = &TypeRef{Name: "Req"}
		s := serviceWith(m)
		expectDiagnostics(t, &s)
	})
	t.Run("rpc with body parameter is rejected", func(t *testing.T) {
		m := httpMethod("DoIt", at("a.proto", 2, 3))
		m.Verb = VerbRPC
		m.Params = []Param{{Name: "b", In: ParamInBody, Type: TypeRef{Name: "Req"}, Location: at("a.proto", 3, 9)}}
		s := serviceWith(m)
		expectDiagnostics(t, &s, `on an rpc method`)
	})
}

func TestValidateStatusCodeRange(t *testing.T) {
	for _, bad := range []int{-1, 99, 700} {
		m := httpMethod("get", at("a.java", 2, 5))
		m.Response.StatusCode = bad
		s := serviceWith(m)
		expectDiagnostics(t, &s, `out of range (want 0 (inferred) or 100..599)`)
	}
	for _, good := range []int{0, 100, 201, 599} {
		m := httpMethod("get", at("a.java", 2, 5))
		m.Response.StatusCode = good
		s := serviceWith(m)
		if ds := s.Validate(); len(ds) != 0 {
			t.Fatalf("status code %d must be accepted, got %+v", good, ds)
		}
	}
	h := ErrorHandler{ExceptionType: "E", StatusCode: 700, Location: at("a.java", 3, 5)}
	s := ApiSurface{ErrorScheme: ErrorScheme{Handlers: []ErrorHandler{h}}}
	expectDiagnostics(t, &s, `status code 700 out of range (want 0 (undeclared) or 100..599)`)
}

func TestValidateParamChecks(t *testing.T) {
	m := httpMethod("get", at("a.java", 2, 5))
	m.Params = []Param{{Name: "", In: ParamInQuery, Type: TypeRef{Name: "String"}, Location: at("a.java", 3, 10)}}
	s := serviceWith(m)
	expectDiagnostics(t, &s, `parameter "": Name must not be empty`)

	m = httpMethod("get", at("a.java", 2, 5))
	m.Params = []Param{{Name: "q", In: ParamIn("cookie"), Type: TypeRef{Name: "String"}, Location: at("a.java", 3, 10)}}
	s = serviceWith(m)
	expectDiagnostics(t, &s, `unknown param location "cookie" (want path, query, header or body)`)

	m = httpMethod("get", at("a.java", 2, 5))
	m.Params = []Param{{Name: "q", In: ParamInQuery, Type: TypeRef{}, Location: at("a.java", 3, 10)}}
	s = serviceWith(m)
	expectDiagnostics(t, &s, `parameter "q": Type.Name must not be empty`)
}

func TestValidatePathChecks(t *testing.T) {
	m := httpMethod("get", at("a.java", 2, 5))
	m.Path = "api/x"
	s := serviceWith(m)
	expectDiagnostics(t, &s, `Path "api/x" must be a normalized absolute path starting with "/"`)

	surface := ApiSurface{Services: []Service{{
		Name:     "S",
		BasePath: "api",
		Location: at("a.java", 1, 1),
		Methods:  []Method{httpMethod("get", at("a.java", 2, 5))},
	}}}
	expectDiagnostics(t, &surface, `BasePath "api" must be empty or start with "/"`)
}

func TestValidateTypeChecks(t *testing.T) {
	s := ApiSurface{Types: []Type{{Name: "D", Kind: TypeKind("struct"), Location: at("a.java", 1, 1)}}}
	expectDiagnostics(t, &s, `unknown kind "struct" (want record or pojo)`)

	s = ApiSurface{Types: []Type{{Name: "D", Kind: KindPOJO, Location: at("a.java", 1, 1), Fields: []Field{
		{Name: "a", Type: TypeRef{}, Location: at("a.java", 2, 12)},
	}}}}
	expectDiagnostics(t, &s, `field "a": Type.Name must not be empty`)
}

func TestValidateGrpcAndHandlerChecks(t *testing.T) {
	s := ApiSurface{GrpcServices: []GrpcService{{Name: "G", Rpcs: []string{""}, Location: at("g.proto", 1, 1)}}}
	expectDiagnostics(t, &s, `rpc 0: name must not be empty`)

	s = ApiSurface{ErrorScheme: ErrorScheme{Handlers: []ErrorHandler{
		{StatusCode: 400, Location: at("a.java", 1, 5)},
	}}}
	expectDiagnostics(t, &s, `ExceptionType must not be empty`)
}

func TestValidateAllowsSameTypeNameInDifferentPackages(t *testing.T) {
	s := ApiSurface{Types: []Type{
		{Name: "D", Kind: KindPOJO, Package: "a", Location: at("a.java", 1, 1)},
		{Name: "D", Kind: KindPOJO, Package: "b", Location: at("b.java", 1, 1)},
	}}
	if ds := s.Validate(); len(ds) != 0 {
		t.Fatalf("same type name in different packages must be allowed, got %+v", ds)
	}
}

func TestValidateIsDeterministic(t *testing.T) {
	s := ApiSurface{
		Services: []Service{{Name: "S", Methods: []Method{{Verb: Verb("BAD")}}}},
		Types:    []Type{{Name: "D", Kind: TypeKind("???")}},
	}
	first := s.Validate()
	second := s.Validate()
	if len(first) < 3 {
		t.Fatalf("expected several violations, got %+v", first)
	}
	a, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal diagnostics: %v", err)
	}
	b, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal diagnostics: %v", err)
	}
	if string(a) != string(b) {
		t.Fatalf("Validate is not deterministic:\nfirst:  %s\nsecond: %s", a, b)
	}
}
