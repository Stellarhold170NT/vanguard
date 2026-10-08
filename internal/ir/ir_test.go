package ir

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Every located node of the inventory implements Node — pinned at compile
// time so a new node type cannot forget its Loc().
var (
	_ Node = Service{}
	_ Node = Method{}
	_ Node = Param{}
	_ Node = Response{}
	_ Node = Pagination{}
	_ Node = Field{}
	_ Node = Type{}
	_ Node = ErrorHandler{}
	_ Node = GrpcService{}
)

// TestLocReturnsTheNodeLocation makes sure Loc() returns exactly the node's
// Location field — the engine anchors findings at Loc().
func TestLocReturnsTheNodeLocation(t *testing.T) {
	loc := Location{File: "src/S.java", Line: 7, Column: 3}
	m := Method{OperationName: "get", Location: loc}
	if m.Loc() != loc {
		t.Fatalf("Method.Loc() = %+v, want %+v", m.Loc(), loc)
	}
	s := Service{Name: "S", Location: loc}
	if s.Loc() != loc {
		t.Fatalf("Service.Loc() = %+v, want %+v", s.Loc(), loc)
	}
}

// TestEveryNodeCarriesALocation is the structural guard for UC1: every
// node struct of the inventory has a Location field of type Location. Pure
// containers (ApiSurface, ErrorScheme, Source, TypeRef) are excluded on
// purpose: findings never anchor on them.
func TestEveryNodeCarriesALocation(t *testing.T) {
	nodes := []any{
		Service{}, Method{}, Param{}, Response{}, Pagination{}, Field{},
		Type{}, ErrorHandler{}, GrpcService{}, Diagnostic{},
	}
	want := reflect.TypeOf(Location{})
	for _, n := range nodes {
		ft := reflect.TypeOf(n)
		f, ok := ft.FieldByName("Location")
		if !ok {
			t.Errorf("%s has no Location field (UC1: every node must be locatable)", ft.Name())
			continue
		}
		if f.Type != want {
			t.Errorf("%s.Location is %s, want ir.Location", ft.Name(), f.Type)
		}
	}
}

// TestMethodOnlyExistsInsideService pins the containment invariant: the
// surface has no slot for a standalone Method — a Method can only be
// reached through its Service (the Resource of the REST reading).
func TestMethodOnlyExistsInsideService(t *testing.T) {
	st := reflect.TypeOf(ApiSurface{})
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if strings.Contains(f.Type.String(), "Method") {
			t.Errorf("ApiSurface.%s has type %s: methods must only live inside Service", f.Name, f.Type)
		}
	}
}

// TestVerbConstantsAreDistinct guards the §5.2 verb enumeration against
// accidental duplicates or empty values, and pins the Valid/IsHTTP helpers
// the R2xx selectors rely on.
func TestVerbConstantsAreDistinct(t *testing.T) {
	verbs := []Verb{VerbGet, VerbPost, VerbPut, VerbPatch, VerbDelete, VerbRPC}
	seen := make(map[Verb]bool, len(verbs))
	for _, v := range verbs {
		if v == "" {
			t.Fatal("verb constant must not be empty")
		}
		if seen[v] {
			t.Fatalf("duplicate verb %q", v)
		}
		seen[v] = true
		if !v.Valid() {
			t.Fatalf("verb %q must report Valid()", v)
		}
	}
	if VerbRPC.IsHTTP() {
		t.Fatal("rpc must not report IsHTTP()")
	}
	for _, v := range []Verb{VerbGet, VerbPost, VerbPut, VerbPatch, VerbDelete} {
		if !v.IsHTTP() {
			t.Fatalf("verb %q must report IsHTTP()", v)
		}
	}
	if Verb("FETCH").Valid() {
		t.Fatal("unknown verb must not report Valid()")
	}
}

// TestPaginationStylesAreDistinct guards the §5.2 pagination enumeration.
func TestPaginationStylesAreDistinct(t *testing.T) {
	styles := []PaginationStyle{PaginationPageable, PaginationParams, PaginationNone}
	seen := make(map[PaginationStyle]bool, len(styles))
	for _, s := range styles {
		if s == "" {
			t.Fatal("pagination style constant must not be empty")
		}
		if seen[s] {
			t.Fatalf("duplicate pagination style %q", s)
		}
		seen[s] = true
		if !s.Valid() {
			t.Fatalf("pagination style %q must report Valid()", s)
		}
	}
	if PaginationStyle("cursor").Valid() {
		t.Fatal("unknown style must not report Valid()")
	}
}

// TestApiSurfaceMarshalIsStable keeps the golden-snapshot prerequisite true
// from day one (charter §5.2, w4-03 determinism): marshalling twice yields
// identical bytes, and the IR needs nothing beyond encoding/json — proof it
// stays pure Go.
func TestApiSurfaceMarshalIsStable(t *testing.T) {
	surface := ApiSurface{
		Source: Source{Lang: "java", Framework: "spring-boot", FrameworkVersion: "4.0.2"},
		Services: []Service{{
			Name:     "YouthResource",
			BasePath: "/api/military-youth",
			Methods: []Method{{
				OperationName: "autoComplete",
				Verb:          VerbGet,
				Path:          "/api/military-youth/auto-complete",
				Pagination:    Pagination{Style: PaginationNone, Location: Location{File: "src/X.java", Line: 1, Column: 2}},
				Response:      Response{Location: Location{File: "src/X.java", Line: 1, Column: 2}},
				Location:      Location{File: "src/X.java", Line: 1, Column: 2},
			}},
			Location: Location{File: "src/X.java", Line: 1, Column: 2},
		}},
	}
	first, err := json.Marshal(surface)
	if err != nil {
		t.Fatalf("marshal ApiSurface: %v", err)
	}
	second, err := json.Marshal(surface)
	if err != nil {
		t.Fatalf("marshal ApiSurface again: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("ApiSurface JSON marshal is not deterministic")
	}
}
